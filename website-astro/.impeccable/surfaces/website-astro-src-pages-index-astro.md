---
version: 1
slug: "website-astro-src-pages-index-astro"
primary_target: "website-astro/src/pages/index.astro"
related_targets: ["website-astro/src/pages/download.astro"]
---

# Homepage chapter opening and downloads

Visitor modes: Persuade (homepage), Operate/Read (downloads). Human players come first, followed by people installing a separate world. Scope: homepage opening, `/download/`, shared copy control, footer discovery, search, and Markdown representation.

Final user correction, October 23, 2026: enlarge the same approved altar image, remove the opening Temple Altar room-description, and center the image, title, and play action like a first chapter. Preserve the real paper texture. This contract supersedes earlier compositions; mock studies remain under `mocks/`.

## Direction contract

THESIS: “You type and the world answers.” opens a used-paperback chapter, with clear paths to play here or run a separate world.

OWN-WORLD: Preserve Haunted Paperback: cream, charcoal rules, serif headings, rare oxblood, machine-only monospace, and the faint real paper scan. The approved charcoal altar image remains the chapter mark. No new global palette, elevation, or motion system.

STORY: Read the centered invitation, enter the shared world or install a separate one. The opening room-description is removed. Downloads explain setup, first immortal, verification, and hosting.

FIRST VIEWPORT: Centered altar image above the two-line title, short lede, and primary play action. A compact Unix installation strip and MUD-client bridge follow. Both actions and the download link stay early on mobile. `/download/` retains its installation/setup split and six platform archives.

FORM: Code-led local extension of the incumbent editorial system. The same transparent image displays at 360×120px desktop and approximately 224×75px mobile. Broad chapter-design inspiration from *The Gunslinger* informs the composition; no existing book illustration was supplied as input.

FINISH: Final responsive captures and site/Go verification are recorded below. Retain source/prompt provenance for both shipping rasters, copy behavior, six download links, and the Markdown endpoint. Preserve global DESIGN.md and design.json for this ordinary extension.

## Implementation and provenance

`index.astro` centers the invitation within a 48rem maximum column. The image is decorative (`alt=""`), with native alpha over the real paper. The paper remains at 10% opacity and is suppressed in forced-colors mode. The installation strip stacks on mobile. `/download/` remains unchanged.

- `src/assets/chapter/temple-altar.png`: original generated interpretation of room 8004, not recovered artwork. Built-in ImageGen exposes no model/version selector. README and `temple-altar.prompt.txt` record origin, exact prompt, 2172×724 RGBA original, and 720×240 RGBA serving asset reduced with Lanczos. Prompt is embedded in the PNG. Only its display size and placement changed in this correction.
- `src/assets/paper/book-page.jpg`: unmodified-image Wikimedia thumbnail of a blank page from the 1902 *Armide* scan, supplied by Internet Archive Book Images. README records source, retrieval date, and “no known copyright restrictions”; JPEG comment also carries provenance. Both assets are imported through Vite.

The shared CopyCommand remains selectable without JavaScript, with success feedback and manual selection on clipboard denial. Downloads retain macOS/Linux/Windows ARM64 and AMD64 archives, verification/setup/hosting guidance, and a Markdown twin.

## Verification and documentation comparison

This documentation pass inspected current source, the updated chapter-asset README, and `.impeccable/review/centered-chapter-user.png`, `centered-chapter-mobile.png`, and `centered-chapter-desktop.png`. The first is the native user viewport; the other two are scaled QA iframe captures whose white parent canvas is not site styling. Captures confirm the centered image/title/action, absent opening room prose, and compact installation strip.

Final implementation handoff reports no overflow or clipped commands at 320/390/768/1280/1440px, with the download link visible within tested 844px phone and 900px desktop heights. Site-check, formatting, Go build/vet/test/lint all pass. These execution results are handoff evidence; this documentation pass inspected source and captures rather than rerunning gates.

Read-only comparison with `origin/main` confirms no changes to `DESIGN.md`, `.impeccable/design.json`, `PRODUCT.md`, or global CSS. Incumbent tokens remain authoritative. Pre-existing documentation drift remains: DESIGN.md describes an earlier front-page composition/title sizing, and the sidecar condenses its narrative. Current surface choices are recorded here; global document repair is outside scope. Only this brief was edited in this documentation pass.
