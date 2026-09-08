package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"

	"github.com/pdfowler/fruit-forwarder/internal/model"
)

// Acquire prevents two bridge processes from applying the same HA command at
// once. The returned function releases the advisory lock.
func Acquire(path string) (func(), error) {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, fmt.Errorf("create lock directory: %w", err)
	}
	if err := validatePrivateDirectory(directory); err != nil {
		return nil, fmt.Errorf("lock directory: %w", err)
	}
	fd, err := unix.Open(path, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open state lock: %w", err)
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = unix.Close(fd)
		return nil, errors.New("open state lock: invalid file descriptor")
	}
	if err := validateExistingPath(path); err != nil {
		_ = file.Close()
		return nil, fmt.Errorf("state lock: %w", err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		file.Close()
		return nil, fmt.Errorf("another bridge process owns the state lock: %w", err)
	}
	return func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
	}, nil
}

// Probe reports whether the advisory state lock is currently available without
// creating the lock file or changing its contents. A busy lock is a normal
// result while the bridge is serving; callers should treat it as a diagnostic
// signal rather than as a failure by itself.
func Probe(path string) (string, error) {
	directory := filepath.Dir(path)
	if err := validatePrivateDirectory(directory); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "available", nil
		}
		return "", fmt.Errorf("lock directory: %w", err)
	}
	fd, err := unix.Open(path, unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if errors.Is(err, os.ErrNotExist) {
		return "available", nil
	}
	if err != nil {
		return "", fmt.Errorf("open state lock: %w", err)
	}
	file := os.NewFile(uintptr(fd), path)
	if file == nil {
		_ = unix.Close(fd)
		return "", errors.New("open state lock: invalid file descriptor")
	}
	defer file.Close()
	if err := validateExistingPath(path); err != nil {
		return "", fmt.Errorf("state lock: %w", err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return "busy", nil
		}
		return "", fmt.Errorf("probe state lock: %w", err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_UN); err != nil {
		return "", fmt.Errorf("release probed state lock: %w", err)
	}
	return "available", nil
}

const maxAppliedCommands = 1000
const maxCommandIDLength = 256
const maxQueueEpochLength = 128
const maxStateBytes = 512 * 1024

type State struct {
	AppliedCommandIDs []string       `json:"applied_command_ids"`
	InFlight          *model.Command `json:"in_flight,omitempty"`
	QueueEpoch        string         `json:"queue_epoch,omitempty"`
}

func Load(path string) (*State, error) {
	if err := validatePrivateDirectory(filepath.Dir(path)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("state directory: %w", err)
	}
	if err := validateExistingPath(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("state file: %w", err)
	}
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return &State{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read state: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxStateBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read state: %w", err)
	}
	if len(data) > maxStateBytes {
		return nil, fmt.Errorf("state exceeds %d bytes", maxStateBytes)
	}
	var result State
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("parse state: %w", err)
	}
	if err := validate(&result); err != nil {
		return nil, fmt.Errorf("validate state: %w", err)
	}
	return &result, nil
}

func (s *State) Has(id string) bool {
	for _, existing := range s.AppliedCommandIDs {
		if existing == id {
			return true
		}
	}
	return false
}

func (s *State) MarkApplied(id string) {
	if s.Has(id) {
		if s.InFlight != nil && s.InFlight.ID == id {
			s.InFlight = nil
		}
		return
	}
	s.AppliedCommandIDs = append(s.AppliedCommandIDs, id)
	if s.InFlight != nil && s.InFlight.ID == id {
		s.InFlight = nil
	}
	if len(s.AppliedCommandIDs) > maxAppliedCommands {
		s.AppliedCommandIDs = append([]string(nil), s.AppliedCommandIDs[len(s.AppliedCommandIDs)-maxAppliedCommands:]...)
	}
}

func (s *State) Save(path string) error {
	if err := validate(s); err != nil {
		return fmt.Errorf("validate state: %w", err)
	}
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	if err := validatePrivateDirectory(directory); err != nil {
		return fmt.Errorf("state directory: %w", err)
	}
	if err := validateExistingPath(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("state file: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".state-*.json")
	if err != nil {
		return fmt.Errorf("create state temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync state temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace state: %w", err)
	}
	directoryFile, err := os.Open(directory)
	if err != nil {
		return fmt.Errorf("open state directory for sync: %w", err)
	}
	defer directoryFile.Close()
	if err := directoryFile.Sync(); err != nil {
		return fmt.Errorf("sync state directory: %w", err)
	}
	return nil
}

func validate(s *State) error {
	if s == nil {
		return errors.New("state is nil")
	}
	if len(s.AppliedCommandIDs) > maxAppliedCommands {
		return fmt.Errorf("applied command ledger exceeds %d entries", maxAppliedCommands)
	}
	if s.QueueEpoch != "" && (strings.TrimSpace(s.QueueEpoch) == "" || len(s.QueueEpoch) > maxQueueEpochLength) {
		return fmt.Errorf("queue epoch must be non-empty and at most %d characters", maxQueueEpochLength)
	}
	seen := make(map[string]struct{}, len(s.AppliedCommandIDs))
	for _, id := range s.AppliedCommandIDs {
		if strings.TrimSpace(id) == "" || len(id) > maxCommandIDLength {
			return errors.New("applied command identifiers must be non-empty and bounded")
		}
		if _, exists := seen[id]; exists {
			return fmt.Errorf("duplicate applied command identifier %q", id)
		}
		seen[id] = struct{}{}
	}
	if s.InFlight != nil {
		if err := validateInFlight(s.InFlight); err != nil {
			return err
		}
		if _, exists := seen[s.InFlight.ID]; exists {
			return errors.New("in-flight command is already acknowledged")
		}
	}
	return nil
}

func validateInFlight(command *model.Command) error {
	if command == nil || strings.TrimSpace(command.ID) == "" || len(command.ID) > maxCommandIDLength {
		return errors.New("in-flight command identifier must be non-empty and bounded")
	}
	if strings.TrimSpace(command.ListID) == "" || len(command.ListID) > maxCommandIDLength {
		return errors.New("in-flight command list identifier must be non-empty and bounded")
	}
	switch command.Action {
	case "create":
		if strings.TrimSpace(command.Item.Summary) == "" {
			return errors.New("in-flight create command requires a summary")
		}
	case "update":
		if strings.TrimSpace(command.Item.UID) == "" || strings.TrimSpace(command.Item.Summary) == "" {
			return errors.New("in-flight update command requires a uid and summary")
		}
	case "complete", "reopen":
		if strings.TrimSpace(command.Item.UID) == "" {
			return errors.New("in-flight completion command requires a uid")
		}
	default:
		return fmt.Errorf("in-flight command has unsupported action %q", command.Action)
	}
	if len(command.Item.Summary) > maxCommandIDLength || len(command.Item.Description) > 4096 || len(command.Item.Due) > maxCommandIDLength {
		return errors.New("in-flight command item fields exceed supported limits")
	}
	return nil
}

func validateExistingPath(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("must be a regular file; symlinks are not accepted")
	}
	if info.Mode().Perm()&0o022 != 0 {
		return errors.New("must not be group- or world-writable")
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); ok && stat.Uid != uint32(os.Getuid()) {
		return errors.New("must be owned by the current user")
	}
	return nil
}

func validatePrivateDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode().Perm()&0o022 != 0 {
		return errors.New("must be an owner-writable private directory")
	}
	if stat, ok := info.Sys().(*syscall.Stat_t); ok && stat.Uid != uint32(os.Getuid()) {
		return errors.New("must be owned by the current user")
	}
	return nil
}
