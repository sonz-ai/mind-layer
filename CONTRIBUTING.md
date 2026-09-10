# Contributing

Run `make verify` before submitting a change. It runs the race-enabled Go suite
and checks every supported capability against named passing tests. Run `make
benchmark` for the local diagnostic. Live provider tests require explicit opt-in
and may incur provider charges; normal tests run without provider keys.

Keep this edition independently runnable. New features must not require Sonzai
accounts, license servers, billing integrations or proprietary services. Update
`docs/capabilities.json`, tests and docs together when adding a supported claim.
Report benchmark dataset, retrieval mode, models and hardware context. Do not
substitute synthetic or retrieval-only scores for public answer-quality results.

Only add authored synthetic fixtures. Do not include customer conversations,
real contact details, personal paths, keys, tokens, database snapshots, screenshots
of customer systems, or `.env` files. Do not publish live test logs containing
provider responses. Preserve upstream notices when extracting more code.

The source is Apache-2.0. Contributions are made under the same license. Use the repository’s issues and pull requests for public contributions.
