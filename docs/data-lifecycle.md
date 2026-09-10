# Data lifecycle and architecture

```text
Your backend / Python example
        |
        v
Local REST service -> scoped Bolt database + BM25 index
        |
        +-- optional direct provider: embeddings / extraction / replies
```

The service has no Sonzai control-plane dependency. The Bolt database stores facts,
session labels, timestamps, personality text and optional vectors. In-memory BM25
indexes are rebuilt from persisted records at startup. Writes use database
transactions and update the index only after a successful commit. One process owns
one database; readers and writers in that process are synchronized.

## What leaves the machine

| Operation | External data flow |
| --- | --- |
| Offline write, lexical search, personality, context, export, delete | None |
| Write with embedding model configured | Fact text to configured provider |
| Semantic/hybrid search | Query text to configured provider |
| Ingest | Supplied transcript to chat provider; extracted facts to embedding provider if configured |
| Chat | Query, retrieved memories and personality to chat provider; query embedding if semantic/hybrid |

No background telemetry, Sonzai callbacks, billing, or license validation run.
Provider keys stay in process memory and are not stored in the database. There
are no automatic retries, so rate limits fail the operation rather than silently
creating repeated charges. Request timeouts are bounded.

## User-controlled memory

Use explicit scope IDs; authorize them in your application's backend. Use stable
memory IDs for corrections. Exact text deduplication is supported, but semantic
deduplication, temporal reasoning and contradiction resolution are outside this
edition. If facts change, update or delete the old fact explicitly.

Only supplied facts or model-extracted facts are stored. The ingest endpoint does
not save the raw transcript, and chat does not save conversation history. Models
can still extract inaccurate facts or follow adversarial instructions. The API
does not execute model tools; review extracted data before relying on it.

## Deletion and backups

Delete removes live records and index entries. It does not securely overwrite Bolt
free pages, filesystem snapshots, exported files, backups, or provider logs.
Use encrypted disks and manage retention outside this process. Never commit
database files or exports to the public repository. This is a local ownership
model, not a compliance certification or a guarantee that stored data lacks PII.
