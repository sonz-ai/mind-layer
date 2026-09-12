package character

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	ml "github.com/sonz-ai/mind-layer"
)

func TestChatPersistenceRecallAndIsolation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "chat.db")
	s, err := ml.Open(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	d, err := New(s, ml.Scope{Agent: "moss", User: "one"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := d.Chat(context.Background(), "My greenhouse is named Fern House.", 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.Mode != "offline" {
		t.Fatal("offline not labeled")
	}
	second, err := d.Chat(context.Background(), "What is my greenhouse named?", 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(second.Answer, "Fern House") || len(second.Context.Memories) != 1 {
		t.Fatal("persisted user memory not recalled")
	}
	if _, err = d.Chat(context.Background(), "duplicate", 1, nil); !errors.Is(err, ErrConflict) {
		t.Fatal("stale chat accepted")
	}
	if _, err = d.Interact(context.Background(), "teach", 0); err != nil {
		t.Fatal("chat changed story revision", err)
	}
	before, _ := d.Chats()
	s.Close()
	s, err = ml.Open(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	d, err = New(s, ml.Scope{Agent: "moss", User: "one"})
	if err != nil {
		t.Fatal(err)
	}
	after, err := d.Chats()
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("chat receipt lost after restart", err)
	}
	other, err := New(s, ml.Scope{Agent: "moss", User: "two"})
	if err != nil {
		t.Fatal(err)
	}
	v, err := other.Chats()
	if err != nil || len(v) != 0 {
		t.Fatal("chat crossed user scope")
	}
}

func TestChatProviderContextAndFailureAtomicity(t *testing.T) {
	fail := false
	var requests [][]ml.Message
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail {
			http.Error(w, "private provider error", 500)
			return
		}
		var body struct {
			Messages []ml.Message `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		requests = append(requests, body.Messages)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"Let’s plant moonflowers beside Fern House."}}]}`))
	}))
	defer server.Close()
	provider, err := ml.NewProvider(server.URL, "", "synthetic-test-model", "")
	if err != nil {
		t.Fatal(err)
	}
	s, err := ml.Open(filepath.Join(t.TempDir(), "chat.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	d, err := New(s, ml.Scope{Agent: "moss", User: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = d.Chat(context.Background(), "My greenhouse is named Fern House.", 0, provider)
	if err != nil {
		t.Fatal(err)
	}
	turn, err := d.Chat(context.Background(), "What is my greenhouse named?", 1, provider)
	if err != nil {
		t.Fatal(err)
	}
	if turn.Mode != "model" || turn.Model != "synthetic-test-model" || len(turn.Context.Memories) != 1 || !reflect.DeepEqual(turn.RecentTurns, []int{1}) {
		t.Fatal("missing context receipt")
	}
	if !strings.Contains(requests[1][0].Content, "fictional curious workshop") || !strings.Contains(requests[1][1].Content, "Fern House") {
		t.Fatal("model did not receive persona and memory")
	}
	if strings.Contains(requests[1][1].Content, "moonflowers") {
		t.Fatal("generated answer indexed as a user fact")
	}
	fail = true
	before, _ := d.Chats()
	if _, err = d.Chat(context.Background(), "Remember a failed turn", 2, provider); err == nil {
		t.Fatal("provider failure hidden")
	}
	after, _ := d.Chats()
	if !reflect.DeepEqual(before, after) {
		t.Fatal("failed provider call partially saved")
	}
}

func TestChatHTTPAndValidation(t *testing.T) {
	s, err := ml.Open(filepath.Join(t.TempDir(), "chat.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	d, err := New(s, ml.Scope{Agent: "moss", User: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	h := Handler(d, false)
	for _, tc := range []struct {
		body   string
		status int
	}{
		{`{"query":"","revision":0}`, 400},
		{`{"query":"My greenhouse is Fern House.","revision":0}`, 200},
		{`{"query":"Again","revision":0}`, 409},
	} {
		r := httptest.NewRequest("POST", "http://127.0.0.1:8090/api/chat", strings.NewReader(tc.body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.status {
			t.Fatalf("got %d want %d: %s", w.Code, tc.status, w.Body.String())
		}
	}
	r := httptest.NewRequest("GET", "http://127.0.0.1:8090/api/chat", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "Fern House") {
		t.Fatal("history endpoint omitted turn")
	}
	if _, err = d.Chat(context.Background(), strings.Repeat("x", 701), 1, nil); !errors.Is(err, ml.ErrInvalid) {
		t.Fatal("oversized input accepted")
	}
}

func TestChatConcurrentRevision(t *testing.T) {
	s, err := ml.Open(filepath.Join(t.TempDir(), "chat.db"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	d, err := New(s, ml.Scope{Agent: "moss", User: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := d.Chat(context.Background(), "My project is Fern House.", 0, nil)
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		} else if !errors.Is(err, ErrConflict) {
			t.Fatal(err)
		}
	}
	turns, _ := d.Chats()
	if successes != 1 || len(turns) != 1 {
		t.Fatal("duplicate turn committed")
	}
}
