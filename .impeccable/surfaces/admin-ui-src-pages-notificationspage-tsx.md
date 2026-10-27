---
version: 1
slug: "admin-ui-src-pages-notificationspage-tsx"
primary_target: "admin-ui/src/pages/NotificationsPage.tsx"
related_targets: []
---

# Notification settings

Scope: extend the existing staff console with the v1 Zach approved on 2026-10-19.
Mode: Operate. HIGOD operators choose five events independently for Pushover and
Discord, check delivery, and optionally enable a hosted Healthchecks heartbeat.
Visual authority: existing player editor, admin-ui/src/index.css, and the Haunted
Paperback system in website-astro/DESIGN.md. No new visual world or system changes.

## Direction contract

THESIS: Make event routing inspectable in a two-destination matrix, with setup and
delivery evidence nearby. A log feed is not the configuration model.

OWN-WORLD: Existing warm paper, cream table headings, charcoal hairline rules,
serif labels and sparse oxblood actions; native checkboxes and square controls.

STORY: See destinations, select different events, save, send a test, then inspect
provider acceptance. Credentials are pasted into write-only inline forms, then shown as Configured.
Replace and Remove are explicit; no environment editing is needed.

FIRST VIEWPORT: Existing sidebar and page header, then the ruled event matrix.
Pushover and Discord own narrow comparison columns. Save stays at the bottom of
the viewport while operating the form. Smaller screens preserve the comparison.

FORM: Direct extension of the incumbent settings/editor language, selected by the
accepted v1 brief; no open composition decision or comp round. seed: none (unavailable).

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

No raster assets ship. Local evidence: desktop at 1272×716 and mobile via an actual
390×640 iframe in a temporary preview harness (preview-only CSP/frame headers
omitted to allow embedding). The production header path was checked on the first
desktop browser pass. Browser settings use a disposable database/config directory.

2026-10-19 extension: write-only credential forms for Pushover, Discord and hosted
Healthchecks. Password-masked inputs use existing paper/ink fields; inline removal
confirmation avoids a modal. Saving credentials preserves event drafts. Secrets
never appear in saved values or screenshots; review uses disposable fake values.

2026-10-27 rebuild (v0.1.7 plan F): one page, one layout, plan A's components.
Every outbound channel is a stacked ruled block: Discord alerts, Pushover, the
gossip mirror, the Healthchecks heartbeat, and the CI-side Discord release
announcement, which links to the operator reference instead of offering controls
it cannot honor. Each block carries its own enable switch, its settings, its
write-only credentials, its delivery evidence and its test send, so the event
matrix and the separate "Destinations & delivery" section are gone. The revisioned
routing settings keep one explicit save, pinned at the top of the working area
rather than the bottom of the viewport: with five blocks, a bottom bar sat where no
block was being edited, which is part of what made the old page read as bolted
together. Credential removal now confirms in the shared AlertDialog. Evidence:
~/Archives/darkpawns/admin-ui/2026-10-27/notifications-rebuild/.
