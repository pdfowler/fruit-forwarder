package state

import (
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
