# Dark Pawns website

The Astro site and its Starlight documentation share the Haunted Paperback design system. Read `DESIGN.md`, `PRODUCT.md`, `../docs/docs-style.md`, and `../docs/brand-voice.md` before changing public prose or interface styling.

## Build locally

Use Node 22.23.4 or newer. From this directory:

```sh
npm ci
npx playwright install chromium
```

From the repository root:

```sh
make build-site
```

This generates world and database assets, checks content and design rules, builds the site, and generates permanent Caddy redirects. It does not deploy. For later Astro-only rebuilds with world assets already present, run `npm run build` in this directory.

## Documentation sources

- Authored pages: `src/content/docs/`. Every page requires `title`, `description`, `type`, `audience`, and `verified`. The loader checks source metadata before applying defaults used by the API plugin.
- Releases: `../docs/releases/v*.md`. The build generates pages and the newest-first release index.
- Commands: `src/content/help/commands/`, the existing parsed in-game help archive. Generated references preserve its body.
- API: `go run ./cmd/admin-openapi` from the repository root. `make admin-api` also saves the offline JSON export. The site separates public and admin operations and retains the complete export under `.generated/api/`.
- Generated files: `.generated/`, ignored by Git. Do not hand-edit them.
- Published screenshots: `../docs/images/generated/`, produced by `make media`. The docs generator copies this canonical tree to the public `/media/docs/` URL.

Mermaid fences render to SVG at build time using Chromium; browsers receive the finished diagram. Set `CHROMIUM_PATH` to use a particular local executable. The renderer also checks the standard Playwright cache. The diagram cache includes the Mermaid version.

Every built docs page has a Markdown twin. Authored and generated Markdown sources remain intact; API twins derive from rendered reference content. `npm run test:docs` checks the built page metadata, twins, source edit links, documentation links, redirects, search index, and update diagram.

## Review screenshots

Serve `dist/` locally, then run from this directory:

```sh
DOCS_REVIEW_URL=http://127.0.0.1:4389 node scripts/capture-docs-review.mjs
```

`DOCS_REVIEW_DIR` chooses the output directory. These are PR review captures; published documentation images use `make media`.
