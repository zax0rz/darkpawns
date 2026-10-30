# Starlight integration review, 2026-10-29

The existing site now serves Starlight documentation under `/docs`. It uses the shared masthead, Haunted Paperback palette and serif typography, with separate named Site and Docs menus on mobile. No theme switch is exposed.

The six sidebar sections are Play, Host, Build, Reference, Project, and Releases. The build produces 24 authored pages/hubs, 165 command pages and their index, six release pages and their index, and 96 API pages: 293 documentation pages total. Every page has a Markdown twin, description, verification line, and an edit link to an existing source file. Release notes and command help remain single-source inputs. The offline API export now also registers the public connection-info operation used by the actual server.

Authored metadata is required before the Starlight/OpenAPI plugin defaults are applied. Removing `audience` from the first-hour source made `astro sync` fail; the source was restored. `test:docs` validates built output and catches broken source edit links, missing Markdown twins, invalid descriptions, missing diagrams/search, and broken docs links. It also verifies all 25 compatibility redirects, including the 11 legacy Markdown URLs. Caddy's generated table contains the permanent redirects.

The renderer upgrade needed for current Starlight makes positions opt-in and reports string offsets. Existing archive link-protection tests caught both changes; the plugin now keeps bare domains inert while preserving explicit links after Unicode text. The global search index also normalizes docs hub paths so its results reach the canonical pages.

## Verification

- `make fmt`, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and `make deadcode-ratchet` passed. Lint also passed with an isolated cache and zero issues.
- `make site-check` and `make build-site` passed. The full world-asset build was run from this worktree. No census is needed for website content and offline tooling changes.
- `npm run test:search` passed four tests, including canonical docs hub URLs. Existing Markdown tests passed eight tests.
- `npm audit --omit=dev` reports zero vulnerabilities. Dependencies are locked; Starlight, OpenAPI, and Mermaid use exact versions.
- The source-only content inventory passes without `.generated` artifacts or Node setup, matching the Go-only CI job.
- Pagefind search for `rollback` finds the update guide, admin tour, and API operation. Mobile Site and Docs menus open; search opens with Enter and closes with Escape; keyboard focus uses the oxblood outline.
- Impeccable detection reports no findings. The independent reader and visual reviews are linked below.

Editorial warnings remain visible in voice lint. The baseline only removes 25 obsolete entries from moved/removed docs; it adds no suppressed findings. Starlight emits no unexplained warnings. This host's Node 22.23.3 builds successfully but npm warns that the transitive Prism package requests 22.23.4; the website README documents the supported minimum.

## Reader and visual review

[Independent reader answers for every authored page](reader-review.md) include prerequisites, expected results, and recovery/next steps. The first-hour route and first fight were checked against rooms 8004–8102 and 16300–16304, trainee-guard resets, and command help. It is explicitly scoped to Kir Drax'in.

[Independent visual review](ui-review.md) covers the shared site identity, navigation, reading hierarchy and 375px layout. [Capture checks](screenshots/checks.json) record dimensions and route/edit-link checks. [Media integration checks](media-checks.json) verify six illustrated pages at both widths: images decode, alt text is present, and there is no horizontal overflow. All 14 built media files match the canonical source hashes. Regenerate the review images with `website-astro/scripts/capture-docs-review.mjs` against a served build.

| Page | Desktop, 1440px | Mobile, 375px |
| --- | --- | --- |
| Docs index | [Capture](screenshots/index-desktop.png) | [Capture](screenshots/index-mobile.png) |
| First hour | [Capture](screenshots/first-hour-desktop.png) | [Capture](screenshots/first-hour-mobile.png) |
| Install | [Capture](screenshots/install-desktop.png) | [Capture](screenshots/install-mobile.png) |
| Admin tour | [Capture](screenshots/admin-console-desktop.png) | [Capture](screenshots/admin-console-mobile.png) |
| Update and rollback diagram | [Capture](screenshots/update-server-desktop.png) | [Capture](screenshots/update-server-mobile.png) |
| Admin API | [Capture](screenshots/api-desktop.png) | [Capture](screenshots/api-mobile.png) |

## Decisions for Zach

- Review the visual captures and the small amount of personality in Play/overview prose. Instructions and recovery paths stay plain.
- The first-hour route covers Kir Drax'in. Other hometowns need their own source-checked routes in later guides.
- Long API property names wrap on mobile; they remain readable. Registered API summaries with no longer description include `reset-zone` ("Reset one zone"), `reset-all-zones` ("Reset every zone"), and `list-triage-summaries` ("List triage summaries"); these deserve a follow-up explaining scope and consequences. Contract generation stays automatic.
- Published media comes from the stacked reproducible media pipeline PR. The docs use its canonical manifest assets; no TODO(media) markers remain.
- No production deployment was performed. Claude can review and stack the PRs in Zach's session.
