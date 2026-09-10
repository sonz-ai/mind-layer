package mindlayer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProviderURLValidation(t *testing.T) {
	for _, url := range []string{"http://example.test/v1", "https://owner:password@example.test/v1", "https://example.test/v1?key=secret", "file:///tmp/model"} {
		if _, err := NewProvider(url, "key", "chat", ""); err == nil {
			t.Fatal("unsafe provider URL accepted")
		}
	}
}
func TestProviderErrorRedaction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte("private provider response with credentials"))
	}))
	defer server.Close()
	p, _ := NewProvider(server.URL, "synthetic-test-key", "chat", "embedding")
	_, err := p.Embed(context.Background(), "synthetic fact")
	if err == nil || !strings.Contains(err.Error(), "401") || strings.Contains(err.Error(), "credentials") || strings.Contains(err.Error(), "synthetic-test-key") {
		t.Fatal("error redaction failed")
	}
}
func TestProviderRejectsRedirect(t *testing.T) {
	called := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	defer target.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 307) }))
	defer server.Close()
	p, _ := NewProvider(server.URL, "synthetic-test-key", "chat", "embedding")
	if _, err := p.Embed(context.Background(), "fact"); err == nil || called {
		t.Fatal("redirect followed or accepted")
	}
}
func TestInvalidProviderOutput(t *testing.T) {
	for _, body := range []string{`{}`, `{"data":[{"embedding":[0,0]}]}`, `{"data":[{"embedding":[]}]}`, `not JSON`} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
			defer server.Close()
			p, _ := NewProvider(server.URL, "", "chat", "embedding")
			if _, err := p.Embed(context.Background(), "fact"); err == nil {
				t.Fatal("bad embedding accepted")
			}
		})
	}
}
func TestExtractionFailureDoesNotPersist(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"facts\":[\"valid\",\"\"]}"}}]}`))
	}))
	defer server.Close()
	p, _ := NewProvider(server.URL, "", "chat", "")
	s := openTest(t, p)
	if _, err := s.Ingest(context.Background(), testScope, "synthetic transcript", ""); err == nil {
		t.Fatal("invalid extraction accepted")
	}
	out, _ := s.Export(testScope)
	if len(out.Memories) != 0 {
		t.Fatal("failed extraction persisted partial output")
	}
}
