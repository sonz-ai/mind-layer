# Publishing

The source repository is public at https://github.com/sonz-ai/mind-layer.
The static docs target a dedicated Cloudflare Pages project, `mind-layer-docs`,
with `docs.sonz.ai` as the intended production hostname.

## First deployment

1. Authenticate Wrangler to the Cloudflare account that owns `sonz.ai`.
2. Run `make verify`, `make docs`, and `python3 scripts/check_release.py`.
3. Create the `mind-layer-docs` Pages project if it does not already exist, with
   `main` as the production branch. Deploy `site/` and verify the Pages preview.
4. Inspect the current DNS, redirect rules and custom-domain attachment for
   `docs.sonz.ai`. Preserve their configuration before replacing the old redirect
   or binding. Attach the hostname to the new project and make only the necessary
   DNS/routing change. The marketing site and its existing `/docs` paths remain
   separate from this standalone documentation deployment.
5. Verify HTTPS, the homepage, character guide, diagrams, raw Markdown and LLM
   bundles through `https://docs.sonz.ai` before reporting deployment complete.

Do not treat a successful Pages upload as proof that the custom hostname has
switched. Keep private account IDs, tokens and routing backups out of this repo.

## Later deployments

The manually triggered **Publish documentation** workflow builds and scans the
docs before uploading them. It requires repository secrets
`CLOUDFLARE_API_TOKEN` (a scoped Pages deployment token) and
`CLOUDFLARE_ACCOUNT_ID`. Configure those in GitHub settings, never in source.
The workflow does not alter DNS or move domains between projects.

For a local authenticated deployment, use Wrangler 4.83.0:

```sh
wrangler pages deploy site --project-name=mind-layer-docs --branch=main
```

The character demo remains a local executable. Only the static documentation is
uploaded; provider keys, databases, logs, and the demo servers are not deployed.
