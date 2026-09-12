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
<textarea id="agent-prompt" readonly rows="3" spellcheck="false">{html.escape(agent_prompt)}</textarea>
<div class="agent-actions"><button type="button" id="copy-agent-prompt">Copy instruction</button>
<a href="https://docs.sonz.ai/agent-setup.md">Read the Markdown guide</a></div>
<p id="agent-copy-status" role="status" aria-live="polite">Paste into ChatGPT, Claude, or your coding agent.</p>
</div>'''
titles = {p: (docs / f"{p}.md").read_text().splitlines()[0].removeprefix("# ") for p in pages}
css = '*{box-sizing:border-box}body{margin:0;background:#fcfbf8;color:#292824;font:19px/1.65 Charter,"Bitstream Charter","Iowan Old Style",Georgia,serif;-webkit-font-smoothing:antialiased}header{max-width:1000px;margin:auto;border-bottom:1px solid #dfded8;padding:24px;display:flex;align-items:baseline;gap:24px}header a{color:inherit;text-decoration:none}header strong{margin-right:auto;font-size:23px;font-weight:500;white-space:nowrap}nav{display:flex;gap:20px;align-items:baseline;font:14px/1.5 system-ui,sans-serif}nav a:hover,summary:hover{text-decoration:underline}nav details{position:relative}nav summary{cursor:pointer}nav details div{position:absolute;right:0;top:30px;z-index:1;min-width:210px;padding:12px 18px;background:#fcfbf8;border:1px solid #dfded8}nav details a{display:block;padding:7px 0}main{max-width:736px;margin:64px auto;padding:0 24px 64px}h1,h2,h3{font-weight:500;line-height:1.2}h1{font-size:clamp(36px,6vw,48px);letter-spacing:-1.2px;margin:0 0 24px}h2{font-size:29px;margin:48px 0 18px}h3{font-size:23px;margin-top:32px}p,li{max-width:68ch}a{color:#345346;text-underline-offset:3px}a:hover{text-decoration-thickness:2px}:focus-visible{outline:2px solid #345346;outline-offset:4px}pre{background:#f1f0eb;padding:18px;overflow:auto;line-height:1.55}code{font:14px/1.6 ui-monospace,SFMono-Regular,Consolas,monospace}table{display:block;overflow-x:auto;border-collapse:collapse;font:14px/1.5 system-ui,sans-serif;margin:25px 0}td,th{border-bottom:1px solid #dfded8;padding:12px;text-align:left;vertical-align:top;min-width:125px}th{font-weight:600;background:#f1f0eb}img,svg{max-width:100%;height:auto}footer{max-width:1000px;margin:auto;border-top:1px solid #dfded8;padding:24px;font:13px/1.6 system-ui,sans-serif;color:#65635b}.skip-link{position:absolute;left:24px;top:-100px}.skip-link:focus{top:8px;background:#fcfbf8;padding:8px}@media(max-width:600px){header{align-items:flex-start;flex-wrap:wrap;gap:14px;padding:20px}header strong{width:100%}nav{gap:22px}nav details div{right:0;left:auto;max-width:calc(100vw - 40px)}main{margin-top:40px;padding:0 20px 48px}body{font-size:18px}}'
for page in pages:
    source = (docs / f"{page}.md").read_text()
    source = re.sub(r"\]\(([a-z-]+)\.md\)", r"](\1.html)", source)
    body = markdown.markdown(source, extensions=["fenced_code", "tables"])
    has_copy_card = "<!-- agent-copy -->" in body
    body = body.replace("<!-- agent-copy -->", copy_card)
    extra_css = '.agent-copy{border-top:1px solid #dfded8;border-bottom:1px solid #dfded8;padding:22px 0;margin:28px 0}.agent-copy label{display:block;font:500 14px/1.5 system-ui,sans-serif;margin-bottom:10px}.agent-copy textarea{width:100%;resize:vertical;padding:14px;font:16px/1.6 Charter,Georgia,serif;color:#292824;background:#fff;border:1px solid #cfcec6;border-radius:2px}.agent-actions{display:flex;gap:18px;align-items:center;flex-wrap:wrap;margin-top:16px;font:14px/1.5 system-ui,sans-serif}.agent-copy button{background:#345346;color:#fff;border:1px solid #345346;border-radius:3px;padding:11px 16px;font:500 14px/1.5 system-ui,sans-serif;cursor:pointer}.agent-copy button:hover{background:#273f35}#agent-copy-status{font:13px/1.5 system-ui,sans-serif;color:#65635b;margin-bottom:0}'
    script = '<script src="assets/agent-copy.js" defer></script>' if has_copy_card else ''
    def nav_link(slug, label):
        current = ' aria-current="page"' if page == slug else ''
        return f'<a href="{slug}.html"{current}>{html.escape(label)}</a>'
    primary = nav_link("use-with-your-agent", "Get started") + nav_link("character-demo", "Moss demo")
    reference = "".join(nav_link(p, titles[p]) for p in pages if p not in {"index", "use-with-your-agent", "character-demo"})
    nav = f'<nav aria-label="Documentation">{primary}<details><summary>Reference</summary><div>{reference}</div></details></nav>'
    (site / f"{page}.html").write_text(f'<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{html.escape(titles[page])} | Mind Layer</title><style>{css}{extra_css}</style>{script}</head><body><a class="skip-link" href="#content">Skip to content</a><header><strong><a href="index.html">Mind Layer</a></strong>{nav}</header><main id="content">{body}</main><footer>Mind Layer is Apache-2.0. Run it locally without a Sonzai account.</footer></body></html>')
    (site / f"{page}.md").write_text((docs / f"{page}.md").read_text().replace("<!-- agent-copy -->", agent_prompt))
(site / "llms.txt").write_text("# Mind Layer standalone docs\n\n" + "\n".join(f"- [{titles[p]}](https://docs.sonz.ai/{p}.md)" for p in pages) + "\n- [Agent project guide](https://docs.sonz.ai/agent-setup.md)\n")
(site / "llms-full.txt").write_text("\n\n".join((site / f"{p}.md").read_text() for p in pages))
print(f"Built {len(pages)} static pages and Markdown/LLM exports in site/ (no analytics).")
