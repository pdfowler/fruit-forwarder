package reminderstore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/pdfowler/icloud-reminders-bridge/internal/config"
	"github.com/pdfowler/icloud-reminders-bridge/internal/model"
)

type request struct {
	CalendarIDs []string   `json:"calendar_ids,omitempty"`
	Start       string     `json:"start,omitempty"`
	End         string     `json:"end,omitempty"`
	Action      string     `json:"action"`
	ListIDs     []string   `json:"list_ids,omitempty"`
	ListID      string     `json:"list_id,omitempty"`
	Item        model.Item `json:"item,omitempty"`
}

type response struct {
	Calendars []model.Calendar `json:"calendars,omitempty"`
	Lists     []model.List     `json:"lists,omitempty"`
	Item      *model.Item      `json:"item,omitempty"`
}

type Runner interface {
	Run(context.Context, request) (response, error)
}

type commandRunner struct {
	helperPath string
}

const (
	maxHelperOutputBytes = 8 << 20
	maxHelperErrorBytes  = 16 << 10
)

type boundedBuffer struct {
	bytes.Buffer
	limit    int
	exceeded bool
}

func (b *boundedBuffer) Write(data []byte) (int, error) {
	if b.Len()+len(data) > b.limit {
		remaining := b.limit - b.Len()
		if remaining > 0 {
			_, _ = b.Buffer.Write(data[:remaining])
		}
		b.exceeded = true
		return len(data), errors.New("output limit exceeded")
	}
	return b.Buffer.Write(data)
}

func (r *commandRunner) Run(ctx context.Context, input request) (response, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	payload, err := json.Marshal(input)
	if err != nil {
		return response{}, fmt.Errorf("encode EventKit request: %w", err)
	}
	cmd := exec.CommandContext(ctx, r.helperPath)
	cmd.Stdin = bytes.NewReader(payload)
	stdout := &boundedBuffer{limit: maxHelperOutputBytes}
	stderr := &boundedBuffer{limit: maxHelperErrorBytes}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	runErr := cmd.Run()
	// Treat a buffer that reaches its ceiling as truncated as well. Some
	// platform pipe implementations stop copying after the writer reports its
	// error, so the flag alone is not sufficient to prove the full output was
	// observed.
	if stdout.exceeded || stderr.exceeded || stdout.Len() >= maxHelperOutputBytes || stderr.Len() >= maxHelperErrorBytes {
		return response{}, fmt.Errorf("EventKit helper output exceeds the supported limit")
	}
	if runErr != nil {
		if ctx.Err() != nil {
			return response{}, fmt.Errorf("EventKit request did not finish; check macOS Reminders access: %w", ctx.Err())
		}
		detail := strings.TrimSpace(stderr.String())
		if len(detail) > 512 {
			detail = detail[:512]
		}
		if detail == "" {
			detail = runErr.Error()
		}
		return response{}, fmt.Errorf("EventKit helper failed: %s", detail)
	}
	var output response
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		return response{}, fmt.Errorf("decode EventKit response: %w", err)
	}
	return output, nil
}

type Store struct {
	calendars          []config.List
	runner             Runner
	allowed            map[string]config.List
	completedRetention time.Duration
}

func New(cfg *config.Config) (*Store, error) {
	helperPath := cfg.EventKitPath()
	if err := validateExecutable(helperPath); err != nil {
		return nil, fmt.Errorf("EventKit helper: %w", err)
	}
	// Start MCP even if EventKit is unavailable; each tool reports its own
	// bounded error rather than preventing the protocol handshake.
	return NewWithRunner(cfg, &commandRunner{helperPath: helperPath}), nil
}

func NewWithRunner(cfg *config.Config, runner Runner) *Store {
	return &Store{runner: runner, allowed: cfg.AllowedIDs(), completedRetention: cfg.CompletedRetentionDuration(), calendars: append([]config.List(nil), cfg.Calendars...)}
}

func Discover(ctx context.Context, helperPath string) ([]model.List, error) {
	if err := validateExecutable(helperPath); err != nil {
		return nil, fmt.Errorf("EventKit helper: %w", err)
	}
	output, err := (&commandRunner{helperPath: helperPath}).Run(ctx, request{Action: "lists"})
	if err != nil {
		return nil, err
	}
	return output.Lists, nil
}

func validateExecutable(path string) error {
	if !filepath.IsAbs(path) {
		return errors.New("path must be absolute")
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return errors.New("path must be a regular executable file")
	}
	if info.Mode().Perm()&0o022 != 0 {
		return errors.New("executable must not be group- or world-writable")
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); ok && stat.Uid != uint32(os.Getuid()) {
		return errors.New("executable must be owned by the current user")
	}
	return nil
}

// ValidateHelperPath applies the same ownership and permission checks used
// before any EventKit request. It is exposed for non-invasive diagnostics.
func ValidateHelperPath(path string) error {
	return validateExecutable(path)
}

func (s *Store) ValidateLists(ctx context.Context) error {
	output, err := s.runner.Run(ctx, request{Action: "lists"})
	if err != nil {
		return fmt.Errorf("list EventKit reminder lists: %w", err)
	}
	byID := make(map[string]model.List, len(output.Lists))
	for _, list := range output.Lists {
		byID[list.ID] = list
	}
	for id, configured := range s.allowed {
		actual, ok := byID[id]
		if !ok {
			return fmt.Errorf("allowlisted reminder list %q (%s) is not available", configured.Name, id)
		}
		if actual.Name != configured.Name {
			return fmt.Errorf("allowlisted reminder list %s changed name from %q to %q", id, configured.Name, actual.Name)
		}
		if actual.ReadOnly {
			return fmt.Errorf("allowlisted reminder list %q is read-only", actual.Name)
		}
	}
	return nil
}

func (s *Store) Lists(ctx context.Context) ([]model.List, error) {
	output, err := s.runner.Run(ctx, request{Action: "lists"})
	if err != nil {
		return nil, fmt.Errorf("list EventKit reminder lists: %w", err)
	}
	result := make([]model.List, 0, len(s.allowed))
	for _, list := range output.Lists {
		if _, ok := s.allowed[list.ID]; ok {
			list.Items = nil
			result = append(result, list)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func (s *Store) Snapshot(ctx context.Context) ([]model.List, error) {
	if len(s.allowed) == 0 {
		return []model.List{}, nil
	}
	ids := make([]string, 0, len(s.allowed))
	for id := range s.allowed {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	output, err := s.runner.Run(ctx, request{Action: "snapshot", ListIDs: ids})
	if err != nil {
		return nil, fmt.Errorf("read EventKit reminder lists: %w", err)
	}
	seen := make(map[string]struct{}, len(output.Lists))
	for i := range output.Lists {
		configured, ok := s.allowed[output.Lists[i].ID]
		if !ok {
			return nil, fmt.Errorf("EventKit returned unexpected list %q", output.Lists[i].ID)
		}
		if output.Lists[i].Name != configured.Name {
			return nil, fmt.Errorf("allowlisted reminder list %s changed name from %q to %q", output.Lists[i].ID, configured.Name, output.Lists[i].Name)
		}
		// EventKit can return legacy reminders with no title. Home Assistant's
		// todo model requires a non-empty summary, so omit records that cannot be
		// represented instead of rejecting the entire list snapshot.
		items := output.Lists[i].Items[:0]
		for _, item := range output.Lists[i].Items {
			if strings.TrimSpace(item.Summary) == "" {
				continue
			}
			if item.Status == "completed" {
				completedAt, err := time.Parse(time.RFC3339, item.Completed)
				cutoff := time.Now().UTC().Add(-s.completedRetention)
				if err != nil || completedAt.Before(cutoff) {
					continue
				}
			}
			items = append(items, item)
		}
		output.Lists[i].Items = items
		seen[output.Lists[i].ID] = struct{}{}
		sort.Slice(output.Lists[i].Items, func(a, b int) bool {
			return output.Lists[i].Items[a].Summary < output.Lists[i].Items[b].Summary
		})
	}
	if len(seen) != len(s.allowed) {
		return nil, errors.New("EventKit snapshot omitted an allowlisted list")
	}
	return output.Lists, nil
}

func (s *Store) Create(ctx context.Context, listID string, item model.Item) (*model.Item, error) {
	if err := s.checkWrite(listID, item, false); err != nil {
		return nil, err
	}
	return s.write(ctx, request{Action: "create", ListID: listID, Item: item})
}

func (s *Store) Update(ctx context.Context, listID string, item model.Item) (*model.Item, error) {
	if err := s.checkWrite(listID, item, true); err != nil {
		return nil, err
	}
	return s.write(ctx, request{Action: "update", ListID: listID, Item: item})
}

func (s *Store) SetCompleted(ctx context.Context, listID, uid string, completed bool) (*model.Item, error) {
	if _, ok := s.allowed[listID]; !ok {
		return nil, fmt.Errorf("list %q is outside the allowlist", listID)
	}
	if strings.TrimSpace(uid) == "" {
		return nil, errors.New("reminder uid is required")
	}
	action := "reopen"
	if completed {
		action = "complete"
	}
	return s.write(ctx, request{Action: action, ListID: listID, Item: model.Item{UID: uid}})
}

func (s *Store) checkWrite(listID string, item model.Item, requireUID bool) error {
	if _, ok := s.allowed[listID]; !ok {
		return fmt.Errorf("list %q is outside the allowlist", listID)
	}
	if requireUID && strings.TrimSpace(item.UID) == "" {
		return errors.New("reminder uid is required")
	}
	if strings.TrimSpace(item.Summary) == "" {
		return errors.New("summary is required")
	}
	if item.Status != "" && item.Status != "needs_action" && item.Status != "completed" {
		return fmt.Errorf("unsupported reminder status %q", item.Status)
	}
	return nil
}

func (s *Store) write(ctx context.Context, input request) (*model.Item, error) {
	output, err := s.runner.Run(ctx, input)
	if err != nil {
		return nil, err
	}
	if output.Item == nil || strings.TrimSpace(output.Item.UID) == "" {
		return nil, errors.New("EventKit helper returned no reminder")
	}
	return output.Item, nil
}

func (s *Store) ApplyCommand(ctx context.Context, command model.Command) error {
	switch command.Action {
	case "create":
		_, err := s.Create(ctx, command.ListID, command.Item)
		return err
	case "update":
		_, err := s.Update(ctx, command.ListID, command.Item)
		return err
	case "complete":
		_, err := s.SetCompleted(ctx, command.ListID, command.Item.UID, true)
		return err
	case "reopen":
		_, err := s.SetCompleted(ctx, command.ListID, command.Item.UID, false)
		return err
	default:
		return fmt.Errorf("unsupported command action %q", command.Action)
	}
}
