package startup

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBeginDetectsPreviousUncleanRunAndMarkClean(t *testing.T) {
	path := filepath.Join(t.TempDir(), "startup-session.json")
	store := NewFileStore(path)
	firstStart := time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC)
	first, err := store.Begin(Options{AppName: "NexusDesk", Version: "1.0.0", Commit: "abc", Now: firstStart, PID: 42})
	if err != nil {
		t.Fatalf("first Begin returned error: %v", err)
	}
	if first.PreviousUnclean {
		t.Fatalf("first run should not report unclean previous status: %#v", first)
	}

	second, err := store.Begin(Options{AppName: "NexusDesk", Version: "1.0.1", Commit: "def", Now: firstStart.Add(time.Hour), PID: 43})
	if err != nil {
		t.Fatalf("second Begin returned error: %v", err)
	}
	if !second.PreviousUnclean || !strings.Contains(second.Message, "did not record a clean exit") {
		t.Fatalf("expected unclean previous run warning: %#v", second)
	}
	if second.Previous.PID != 42 || second.Previous.Commit != "abc" {
		t.Fatalf("expected previous session metadata, got %#v", second.Previous)
	}
	if err := store.MarkClean(second.CurrentID, firstStart.Add(2*time.Hour)); err != nil {
		t.Fatalf("MarkClean returned error: %v", err)
	}
	third, err := store.Begin(Options{Now: firstStart.Add(3 * time.Hour), PID: 44})
	if err != nil {
		t.Fatalf("third Begin returned error: %v", err)
	}
	if third.PreviousUnclean {
		t.Fatalf("cleanly closed previous run should not warn: %#v", third)
	}
}

func TestMarkCleanIgnoresMismatchedSessionID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "startup-session.json")
	store := NewFileStore(path)
	started := time.Date(2026, 5, 28, 10, 0, 0, 0, time.UTC)
	status, err := store.Begin(Options{Now: started, PID: 42})
	if err != nil {
		t.Fatalf("Begin returned error: %v", err)
	}
	if err := store.MarkClean("different", started.Add(time.Minute)); err != nil {
		t.Fatalf("MarkClean returned error: %v", err)
	}
	next, err := store.Begin(Options{Now: started.Add(time.Hour), PID: 43})
	if err != nil {
		t.Fatalf("second Begin returned error: %v", err)
	}
	if !next.PreviousUnclean || next.Previous.ID != status.CurrentID {
		t.Fatalf("mismatched clean marker should not close prior session: %#v", next)
	}
}

func TestBeginCreatesParentDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "startup-session.json")
	store := NewFileStore(path)
	if _, err := store.Begin(Options{}); err != nil {
		t.Fatalf("Begin returned error: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected marker file to exist: %v", err)
	}
}
