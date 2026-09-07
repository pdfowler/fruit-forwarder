package hasync

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pdfowler/icloud-reminders-bridge/internal/config"
	"github.com/pdfowler/icloud-reminders-bridge/internal/model"
	"github.com/pdfowler/icloud-reminders-bridge/internal/state"
)

const maxResponseBytes = 1 << 20
const maxCommands = 1000
const maxCommandString = 256
const maxRetryBackoff = 15 * time.Minute

type ReminderStore interface {
	Snapshot(context.Context) ([]model.List, error)
	ApplyCommand(context.Context, model.Command) error
}

type Client struct {
	cfg        *config.Config
	version    string
	token      string
	store      ReminderStore
	state      *state.State
	httpClient *http.Client
	logger     *slog.Logger
	random     *rand.Rand
}

func New(cfg *config.Config, token string, store ReminderStore, bridgeState *state.State, logger *slog.Logger) *Client {
	return NewWithVersion(cfg, token, store, bridgeState, logger, "dev")
}

func NewWithVersion(cfg *config.Config, token string, store ReminderStore, bridgeState *state.State, logger *slog.Logger, version string) *Client {
	httpClient := &http.Client{
		Timeout: 20 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return &Client{
		cfg: cfg, version: version, token: token, store: store, state: bridgeState,
		httpClient: httpClient, logger: logger,
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (c *Client) Run(ctx context.Context) error {
	failures := 0
	for {
		if _, err := c.SyncOnce(ctx); err != nil {
			failures++
			c.logger.Error("Home Assistant sync failed", "error", err, "consecutive_failures", failures)
		} else {
			failures = 0
		}
		delay := retryDelay(c.cfg.Interval(), failures, c.random.Float64())
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil
		case <-timer.C:
		}
	}
}

// retryDelay backs off only after a failed sync. Jitter prevents a group of
// bridges recovering at the same time from synchronizing in lockstep, while
// the cap ensures a failure cannot make recovery effectively indefinite.
func retryDelay(base time.Duration, consecutiveFailures int, jitter float64) time.Duration {
	if consecutiveFailures <= 0 {
		return base
	}
	if jitter < 0 {
		jitter = 0
	}
	if jitter > 1 {
		jitter = 1
	}
	delay := base
	for i := 0; i < consecutiveFailures; i++ {
		if delay >= maxRetryBackoff/2 {
			delay = maxRetryBackoff
			break
		}
		delay *= 2
	}
	if delay > maxRetryBackoff {
		delay = maxRetryBackoff
	}
	// Keep jitter within +/-10% of the exponential delay and never make a
	// retry faster than the configured normal interval.
	factor := 0.9 + (0.2 * jitter)
	result := time.Duration(float64(delay) * factor)
	if result < base {
		return base
	}
	if result > maxRetryBackoff {
		return maxRetryBackoff
	}
	return result
}

func (c *Client) SyncOnce(ctx context.Context) (int, error) {
	if c.state.InFlight != nil {
		return 0, fmt.Errorf("command %s has an uncertain outcome; resolve it before syncing again", c.state.InFlight.ID)
	}
	lists, err := c.store.Snapshot(ctx)
	if err != nil {
		return 0, err
	}
	response, err := c.postSnapshot(ctx, lists)
	if err != nil {
		return 0, err
	}
	applied := 0
	for _, command := range response.Commands {
		if c.state.Has(command.ID) {
			continue
		}
		c.state.InFlight = &command
		if err := c.state.Save(c.cfg.StatePath); err != nil {
			return applied, fmt.Errorf("persist in-flight command %s: %w", command.ID, err)
		}
		if err := c.store.ApplyCommand(ctx, command); err != nil {
			c.logger.Error("Home Assistant command failed", "command_id", command.ID, "action", command.Action, "error", err)
			return applied, fmt.Errorf("command %s outcome is uncertain: %w", command.ID, err)
		}
		c.state.MarkApplied(command.ID)
		if err := c.state.Save(c.cfg.StatePath); err != nil {
			return applied, fmt.Errorf("persist applied command %s: %w", command.ID, err)
		}
		applied++
	}
	if applied > 0 {
		// Immediately publish the EventKit-confirmed state and acknowledgements so
		// HA does not need to wait for the next polling interval.
		lists, err = c.store.Snapshot(ctx)
		if err != nil {
			return applied, err
		}
		if _, err := c.postSnapshot(ctx, lists); err != nil {
			return applied, err
		}
	}
	c.logger.Info("Home Assistant sync complete", "lists", len(lists), "commands_applied", applied)
	return applied, nil
}

func (c *Client) postSnapshot(ctx context.Context, lists []model.List) (*model.SyncResponse, error) {
	// A nil calendar slice is intentional: omitting calendars tells the HA
	// runtime to retain its last known calendar snapshot when a calendar read
	// fails. Reminder publication must not be blocked by an independent
	// Calendar permission or EventKit failure.
	var calendars []model.Calendar
	if len(c.cfg.Calendars) > 0 {
		reader, ok := c.store.(interface {
			CalendarEvents(context.Context, string, string, string) ([]model.Event, error)
		})
		if !ok {
			c.logger.Warn("calendar sync unavailable; publishing reminders without replacing the cached calendar snapshot")
		} else {
			now := time.Now().UTC()
			start, end := now.Add(-30*24*time.Hour).Format(time.RFC3339), now.Add(90*24*time.Hour).Format(time.RFC3339)
			candidate := make([]model.Calendar, 0, len(c.cfg.Calendars))
			calendarErr := false
			for _, calendar := range c.cfg.Calendars {
				events, err := reader.CalendarEvents(ctx, calendar.ID, start, end)
				if err != nil {
					calendarErr = true
					break
				}
				candidate = append(candidate, model.Calendar{ID: calendar.ID, Name: calendar.Name, Events: events, WindowStart: start, WindowEnd: end})
			}
			if calendarErr {
				c.logger.Warn("calendar sync failed; publishing reminders without replacing the cached calendar snapshot")
			} else {
				calendars = candidate
			}
		}
	}
	payload := model.Snapshot{
		Calendars:         calendars,
		Version:           model.ProtocolVersion,
		BridgeID:          c.cfg.BridgeID,
		SentAt:            time.Now().UTC(),
		Lists:             lists,
		AppliedCommandIDs: append([]string(nil), c.state.AppliedCommandIDs...),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode snapshot: %w", err)
	}
	endpoint := strings.TrimRight(c.cfg.HomeAssistantURL, "/") + "/api/webhook/" + c.token
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create sync request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("fruit-forwarder/%s", c.version))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// net/http wraps failures with the request URL, which contains our token.
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return nil, fmt.Errorf("post snapshot: %w", err)
	}
	defer resp.Body.Close()
	limited := io.LimitReader(resp.Body, maxResponseBytes+1)
	responseBody, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("read sync response: %w", err)
	}
	if len(responseBody) > maxResponseBytes {
		return nil, errors.New("sync response exceeds 1 MiB")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Home Assistant returned HTTP %d", resp.StatusCode)
	}
	var result model.SyncResponse
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("decode sync response: %w", err)
	}
	if result.Version != model.ProtocolVersion {
		return nil, fmt.Errorf("unsupported sync protocol version %d", result.Version)
	}
	if len(result.Commands) > maxCommands {
		return nil, fmt.Errorf("sync response contains too many commands")
	}
	for _, command := range result.Commands {
		if c.state.Has(command.ID) {
			continue
		}
		if err := validateCommand(command); err != nil {
			return nil, fmt.Errorf("invalid Home Assistant command: %w", err)
		}
	}
	return &result, nil
}

func validateCommand(command model.Command) error {
	if !boundedCommandString(command.ID) || !boundedCommandString(command.ListID) {
		return errors.New("command id and list_id must be non-empty and bounded")
	}
	switch command.Action {
	case "create":
		if strings.TrimSpace(command.Item.Summary) == "" {
			return errors.New("create command requires a summary")
		}
	case "update":
		if !boundedCommandString(command.Item.UID) || strings.TrimSpace(command.Item.Summary) == "" {
			return errors.New("update command requires a uid and summary")
		}
	case "complete", "reopen":
		if !boundedCommandString(command.Item.UID) {
			return errors.New("completion command requires a uid")
		}
	default:
		return fmt.Errorf("unsupported action %q", command.Action)
	}
	if len(command.Item.Summary) > maxCommandString || len(command.Item.Description) > 4096 || len(command.Item.Due) > maxCommandString {
		return errors.New("command item fields exceed supported limits")
	}
	return nil
}

func boundedCommandString(value string) bool {
	return strings.TrimSpace(value) != "" && len(value) <= maxCommandString
}
