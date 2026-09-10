# REST API

Base URL: `http://127.0.0.1:8080`. All POST/PUT requests require
`Content-Type: application/json`. If configured, send the service owner token
as `Authorization: Bearer <owner-token>` from your backend.

Every memory operation requires an explicit scope:

```json
{"scope":{"agent":"assistant","user":"synthetic-user"}}
```

Agent/user IDs accept 1–128 ASCII letters, digits, underscores, dots and hyphens.
They are namespaces, not authentication claims. The caller must authorize them.
POST/PUT request bodies have a 1 MiB limit and reject unknown fields or trailing JSON.

## Store or update memory

`POST /v1/memories`

```json
{
  "scope":{"agent":"assistant","user":"synthetic-user"},
  "id":"diet",
  "text":"The user is vegetarian and cooks dinner for two people.",
  "session":"first-conversation"
}
```

Returns the stored memory. A supplied ID updates that memory inside this scope.
Omit the ID to derive one from trimmed text: exact duplicates collapse, but
semantic duplicates and contradictions are not automatically reconciled. Text
must be nonempty and at most 8,192 bytes. Session labels are optional provenance,
at most 128 bytes; they do not create a managed conversation runtime.

## Search and build context

`POST /v1/search` returns scored memories. `POST /v1/context` returns the same
retrieval results plus the scope's configured personality.

```json
{
  "scope":{"agent":"assistant","user":"synthetic-user"},
  "query":"vegetarian dinner",
  "mode":"lexical",
  "limit":5
}
```

Modes: `lexical` (default), `semantic`, `hybrid`. Limit defaults to 5, range 1–100.
Queries are nonempty and at most 8,192 bytes. Semantic/hybrid require embeddings.
Hybrid combines lexical and vector ranks with reciprocal rank fusion (constant 60).
Scores differ across modes and are not calibrated probabilities. Semantic retrieval
can return irrelevant positive-similarity hits; there is no calibrated abstention gate.

## Configure personality

`PUT /v1/personality`

```json
{"scope":{"agent":"assistant","user":"synthetic-user"},"personality":"Be warm, concise and practical."}
```

Personality is explicitly configured text (up to 8,192 bytes), persisted per scope.
There is no Big Five inference, automatic trait evolution or mood simulation.

## Extract and chat

`POST /v1/ingest` uses the configured chat provider to extract up to 20 durable
facts from supplied text. The raw transcript is not stored by this endpoint.
Provider retention is separate from local retention.

```json
{"scope":{"agent":"assistant","user":"synthetic-user"},"text":"User: I am vegetarian and cook for two.","session":"first-conversation"}
```

`POST /v1/chat` accepts the same input as `/v1/search`. It returns
`{"answer":"...","context":{"personality":"...","memories":[]}}`.
It is a single non-streaming completion using retrieved context, not a managed
multi-turn conversation. It does not automatically store the query or response.
Call ingest explicitly for conversations you choose to remember. Extraction can
be wrong; review facts and use explicit ID updates for corrections.

## Export and delete

| Method and path | Result |
| --- | --- |
| `GET /v1/export?agent=assistant&user=synthetic-user` | All live memories, vectors and personality for the scope |
| `DELETE /v1/memories?agent=assistant&user=synthetic-user&id=diet` | Delete one live record and its search entry |
| `DELETE /v1/scope?agent=assistant&user=synthetic-user` | Delete all live records, indexes and personality for the scope |
| `GET /healthz` | Process liveness; not a provider connectivity test |

Deletion is logical, not secure disk erasure. Export can contain personal data
you intentionally supplied; treat exports as private. There is no bulk restore
endpoint, but explicit memories can be reinserted with their IDs through the API.

Successful operations return HTTP 200. Invalid inputs return 400; unauthorized
owner-token requests 401; rejected browser/host requests 403; a missing memory
delete returns 404. Store/provider failures return a generic 500 without provider
response bodies. Unsupported methods/routes use standard HTTP 405/404 responses.
