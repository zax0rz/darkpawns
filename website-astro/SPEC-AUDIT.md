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

## Next audit pass

Run required and recommended checks for foundations, SEO, accessibility, security, performance, privacy, and resilience. Record exceptions here with the reason they do not apply.
