# Benchmarks

These are measured diagnostics for this standalone edition. They are not scores
for the full hosted platform and do not establish general memory quality.

## Retrieval quality: synthetic-v1

The unchanged authored dataset contains 12 synthetic facts and
12 queries, including exact terms, inflection, paraphrases and one
two-evidence query. Both runs use k=3. No real users or customer data are
included. The query labels are used only for scoring, not ingestion or retrieval.

| Retrieval mode | Mean recall@3 | MRR@3 | Provider |
| --- | --- | --- | --- |
| Most recently inserted 3 facts baseline | 25.00% | Not measured | None |
| BM25 lexical | 75.00% | 0.7500 | None |
| Hybrid BM25 + embeddings | 100.00% | 0.9583 | gemini-embedding-001 |

Recall@3 is the fraction of required evidence memories retrieved, averaged over
queries. MRR@3 is the reciprocal rank of the first relevant memory, or zero if
none is retrieved. A high score on this tiny authored set is not a statistically
reliable quality estimate. Lexical search misses several paraphrases; hybrid
retrieval recovers them in this run. There is no answer-generation judge here.

Dataset SHA-256: `297811d32167a59cfb1fe7bd1bfcdc03633c5b8cc1606e13dfff1d810fcd489d`.
Lexical measured: `2026-09-10T19:56:11Z`. Hybrid measured: `2026-09-10T19:57:06Z`.
Environment: darwin/arm64, 14 logical CPUs,
go1.26.0. Models and machines can change results.

## Local latency

The separate scale test contains 1,000 authored synthetic records,
with 30 sequential warmed-up keyword queries in one process.
It includes database reads and retrieval, without HTTP or provider calls.

| Measurement | Result |
| --- | --- |
| p50 query latency | 3.267 ms |
| p95 query latency | 4.380 ms |
| 1,000 individual durable writes | 9.275 seconds |

This is not cold-start latency, concurrent throughput, a vector-index scaling
test, or a provider-latency claim. Multilingual tokenizers load models lazily.
Semantic search scans the selected scope's vectors; it is not an ANN service.

## Reproduce

```sh
go run ./cmd/benchmark
# Requires configured provider environment and incurs provider usage:
go run ./cmd/benchmark -mode hybrid -live -iterations 0
# Optional separate provider smoke test (synthetic data only):
MIND_LAYER_LIVE_TEST=1 go test -run '^TestLiveProvider$' -count=1 -v .
```

Raw per-query results live in `benchmarks/results/lexical.json` and
`benchmarks/results/hybrid.json`. The fixture is `benchmarks/synthetic.json`.
The live smoke report is `benchmarks/results/live-smoke.json`; it records one
passed Gemini extraction/retrieval/answer test, not a quality leaderboard score.
No token-price estimates or competitor numbers are invented.

## Character chat smoke eval

`make chat-eval` runs five authored messages through a real local HTTP server,
then restarts it to verify exact chat receipts. It checks original fact and
correction retrieval, mode labels, and unchanged story traits. To enable model
replies explicitly, run `python3 scripts/run_chat_eval.py --live` with provider
configuration. This can incur provider charges.

Recorded results are `benchmarks/results/chat-offline.json` and
`benchmarks/results/chat-live.json`. Read the unedited live responses to assess
correction handling and unknown-fact honesty. The automated checks measure
structure and context selection; they do not judge answer quality. This small
fixture is newly authored for the standalone chat path. Existing hosted monolith
eval suites depend on services outside this release and are not represented by
these scores.

## Public benchmarks: not yet measured

| Benchmark | Why relevant | This edition's status |
| --- | --- | --- |
| [LongMemEval](https://github.com/xiaowu0162/LongMemEval) | Long-term memory QA, updates, temporal reasoning, multi-session evidence and abstention | Not run; no score claimed |
| [LoCoMo](https://github.com/snap-research/locomo) | Long conversational histories with QA and evidence annotations | Not run; no score claimed |

For a comparable public result, pin the dataset revision and split, include every
evaluation case (including abstention), document ingestion/chunking and retrieval
settings, and use the benchmark's official scoring protocol. Report provider
models, token usage/cost, timing and all failures. Retrieval recall alone must not
be presented as answer accuracy. External datasets and generated transcripts
must remain outside the public source tree; do not bundle them as fixtures.
