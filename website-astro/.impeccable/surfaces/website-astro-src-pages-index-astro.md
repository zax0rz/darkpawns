---
version: 1
slug: "website-astro-src-pages-index-astro"
primary_target: "website-astro/src/pages/index.astro"
related_targets: ["website-astro/src/pages/download.astro"]
---

# Homepage chapter opening and downloads

Visitor modes: Persuade (homepage), Operate/Read (downloads). Human players come first, followed by people installing a separate world. Scope: homepage opening, `/download/`, shared copy control, footer discovery, search, and Markdown representation.

User-approved chapter direction, October 23, 2026: enlarge the same approved altar image, remove the opening Temple Altar room-description, and center the image, title, and play action like a first chapter. Preserve the real paper texture. The latest flow refinement replaces the crowded MUD bar with a native disclosure, gives world facts their own section, and moves news to the end. This contract supersedes earlier compositions; mock studies remain under `mocks/`.

## Direction contract

THESIS: “You type and the world answers.” opens a used-paperback chapter, with clear paths to play here or run a separate world.

OWN-WORLD: Preserve Haunted Paperback: cream, charcoal rules, serif headings, rare oxblood, machine-only monospace, and the faint real paper scan. The approved charcoal altar image remains the chapter mark. No new global palette, elevation, or motion system.

STORY: Read the centered invitation, enter the shared world or install a separate one. The opening room-description is removed. Continue through Inside Dark Pawns (rules and sourced counts), original world excerpts, field guides, history/restoration, and finally news. Downloads explain setup, first immortal, verification, and hosting.

FIRST VIEWPORT: Centered altar image above the two-line title, short lede, and primary play action. A compact Unix installation strip and centered “Playing with a MUD client?” disclosure follow. Both actions and the download link stay early on mobile. `/download/` retains its installation/setup split and six platform archives.

FORM: Code-led local extension of the incumbent editorial system. Native details/summary reveals labeled Host, TLS port, Classic telnet, and the connection guide. A single Explore the world link belongs to Inside Dark Pawns; history navigation stays in the history section, without the redundant early bullet-linked pair. The same transparent image displays at 360×120px desktop and approximately 224×75px mobile. Broad chapter-design inspiration from *The Gunslinger* informs the composition; no existing book illustration was supplied as input.

FINISH: Responsive captures and site/Go verification are recorded below. Fresh independent `landing_flow_finish_review` disposition: **ready**, no material findings, after inspecting source and all four current captures. Retain source/prompt provenance for both shipping rasters, copy behavior, six download links, and the Markdown endpoint. Preserve global DESIGN.md and design.json for this ordinary extension.

## Implementation and provenance

`index.astro` centers the invitation within a 48rem maximum column. The image is decorative (`alt=""`), with native alpha over the real paper. The paper remains at 10% opacity and is suppressed in forced-colors mode. The installation strip stacks on mobile. The connection disclosure works natively without JavaScript, has explicit keyboard focus, and changes its settings grid into labeled rows on mobile. Rules and derived counts form a semantic section headed Inside Dark Pawns, with one Explore the world action. The full-history link remains in the archive/history section; Latest from the restoration now follows it at the end. `/download/` remains unchanged.

- `src/assets/chapter/temple-altar.png`: original generated interpretation of room 8004, not recovered artwork. Built-in ImageGen exposes no model/version selector. README and `temple-altar.prompt.txt` record origin, exact prompt, 2172×724 RGBA original, and 720×240 RGBA serving asset reduced with Lanczos. Prompt is embedded in the PNG. Only its display size and placement changed in this correction.
- `src/assets/paper/book-page.jpg`: unmodified-image Wikimedia thumbnail of a blank page from the 1902 *Armide* scan, supplied by Internet Archive Book Images. README records source, retrieval date, and “no known copyright restrictions”; JPEG comment also carries provenance. Both assets are imported through Vite.

The shared CopyCommand remains selectable without JavaScript, with success feedback and manual selection on clipboard denial. Downloads retain macOS/Linux/Windows ARM64 and AMD64 archives, verification/setup/hosting guidance, and a Markdown twin.

## Verification and documentation comparison

This documentation pass inspected the current source and `.impeccable/review/landing-flow-user.png`, `landing-flow-expanded.png`, `landing-flow-mobile-expanded.png`, and `landing-flow-desktop.png`. The captures show the collapsed and expanded disclosure, labeled settings, focus outline, and distinct world section. Mobile/desktop QA iframe captures include a white parent canvas that is not site styling. Hero assets and their provenance remain unchanged from the centered-chapter implementation.

Final implementation handoff reports no overflow or clipped commands at 320/390/768/1280/1440px, with opening download positions unchanged and visible within tested 844px phone and 900px desktop heights. Enter/Space toggling and focus were verified. Site-check, formatting, Go build/vet/test/lint all pass; scoped layout detector reports `[]` before and after. These execution results are handoff evidence; this documentation pass inspected source and captures rather than rerunning gates.

Read-only comparison with `origin/main` confirms no changes to `DESIGN.md`, `.impeccable/design.json`, `PRODUCT.md`, or global CSS. Incumbent tokens remain authoritative. Pre-existing documentation drift remains: DESIGN.md describes an earlier front-page composition/title sizing, and the sidecar condenses its narrative. Current surface choices are recorded here; global document repair is outside scope. Only this brief was edited in this documentation pass.
