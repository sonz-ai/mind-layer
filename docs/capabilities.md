# Tested capabilities

This is a scoped standalone edition, not hosted-platform parity. Each supported
claim below maps to named tests. `make verify` requires all of them to pass
with the race detector. Mock-provider tests check integration contracts; the
separate live smoke and benchmark reports measure actual provider behavior.

| Capability | Provider requirement | Tests |
| --- | --- | --- |
| Store and retrieve explicit memories without provider keys or a Sonzai account | none | `TestNoProviderRequired` |
| Memories and personality survive database close and reopen | none | `TestDurableMemory` |
| Separate agent/user scopes for search, export and deletion | none | `TestScopeIsolation`, `TestSemanticAndHybridRecall` |
| BM25 keyword retrieval with stemming and no-match empty results | none | `TestLexicalRecall` |
| Direct-provider embeddings, cosine retrieval and reciprocal-rank hybrid search | embedding model | `TestSemanticAndHybridRecall`, `TestModelMismatchFailsClosed` |
| Update by memory ID and deduplicate exact trimmed text | none; embedding calls when configured | `TestExplicitUpdatesAndDeduplication` |
| Export a scope and logically delete individual memories or the whole scope | none | `TestExportAndDeletion` |
| Persist a manually configured personality and include it in retrieved context | none | `TestPersonalityContext`, `TestDurableMemory` |
| Extract facts from supplied text and answer with selected memories through the configured provider | chat model; optional embedding model | `TestExtractionAndGroundedChat`, `TestExtractionFailureDoesNotPersist` |
| REST memory, personality, retrieval, context, ingest, chat, export and delete endpoints | depends on endpoint | `TestHTTPMemoryLifecycle`, `TestHTTPProviderEndpoints` |
| Loopback defaults, owner-token authentication, browser-origin rejection and public-bind checks | none | `TestHTTPAuthenticationAndBrowserIsolation`, `TestHTTPRejectsInvalidInput` |
| Direct provider transport with URL validation, error redaction and no redirect following | mock provider for tests | `TestProviderURLValidation`, `TestProviderErrorRedaction`, `TestProviderRejectsRedirect`, `TestInvalidProviderOutput` |
| Concurrent in-process access and validation before batch writes | none | `TestConcurrentMemoryAccess`, `TestInputValidationAndAtomicBatch` |
| Commit application-driven personality changes with event memories atomically | none; embeddings if configured | `TestAtomicScopeUpdate` |
| Moss example evolves through explicit interactions, recalls them, persists after restart and rejects stale turns | none; optional generated dialogue | `TestCharacterEvolution`, `TestCharacterRestartAndIsolation`, `TestCharacterConcurrentRevision`, `TestCharacterBoundsAndInvalidAction`, `TestCharacterHTTP` |

## Relationship to existing hosted docs

The following inventory maps the top-level English pages from the existing
Mind Layer docs to this release. It does not import their hosted API claims,
screenshots or examples. Nested hosted guides and reference endpoints are
not included unless this edition's API and tests explicitly document them.

| Existing page | Status | Standalone scope |
| --- | --- | --- |
| Advance Time | not included | Not implemented or promised by this standalone edition. |
| Agent Insights | not included | Not implemented or promised by this standalone edition. |
| Architecture | limited subset | Replaced by the standalone single-process architecture. |
| Pattern 6: Hermes | not included | Not implemented or promised by this standalone edition. |
| Pattern 1: Managed Agent Runtime | not included | Not implemented or promised by this standalone edition. |
| Pattern 2: MCP | not included | Not implemented or promised by this standalone edition. |
| Pattern 3: OpenClaw | not included | Not implemented or promised by this standalone edition. |
| Pattern 5: Standalone Memory (Batch) | limited subset | Library batch writes are atomic; no hosted batch processing service. |
| Pattern 4: Standalone Memory (Real-Time) | limited subset | New local REST API, incompatible with the hosted SDK/API. |
| Conversations | limited subset | Single non-streaming reply with retrieved context; no managed conversation runtime. |
| Custom State | not included | Not implemented or promised by this standalone edition. |
| Custom Tools | not included | Not implemented or promised by this standalone edition. |
| Emotions & Mood | not included | Not implemented or promised by this standalone edition. |
| Events & Multi-Agent Dialogue | not included | Not implemented or promised by this standalone edition. |
| Generation | not included | Not implemented or promised by this standalone edition. |
| Sonzai Mind Layer | limited subset | Replaced by the tested standalone capability list. |
| Instances | limited subset | Agent/user namespace isolation only; no hosted workspace provisioning. |
| Inventory | not included | Not implemented or promised by this standalone edition. |
| Knowledge Analytics | not included | Not implemented or promised by this standalone edition. |
| Knowledge Base | limited subset | Plain text can be inserted as facts; no file ingestion or graph-building pipeline. |
| Memory | limited subset | Explicit facts, BM25 and optional semantic/hybrid retrieval only. |
| Multiplayer Memory | not included | Not implemented or promised by this standalone edition. |
| Notifications (Polling) | not included | Not implemented or promised by this standalone edition. |
| Omnichannel Messaging | not included | Not implemented or promised by this standalone edition. |
| Organization-Global Knowledge Base | not included | Not implemented or promised by this standalone edition. |
| Personality System | limited subset | Manually configured personality text only; no automatic Big Five evolution. |
| Priming | not included | Not implemented or promised by this standalone edition. |
| Proactive Messaging | not included | Not implemented or promised by this standalone edition. |
| Scheduled Reminders | not included | Not implemented or promised by this standalone edition. |
| Self-Improvement (Post-Processing) | not included | Not implemented or promised by this standalone edition. |
| Sessions | limited subset | Session provenance labels only; no session lifecycle or automatic end processing. |
| Shared Memory | not included | Not implemented or promised by this standalone edition. |
| User Personas | limited subset | Manual per-scope personality/context only; no inferred user persona. |
| Voice | not included | Not implemented or promised by this standalone edition. |
| Wakeups | not included | Not implemented or promised by this standalone edition. |
| Webhooks | not included | Not implemented or promised by this standalone edition. |

## Explicit limitations

No automatic personality evolution, graph reasoning, semantic contradiction
resolution, memory decay/consolidation jobs, shared organization memory,
proactive messages, voice, external tool execution, MCP server, or hosted SDK
compatibility is claimed. Personality is configured text. Sessions are labels.
Single-owner authentication is not end-user authorization. Logical deletion
is not secure erasure. This release includes no production-scale certification.
