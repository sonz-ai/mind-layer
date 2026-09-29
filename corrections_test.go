package mindlayer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func ids(hits []Hit) []string {
	out := []string{}
	for _, h := range hits {
		out = append(out, h.Memory.ID)
	}
	return out
}

func TestCorrectionsSupersedeOlderMemories(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.db")
	s, err := Open(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	putTest(t, s, testScope, Input{ID: "budget-1", Text: "The garden budget is 180 euros."})
	fix := putTest(t, s, testScope, Input{ID: "budget-2", Text: "The garden budget is now 120 euros.", Supersedes: "budget-1"})
	if fix.Supersedes != "budget-1" {
		t.Fatal("correction link not returned")
	}
	if got := ids(findTest(t, s, testScope, "garden budget", "lexical")); len(got) != 1 || got[0] != "budget-2" {
		t.Fatalf("superseded memory still retrieved: %v", got)
	}
	c, err := s.BuildContext(context.Background(), testScope, "garden budget", "lexical", 5)
	if err != nil || len(c.Memories) != 1 || c.Memories[0].Memory.ID != "budget-2" {
		t.Fatal("context included superseded memory", err)
	}

	// The older memory stays in export for audit, and survives a restart hidden.
	s.Close()
	if s, err = Open(path, nil); err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	snap, _ := s.Export(testScope)
	byID := map[string]Memory{}
	for _, m := range snap.Memories {
		byID[m.ID] = m
	}
	if len(snap.Memories) != 2 || byID["budget-1"].SupersededBy != "budget-2" {
		t.Fatalf("export lost correction history: %+v", snap.Memories)
	}
	if got := ids(findTest(t, s, testScope, "garden budget", "lexical")); len(got) != 1 || got[0] != "budget-2" {
		t.Fatalf("correction not durable: %v", got)
	}

	// Updating the correction without naming a target keeps the link.
	putTest(t, s, testScope, Input{ID: "budget-2", Text: "The garden budget is now 120 euros, no lights."})
	if got := ids(findTest(t, s, testScope, "garden budget", "lexical")); len(got) != 1 || got[0] != "budget-2" {
		t.Fatalf("update dropped correction link: %v", got)
	}

	// Deleting the correction restores the memory it replaced.
	if err := s.Delete(testScope, "budget-2"); err != nil {
		t.Fatal(err)
	}
	if got := ids(findTest(t, s, testScope, "garden budget", "lexical")); len(got) != 1 || got[0] != "budget-1" {
		t.Fatalf("original not restored after deleting correction: %v", got)
	}
}

func TestCorrectionValidation(t *testing.T) {
	s := openTest(t, nil)
	ctx := context.Background()
	putTest(t, s, testScope, Input{ID: "a", Text: "Fact A."})
	if _, err := s.Put(ctx, testScope, Input{ID: "b", Text: "Fact B.", Supersedes: "missing"}); !errors.Is(err, ErrNotFound) {
		t.Fatal("missing target accepted", err)
	}
	if _, err := s.Put(ctx, testScope, Input{ID: "a", Text: "Fact A.", Supersedes: "a"}); !errors.Is(err, ErrInvalid) {
		t.Fatal("self-supersession accepted", err)
	}
	if _, err := s.Put(ctx, Scope{"other", "user-a"}, Input{ID: "c", Text: "Fact C.", Supersedes: "a"}); !errors.Is(err, ErrNotFound) {
		t.Fatal("cross-scope supersession accepted", err)
	}
	putTest(t, s, testScope, Input{ID: "b", Text: "Fact B.", Supersedes: "a"})
	if _, err := s.Put(ctx, testScope, Input{ID: "a", Text: "Fact A.", Supersedes: "b"}); !errors.Is(err, ErrInvalid) {
		t.Fatal("supersession cycle accepted", err)
	}
	if got := ids(findTest(t, s, testScope, "fact", "lexical")); len(got) != 1 || got[0] != "b" {
		t.Fatalf("failed write changed state: %v", got)
	}

	// A batch can store a fact and its correction together.
	if _, err := s.PutBatch(ctx, testScope, []Input{{ID: "x1", Text: "Moss is green."}, {ID: "x2", Text: "Moss is now blue.", Supersedes: "x1"}}); err != nil {
		t.Fatal(err)
	}
	if got := ids(findTest(t, s, testScope, "moss", "lexical")); len(got) != 1 || got[0] != "x2" {
		t.Fatalf("batch correction failed: %v", got)
	}
}

// correctionProvider returns a fixed extraction and records the prompt it saw.
func correctionProvider(t *testing.T, content string, seen *[]Message) *Provider {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Messages []Message `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&in)
		*seen = in.Messages
		json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": Message{"assistant", content}}}})
	}))
	t.Cleanup(server.Close)
	p, err := NewProvider(server.URL, "synthetic-test-key", "fixture-chat", "")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestIngestMarksCorrections(t *testing.T) {
	var seen []Message
	s := openTest(t, correctionProvider(t, `{"facts":[{"text":"The garden budget is 120 euros.","supersedes":"budget"},"Inez dislikes electric lights."]}`, &seen))
	putTest(t, s, testScope, Input{ID: "budget", Text: "The garden budget is 180 euros."})
	out, err := s.Ingest(context.Background(), testScope, "User: change of plan, the garden budget is now 120 euros and no electric lights.", "s2")
	if err != nil || len(out) != 2 || out[0].Supersedes != "budget" {
		t.Fatal("ingest did not record correction", err, out)
	}
	if len(seen) != 3 || !strings.Contains(seen[1].Content, `"id":"budget"`) {
		t.Fatal("extractor was not shown related existing memories")
	}
	if got := ids(findTest(t, s, testScope, "garden budget", "lexical")); len(got) != 1 || got[0] != out[0].ID {
		t.Fatalf("old budget still retrieved after ingest: %v", got)
	}
}

func TestIngestRejectsUnknownSupersedes(t *testing.T) {
	var seen []Message
	s := openTest(t, correctionProvider(t, `{"facts":[{"text":"Budget is 120 euros.","supersedes":"not-shown"}]}`, &seen))
	putTest(t, s, testScope, Input{ID: "budget", Text: "The garden budget is 180 euros."})
	if _, err := s.Ingest(context.Background(), testScope, "User: budget is 120 euros.", "s2"); err == nil {
		t.Fatal("invented supersedes id accepted")
	}
	snap, _ := s.Export(testScope)
	if len(snap.Memories) != 1 || snap.Memories[0].SupersededBy != "" {
		t.Fatal("failed extraction changed stored memories")
	}
}

func TestHTTPCorrection(t *testing.T) {
	s := openTest(t, nil)
	h := Handler(s, "")
	call := func(method, path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://127.0.0.1:8080"+path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	scope := `"scope":{"agent":"assistant","user":"user-a"}`
	if w := call("POST", "/v1/memories", `{`+scope+`,"id":"name-1","text":"The greenhouse is called Fern House."}`); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w := call("POST", "/v1/memories", `{`+scope+`,"id":"name-2","text":"The greenhouse is now called Cedar Room.","supersedes":"name-1"}`); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w := call("POST", "/v1/memories", `{`+scope+`,"id":"name-3","text":"Other.","supersedes":"missing"}`); w.Code != 404 {
		t.Fatal("missing target should be 404", w.Code)
	}
	w := call("POST", "/v1/search", `{`+scope+`,"query":"greenhouse called"}`)
	if w.Code != 200 || strings.Contains(w.Body.String(), "Fern House") || !strings.Contains(w.Body.String(), "Cedar Room") {
		t.Fatal("search returned superseded memory", w.Body.String())
	}
}
