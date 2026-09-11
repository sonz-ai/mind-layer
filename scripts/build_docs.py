#!/usr/bin/env python3
"""Build static docs and plaintext bundles from reviewed Markdown and evidence."""
import html
import json
from pathlib import Path
import re
import shutil

import markdown
from build_diagrams import build as build_diagrams

root = Path(__file__).resolve().parents[1]
docs = root / "docs"
site = root / "site"
site.mkdir(exist_ok=True)
build_diagrams(root)
(site / "assets").mkdir(exist_ok=True)
for asset in (docs / "assets").glob("*.svg"):
    shutil.copyfile(asset, site / "assets" / asset.name)
caps = json.loads((docs / "capabilities.json").read_text())
legacy = json.loads((docs / "hosted-scope.json").read_text())
lines = ["# Tested capabilities", "", "This is a scoped standalone edition, not hosted-platform parity. Each supported",
         "claim below maps to named tests. `make verify` requires all of them to pass",
         "with the race detector. Mock-provider tests check integration contracts; the",
         "separate live smoke and benchmark reports measure actual provider behavior.", "",
         "| Capability | Provider requirement | Tests |", "| --- | --- | --- |"]
for cap in caps:
    lines.append(f"| {cap['claim']} | {cap['provider']} | " + ", ".join(f"`{t}`" for t in cap['tests']) + " |")
lines += ["", "## Relationship to existing hosted docs", "",
          "The following inventory maps the top-level English pages from the existing",
          "Mind Layer docs to this release. It does not import their hosted API claims,",
          "screenshots or examples. Nested hosted guides and reference endpoints are",
          "not included unless this edition's API and tests explicitly document them.", "",
          "| Existing page | Status | Standalone scope |", "| --- | --- | --- |"]
for item in legacy:
    lines.append(f"| {item['title']} | {item['status']} | {item['scope']} |")
lines += ["", "## Explicit limitations", "",
          "No automatic personality evolution, graph reasoning, semantic contradiction",
          "resolution, memory decay/consolidation jobs, shared organization memory,",
          "proactive messages, voice, external tool execution, MCP server, or hosted SDK",
          "compatibility is claimed. Personality is configured text. Sessions are labels.",
          "Single-owner authentication is not end-user authorization. Logical deletion",
          "is not secure erasure. This release includes no production-scale certification.", ""]
(docs / "capabilities.md").write_text("\n".join(lines))

lex = json.loads((root / "benchmarks/results/lexical.json").read_text())
hybrid = json.loads((root / "benchmarks/results/hybrid.json").read_text())
scale = lex["scale_latency"]
bench = f'''# Benchmarks

These are measured diagnostics for this standalone edition. They are not scores
for the full hosted platform and do not establish general memory quality.

## Retrieval quality: synthetic-v1

The unchanged authored dataset contains {lex['memories']} synthetic facts and
{lex['queries']} queries, including exact terms, inflection, paraphrases and one
two-evidence query. Both runs use k={lex['k']}. No real users or customer data are
included. The query labels are used only for scoring, not ingestion or retrieval.

| Retrieval mode | Mean recall@3 | MRR@3 | Provider |
| --- | --- | --- | --- |
| Most recently inserted 3 facts baseline | {lex['recency_baseline_recall_at_k']:.2%} | Not measured | None |
| BM25 lexical | {lex['mean_recall_at_k']:.2%} | {lex['mrr_at_k']:.4f} | None |
| Hybrid BM25 + embeddings | {hybrid['mean_recall_at_k']:.2%} | {hybrid['mrr_at_k']:.4f} | {hybrid['embedding_model']} |

Recall@3 is the fraction of required evidence memories retrieved, averaged over
queries. MRR@3 is the reciprocal rank of the first relevant memory, or zero if
none is retrieved. A high score on this tiny authored set is not a statistically
reliable quality estimate. Lexical search misses several paraphrases; hybrid
retrieval recovers them in this run. There is no answer-generation judge here.

Dataset SHA-256: `{lex['dataset_sha256']}`.
Lexical measured: `{lex['measured_at']}`. Hybrid measured: `{hybrid['measured_at']}`.
Environment: {lex['os']}/{lex['arch']}, {lex['logical_cpus']} logical CPUs,
{lex['go_version']}. Models and machines can change results.

## Local latency

The separate scale test contains {scale['records']:,} authored synthetic records,
with {scale['queries']} sequential warmed-up keyword queries in one process.
It includes database reads and retrieval, without HTTP or provider calls.

| Measurement | Result |
| --- | --- |
| p50 query latency | {scale['p50_ms']:.3f} ms |
| p95 query latency | {scale['p95_ms']:.3f} ms |
| {scale['records']:,} individual durable writes | {scale['individual_durable_writes_total_ms'] / 1000:.3f} seconds |

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
'''
(docs / "benchmarks.md").write_text(bench)

pages = ["index", "use-with-your-agent", "character-demo", "capabilities", "configuration", "api", "benchmarks", "data-lifecycle"]
agent_prompt = (docs / "agent-prompt.txt").read_text().strip()
for name in ["agent-setup.md", "agent-prompt.txt"]:
    shutil.copyfile(docs / name, site / name)
(site / "_headers").write_text("/agent-setup.md\n  Content-Type: text/plain; charset=utf-8\n/agent-prompt.txt\n  Content-Type: text/plain; charset=utf-8\n")
shutil.copyfile(docs / "assets/agent-copy.js", site / "assets/agent-copy.js")
copy_card = f'''<div class="agent-copy">
<label for="agent-prompt">Instruction for your agent</label>
<textarea id="agent-prompt" readonly rows="7" spellcheck="false">{html.escape(agent_prompt)}</textarea>
<div class="agent-actions"><button type="button" id="copy-agent-prompt">Copy instruction</button>
<a href="https://docs.sonz.ai/agent-setup.md">Read the Markdown guide</a></div>
<p id="agent-copy-status" role="status" aria-live="polite">Paste into ChatGPT, Claude, or your coding agent.</p>
</div>'''
titles = {p: (docs / f"{p}.md").read_text().splitlines()[0].removeprefix("# ") for p in pages}
css = """body{margin:0;background:#fafaf8;color:#242622;font:17px/1.65 system-ui,sans-serif}header{border-bottom:1px solid #deded8;padding:22px 5%;display:flex;gap:25px;flex-wrap:wrap}header a{color:#244d3e;text-decoration:none}header strong{margin-right:auto}main{max-width:1050px;margin:48px auto;padding:0 24px 80px}h1{font-size:44px;line-height:1.1;letter-spacing:-1.5px}h2{margin-top:45px;line-height:1.25}p,li{max-width:85ch}a{color:#21674d}pre{background:#eeeFEA;padding:20px;border-radius:8px;overflow:auto}code{font-size:14px}table{display:block;overflow-x:auto;border-collapse:collapse;font-size:14px;margin:25px 0}td,th{border:1px solid #d9ddd5;padding:12px;text-align:left;vertical-align:top;min-width:125px}th{background:#edf1e9}footer{border-top:1px solid #deded8;padding:25px 5%;font-size:14px}@media(max-width:600px){header{gap:12px;font-size:14px}h1{font-size:34px}main{margin-top:30px}}"""
for page in pages:
    source = (docs / f"{page}.md").read_text()
    source = re.sub(r"\]\(([a-z-]+)\.md\)", r"](\1.html)", source)
    body = markdown.markdown(source, extensions=["fenced_code", "tables"])
    has_copy_card = "<!-- agent-copy -->" in body
    body = body.replace("<!-- agent-copy -->", copy_card)
    extra_css = '.agent-copy{border:1px solid #cbd7cc;border-radius:10px;padding:24px;margin:28px 0;background:#f0f4ed}.agent-copy label{display:block;font-weight:600;margin-bottom:12px}.agent-copy textarea{box-sizing:border-box;width:100%;resize:vertical;padding:14px;font:15px/1.6 system-ui;color:#242622;background:#fff;border:1px solid #b9c9bd;border-radius:6px}.agent-actions{display:flex;gap:20px;align-items:center;flex-wrap:wrap;margin-top:16px}.agent-copy button{background:#244d3e;color:white;border:0;border-radius:6px;padding:13px 20px;font:600 16px system-ui;cursor:pointer}.agent-copy button:hover{background:#183e2e}.agent-copy :focus-visible{outline:3px solid #b07717;outline-offset:3px}#agent-copy-status{font-size:14px;margin-bottom:0}'
    script = '<script src="assets/agent-copy.js" defer></script>' if has_copy_card else ''
    nav = "".join(f'<a href="{p}.html">{html.escape(titles[p])}</a>' for p in pages if p != "index")
    (site / f"{page}.html").write_text(f'<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{html.escape(titles[page])} | Mind Layer</title><style>{css}{extra_css}</style>{script}</head><body><header><strong><a href="index.html">MIND LAYER</a></strong>{nav}</header><main>{body}</main><footer>Apache-2.0 · Run locally · No Sonzai account required</footer></body></html>')
    (site / f"{page}.md").write_text((docs / f"{page}.md").read_text().replace("<!-- agent-copy -->", agent_prompt))
(site / "llms.txt").write_text("# Mind Layer standalone docs\n\n" + "\n".join(f"- [{titles[p]}](https://docs.sonz.ai/{p}.md)" for p in pages) + "\n- [Agent project guide](https://docs.sonz.ai/agent-setup.md)\n")
(site / "llms-full.txt").write_text("\n\n".join((site / f"{p}.md").read_text() for p in pages))
print(f"Built {len(pages)} static pages and Markdown/LLM exports in site/ (no analytics).")
