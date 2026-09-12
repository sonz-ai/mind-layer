# Memory for your agent

Mind Layer stores facts your agent can retrieve in later conversations. Run it
locally, keep the database, and inspect the context used for a reply. No Sonzai
account is required.

[Use it with your agent](use-with-your-agent.md) or try Moss, a character that
remembers your interactions.

## Meet Moss

Run `make demo` and open `http://127.0.0.1:8090`. Choose an interaction, talk to
Moss, and look at the memories behind a reply. Story choices change its traits
through explicit rules. Offline mode recalls stored facts; open-ended dialogue
uses your configured model. [Read the demo guide](character-demo.md).

## Run Mind Layer

With Go 1.26 installed, start the service from the repository root:

```sh
go run ./cmd/mind-layer
```

Then run the example in another terminal:

```sh
python3 examples/quickstart.py
```

No provider key is needed. Facts persist in `data/memory.db` across restarts.
The first build downloads dependencies; keyword memory operations run locally
without network calls.

## Choose how to retrieve memories

Keyword search can find a stored fact about “vegetarian dinner” when your agent
asks for those words. For differently worded questions, configure an embedding
model and use semantic or hybrid search. You can also configure a chat model to
extract facts and write replies. [Configure a provider](configuration.md).

Mind Layer runs as one Go process with an embedded database. It is Apache-2.0,
with no Sonzai subscription, license server, metering or telemetry dependency.
Optional model usage is billed by your provider; you can also use a compatible
local model server. You cover your own hardware and hosting.

This standalone edition reuses Sonzai’s BM25 and multilingual tokenization code;
its persistence, API and provider integration are new. It includes a subset of
the hosted platform. Read the [tested capabilities](capabilities.md),
[API reference](api.md), [benchmarks](benchmarks.md), and
[data lifecycle](data-lifecycle.md) for the exact scope.
