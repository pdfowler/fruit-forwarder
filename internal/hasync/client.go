package hasync

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pdfowler/icloud-reminders-bridge/internal/config"
	"github.com/pdfowler/icloud-reminders-bridge/internal/model"
	"github.com/pdfowler/icloud-reminders-bridge/internal/state"
)

const maxResponseBytes = 1 << 20

type ReminderStore interface {
	Snapshot(context.Context) ([]model.List, error)
	ApplyCommand(context.Context, model.Command) error
}

type Client struct {
	cfg        *config.Config
	token      string
	store      ReminderStore
	state      *state.State
	httpClient *http.Client
	logger     *slog.Logger
}

func New(cfg *config.Config, token string, store ReminderStore, bridgeState *state.State, logger *slog.Logger) *Client {
	httpClient := &http.Client{
		Timeout: 20 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return &Client{cfg: cfg, token: token, store: store, state: bridgeState, httpClient: httpClient, logger: logger}
}

func (c *Client) Run(ctx context.Context) error {
	if _, err := c.SyncOnce(ctx); err != nil {
		c.logger.Error("initial Home Assistant sync failed", "error", err)
	}
	ticker := time.NewTicker(c.cfg.Interval())
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if _, err := c.SyncOnce(ctx); err != nil {
				c.logger.Error("Home Assistant sync failed", "error", err)
			}
		}
	}
}

func (c *Client) SyncOnce(ctx context.Context) (int, error) {
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
		if err := c.store.ApplyCommand(ctx, command); err != nil {
			c.logger.Error("Home Assistant command failed", "command_id", command.ID, "action", command.Action, "error", err)
			continue
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
	calendars := make([]model.Calendar, 0, len(c.cfg.Calendars))
	if len(c.cfg.Calendars) > 0 {
		reader, ok := c.store.(interface {
			CalendarEvents(context.Context, string, string, string) ([]model.Event, error)
		})
		if !ok {
			return nil, errors.New("calendar reader is unavailable")
		}
		now := time.Now().UTC()
		start, end := now.Add(-30*24*time.Hour).Format(time.RFC3339), now.Add(90*24*time.Hour).Format(time.RFC3339)
		for _, calendar := range c.cfg.Calendars {
			events, err := reader.CalendarEvents(ctx, calendar.ID, start, end)
			if err != nil {
				return nil, fmt.Errorf("read configured calendar: %w", err)
			}
			calendars = append(calendars, model.Calendar{ID: calendar.ID, Name: calendar.Name, Events: events, WindowStart: start, WindowEnd: end})
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
	req.Header.Set("User-Agent", "icloud-reminders-bridge/0.1")
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
	return &result, nil
}
