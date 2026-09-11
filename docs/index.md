# Mind Layer

Give an agent persistent memory without a Sonzai account or hosted subscription.
Run one Go process with an embedded database. Store facts locally, retrieve them
by keyword, and optionally use your own model-provider key for semantic retrieval,
fact extraction and replies.

This standalone edition reuses Sonzai's BM25 and multilingual tokenization code.
Its persistence, API and provider integration are new. It does not include the
full hosted platform. The [capability matrix](capabilities.md) records the exact
scope and the tests that support it.

## Ask your agent about your project

Copy this into ChatGPT, Claude, or your coding agent. It will read the guide,
ask about what you're building, and suggest useful features with links to the docs.

<!-- agent-copy -->

[How to use this with your agent](use-with-your-agent.md).

## Try an evolving character

Run `make demo`, open `http://127.0.0.1:8090`, and choose interactions with Moss.
The example changes fictional traits, personality and replies while retaining
shared memories. Chat and inspect the context behind each reply. Offline recall needs no provider; open-ended dialogue uses your configured model. [Open the character guide](character-demo.md).

![Character evolution](assets/character-evolution.svg)

## Quick start

From the repository root, with Go 1.26 installed:

```sh
go run ./cmd/mind-layer
```

In another terminal:

```sh
python3 examples/quickstart.py
```

No key is needed for this example. Facts persist in `data/memory.db` across restarts.
The initial Go build needs network access to download dependencies. Runtime
keyword memory operations do not make network calls.

## A conversation ends; the memory stays

1. Store: “The user is vegetarian and cooks dinner for two people.”
2. Later, query: “vegetarian dinner”.
3. Supply the retrieved fact to your agent, or call the optional chat endpoint.

For paraphrases such as “What should guide my meal suggestions?”, configure an
embedding model and use semantic or hybrid search. Keyword search alone cannot
reliably connect phrases that use different words; the benchmark demonstrates this.

## Ownership and cost

The source is Apache-2.0. There is no Sonzai payment, license key, license server,
metering or telemetry dependency. You own and operate the database. Optional
provider usage is billed by your provider; a compatible local model server can
run without a paid API. Hardware and hosting remain your responsibility.

Continue with [Configuration](configuration.md), [API](api.md),
[Capabilities](capabilities.md), [Benchmarks](benchmarks.md), and
[Data lifecycle](data-lifecycle.md).
