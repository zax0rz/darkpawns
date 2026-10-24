---
version: 1
slug: "website-astro-src-components-footer-astro"
primary_target: "website-astro/src/components/Footer.astro"
related_targets: ["website-astro/src/pages/index.astro", "website-astro/src/pages/download.astro", "website-astro/src/pages/connect.astro"]
---

# Footer directory and community

User explicitly requested footer refinement, October 23, 2026. Visitor mode: Operate/Find. Human players need direct routes to playing, connecting, and hosting; archive readers and project visitors need a compact directory. Scope is the shared footer only. Homepage/header/download content and Base remain unchanged.

## Direction contract

THESIS: End each page with a legible directory and a clear community invitation, using the site's existing printed hierarchy.

OWN-WORLD: Preserve Haunted Paperback: warm cream, charcoal rules, display/body serifs, rare oxblood social links, and monospace status. Reuse incumbent tokens, flat surfaces, and existing monochrome SVGs; no new raster assets or global design changes.

STORY: Recognize Dark Pawns and its dates, find Discord/GitHub and the existing server status, choose Play/Explore/Archive/Project, then reach privacy and license details.

FIRST VIEWPORT: The footer opens with identity and dates alongside social links/status. Four directory columns on desktop become two at 700px and below. At 480px and below the identity/community header stacks. Legal content ends the footer. Preserve compact scanning without the former long single-column mobile directory.

FORM: Semantic footer with labeled community, directory, and legal navigation. Preserve every previous destination, deduplicate GitHub, and add `/connect/` discovery. At widths through 900px or with a coarse pointer, all footer links meet 44px minimum targets; fine-pointer desktop directory rows remain 36px with normal legal-link density.

FINISH: Source/capture comparison and reported validation below; fresh independent `footer_finish_review` verdict: **SHIP**, no material footer defects. The reviewer inspected source, DESIGN.md, global focus styles, and all three captures; labels wrap without collision/truncation, 320px is dense but usable, and 390px is more comfortable. Existing destinations and status hooks are preserved. Measurements and gates remain implementation handoff evidence, not reviewer reruns. Preserve global DESIGN.md, PRODUCT.md, and design.json. No claim of physical-device or zoom testing.

## Implementation evidence

`Footer.astro` holds four explicit link groups: Play (including browser, client guide, download/host, Mudlet), Explore (world/map/database/zones/help), Archive (archive/blog/community captures/lineage), and Project (about/docs/research/credits). Privacy and the existing license URL move into the legal row. Discord retains the exact user invite `https://discord.gg/supVzU9yN`; GitHub has one footer destination. Existing status IDs, data attributes, live region, and online/offline styling remain intact.

This documentation pass inspected Footer.astro, incumbent DESIGN.md/PRODUCT.md, and `.impeccable/review/footer-redesign-user.png`, `footer-redesign-mobile-top.png`, and `footer-redesign-mobile-bottom.png`. Captures show the four/two-column directory, visible community links, one status, and legal separation. Mobile captures are scaled QA iframe views; their white parent canvas is outside the site. No material documentation/design-extension finding was observed.

Implementation handoff reports old hrefs are a subset of the new hrefs plus `/connect/`; widths 320/390/480/481/700/701/900/1440px have no overflow, links meet 44px through 900px, and exactly one status remains. Site-check, formatting, Go build/vet/test/lint pass; scoped layout scans report `[]` before/after. Five stale voice-baseline findings were removed without adding exceptions. These execution results are handoff evidence, not gates rerun by this documentation pass.

Read-only comparison confirms DESIGN.md, PRODUCT.md, and `.impeccable/design.json` remain unchanged. The footer extends the incumbent system without changing global tokens. Previously noted global-document drift is outside scope. Only this new footer brief was created; reviewer history was not added to the homepage brief.
