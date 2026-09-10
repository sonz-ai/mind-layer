// Benchmark runs retrieval-only diagnostics, never an answer-quality judge.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	ml "github.com/sonz-ai/mind-layer"
)

type Query struct {
	ID       string   `json:"id"`
	Query    string   `json:"query"`
	Relevant []string `json:"relevant"`
	Category string   `json:"category"`
}
type Dataset struct {
	Name     string     `json:"name"`
	Memories []ml.Input `json:"memories"`
	Queries  []Query    `json:"queries"`
}
type Result struct {
	ID           string   `json:"id"`
	Category     string   `json:"category"`
	Recall       float64  `json:"recall_at_k"`
	RR           float64  `json:"reciprocal_rank_at_k"`
	Retrieved    []string `json:"retrieved_ids"`
	Milliseconds float64  `json:"milliseconds"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "benchmark failed:", err)
		os.Exit(1)
	}
}
func run() error {
	dataset := flag.String("dataset", "benchmarks/synthetic.json", "authored synthetic dataset path")
	mode := flag.String("mode", "lexical", "lexical, semantic, hybrid")
	live := flag.Bool("live", false, "allow provider calls and charges for semantic/hybrid")
	k := flag.Int("k", 3, "number of retrieved memories")
	iterations := flag.Int("iterations", 30, "local scale-latency queries; zero disables")
	size := flag.Int("size", 1000, "synthetic records in separate local scale test")
	flag.Parse()
	if *k < 1 || *k > 100 || *iterations < 0 || *size < 1 || *size > 100000 {
		return fmt.Errorf("invalid benchmark arguments")
	}
	if *mode != "lexical" && *mode != "semantic" && *mode != "hybrid" {
		return fmt.Errorf("invalid mode")
	}
	var p *ml.Provider
	if *mode != "lexical" {
		if !*live {
			return fmt.Errorf("semantic/hybrid require explicit -live consent")
		}
		var err error
		p, err = ml.NewProvider(os.Getenv("MIND_LAYER_BASE_URL"), os.Getenv("MIND_LAYER_API_KEY"), "", os.Getenv("MIND_LAYER_EMBEDDING_MODEL"))
		if err != nil {
			return err
		}
	}
	raw, err := os.ReadFile(*dataset)
	if err != nil {
		return err
	}
	var data Dataset
	if err = json.Unmarshal(raw, &data); err != nil {
		return err
	}
	if len(data.Queries) == 0 || len(data.Memories) == 0 {
		return fmt.Errorf("empty dataset")
	}
	ids := map[string]bool{}
	for _, m := range data.Memories {
		if m.ID == "" || ids[m.ID] {
			return fmt.Errorf("missing or duplicate memory ID")
		}
		ids[m.ID] = true
	}
	for _, q := range data.Queries {
		if len(q.Relevant) == 0 {
			return fmt.Errorf("every query requires known evidence IDs")
		}
		seen := map[string]bool{}
		for _, id := range q.Relevant {
			if !ids[id] || seen[id] {
				return fmt.Errorf("invalid evidence ID")
			}
			seen[id] = true
		}
	}
	dir, err := os.MkdirTemp("", "mind-layer-benchmark-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	s, err := ml.Open(filepath.Join(dir, "quality.db"), p)
	if err != nil {
		return err
	}
	defer s.Close()
	ctx := context.Background()
	scope := ml.Scope{Agent: "benchmark", User: "synthetic"}
	started := time.Now()
	for _, m := range data.Memories {
		if _, err := s.Put(ctx, scope, m); err != nil {
			return err
		}
	}
	ingestMS := float64(time.Since(started).Microseconds()) / 1000
	results := []Result{}
	recall, mrr, baseline := 0.0, 0.0, 0.0
	for _, q := range data.Queries {
		start := time.Now()
		hits, err := s.Search(ctx, scope, q.Query, *mode, *k)
		if err != nil {
			return err
		}
		r := Result{ID: q.ID, Category: q.Category, Retrieved: []string{}, Milliseconds: float64(time.Since(start).Microseconds()) / 1000}
		wanted := map[string]bool{}
		for _, id := range q.Relevant {
			wanted[id] = true
		}
		for i, h := range hits {
			r.Retrieved = append(r.Retrieved, h.Memory.ID)
			if wanted[h.Memory.ID] {
				r.Recall += 1 / float64(len(wanted))
				if r.RR == 0 {
					r.RR = 1 / float64(i+1)
				}
			}
		}
		for i := len(data.Memories) - 1; i >= 0 && i >= len(data.Memories)-*k; i-- {
			if wanted[data.Memories[i].ID] {
				baseline += 1 / float64(len(wanted))
			}
		}
		recall += r.Recall
		mrr += r.RR
		results = append(results, r)
	}
	scale := map[string]any{"status": "not_run"}
	if *iterations > 0 {
		local, err := ml.Open(filepath.Join(dir, "scale.db"), nil)
		if err != nil {
			return err
		}
		defer local.Close()
		start := time.Now()
		for i := 0; i < *size; i++ {
			if _, err := local.Put(ctx, scope, ml.Input{ID: fmt.Sprintf("fact-%06d", i), Text: fmt.Sprintf("Synthetic project item %d concerns gardening and plants.", i)}); err != nil {
				return err
			}
		}
		writeMS := float64(time.Since(start).Microseconds()) / 1000
		if _, err := local.Search(ctx, scope, "gardening plants", "lexical", *k); err != nil {
			return err
		}
		times := make([]float64, 0, *iterations)
		for i := 0; i < *iterations; i++ {
			start := time.Now()
			if _, err := local.Search(ctx, scope, "gardening plants", "lexical", *k); err != nil {
				return err
			}
			times = append(times, float64(time.Since(start).Microseconds())/1000)
		}
		sort.Float64s(times)
		scale = map[string]any{"status": "measured", "mode": "lexical", "records": *size, "queries": *iterations, "p50_ms": times[(len(times)-1)*50/100], "p95_ms": times[(len(times)-1)*95/100], "individual_durable_writes_total_ms": writeMS, "note": "Warm single-process sequential latency including database reads; not HTTP, cold start, concurrency or provider latency."}
	}
	sum := sha256.Sum256(raw)
	n := float64(len(results))
	report := map[string]any{"dataset": data.Name, "dataset_sha256": hex.EncodeToString(sum[:]), "measured_at": time.Now().UTC().Format(time.RFC3339), "go_version": runtime.Version(), "os": runtime.GOOS, "arch": runtime.GOARCH, "logical_cpus": runtime.NumCPU(), "mode": *mode, "k": *k, "queries": len(results), "memories": len(data.Memories), "mean_recall_at_k": recall / n, "mrr_at_k": mrr / n, "recency_baseline_recall_at_k": baseline / n, "ingest_ms": ingestMS, "results": results, "scale_latency": scale, "limitations": "Small authored retrieval diagnostic, not LoCoMo/LongMemEval, not answer accuracy. No statistical confidence claim. Provider pricing is not estimated."}
	if p != nil {
		report["embedding_model"] = os.Getenv("MIND_LAYER_EMBEDDING_MODEL")
		report["provider_base_url"] = os.Getenv("MIND_LAYER_BASE_URL")
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}
