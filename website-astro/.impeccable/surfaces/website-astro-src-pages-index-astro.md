---
version: 1
slug: "website-astro-src-pages-index-astro"
primary_target: "website-astro/src/pages/index.astro"
related_targets: ["website-astro/src/pages/download.astro"]
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
