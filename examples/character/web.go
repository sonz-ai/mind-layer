package character

import (
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"

	ml "github.com/sonz-ai/mind-layer"
)

//go:embed index.html
var page []byte

//go:embed showcase.html
var showcase []byte

func Handler(d *Demo, live bool, providers ...*ml.Provider) http.Handler {
	var provider *ml.Provider
	if live && len(providers) > 0 {
		provider = providers[0]
	}
	mux := http.NewServeMux()
	send := func(w http.ResponseWriter, v any, err error) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		if err != nil {
			status, message := 500, "Operation failed; your stored history is preserved."
			if errors.Is(err, ErrConflict) {
				status, message = 409, "State changed; reload and try again."
			}
			if errors.Is(err, ErrReplyLimit) {
				status, message = 502, "Model reply was too long. Ask for a shorter response and try again."
			}
			if errors.Is(err, ml.ErrInvalid) {
				status, message = 400, "Invalid request."
			}
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(map[string]string{"error": message})
			return
		}
		json.NewEncoder(w).Encode(v)
	}
	type input struct {
		Action   string `json:"action"`
		Revision int    `json:"revision"`
		Query    string `json:"query"`
	}
	decode := func(w http.ResponseWriter, r *http.Request) (input, bool) {
		var in input
		typ, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if typ != "application/json" {
			send(w, nil, ml.ErrInvalid)
			return in, false
		}
		r.Body = http.MaxBytesReader(w, r.Body, 16384)
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		if dec.Decode(&in) != nil {
			send(w, nil, ml.ErrInvalid)
			return in, false
		}
		var extra any
		if dec.Decode(&extra) != io.EOF {
			send(w, nil, ml.ErrInvalid)
			return in, false
		}
		return in, true
	}
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(page)
	})
	mux.HandleFunc("GET /showcase", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(showcase)
	})
	mux.HandleFunc("GET /api/state", func(w http.ResponseWriter, r *http.Request) {
		v, err := d.View()
		send(w, map[string]any{"character": v, "live": live, "model": provider.ChatModel()}, err)
	})
	mux.HandleFunc("POST /api/interact", func(w http.ResponseWriter, r *http.Request) {
		in, ok := decode(w, r)
		if !ok {
			return
		}
		v, err := d.Interact(r.Context(), in.Action, in.Revision)
		send(w, v, err)
	})
	mux.HandleFunc("GET /api/chat", func(w http.ResponseWriter, r *http.Request) {
		turns, err := d.Chats()
		send(w, turns, err)
	})
	mux.HandleFunc("POST /api/chat", func(w http.ResponseWriter, r *http.Request) {
		in, ok := decode(w, r)
		if !ok {
			return
		}
		turn, err := d.Chat(r.Context(), in.Query, in.Revision, provider)
		send(w, turn, err)
	})
	mux.HandleFunc("POST /api/recall", func(w http.ResponseWriter, r *http.Request) {
		in, ok := decode(w, r)
		if !ok {
			return
		}
		v, err := d.Recall(r.Context(), in.Query)
		send(w, v, err)
	})
	mux.HandleFunc("POST /api/reply", func(w http.ResponseWriter, r *http.Request) {
		if !live {
			w.WriteHeader(503)
			return
		}
		in, ok := decode(w, r)
		if !ok {
			return
		}
		answer, ctx, err := d.Reply(r.Context(), in.Query)
		send(w, map[string]any{"answer": answer, "context": ctx}, err)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		ip := net.ParseIP(host)
		if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
			http.Error(w, "local demo only", 403)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" && origin != "http://"+r.Host {
			http.Error(w, "cross-origin request rejected", 403)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; connect-src 'self'; img-src 'self' data:; frame-ancestors 'none'")
		mux.ServeHTTP(w, r)
	})
}
