package mindlayer

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

// Opt-in only: this sends authored synthetic data to the configured provider.
func TestLiveProvider(t *testing.T) {
	if os.Getenv("MIND_LAYER_LIVE_TEST") != "1" {
		t.Skip("set MIND_LAYER_LIVE_TEST=1 and provider configuration to opt into provider charges")
	}
	p, err := NewProvider(os.Getenv("MIND_LAYER_BASE_URL"), os.Getenv("MIND_LAYER_API_KEY"), os.Getenv("MIND_LAYER_CHAT_MODEL"), os.Getenv("MIND_LAYER_EMBEDDING_MODEL"))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()
	s := openTest(t, p)
	s.SetPersonality(testScope, "Be concise. Use one sentence.")
	facts, err := s.Ingest(ctx, testScope, "User: I am vegetarian and I cook dinner for two people.", "synthetic-session-one")
	if err != nil {
		t.Fatal(err)
	}
	if len(facts) == 0 {
		t.Fatal("provider extracted no facts")
	}
	hits, err := s.Search(ctx, testScope, "What dietary preference should guide meal suggestions?", "semantic", 3)
	if err != nil || len(hits) == 0 {
		t.Fatal("live semantic recall failed", err)
	}
	answer, context, err := s.Chat(ctx, testScope, "What is my dietary preference?", "hybrid", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(context.Memories) == 0 || !strings.Contains(strings.ToLower(answer), "vegetarian") {
		t.Fatal("live answer did not use the synthetic dietary preference")
	}
	t.Logf("live synthetic extraction, semantic retrieval and grounded answer passed; extracted facts=%d", len(facts))
}
