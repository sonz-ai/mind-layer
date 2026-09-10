package mindlayer

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPMemoryLifecycle(t *testing.T) {
	s := openTest(t, nil)
	h := Handler(s, "")
	for _, step := range []struct{ method, path, body, contains string }{
		{"POST", "/v1/memories", `{"scope":{"agent":"assistant","user":"user-a"},"id":"diet","text":"Vegetarian dinners"}`, `"id":"diet"`},
		{"PUT", "/v1/personality", `{"scope":{"agent":"assistant","user":"user-a"},"personality":"Warm"}`, `"saved":true`},
		{"POST", "/v1/context", `{"scope":{"agent":"assistant","user":"user-a"},"query":"vegetarian"}`, `"personality":"Warm"`},
		{"GET", "/v1/export?agent=assistant&user=user-a", "", `"text":"Vegetarian dinners"`},
		{"DELETE", "/v1/memories?agent=assistant&user=user-a&id=diet", "", `"deleted":true`},
		{"DELETE", "/v1/scope?agent=assistant&user=user-a", "", `"deleted":true`},
	} {
		r := httptest.NewRequest(step.method, "http://127.0.0.1:8080"+step.path, strings.NewReader(step.body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 || !strings.Contains(w.Body.String(), step.contains) {
			t.Fatalf("%s %s: %d %s", step.method, step.path, w.Code, w.Body.String())
		}
	}
}
func TestHTTPAuthenticationAndBrowserIsolation(t *testing.T) {
	s := openTest(t, nil)
	token := strings.Repeat("synthetic", 5)
	for _, test := range []struct {
		token, auth, host, origin string
		want                      int
	}{
		{token, "", "127.0.0.1", "", 401}, {token, "Bearer " + token, "127.0.0.1", "", 200},
		{"", "", "attacker.example.test", "", 403}, {"", "", "127.0.0.1", "https://attacker.example.test", 403},
	} {
		r := httptest.NewRequest("GET", "http://"+test.host+"/healthz", nil)
		r.Header.Set("Authorization", test.auth)
		r.Header.Set("Origin", test.origin)
		w := httptest.NewRecorder()
		Handler(s, test.token).ServeHTTP(w, r)
		if w.Code != test.want {
			t.Fatalf("got %d, want %d", w.Code, test.want)
		}
	}
	if ValidateListen("0.0.0.0:8080", "") == nil {
		t.Fatal("unauthenticated public binding allowed")
	}
	if ValidateListen("127.0.0.1:8080", "") != nil {
		t.Fatal("local binding rejected")
	}
}
func TestHTTPRejectsInvalidInput(t *testing.T) {
	s := openTest(t, nil)
	h := Handler(s, "")
	for _, body := range []string{`{}`, `{"scope":{"agent":"assistant","user":"user-a"},"text":"hello","secret":true}`, `{"scope":{"agent":"assistant","user":"user-a"},"text":"hello"}{}`, strings.Repeat("x", 1<<20+1)} {
		r := httptest.NewRequest(http.MethodPost, "http://127.0.0.1/v1/memories", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 400 {
			t.Fatalf("invalid body accepted: %d", w.Code)
		}
	}
}

func TestHTTPProviderEndpoints(t *testing.T) {
	s := openTest(t, fakeProvider(t))
	s.SetPersonality(testScope, "Be concise.")
	h := Handler(s, "")
	for _, step := range []struct{ path, body, contains string }{
		{"/v1/ingest", `{"scope":{"agent":"assistant","user":"user-a"},"text":"User: I avoid animal products."}`, "animal products"},
		{"/v1/search", `{"scope":{"agent":"assistant","user":"user-a"},"query":"vegan","mode":"hybrid"}`, "animal products"},
		{"/v1/chat", `{"scope":{"agent":"assistant","user":"user-a"},"query":"dinner","mode":"semantic"}`, "plant-based dinner"},
	} {
		r := httptest.NewRequest("POST", "http://127.0.0.1"+step.path, strings.NewReader(step.body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 || !strings.Contains(w.Body.String(), step.contains) {
			t.Fatalf("%s: %d %s", step.path, w.Code, w.Body.String())
		}
	}
}
