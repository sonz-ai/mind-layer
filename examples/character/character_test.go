package character

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	ml "github.com/sonz-ai/mind-layer"
)

func TestCharacterEvolution(t *testing.T) {
	s, err := ml.Open(filepath.Join(t.TempDir(), "demo.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	d, err := New(s, ml.Scope{Agent: "moss", User: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	expected := []Traits{{32, 45, 50}, {37, 50, 65}, {57, 60, 70}, {32, 40, 60}, {44, 48, 65}}
	for i, a := range Actions() {
		v, err := d.Interact(context.Background(), a.ID, i)
		if err != nil {
			t.Fatal(err)
		}
		if v.Traits != expected[i] || len(v.Events) != i+1 {
			t.Fatalf("step %d: %+v", i, v)
		}
	}
	ctx, err := d.Recall(context.Background(), "promise sensor weather")
	if err != nil || len(ctx.Memories) == 0 {
		t.Fatal("shared experience not recalled", err)
	}
	before, err := d.View()
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.Interact(context.Background(), "encourage", 0)
	if !errors.Is(err, ErrConflict) {
		t.Fatal("stale interaction not rejected")
	}
	after, _ := d.View()
	if !reflect.DeepEqual(before, after) {
		t.Fatal("stale request changed history")
	}
}
func TestCharacterRestartAndIsolation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "demo.db")
	s, err := ml.Open(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	scope := ml.Scope{Agent: "moss", User: "demo"}
	d, err := New(s, scope)
	if err != nil {
		t.Fatal(err)
	}
	before, err := d.Interact(context.Background(), "teach", 0)
	if err != nil {
		t.Fatal(err)
	}
	s.Close()
	s, err = ml.Open(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	d, err = New(s, scope)
	if err != nil {
		t.Fatal(err)
	}
	after, err := d.View()
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("character did not survive restart")
	}
	other, err := New(s, ml.Scope{Agent: "moss", User: "other"})
	if err != nil {
		t.Fatal(err)
	}
	v, _ := other.View()
	if v.Revision != 0 || v.Traits != initial() {
		t.Fatal("character crossed user scope")
	}
}
func TestCharacterConcurrentRevision(t *testing.T) {
	s, err := ml.Open(filepath.Join(t.TempDir(), "demo.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	d, err := New(s, ml.Scope{Agent: "moss", User: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := d.Interact(context.Background(), "encourage", 0); errs <- err }()
	}
	wg.Wait()
	close(errs)
	successes := 0
	for err := range errs {
		if err == nil {
			successes++
		} else if !errors.Is(err, ErrConflict) {
			t.Fatal(err)
		}
	}
	if successes != 1 {
		t.Fatal("revision applied twice")
	}
}
func TestCharacterBoundsAndInvalidAction(t *testing.T) {
	if evolve(Traits{95, 95, 95}, Traits{20, 20, 20}) != (Traits{100, 100, 100}) || evolve(Traits{1, 1, 1}, Traits{-20, -20, -20}) != (Traits{}) {
		t.Fatal("trait bounds failed")
	}
	s, err := ml.Open(filepath.Join(t.TempDir(), "demo.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	d, err := New(s, ml.Scope{Agent: "moss", User: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = d.Interact(context.Background(), "unknown", 0); !errors.Is(err, ml.ErrInvalid) {
		t.Fatal("unknown action accepted")
	}
	v, _ := d.View()
	if v.Revision != 0 {
		t.Fatal("invalid action changed state")
	}
}
