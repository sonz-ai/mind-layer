package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	mindlayer "github.com/sonz-ai/mind-layer"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "listen address")
	data := flag.String("data", "data/memory.db", "embedded database path")
	flag.Parse()
	token := os.Getenv("MIND_LAYER_TOKEN")
	if err := mindlayer.ValidateListen(*addr, token); err != nil {
		log.Fatal(err)
	}
	var provider *mindlayer.Provider
	if base := os.Getenv("MIND_LAYER_BASE_URL"); base != "" {
		var err error
		provider, err = mindlayer.NewProvider(base, os.Getenv("MIND_LAYER_API_KEY"), os.Getenv("MIND_LAYER_CHAT_MODEL"), os.Getenv("MIND_LAYER_EMBEDDING_MODEL"))
		if err != nil {
			log.Fatal(err)
		}
	} else if os.Getenv("MIND_LAYER_CHAT_MODEL") != "" || os.Getenv("MIND_LAYER_EMBEDDING_MODEL") != "" || os.Getenv("MIND_LAYER_API_KEY") != "" {
		log.Fatal("set MIND_LAYER_BASE_URL when configuring a provider")
	}
	store, err := mindlayer.Open(*data, provider)
	if err != nil {
		log.Fatal("cannot open memory database")
	}
	defer store.Close()
	server := &http.Server{Addr: *addr, Handler: mindlayer.Handler(store, token), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 180 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		deadline, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(deadline)
	}()
	log.Print("Mind Layer ready; local persistence enabled; no Sonzai account required")
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("HTTP server failed")
	}
}
