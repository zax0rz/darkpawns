# Documentation review pass 2

The same stacked PRs, #2229 and #2230, implement the October 30 brief. No deploy or Claude review was run. The foundation stays in Starlight overrides and the existing Astro build.

## What changed

Play leads with the game's help and a three-character party. The first-hour route is verified against active world files and explicitly applies to Kir Drax'in. Host has a measured hosting overview, bootstrap-first install with four alternate-method tabs, and media throughout. Build has five webOLC task guides and focused UI captures. Project links the repository, fidelity rules, research and existing history/about/credits surfaces. Release announcements moved into the maintainer publishing document.

The old help museum is now 12 command groups: all 429 bodies are read directly from `lib/text/help/*.hlp` and byte-compared independently. Every old help topic and alias redirects to its group anchor; global search uses those canonical destinations. There are 3,104 docs/help redirect checks.

The same collections and resolved Starlight sidebar generate `/llms.txt` at build time. The build emits Markdown twins and `/llms-full.txt`, with discovery tags and a visible Markdown link on every published docs page. The shared site footer has a compact docs variant. Search controls share tokens, including their measured height.

## Validation

- `make site-check`, including the Astro build: 146 published docs pages, all twins, agent-index entries, alternate tags, source links, redirects, search and diagrams pass.
- All 429 raw help bodies match byte-for-byte; ASSIST preserves literal angle-bracket text rather than museum link transforms.
- Formatting, build, vet, tests, lint and deadcode gates passed independently on both restacked branches. Lint was serialized after its process lock refused a parallel invocation.
- Full media regeneration passed on current main; all 19 outputs pass the canonical media freshness check.
- Twelve full-page screenshots at 1440 and 375 pixels have decoded images and no horizontal overflow.
- Browser interaction checks confirm matching search-control styles, working Pagefind results, all four install tabs, complete install Markdown, and mobile Overview navigation; no page errors. See `browser-checks.json` and `screenshots/checks.json`.

## Failure proofs

Removing the alternate tag from the actual Head override made **`make site-check` exit nonzero**. The source was restored in a finally block and the full positive gate rerun. Removing a built index entry and a built Markdown twin each made the docs gate fail too; both were restored. The signing-key test also rejected a substituted production pin through an isolated build overlay. Full logs remain in `/home/zach/Archives/darkpawns/docs-pass-2/`.

### site-negative-alternate

```text
AssertionError [ERR_ASSERTION]: /home/zach/wt/codex-docs-starlight/website-astro/dist/docs/build/edit-lua-scripts/index.html: missing Markdown alternate link
```

### site-negative-index

```text
AssertionError [ERR_ASSERTION]: /home/zach/wt/codex-docs-starlight/website-astro/dist/docs/play/first-hour/index.html: published docs page missing from llms.txt
```

### site-negative-twin

```text
AssertionError [ERR_ASSERTION]: /home/zach/wt/codex-docs-starlight/website-astro/dist/docs/play/first-hour/index.html: missing Markdown twin
```

### capture-key-negative

```text
    capture_key_test.go:30: documentation capture key must never be the production publisher key
```

## Reader and source evidence

- [Independent reader pass and resolutions](../2026-10-30-docs-pass-2-reader.md)
- [Play, client and hosting facts](../2026-10-30-docs-pass-2-play-host.md)
- [Live webOLC checks and capture comparisons](../2026-10-30-docs-pass-2-build.md)
- [Help classification, raw source and redirects](../2026-10-30-docs-pass-2-commands.md)
- [Media and key guard](../2026-10-30-docs-media-pass-2.md)

The hosting resource figures are one measured idle local run; planning headroom is labeled as an estimate. Only the Kir Drax'in player route is covered. Linux capture dependencies and live terminal timing limits remain documented. Existing editorial lint warnings remain visible; 22 obsolete baseline entries were removed, with no new suppression.
