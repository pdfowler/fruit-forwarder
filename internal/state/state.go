package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// Acquire prevents two bridge processes from applying the same HA command at
// once. The returned function releases the advisory lock.
func Acquire(path string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create lock directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open state lock: %w", err)
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

type State struct {
	AppliedCommandIDs []string `json:"applied_command_ids"`
}

func Load(path string) (*State, error) {
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
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create state directory: %w", err)
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
