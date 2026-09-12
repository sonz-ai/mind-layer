package character

import (
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	ml "github.com/sonz-ai/mind-layer"
)

func TestCharacterHTTP(t *testing.T) {
	s, err := ml.Open(filepath.Join(t.TempDir(), "demo.db"), nil)
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
		method, path, body, origin string
		status                     int
		contains                   string
	}{
		{"GET", "/", "", "", 200, "Meet Moss."},
		{"GET", "/showcase", "", "", 200, "Memory, made visible."},
		{"GET", "/api/state", "", "", 200, `"revision":0`},
		{"POST", "/api/interact", `{"action":"encourage","revision":0}`, "http://127.0.0.1:8090", 200, `"confidence":45`},
		{"POST", "/api/interact", `{"action":"encourage","revision":0}`, "", 409, "State changed"},
		{"POST", "/api/recall", `{"query":"encouraged"}`, "", 200, "encouraged Moss"},
		{"POST", "/api/reply", `{"query":"hello"}`, "", 503, ""},
		{"POST", "/api/interact", `{"action":"teach","revision":1}`, "https://untrusted.example.test", 403, "cross-origin"},
		{"POST", "/api/interact", `{"action":"teach","revision":1,"unknown":true}`, "", 400, "Invalid"},
	} {
		r := httptest.NewRequest(tc.method, "http://127.0.0.1:8090"+tc.path, strings.NewReader(tc.body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", tc.origin)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.contains) {
			t.Fatalf("%s: %d %s", tc.path, w.Code, w.Body.String())
		}
	}
}
