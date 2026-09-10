package mindlayer

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
)

type request struct {
	Scope       Scope  `json:"scope"`
	ID          string `json:"id,omitempty"`
	Text        string `json:"text,omitempty"`
	Session     string `json:"session,omitempty"`
	Query       string `json:"query,omitempty"`
	Mode        string `json:"mode,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Personality string `json:"personality,omitempty"`
}

// Handler serves a single owner's scopes. The owner token authorizes ALL scopes.
// Do not give this token to untrusted end users; authorize scopes in your backend.
func Handler(s *Store, token string) http.Handler {
	mux := http.NewServeMux()
	respond := func(w http.ResponseWriter, v any, err error) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if err != nil {
			code, message := http.StatusInternalServerError, "operation failed; check provider configuration and database availability"
			if errors.Is(err, ErrInvalid) {
				code, message = 400, "invalid input"
			}
			if errors.Is(err, ErrNotFound) {
				code, message = 404, "not found"
			}
			w.WriteHeader(code)
			json.NewEncoder(w).Encode(map[string]string{"error": message})
			return
		}
		json.NewEncoder(w).Encode(v)
	}
	decode := func(w http.ResponseWriter, r *http.Request) (request, bool) {
		var in request
		media, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if media != "application/json" {
			respond(w, nil, ErrInvalid)
			return in, false
		}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&in); err != nil {
			respond(w, nil, ErrInvalid)
			return in, false
		}
		var extra any
		if dec.Decode(&extra) != io.EOF {
			respond(w, nil, ErrInvalid)
			return in, false
		}
		if _, err := in.Scope.key(); err != nil {
			respond(w, nil, ErrInvalid)
			return in, false
		}
		if in.Limit == 0 {
			in.Limit = 5
		}
		return in, true
	}
	queryScope := func(r *http.Request) Scope { return Scope{r.URL.Query().Get("agent"), r.URL.Query().Get("user")} }
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { respond(w, map[string]string{"status": "ok"}, nil) })
	mux.HandleFunc("POST /v1/memories", func(w http.ResponseWriter, r *http.Request) {
		in, ok := decode(w, r)
		if !ok {
			return
		}
		out, err := s.Put(r.Context(), in.Scope, Input{in.ID, in.Text, in.Session})
		respond(w, out, err)
	})
	mux.HandleFunc("DELETE /v1/memories", func(w http.ResponseWriter, r *http.Request) {
		err := s.Delete(queryScope(r), r.URL.Query().Get("id"))
		respond(w, map[string]bool{"deleted": err == nil}, err)
	})
	mux.HandleFunc("POST /v1/search", func(w http.ResponseWriter, r *http.Request) {
		in, ok := decode(w, r)
		if !ok {
			return
		}
		out, err := s.Search(r.Context(), in.Scope, in.Query, in.Mode, in.Limit)
		respond(w, out, err)
	})
	mux.HandleFunc("POST /v1/context", func(w http.ResponseWriter, r *http.Request) {
		in, ok := decode(w, r)
		if !ok {
			return
		}
		out, err := s.BuildContext(r.Context(), in.Scope, in.Query, in.Mode, in.Limit)
		respond(w, out, err)
	})
	mux.HandleFunc("POST /v1/ingest", func(w http.ResponseWriter, r *http.Request) {
		in, ok := decode(w, r)
		if !ok {
			return
		}
		out, err := s.Ingest(r.Context(), in.Scope, in.Text, in.Session)
		respond(w, out, err)
	})
	mux.HandleFunc("POST /v1/chat", func(w http.ResponseWriter, r *http.Request) {
		in, ok := decode(w, r)
		if !ok {
			return
		}
		answer, context, err := s.Chat(r.Context(), in.Scope, in.Query, in.Mode, in.Limit)
		respond(w, map[string]any{"answer": answer, "context": context}, err)
	})
	mux.HandleFunc("PUT /v1/personality", func(w http.ResponseWriter, r *http.Request) {
		in, ok := decode(w, r)
		if !ok {
			return
		}
		err := s.SetPersonality(in.Scope, in.Personality)
		respond(w, map[string]bool{"saved": err == nil}, err)
	})
	mux.HandleFunc("GET /v1/export", func(w http.ResponseWriter, r *http.Request) {
		out, err := s.Export(queryScope(r))
		respond(w, out, err)
	})
	mux.HandleFunc("DELETE /v1/scope", func(w http.ResponseWriter, r *http.Request) {
		err := s.DeleteScope(queryScope(r))
		respond(w, map[string]bool{"deleted": err == nil}, err)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No browser-origin access. Prevent local CSRF and DNS rebinding.
		if r.Header.Get("Origin") != "" {
			http.Error(w, "browser-origin requests are disabled", 403)
			return
		}
		if token != "" {
			want := sha256.Sum256([]byte("Bearer " + token))
			got := sha256.Sum256([]byte(r.Header.Get("Authorization")))
			if subtle.ConstantTimeCompare(want[:], got[:]) != 1 {
				http.Error(w, "unauthorized", 401)
				return
			}
		} else {
			host := r.Host
			if h, _, err := net.SplitHostPort(host); err == nil {
				host = h
			}
			ip := net.ParseIP(strings.Trim(host, "[]"))
			if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
				http.Error(w, "invalid local host", 403)
				return
			}
		}
		mux.ServeHTTP(w, r)
	})
}

// ValidateListen prevents accidentally exposing an unauthenticated database.
func ValidateListen(addr, token string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return ErrInvalid
	}
	p, err := strconv.Atoi(port)
	if err != nil || p < 0 || p > 65535 {
		return ErrInvalid
	}
	ip := net.ParseIP(host)
	if host != "localhost" && (ip == nil || !ip.IsLoopback()) && len(token) < 32 {
		return errors.New("non-loopback listening requires MIND_LAYER_TOKEN of at least 32 characters")
	}
	return nil
}
