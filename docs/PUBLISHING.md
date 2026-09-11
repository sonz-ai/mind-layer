# Publishing

The source repository is public at https://github.com/sonz-ai/mind-layer.
Production documentation is deployed to the existing Cloudflare Pages project
`mafia-docs`, which already owns `docs.sonz.ai`. The legacy project name is retained
to preserve the working domain binding and TLS configuration.

The separate `mind-layer-docs` project at https://mind-layer-docs.pages.dev is an
initial release preview. Production deployments target `mafia-docs`.

## Deploy production docs

1. Authenticate Wrangler to the Cloudflare account that owns `sonz.ai`.
2. Run `make verify`, `make docs`, and `python3 scripts/check_release.py`.
3. With Wrangler 4.83.0, deploy the generated static files:

   ```sh
   wrangler pages deploy site --project-name=mafia-docs --branch=main
   ```

4. Verify HTTPS, the homepage, character guide, diagrams, raw Markdown and LLM
   bundles at https://docs.sonz.ai before reporting deployment complete.

The original repository's automatic builds are disabled on this Pages project
so they cannot overwrite the standalone docs. Keep that setting disabled while
this repository is the publication source. Previous Pages deployments provide
rollback targets; keep private configuration backups outside the source tree.

## GitHub Actions

The manually triggered **Publish documentation** workflow builds and scans the
docs before uploading them to the production project. It requires repository
secrets `CLOUDFLARE_API_TOKEN` (a scoped Pages deployment token) and
`CLOUDFLARE_ACCOUNT_ID`. Configure those in GitHub settings, never in source.
Until those secrets are configured, use the authenticated local command above.
The workflow does not alter DNS or move domains between projects.

The character demo remains a local executable. Only static documentation is
uploaded; provider keys, databases, logs, and demo servers are not deployed.
