# Evolving character project

Moss is a fictional workshop companion. The project demonstrates how an
application can evolve a character using durable Mind Layer memories and
personality configuration. The core does not infer personality changes by itself.

## Run

```sh
make demo
```

Open `http://127.0.0.1:8090`. No other server or API key is needed. Choose
interactions individually or click **Run the five-interaction story**. The page
shows the character's traits, dialogue, a live trend graph, causal explanations,
and the remembered event behind each change. Use **Recall shared experiences**
to query the actual stored memories.

![Moss changes after interactions](assets/character-evolution.svg)

The recorded story starts at trust 20, confidence 30, curiosity 45. Encouragement,
teaching and a kept promise raise these values. Dismissal lowers them. An apology
restores only part of the loss: the final state is **44 / 48 / 65**. These are
explicit fictional game rules, not scientifically validated personality scores.

## Persistent chat

**Talk to Moss** saves your message, reply, and a context receipt together as one
Mind Layer memory record. Use **Inspect this reply** to see the retrieved user
messages/story events, personality snapshot, and IDs of the recent turns used.
Those receipts describe the actual request context, not a model-generated
explanation or hidden reasoning. Offline mode shows local recall; `-live` enables
open-ended dialogue through the configured provider.

Chat records live in a separate `<agent>-chat/<user>` scope in the same demo
database. Their user messages rebuild a small BM25 index on each request. The
three highest-ranked matches from story memory and chat recall are supplied to
the model, along with the last four complete conversation turns. Scores from the
two corpora are ranking hints, not calibrated confidence. Generated replies are
preserved for conversational continuity but are never indexed as user facts.

The chat scope stores full receipts for inspection, so the general memory API
must not be exposed over this dedicated demo database. A rejected provider call
does not store half a turn. Stale turn numbers return HTTP 409; reload to recover
if another browser sent a message or a response was lost. The demo allows 100 chat
turns, messages up to 700 UTF-8 bytes, and replies up to 2,500 bytes. Context receipts
must also fit the core's 8 KiB memory-record limit. Start a new database for a new
conversation. Data is local but not application-encrypted; provider-enabled chat
sends messages and selected context to your provider. Its charges and retention apply.

Try the fictional greenhouse example in the README. A later correction remains
an explicit dated user message; the model is instructed to prefer it. This is
not a general contradiction-resolution system or a promise that every correction
will always be retrieved. Free text does not modify the character's trait policy.

Reproduce the structural checks (fresh temporary database; no personal data):

```sh
make chat-eval
# With configured provider variables; incurs provider usage:
python3 scripts/run_chat_eval.py --live
```

The recorded offline and live results are `benchmarks/results/chat-offline.json`
and `benchmarks/results/chat-live.json`. They check complete turns, exact restart
persistence, original fact and correction retrieval, mode labels, and unchanged
story traits. Five synthetic messages are not a general answer-quality benchmark.

## What changes and why

| Interaction | Trust delta | Confidence delta | Curiosity delta |
| --- | --- | --- | --- |
| Encourage | +12 | +15 | +5 |
| Teach | +5 | +5 | +15 |
| Keep promise | +20 | +10 | +5 |
| Dismiss | -25 | -20 | -10 |
| Repair | +12 | +8 | +5 |

Values clamp to 0–100. At trust 30 Moss starts warming up; at 50 it becomes a
trusting teammate. The kept-promise response changes with the current trust level.
All default dialogue is authored/scripted and labeled as such in the UI.

The application policy lives in `examples/character/character.go`. Each event is
an explicit fact in the dedicated `moss/demo-user` scope. `Store.UpdateScope`
commits that memory and the newly rendered personality in one transaction.
Traits and the chart are reconstructed by replaying the event history; no hidden
state file is required. Revision checks reject duplicate/stale requests.

## Persistence and fresh stories

The default database is `data/character-demo.db`. Reloading the page or restarting
the process preserves the character. To start another story without deleting the
old one, choose a new database path:

```sh
go run ./cmd/character-demo -data data/moss-second-story.db
```

The demonstration allows up to 100 story interactions and 100 chat turns. It binds only to a loopback IP,
rejects cross-origin requests, and does not expose the general memory API. The
demo database must remain separate from other application data.

## Optional model-generated dialogue

With a configured direct provider (`MIND_LAYER_BASE_URL`, `MIND_LAYER_API_KEY`,
`MIND_LAYER_CHAT_MODEL`), start:

```sh
go run ./cmd/character-demo -live
```

The separate **Generate a model reply** button under Ask memory sends the question, current personality and retrieved
memories directly to that provider. It does not infer trait deltas, store the
question or rewrite the event history. This keeps model creativity separate from
the deterministic evolution policy. Provider charges and retention apply.
The demo does not use embeddings; its memory queries are lexical in both modes.
Without `-live`, provider variables are ignored and the entire example is offline.

### Observed live replies

The same synthetic question was sent to Gemini 2.5 Flash before interaction,
after the kept promise, after dismissal and after repair. Initially Moss said it
had no record of a promise. After the promise it recalled the sensor and proposed
integrating it. After dismissal its reply expressed uncertainty about contributing.
After repair it recalled the encouragement and lesson and cautiously offered help.

These four qualitative observations are recorded in
`benchmarks/results/character-live.json`. They demonstrate the actual provider
path using changing persisted state; they are not a statistical evaluation of
personality quality. The UI marks model-generated replies separately from the
offline scripted replies.

## Test and reproduce

```sh
go test -race ./examples/character
make demo-scenario
# Regenerate the authored scenario receipt and graph:
python3 scripts/run_character_scenario.py --record
.venv/bin/python scripts/build_docs.py
```

The scenario command uses a fresh temporary database and exercises the real
executable. Its checked-in receipt is `benchmarks/results/character-scenario.json`.
Tests prove the exact deltas, memory recall, persistence, scope isolation, bounds,
invalid/stale request rejection, concurrent revision handling and HTTP behavior.
`TestAtomicScopeUpdate` proves personality and event memory stay together across
restart and that rejected updates do not partially apply.
