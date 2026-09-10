package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	ml "github.com/sonz-ai/mind-layer"
	"github.com/sonz-ai/mind-layer/examples/character"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8090", "local demo address")
	data := flag.String("data", "data/character-demo.db", "dedicated character database")
	live := flag.Bool("live", false, "enable optional provider-generated replies (provider charges may apply)")
	scenario := flag.Bool("scenario", false, "run five interactions on a fresh database and print JSON instead of serving")
	flag.Parse()
	host, _, err := net.SplitHostPort(*addr)
	ip := net.ParseIP(host)
	if err != nil || ip == nil || !ip.IsLoopback() {
		log.Fatal("character demo must bind to a loopback IP")
	}
	var provider *ml.Provider
	if *live {
		provider, err = ml.NewProvider(os.Getenv("MIND_LAYER_BASE_URL"), os.Getenv("MIND_LAYER_API_KEY"), os.Getenv("MIND_LAYER_CHAT_MODEL"), "")
		if err != nil {
			log.Fatal(err)
		}
	}
	store, err := ml.Open(*data, provider)
	if err != nil {
		log.Fatal("cannot open demo database")
	}
	defer store.Close()
	demo, err := character.New(store, ml.Scope{Agent: "moss", User: "demo-user"})
	if err != nil {
		log.Fatal(err)
	}
	if *scenario {
		v, err := demo.View()
		if err != nil {
			log.Fatal(err)
		}
		if v.Revision != 0 {
			log.Fatal("scenario requires a fresh database; choose a new -data path")
		}
		for i, a := range character.Actions() {
			v, err = demo.Interact(context.Background(), a.ID, i)
			if err != nil {
				log.Fatal(err)
			}
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if enc.Encode(v) != nil {
			log.Fatal("cannot write scenario output")
		}
		return
	}
	server := &http.Server{Addr: *addr, Handler: character.Handler(demo, *live), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 90 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()
	log.Printf("Moss character demo: http://%s (live dialogue: %t)", *addr, *live)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal("demo server failed")
	}
}
