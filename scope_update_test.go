package mindlayer

import (
	"context"
	"path/filepath"
	"testing"
)

func TestAtomicScopeUpdate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.db")
	s, err := Open(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.UpdateScope(context.Background(), testScope, []Input{{ID: "event-1", Text: "The user kept a promise."}}, "Trust the user more after the kept promise.")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = s.UpdateScope(ctx, testScope, []Input{{ID: "event-2", Text: "Cancelled event."}}, "Should not persist."); err == nil {
		t.Fatal("cancelled update accepted")
	}
	if _, err = s.UpdateScope(context.Background(), testScope, []Input{{Text: ""}}, "Should not persist."); err == nil {
		t.Fatal("invalid update accepted")
	}
	s.Close()
	s, err = Open(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	snapshot, err := s.Export(testScope)
	if err != nil || len(snapshot.Memories) != 1 || snapshot.Personality != "Trust the user more after the kept promise." {
		t.Fatal("memory and personality diverged")
	}
}
