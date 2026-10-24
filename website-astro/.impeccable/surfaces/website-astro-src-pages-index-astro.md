---
version: 1
slug: "website-astro-src-pages-index-astro"
primary_target: "website-astro/src/pages/index.astro"
related_targets: ["website-astro/src/pages/download.astro", "website-astro/src/components/Footer.astro", "website-astro/src/components/Header.astro"]
---

# Homepage chapter opening and downloads

Visitor modes: Persuade (homepage), Operate/Read (downloads). Human players come first, followed by people installing a separate world. Scope: homepage opening, `/download/`, shared copy control, footer discovery, search, and Markdown representation.

User-approved chapter direction, October 23, 2026: enlarge the same approved altar image, remove the opening Temple Altar room-description, and center the image, title, and play action like a first chapter. Preserve the real paper texture. The latest correction gives always-visible MUD-client instructions equal importance to hosting, restores news immediately after the world overview, removes unsupported geography from map summaries, and uses the evergreen history heading “Over thirty years in the record”. This contract supersedes earlier compositions; mock studies remain under `mocks/`.

## Direction contract

THESIS: “You type and the world answers.” opens a used-paperback chapter, with clear paths to play here or run a separate world.

OWN-WORLD: Preserve Haunted Paperback: cream, charcoal rules, serif headings, rare oxblood, machine-only monospace, and the faint real paper scan. The approved charcoal altar image remains the chapter mark. No new global palette, elevation, or motion system.

STORY: Read the centered invitation, enter the shared world or install a separate one. The opening room-description is removed. Continue through Inside Dark Pawns (rules and sourced counts), Latest from the restoration, original world excerpts, field guides, and history/restoration. Downloads explain setup, first immortal, verification, and hosting.

FIRST VIEWPORT: Centered altar image above the two-line title, short lede, and primary play action. Adjacent equally titled MUD-client and hosting sections follow. Host, TLS port, classic telnet port, and client guide are always visible. On mobile client instructions come first, hosting follows; download placement below the fold is accepted to preserve client visibility. `/download/` retains its installation/setup split and six platform archives.

FORM: Code-led local extension of the incumbent editorial system. Semantic sections give clients and hosting equal heading weight, without a disclosure. A single Explore the world link belongs to Inside Dark Pawns; history navigation stays with the history section. The same transparent image displays at 360×120px desktop and approximately 224×75px mobile. Broad chapter-design inspiration from *The Gunslinger* informs the composition; no existing book illustration was supplied as input.

FINISH: Current source/capture evidence is recorded below. Fresh independent `client_host_finish_review` verdict: **SHIP**, no material findings, after inspecting current source and all three captures. The mobile scroll tradeoff follows the user’s explicit client-visibility priority. Retain source/prompt provenance for both shipping rasters, copy behavior, six download links, and the Markdown endpoint. Preserve global DESIGN.md and design.json for this ordinary extension.

## Implementation and provenance

`index.astro` centers the invitation within a 48rem maximum column. The image is decorative (`alt=""`), with native alpha over the real paper. The paper remains at 10% opacity and is suppressed in forced-colors mode. The always-visible client section labels Host (`darkpawns.org`), TLS port (`7778`), and Classic telnet (`7777`) beside the hosting section; mobile stacks client then host. Rules and derived counts form Inside Dark Pawns, followed by news, excerpts, guides, and history. The room figure detail is “Every room and connection on the map”; the field-guide description omits the unsupported three-continents claim. The archive heading is “Over thirty years in the record”. `/download/` is unchanged.

- `src/assets/chapter/temple-altar.png`: original generated interpretation of room 8004, not recovered artwork. Built-in ImageGen exposes no model/version selector. README and `temple-altar.prompt.txt` record origin, exact prompt, 2172×724 RGBA original, and 720×240 RGBA serving asset reduced with Lanczos. Prompt is embedded in the PNG. Only its display size and placement changed in this correction.
- `src/assets/paper/book-page.jpg`: unmodified-image Wikimedia thumbnail of a blank page from the 1902 *Armide* scan, supplied by Internet Archive Book Images. README records source, retrieval date, and “no known copyright restrictions”; JPEG comment also carries provenance. Both assets are imported through Vite.

The shared CopyCommand remains selectable without JavaScript, with success feedback and manual selection on clipboard denial. Downloads retain macOS/Linux/Windows ARM64 and AMD64 archives, verification/setup/hosting guidance, and a Markdown twin.

## Verification and documentation comparison

This documentation pass inspected current source and `.impeccable/review/client-host-user.png`, `client-host-mobile.png`, and `client-host-desktop.png`. Captures show adjacent client/hosting sections on desktop and visible client instructions before hosting on mobile. The latter two are scaled QA iframe captures; their white parent canvas is not site styling. Hero assets and provenance are unchanged. Independent finish review confirms SHIP with no material findings.

Final implementation handoff reports site-check passing after the final room-summary correction, plus Go build/vet/test/lint passing. Widths 320/390/768/1280/1440px show no overflow or clipped command. Download-link bottoms are 1144/1094/843/851/863px respectively: phone hosting/downloads require scrolling to preserve higher-priority visible client information. Accessibility-tree inspection confirms news before original excerpts, the evergreen history heading, and neutral room caption. These execution results are handoff evidence; this documentation pass inspected source/captures rather than rerunning gates.

Read-only comparison with `origin/main` confirms no changes to `DESIGN.md`, `.impeccable/design.json`, `PRODUCT.md`, or global CSS. Incumbent tokens remain authoritative. Pre-existing documentation drift remains: DESIGN.md describes an earlier front-page composition/title sizing, and the sidecar condenses its narrative. Current surface choices are recorded here; global document repair is outside scope. Only this brief was edited in this documentation pass.


## Record-link and terminology refinement

The world figure now says “mobs”; the Database field guide says “Every mob and object”. Archive navigation uses two stacked primary serif rows, Browse the archive (with the derived record count) and Read the full history. DPReturns history and Original staff & credits remain as smaller muted underlined references without arrows. All four destinations are preserved; other sections, hero assets, and download behavior are unchanged.

This pass inspected current source plus `.impeccable/review/record-links-user.png` and `record-links-mobile.png`; the mobile image is a scaled QA iframe capture. The two primary rows and quieter related references maintain the incumbent type, ink rules, and focus treatment. Implementation handoff reports keyboard focus/order verified, no overflow at 320/390/768/1440px, site-check/formatting/Go build/vet/test/lint passing, and layout detector `[]`. Fresh independent `record_links_finish_review` verdict: **SHIP**, no material findings, after inspecting source and both captures; all four routes and the derived count are preserved, and the hierarchy fits the incumbent system. Earlier captures remain evidence for the unchanged hero. Global design documents and their previously noted drift remain untouched; only this brief was edited.


## Discord footer addition

The user-requested Discord badge follows GitHub in `Footer.astro`, linking to the exact invite `https://discord.gg/supVzU9yN`. It reuses `footer-social-badge`/`footer-icon`, with a 14×14 monochrome Bootstrap Icons SVG, decorative `aria-hidden`, visible Discord text, and accessible name “Dark Pawns on Discord”. No other UI changes are included in this addition.

Read-only inspection of the source diff and `.impeccable/review/discord-footer-user.png` / `discord-footer-mobile.png` found no material issue; the badge fits the existing footer treatment and wraps on narrow screens. Implementation handoff verifies no overflow at 320px, keyboard reachability, exact href, and passing site-check/formatting/Go build/vet/test/lint. This scoped evidence does not claim broader critique findings were fixed. Global design files remain unchanged; only this surface brief was edited in this documentation pass.


## Clarity and touch refinement

The authorized copy pass explains remort as starting again at level 1 in an advanced class while keeping equipment, gold, and statistics; the source is `src/content/help/info/remort-immort-remorting-immorting-immortality.md`. The objects detail now explains quitting in your temple to save gear without rent, supported by `src/content/archive/features.md:22` and `lib/text/help/commands.hlp:1252`; the duplicate rent claim is removed from Wyldlands. The Database guide replaces VNUM with “in-game ID”. This pass inspected those source passages and the implementation diff.

At widths through 900px or with a coarse pointer, homepage standalone actions/blog headlines, footer column links/social badges/license, and header navigation/search have 44px minimum targets. Mobile menu summary has a 44px target and explicit focus treatment. Inline prose links retain reading flow. GitHub and Discord are grouped in the labeled Community and source code navigation, allowing them to wrap as a group. Hero, blog placement, client visibility, download content, assets, and availability/status behavior are unchanged.

Inspected `.impeccable/review/clarity-touch-menu.png`, `clarity-touch-world-mobile.png`, the final unfocused `clarity-touch-world-user.png`, and `clarity-touch-footer-mobile.png`; scaled QA iframe captures include a white parent canvas outside site styling. Implementation handoff reports no overflow at 320/390/640/641/768/900/1440px, measured target groups at least 44px through 900px with fine-pointer desktop density preserved, menu Enter/Space and focus verified, and shared header/footer checks on `/download/` and `/connect/` at 390px. Site-check, formatting, Go build/vet/test/lint pass; one scoped layout detector run reports `[]`. Final site-check after the footer navigation wrapper passed. The final footer capture shows GitHub/Discord side by side; handoff measurements confirm all target groups at least 44px and no overflow at 390px. Fresh independent `clarity_touch_finish_review` verdict: **SHIP**, no material findings, after inspecting the three-file diff, DESIGN.md, primary remort/rent texts, and all four final captures. Final implementation checks confirm no footer-group overflow at 320/1440px and all 22 footer links at least 44px high at 320px. Availability remains untouched; this does not close every earlier critique priority. Global DESIGN.md/design.json/PRODUCT.md remain unchanged, as confirmed by read-only comparison. Only this brief was edited.
