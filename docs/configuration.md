# Configuration

## Offline memory

```sh
go run ./cmd/mind-layer -addr 127.0.0.1:8080 -data data/memory.db
```

Explicit storage, BM25 search, personality, context, export and deletion work
without any provider. Data is stored in one Bolt database file. A second process
cannot open the same database for writing; the open attempt times out. Multiple
requests in one process are supported. Horizontal scaling is outside this edition.

## Direct providers

| Environment variable | Purpose |
| --- | --- |
| `MIND_LAYER_BASE_URL` | Trusted OpenAI-compatible API base URL, without `/chat/completions` or `/embeddings` |
| `MIND_LAYER_API_KEY` | Your provider credential; optional for local servers |
| `MIND_LAYER_CHAT_MODEL` | Explicit model for JSON fact extraction and replies |
| `MIND_LAYER_EMBEDDING_MODEL` | Explicit model for stored/query vectors |
| `MIND_LAYER_TOKEN` | Optional service-owner token; 32+ characters required for public binding |

The server reads process environment variables. It does not automatically load
`.env` files. Set a base URL and at least one model to enable a provider. Model
defaults are deliberately not guessed. HTTPS is required except for a loopback
IP endpoint such as `http://127.0.0.1:11434/v1`. Redirects are refused.

### Gemini example, verified with a synthetic smoke test

```sh
export MIND_LAYER_BASE_URL=https://generativelanguage.googleapis.com/v1beta/openai
export MIND_LAYER_API_KEY="$GEMINI_API_KEY"
export MIND_LAYER_CHAT_MODEL=gemini-3.8-flash
export MIND_LAYER_EMBEDDING_MODEL=gemini-embedding-001
go run ./cmd/mind-layer -data data/semantic.db
```

This uses Google's documented [OpenAI-compatible API](https://ai.google.dev/gemini-api/docs/openai).
It calls Google directly. Provider availability and charges depend on your account.

### Other compatible providers

Set the base URL, credential and models supplied by your provider. The adapter
uses [chat completions](https://developers.openai.com/api/reference/resources/chat/subresources/completions/methods/create)
and [float embeddings](https://developers.openai.com/api/reference/resources/embeddings/methods/create).
Extraction requires `response_format: {"type":"json_object"}`. Support varies
between compatible implementations. The protocol is contract-tested locally;
only the recorded Gemini configuration has been live-tested in this preparation.

## Embedding lifecycle

Configuring an embedding model makes new writes embed their text before commit.
Embedding errors fail the write. Extraction embeds the returned facts when enabled.
Lexical search remains offline; semantic and hybrid queries call the embedding
provider. Chat calls the chat model after retrieval. No background jobs run.

Vectors retain their provider-base/model identity. Semantic/hybrid search rejects
a scope containing missing or incompatible vectors. Reinsert those memories
under the same IDs with the current model, or use a fresh database. There is no
automatic re-embedding job. A model alias changing behavior under the same name
cannot be detected; use stable model versions where available.

## Remote access

Bind to loopback unless your application needs remote access. For remote access,
set an unpredictable owner token of at least 32 characters and use TLS at a trusted
reverse proxy. The token grants access to every scope. It does not authenticate
individual end users. Browser-origin requests are rejected; use your backend.
