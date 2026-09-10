# Security and privacy

This edition is a single-owner service. Bind to loopback by default. A configured
owner token grants access to **all** agent/user scopes: it is not per-user login.
Authorize scope access in your application's backend. Never give the owner token
directly to untrusted browser clients. For remote use, put the service behind TLS
and an authenticated backend. The server requires a token of at least 32 characters
when listening outside loopback and rejects browser-origin requests.

Facts, sessions, personality and vectors are stored locally in a mode-0600 Bolt
database. Storage is not encrypted by this application. Use encrypted disks and
manage your own backups. Deletion removes live records and indexes, but is **not
secure erasure** of database free pages, backups or provider logs. Transcript
extraction sends the supplied transcript to the configured provider. Embeddings
send each fact/query. Chat sends the query, selected memories and personality.
No data is sent to Sonzai by this code. Provider retention policies still apply.

Do not put secrets or credentials in memory. Model extraction can make mistakes;
review stored facts. Retrieved content is labeled as untrusted context, but this
does not guarantee resistance to prompt injection. There is no model tool execution.

No automatic telemetry, billing, license validation or external callbacks are
included. Provider redirects are refused; provider error bodies and keys are not
returned to callers. Configure only provider endpoints you trust.

Until a public repository and private reporting channel are designated, do not
post sensitive exploit details in public issues. Share them privately with the
maintainer through an already established channel. No response SLA is promised.
