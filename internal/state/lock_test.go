package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAcquireRejectsConcurrentProcessLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.lock")
	release, err := Acquire(path)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if second, err := Acquire(path); err == nil {
		second()
		t.Fatal("second lock unexpectedly succeeded")
	}
}

func TestAcquireReleasesLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.lock")
	release, err := Acquire(path)
	if err != nil {
		t.Fatal(err)
	}
	release()
	second, err := Acquire(path)
	if err != nil {
		t.Fatal(err)
	}
	second()
}

func TestLockRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "lock")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if release, err := Acquire(link); err == nil {
		release()
		t.Fatal("symlinked lock accepted")
	}
}
