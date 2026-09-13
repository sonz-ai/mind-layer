# Mind Layer

Memory for AI characters and agents. Store facts, retrieve them in later
conversations, and inspect the context supplied to your model.

Run it locally, keep the database, and connect your model. Mind Layer is
Apache-2.0 and needs no Sonzai account or subscription. Keyword retrieval works
without API keys; optional model and embedding calls may incur provider costs.

[Watch Moss · 32-second demo](https://sonz.ai/demo) ·
[Documentation](https://docs.sonz.ai) ·
[Use it with your coding agent](https://docs.sonz.ai/use-with-your-agent)

## Try it with Moss

Moss is the included workshop companion. Share a fact, change a detail, and
inspect the memories and recent conversation turns behind its reply.

With **Go 1.26+** installed:

```sh
git clone https://github.com/sonz-ai/mind-layer.git
cd mind-layer
make demo
```

Open **http://127.0.0.1:8090**. No API key, Docker, Redis, or separate database
installation is needed. The first build downloads Go dependencies.

Offline chat saves messages and quotes matching memories. For open-ended replies,
[configure a model provider](docs/configuration.md), stop the offline demo, and run:

```sh
go run ./cmd/character-demo -live
```

Try telling Moss “My greenhouse is called Fern House,” then rename it to
“Cedar Room” and ask what it is called. Open **Inspect this reply** to see the
context it received. The last four conversation turns also contribute to replies;
the demo is not an isolated retrieval benchmark.

Story interactions change fictional traits through explicit game rules. The
model can write dialogue, but it does not choose those trait changes. Messages,
memories, and traits persist in `data/character-demo.db` across restarts.
[Demo setup and limits](docs/character-demo.md).

## Use it in your project

Run the local REST API:

```sh
make run
```

In another terminal, run the example:

```sh
python3 examples/quickstart.py
```

The API listens on `127.0.0.1:8080` and stores data in `data/memory.db`.
The example saves and retrieves a synthetic dietary preference without a provider
key. You can also use Mind Layer as a Go library.
[API reference and examples](docs/api.md).

Mind Layer provides:

- **Persistent memory:** store, update, export, and delete records within agent/user scopes.
- **Keyword retrieval:** local BM25 search with multilingual tokenization.
- **Semantic and hybrid retrieval:** optional embeddings from your configured provider.
- **Explicit personality settings:** persist personality and commit memory/personality changes atomically.
- **Context inspection in Moss:** see retrieved memories, personality, and recent turns supplied for a reply.

[Capabilities and named tests](docs/capabilities.md) describe the supported behavior.
This standalone edition includes a subset of the hosted Sonzai platform.

## How it runs

One Go process owns an embedded Bolt database. The keyword index rebuilds from
stored records at startup. No Sonzai billing, telemetry, license server, or
control-plane service is required.

Model requests go directly to the provider you configure. Compatible local model
servers are also supported. Existing keyword-only memories need embeddings before
semantic retrieval can use them; changing embedding models requires re-embedding.
Nothing silently sends your existing database to a provider.
[Provider configuration](docs/configuration.md).

Agent/user scope IDs separate records; your application must authorize access.
The optional service token is an owner credential for all scopes. Storage is not
application-encrypted, and logical deletion is not secure erasure of backups or
provider logs. [Security](SECURITY.md) · [Data lifecycle](docs/data-lifecycle.md).

## Tests and benchmarks

```sh
make verify          # named capability tests, race detector, and go vet
make chat-eval       # synthetic chat, correction recall, and server restart
make demo-scenario   # reproducible character story
make benchmark       # offline retrieval diagnostic and latency measurement
```

Recorded results from the included synthetic diagnostic:

| Measurement | Recorded result | Conditions |
| --- | ---: | --- |
| Keyword mean recall@3 | 75% | 12 authored synthetic queries |
| Hybrid mean recall@3 | 100% | Same 12 queries, Gemini embeddings |
| Recency baseline recall@3 | 25% | Most recently inserted 3 facts |
| Keyword query p50 / p95 | 3.27 / 4.38 ms | 1,000 records; 30 warm sequential queries |

These are small diagnostics recorded on macOS/ARM64, not general answer accuracy
or production load tests. LoCoMo and LongMemEval have not been run. Live provider
checks are opt-in and may cost money.
[Methodology and reproduction](docs/benchmarks.md) · [Raw results](benchmarks/results/).

## Documentation and contributing

Start with the [docs](https://docs.sonz.ai), or give your coding agent the
[Markdown setup guide](docs/agent-setup.md) to plan a small integration.

For development, read [Contributing](CONTRIBUTING.md) and the
[public writing guide](PUBLIC_STYLE.md). To build the static docs locally:

```sh
python3 -m venv .venv
.venv/bin/pip install -r requirements-docs.txt
make docs
```

Before publishing, run `python3 scripts/check_release.py` with Gitleaks installed
and review the final diff. Automated scanning cannot prove every kind of PII is
absent. [Publishing guide](docs/PUBLISHING.md).

Licensed under [Apache-2.0](LICENSE), with [upstream notices](THIRD_PARTY_NOTICES.md).
