package mindlayer

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var testScope = Scope{"assistant", "user-a"}

func openTest(t *testing.T, p *Provider) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "memory.db"), p)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
func putTest(t *testing.T, s *Store, scope Scope, in Input) Memory {
	t.Helper()
	m, err := s.Put(context.Background(), scope, in)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
func findTest(t *testing.T, s *Store, scope Scope, q, mode string) []Hit {
	t.Helper()
	hits, err := s.Search(context.Background(), scope, q, mode, 5)
	if err != nil {
		t.Fatal(err)
	}
	return hits
}

func TestDurableMemory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "memory.db")
	s, err := Open(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	m := putTest(t, s, testScope, Input{Text: "The user prefers vegetarian dinners.", Session: "session-one"})
	if err := s.SetPersonality(testScope, "Warm and concise."); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	hits := findTest(t, s, testScope, "vegetarian", "lexical")
	if len(hits) != 1 || hits[0].Memory.ID != m.ID || hits[0].Memory.Session != "session-one" {
		t.Fatalf("memory did not survive restart: %+v", hits)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("database permissions")
	}
	exported, err := s.Export(testScope)
	if err != nil || exported.Personality != "Warm and concise." {
		t.Fatal("personality did not persist")
	}
}
func TestScopeIsolation(t *testing.T) {
	s := openTest(t, nil)
	putTest(t, s, testScope, Input{Text: "Private vegetarian preference"})
	for _, other := range []Scope{{"assistant", "user-b"}, {"other-agent", "user-a"}} {
		if len(findTest(t, s, other, "vegetarian", "lexical")) != 0 {
			t.Fatal("cross-scope memory leak")
		}
		exported, err := s.Export(other)
		if err != nil || len(exported.Memories) != 0 {
			t.Fatal("cross-scope export leak")
		}
		if err := s.DeleteScope(other); err != nil {
			t.Fatal(err)
		}
	}
	if len(findTest(t, s, testScope, "vegetarian", "lexical")) != 1 {
		t.Fatal("other-scope deletion affected owner")
	}
}
func TestLexicalRecall(t *testing.T) {
	s := openTest(t, nil)
	putTest(t, s, testScope, Input{Text: "The user runs every morning."})
	putTest(t, s, testScope, Input{Text: "The user enjoys painting."})
	hits := findTest(t, s, testScope, "running", "lexical")
	if len(hits) != 1 || !strings.Contains(hits[0].Memory.Text, "runs") {
		t.Fatal("stemmed lexical recall failed")
	}
	if len(findTest(t, s, testScope, "zxqvnonexistent", "lexical")) != 0 {
		t.Fatal("no-match query should return empty")
	}
}
func TestExplicitUpdatesAndDeduplication(t *testing.T) {
	s := openTest(t, nil)
	first := putTest(t, s, testScope, Input{ID: "diet", Text: "User eats meat."})
	second := putTest(t, s, testScope, Input{ID: "diet", Text: "User is vegetarian."})
	if !first.CreatedAt.Equal(second.CreatedAt) {
		t.Fatal("update reset creation time")
	}
	if len(findTest(t, s, testScope, "meat", "lexical")) != 0 {
		t.Fatal("stale memory remained indexed")
	}
	a := putTest(t, s, testScope, Input{Text: "User likes jazz."})
	b := putTest(t, s, testScope, Input{Text: " User likes jazz. "})
	if a.ID != b.ID {
		t.Fatal("exact duplicate was not deduplicated")
	}
	out, err := s.Export(testScope)
	if err != nil || len(out.Memories) != 2 {
		t.Fatal("unexpected number of memories")
	}
}
func TestExportAndDeletion(t *testing.T) {
	s := openTest(t, nil)
	m := putTest(t, s, testScope, Input{Text: "Synthetic memory for export."})
	exported, err := s.Export(testScope)
	if err != nil || len(exported.Memories) != 1 || exported.Memories[0].ID != m.ID {
		t.Fatal("export mismatch")
	}
	if err := s.Delete(Scope{"assistant", "user-b"}, m.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("cross-scope delete succeeded")
	}
	if err := s.Delete(testScope, m.ID); err != nil {
		t.Fatal(err)
	}
	if len(findTest(t, s, testScope, "synthetic", "lexical")) != 0 {
		t.Fatal("deleted memory retrieved")
	}
	if err := s.SetPersonality(testScope, "A configured personality"); err != nil {
		t.Fatal(err)
	}
	putTest(t, s, testScope, Input{Text: "Another memory."})
	if err := s.DeleteScope(testScope); err != nil {
		t.Fatal(err)
	}
	out, err := s.Export(testScope)
	if err != nil || len(out.Memories) != 0 || out.Personality != "" {
		t.Fatal("scope was not logically deleted")
	}
}
func TestPersonalityContext(t *testing.T) {
	s := openTest(t, nil)
	s.SetPersonality(testScope, "Answer concisely and warmly.")
	putTest(t, s, testScope, Input{Text: "User prefers tea."})
	out, err := s.BuildContext(context.Background(), testScope, "tea", "lexical", 3)
	if err != nil || out.Personality != "Answer concisely and warmly." || len(out.Memories) != 1 {
		t.Fatalf("bad context: %+v %v", out, err)
	}
}
func TestInputValidationAndAtomicBatch(t *testing.T) {
	s := openTest(t, nil)
	_, err := s.PutBatch(context.Background(), testScope, []Input{{Text: "valid"}, {Text: ""}})
	if !errors.Is(err, ErrInvalid) {
		t.Fatal("invalid batch accepted")
	}
	out, _ := s.Export(testScope)
	if len(out.Memories) != 0 {
		t.Fatal("partial batch committed")
	}
	for _, scope := range []Scope{{"", "user"}, {"../escape", "user"}, {"agent", "a/b"}} {
		if _, err := s.Put(context.Background(), scope, Input{Text: "hello"}); !errors.Is(err, ErrInvalid) {
			t.Fatal("invalid scope accepted")
		}
	}
	if _, err := s.Search(context.Background(), testScope, "hello", "invalid", 5); !errors.Is(err, ErrInvalid) {
		t.Fatal("bad search mode accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Put(ctx, testScope, Input{Text: "cancelled"}); err == nil {
		t.Fatal("cancelled write succeeded")
	}
}
func TestConcurrentMemoryAccess(t *testing.T) {
	s := openTest(t, nil)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				if _, err := s.Put(context.Background(), testScope, Input{Text: "Concurrent synthetic preference."}); err != nil {
					t.Error(err)
				}
				if _, err := s.Search(context.Background(), testScope, "preference", "lexical", 2); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
	out, _ := s.Export(testScope)
	if len(out.Memories) != 1 {
		t.Fatal("concurrent deduplication failed")
	}
}

func fakeProvider(t *testing.T) *Provider {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer synthetic-test-key" {
			t.Error("provider authentication missing")
		}
		var in map[string]json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			t.Error(err)
		}
		switch r.URL.Path {
		case "/embeddings":
			var text string
			json.Unmarshal(in["input"], &text)
			v := []float64{1, 0}
			if strings.Contains(text, "painting") {
				v = []float64{0, 1}
			}
			json.NewEncoder(w).Encode(map[string]any{"data": []any{map[string]any{"embedding": v}}})
		case "/chat/completions":
			content := `{"facts":["User avoids animal products."]}`
			if _, ok := in["response_format"]; !ok {
				var messages []Message
				json.Unmarshal(in["messages"], &messages)
				if len(messages) != 3 || !strings.Contains(messages[0].Content, "concise") || !strings.Contains(messages[1].Content, "animal products") {
					t.Error("chat missing personality or retrieved memory")
				}
				content = "Try a plant-based dinner."
			}
			json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": Message{"assistant", content}}}})
		default:
			t.Error("unexpected provider endpoint")
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	p, err := NewProvider(server.URL, "synthetic-test-key", "fixture-chat", "fixture-embedding")
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func TestSemanticAndHybridRecall(t *testing.T) {
	s := openTest(t, fakeProvider(t))
	putTest(t, s, testScope, Input{ID: "diet", Text: "User avoids animal products."})
	putTest(t, s, testScope, Input{ID: "art", Text: "User enjoys painting."})
	for _, mode := range []string{"semantic", "hybrid"} {
		hits := findTest(t, s, testScope, "vegan", ""+mode)
		if len(hits) == 0 || hits[0].Memory.ID != "diet" {
			t.Fatal("semantic match missing")
		}
	}
	if len(findTest(t, s, Scope{"other", "user-a"}, "vegan", "semantic")) != 0 {
		t.Fatal("semantic cross-scope leak")
	}
}
func TestExtractionAndGroundedChat(t *testing.T) {
	s := openTest(t, fakeProvider(t))
	s.SetPersonality(testScope, "Be concise.")
	memories, err := s.Ingest(context.Background(), testScope, "User: I do not eat animal products.", "session-one")
	if err != nil || len(memories) != 1 {
		t.Fatal("extraction failed", err)
	}
	answer, context, err := s.Chat(context.Background(), testScope, "dinner", "semantic", 3)
	if err != nil || answer != "Try a plant-based dinner." || len(context.Memories) != 1 {
		t.Fatal("chat failed", err)
	}
}
func TestNoProviderRequired(t *testing.T) {
	s := openTest(t, nil)
	putTest(t, s, testScope, Input{Text: "Offline memory."})
	if len(findTest(t, s, testScope, "offline", "lexical")) != 1 {
		t.Fatal("offline memory failed")
	}
	if _, err := s.Ingest(context.Background(), testScope, "User: hello", "session"); err == nil {
		t.Fatal("unconfigured extraction should fail clearly")
	}
	if _, err := s.Search(context.Background(), testScope, "hello", "semantic", 1); err == nil {
		t.Fatal("unconfigured semantic search should fail clearly")
	}
}
func TestModelMismatchFailsClosed(t *testing.T) {
	p := fakeProvider(t)
	s := openTest(t, p)
	putTest(t, s, testScope, Input{Text: "Synthetic fact."})
	p.embeddingModel = "different-model"
	if _, err := s.Search(context.Background(), testScope, "fact", "semantic", 1); err == nil {
		t.Fatal("incompatible vectors were searched")
	}
	if len(findTest(t, s, testScope, "fact", "lexical")) != 1 {
		t.Fatal("lexical fallback must remain explicitly available")
	}
}
