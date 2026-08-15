# Website Specification audit

Last run: 2026-08-15

Scope: Astro migration, with an initial focus on the [agent-readiness category](https://specification.website/spec/agent-readiness/).

This is a release checklist, not a promise to implement every emerging protocol. Re-run the audit before replacing the Hugo deployment.

## Implemented

- Stable canonical URLs on every HTML page.
- An HTML-only sitemap index at `/sitemap-index.xml`.
- An accurate, curated `/llms.txt`.
- Explicit crawler policy and AI crawler rules in `/robots.txt`.
- Agent instructions at `/.well-known/agent-skills/`.
- Predictable Markdown representations for docs, help, world, blog, and archive entries.
- JSON representations for individual mob and item records.
- JSON-LD for the site and database records.
- HTML discovery links for the sitemap, `llms.txt`, agent skills, Markdown, and database JSON.
- Caddy `Link` headers for the sitemap, `llms.txt`, and agent skills.

## Partial or awaiting deployment

- Stable URLs: database parity is complete, but the remaining Hugo compatibility routes still need review.
- Structured data: database records and the site have typed data. Editorial page types can be added after their content is settled.
- HTTP discovery headers: configured in Caddy and must be verified after deployment.

## Deliberately deferred

- `llms-full.txt`: the site is too large for one useful concatenated file. Per-page Markdown is cheaper and more precise.
- AI Catalog, A2A, NLWeb, and MCP discovery: add these only when Dark Pawns exposes the corresponding public service.
- WebMCP: wait for browser support and a concrete in-page tool.
- DNS-AID and Web Bot Auth: infrastructure work with no migration requirement.
- OKF and schemamap: optional competing representations without a current consumer.
- TDM reservation: unnecessary while the project explicitly permits agent use and training.

## Pre-launch audit

### Foundations and SEO

Implemented:

- Doctype, language, UTF-8, viewport, unique titles, descriptions, and canonical URLs.
- Open Graph and Twitter summary metadata.
- Theme color, color-scheme declaration, SVG favicon, and Apple touch icon.
- Static HTML for primary content, an XML sitemap, redirects, robots policy, and internal links.
- RSS discovery and a generated feed for project notes.

Partial:

- Breadcrumbs are visible on hierarchical pages but do not yet have `BreadcrumbList` JSON-LD.
- Editorial structured data can be typed as `Article` after the content pass is complete.
- A raster social image exists, but a purpose-built 1200 by 630 share image would produce better previews.
- Every HTML route in the generated Hugo site has an Astro page or redirect. `make route-parity` enforces this against both build directories.
- Static redirect documents are mirrored into Caddy permanent redirects by `website-astro/scripts/caddy_redirects.py`, preserving direct browser fallbacks and HTTP migration signals.

### Accessibility

Implemented:

- One main landmark, a skip link, a page heading, semantic navigation, and visible focus treatment.
- Reduced-motion and forced-colors handling.
- Labels for authored form controls and captions for generated data tables.
- Mobile form controls remain at a readable size.
- No informational raster images or audio/video content currently require alternatives or transcripts.

Partial:

- The map and database have keyboard controls and static HTML alternatives, but their complete dynamic interaction needs a manual screen-reader and keyboard pass.
- Color contrast and touch targets need measurement in the rendered site, not inspection alone.
- Contact failures are announced in a live status region. Field-specific server errors would be an improvement if validation becomes more complex.

### Security

Implemented:

- Astro upgraded from 5.18.2 to the current 7.x release after the audit found active advisories. `npm audit` reports no known vulnerabilities.
- Caddy sends HSTS, `nosniff`, frame protection, referrer policy, permissions policy, and a restrictive source policy compatible with the current client and contact form.
- Text compression and immutable caching for fingerprinted Astro assets.
- Subresource integrity on the jsDelivr scripts and styles loaded by the map and game client.
- A contact-only `security.txt` that does not expose a private email address.

Partial:

- The source policy permits inline script and style because the existing map, database, and Astro output use them. Remove those allowances only after the affected code is migrated to nonce or hash based loading.
- TLS versions, HTTP-to-HTTPS redirects, HTTP/2 or HTTP/3, CAA, and final headers must be verified at the public Cloudflare edge after deployment.
- Cross-origin isolation and Trusted Types are not enabled because the current third-party client resources and DOM rendering code are not compatible without additional work.

### Performance

Implemented:

- Static generation, fingerprinted assets, compression, immutable asset caching, deferred client scripts where supported, and stable scrollbar space.
- The site currently has no content images requiring responsive image generation or lazy loading.

Partial:

- Core Web Vitals require field or production lab measurements.
- Fonts are still served by Google. Self-hosted, subset WOFF2 files are the next useful loading and privacy improvement.
- The map, database, and web client retain third-party JavaScript. Their failure leaves navigation and static record pages available, but not the interactive tool itself.

### Privacy

Implemented:

- A public privacy page covering contact messages, Turnstile, temporary rate limiting, server records, storage, and archive removal requests.
- No advertising or behavioral analytics.
- No consent banner because the site does not set non-essential tracking cookies.

Partial:

- Verify Cloudflare and host log retention before launch and add the actual period if one is configured.
- Global Privacy Control has no sale or sharing behavior to disable. Continue to avoid adding third-party analytics that would change that answer.

### Resilience

Implemented:

- A custom static 404 page with useful routes back into the site.
- Caddy preserves error status codes and serves the custom error document.
- Primary navigation, articles, help, archive, mob records, and item records work without JavaScript.

Deferred:

- A maintenance page and external uptime monitor belong to deployment operations.
- A PWA manifest is unnecessary unless installing the web client becomes a supported product surface.
- `Redirect-By` is optional and adds little while redirects come from one Astro configuration.

### Deployment pipeline

Implemented:

- `make build-site` regenerates the shared world and database assets, runs the site checks, and builds Astro.
- `make deploy-site` retains explicit host credentials and syncs `website-astro/dist/` to the existing Caddy document root.
- The old help interlinker is no longer part of deployment because it only rewrites the retired Hugo content tree.

## Deployment verification

- Confirm the live map loads its fingerprinted Astro script. The Hugo deployment referenced `/js/map.js?v=3`, which returned the site's 404 document during the migration audit.
- Confirm HTTP redirects to HTTPS and only TLS 1.2 or 1.3 is accepted.
- Confirm HSTS, CSP, `nosniff`, referrer, permissions, cache, compression, and discovery headers at the public URL.
- Confirm `/404-test` returns the custom page with status 404.
- Confirm `/rss.xml`, `/sitemap-index.xml`, `/llms.txt`, Markdown records, JSON records, agent skills, and `security.txt` have correct content types.
- Run keyboard, screen-reader, contrast, touch-target, and Core Web Vitals checks against production.
- Compare every Hugo URL with the Astro output and redirect table.
