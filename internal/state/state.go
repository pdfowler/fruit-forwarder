package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"
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

const maxAppliedCommands = 1000
const maxCommandIDLength = 256

type State struct {
	AppliedCommandIDs []string `json:"applied_command_ids"`
}

func Load(path string) (*State, error) {
	if err := validatePrivateDirectory(filepath.Dir(path)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("state directory: %w", err)
	}
	if err := validateExistingPath(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("state file: %w", err)
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &State{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read state: %w", err)
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
		return
	}
	s.AppliedCommandIDs = append(s.AppliedCommandIDs, id)
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
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace state: %w", err)
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
