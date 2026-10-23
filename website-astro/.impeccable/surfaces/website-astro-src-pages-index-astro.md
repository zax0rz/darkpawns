---
version: 1
slug: "website-astro-src-pages-index-astro"
primary_target: "website-astro/src/pages/index.astro"
related_targets: ["website-astro/src/pages/download.astro"]
---

# Homepage chapter opening and downloads

Visitor modes: Persuade (homepage), Operate/Read (downloads). Human players come first, followed by people installing a separate world. Scope: homepage opening, `/download/`, shared copy control, footer discovery, search, and Markdown representation. Final user direction, October 23, 2026: a subtle chapter opening with a small illustration, broad book-design inspiration from *The Gunslinger*, and the existing used-paper texture. This contract supersedes earlier mock compositions; studies and their archived build state remain under `mocks/`.

## Direction contract

THESIS: “You type and the world answers.” reads as an invitation at the opening of a used paperback chapter, with clear paths to play here or run a separate world.

OWN-WORLD: Preserve Haunted Paperback: cream, charcoal rules, serif headings, rare oxblood, machine-only monospace. Retain the faint real paper scan. A tiny charcoal altar vignette supplies the chapter mark; no new global palette, elevation, or motion system.

STORY: Encounter the invitation and the original Temple Altar prose, then play in the shared world or install a separate one. Downloads explain setup, first immortal, verification, and hosting.

FIRST VIEWPORT: Small vignette above the two-line title and primary play action, beside unboxed room prose. Compact Unix installation strip and MUD-client bridge follow. Mobile places play and installation actions before the passage. `/download/` retains its installation/setup split and six explicit platform archives.

FORM: Code-led local extension of the incumbent editorial system. Keep the vignette at 180×60px desktop and 132×44px mobile: enlarging its scratchy detail weakens it. Broad chapter-design inspiration does not authorize copying an existing book illustration.

FINISH: Fresh independent `chapter_finish_reviewer` verdict **SHIP**, no material findings. Retain source/prompt provenance for both shipping rasters, responsive captures, command/copy checks, six download links, Markdown endpoint, and verbatim room evidence. Global DESIGN.md and design.json remain intact for this ordinary extension.

## Implementation and provenance

`index.astro` imports both assets through Vite. The altar is decorative (`alt=""`), with native alpha over the real paper; the texture remains at 10% opacity and is suppressed in forced-colors mode. The final addition preserves the paperback layout/content, apart from reducing mobile installation-strip top spacing by 8px to keep actions early. `/download/` is unchanged.

- `src/assets/chapter/temple-altar.png`: original generated interpretation of room 8004, not recovered artwork. Built-in ImageGen exposes no model/version selector; no artist or book image was supplied as input. The README and exact `temple-altar.prompt.txt` describe origin, intent, 2172×724 RGBA original, and 720×240 RGBA serving asset reduced with Lanczos. Prompt is embedded in the PNG.
- `src/assets/paper/book-page.jpg`: unmodified-image Wikimedia thumbnail of a blank page from the 1902 *Armide* scan, supplied by Internet Archive Book Images. Its README records source, download, retrieval date, and “no known copyright restrictions”; the JPEG comment carries provenance too.

The shared CopyCommand remains selectable without JavaScript, with success feedback and manual selection on clipboard denial. Downloads retain macOS/Linux/Windows ARM64 and AMD64 archives, verification/setup/hosting guidance, and a Markdown twin.

## Verification and documentation comparison

This pass inspected current source, both asset READMEs, the exact generation prompt, and `.impeccable/review/chapter-user.png` (native 987×816 user viewport), `chapter-mobile.png`, and `chapter-desktop.png`. The latter two are scaled QA iframe captures: their white parent canvas is not site styling. Previous source comparison independently confirmed the room passage matches `lib/world/wld/80.wld` room 8004 after whitespace folding; that passage is unchanged.

Final implementation/reviewer handoff reports no overflow or clipped commands at 320/390/768/1280/1440px; download-link bottoms are respectively 824/774/834/806/818px. Site-check, formatting, Go build/vet/test/lint pass; tests were rerun successfully after an interrupted exit 143. Raster provenance scan is clean. Nine type-size detector findings are advisory and nonblocking. These execution results are handoff evidence, not gates rerun by this documentation pass.

Read-only comparison with `origin/main` confirms no changes to `DESIGN.md`, `.impeccable/design.json`, `PRODUCT.md`, or global CSS. The final extension preserves incumbent tokens. Pre-existing documentation drift remains: DESIGN.md describes an earlier front-page composition/title sizing, and the sidecar condenses its narrative. This task records current surface choices here without repairing global documents. Only this brief was edited in this documentation pass.
