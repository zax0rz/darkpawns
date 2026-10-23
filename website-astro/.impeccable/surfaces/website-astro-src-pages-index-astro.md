---
version: 1
slug: "website-astro-src-pages-index-astro"
primary_target: "website-astro/src/pages/index.astro"
related_targets: ["website-astro/src/pages/download.astro"]
---

# Homepage opening and downloads

Visitor modes: Persuade (homepage), Operate/Read (downloads).
Audience: players first, then people installing and hosting their own world.
Scope: homepage opening, /download/, shared copy control, footer discovery, search and Markdown representation. Existing world content and visual system remain authoritative.
User approved the composition and copy direction in chat on October 23, 2026.

## Direction contract

THESIS: Make playing here and running a separate world immediately understandable, using direct actions and real release assets.

OWN-WORLD: Preserve Haunted Paperback: cream, charcoal rules, serif headings, oxblood primary play action, monospace commands. No new imagery or decorative UI.

STORY: Recognize a playable text world from 1994; enter the shared server or install a local world. Download instructions explain setup, first immortal, verification, and the separate hosting step.

FIRST VIEWPORT: Two-line existing headline above a short lede; play action left, installation command and copy button right. On mobile, compact stacked choices. Ports follow. The download page starts with installation beside setup steps, followed by six explicit platform downloads.

FORM: User-selected editorial split and platform download table; narrow extension of the incumbent world. Code-led implementation. No concept roll: composition explicitly approved by user.

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance

No raster assets are created. Verify clipboard success, clipboard denial/manual selection, no-JS command availability, keyboard focus, 320px through desktop widths, six release URLs, and Markdown endpoint.


## Earlier split implementation and design-extension evidence

Documented October 23, 2026. Finish review disposition: **SHIP**, with no material findings. This is an ordinary extension of the incumbent Haunted Paperback system; the approved surface composition does not replace the visual world.

The homepage keeps the historical headline and puts the oxblood browser-play action first. A ruled editorial split adds the Unix installation command and the route to Windows, release archives, and hosting instructions. It stacks into two compact choices on mobile. Existing world excerpts, derived figures, Dispatch, field guides, archive, and restoration content remain below the opening.

The download page extends the same paper, ink rules, display/body serifs, and machine-only monospace. Its installation/setup split becomes a single column on narrow screens. A semantic table names six OS/processor archives, followed by verification, operation, and hosting guidance. The shared CopyCommand uses a cream field and square ink border; its button inverts to ink on hover, and the global focus treatment applies. The command remains a read-only selectable textarea without JavaScript. Script inspection confirms success feedback and a denial path that selects the command for manual copying; interactive browser verification belongs to the finish review.

Evidence checked for this documentation pass:

- Read `src/pages/index.astro`, `src/pages/download.astro`, `src/components/CopyCommand.astro`, `src/scripts/copy-command.ts`, `src/lib/download.ts`, and `src/pages/download.md.ts` against this direction contract, `PRODUCT.md`, `DESIGN.md`, `.impeccable/design.json`, and the global CSS tokens/focus rule.
- Visually inspected `.impeccable/review/home-desktop-fold.png`, `home-mobile-fold.png`, `download-desktop.png`, and `download-mobile.png`. The screenshots show the approved split, compact stacked mobile choices, readable commands, setup sequence, six archive rows, and the incumbent flat editorial treatment. They are review captures, not shipping raster assets.
- Inspected built `dist/index.html`, `dist/download/index.html`, and the presence of `dist/download.md`. The built download HTML contains exactly six distinct v0.1.4 archive links: macOS/Linux/Windows, each ARM64 and AMD64. This confirms generated link structure; release availability and interaction testing are separate finish-review evidence.
- Read-only comparison with `origin/main` confirms no changes to `DESIGN.md`, `.impeccable/design.json`, `PRODUCT.md`, or global CSS. No new global colors, fonts, elevation, shape vocabulary, or motion system were introduced. No new raster artwork requires provenance.

Documentation drift retained: the incumbent `DESIGN.md` front-page description records its earlier title sizing and title/connection composition; the selected surface contract and source now hold the approved opening. The existing sidecar narrative condenses the Markdown narrative rather than copying it verbatim. Neither condition was created as a global system change by this extension. Preserve both global documents for this scope; any later system-document refresh is a separate task.


## Homepage refinement

User-directed follow-up: omit the Serapis/Tracer restoration introduction and “at Hero”; use the MUD-client connection as the bridge into world details. The connection guide, host, and ports now share a full-width Paper-Deep strip below the play/install choices. Download page unchanged.

Evidence: production `make site-check`; Go build/vet/test/lint and formatting; fresh visible user-viewport screenshot `website-astro/.impeccable/review/home-bridge-user.png`. Independent follow-up review: **ship**, no material findings. The bridge reuses existing cream, rule, and type tokens, so DESIGN.md and design.json remain unchanged.


## Used-paperback direction (supersedes illustrated mockups)

User feedback: “honestly I like the first one but I'm kind of skeptical of AI generated imagery. I think we need to go more subtle, more paperback, like it's been printed, but also, this game is old, text-based, etc... it should feel like a USED book someone picked up off the shelf”.

The three generated studies remain as research references. None is an approved spatial comp. Their pending build state is archived alongside the studies; this refinement is code-led. The invitation/compact installation composition of the first study carries forward, with original game text replacing the illustration.

THESIS: A worn paperback title page earns the invitation through the game's own words and a real paper surface.
OWN-WORLD: Existing cream/charcoal/oxblood and serif typography, with a faint sourced scan of a blank 1902 book page. No generated hero artwork, new palette, shadows, gradients, or fake terminal framing.
STORY: Read the invitation, enter the shared world or install a separate one, and encounter the original Temple Altar description as printed prose.
FIRST VIEWPORT: Invitation and primary play action beside the unboxed original room passage; a compact install strip and client bridge underneath. On mobile both actions precede the passage.
FORM: Code-led local extension based on user-selected frontispiece balance; no approved generated comp. Download page remains unchanged.
FINISH: independent review, responsive capture, copy verification, source-text verification, and source provenance for the single new raster.


## Final used-paperback implementation and evidence

October 23, 2026. The used-paperback direction supersedes the earlier opening compositions and their no-raster assumption. Fresh independent `paperback_finish_reviewer` disposition: **SHIP**, no material findings.

The final code-led opening places “You type and the world answers.” beside the unboxed Temple Altar passage. The compact installation strip and MUD-client bridge follow; mobile places both play and installation actions before the passage. `/download/` remains unchanged. The three generated studies remain in `mocks/`, with their unapproved composition/build state archived in `mocks/illustrated-build-state/`; no generated hero artwork ships.

Documentation verification: inspected current `index.astro`, the asset README and embedded JPEG provenance comment, plus `paperback-user-final.png`, `paperback-mobile-final.png`, and `paperback-desktop-final.png` under `.impeccable/review/`. The latter two are scaled QA iframe views within a white parent canvas; that canvas is not site styling. The invitation, printed room prose, compact command, and restrained paper surface retain the incumbent hierarchy and palette. Independently confirmed the opening quote matches room 8004 in `lib/world/wld/80.wld` after whitespace folding.

The sole new shipping raster, `src/assets/paper/book-page.jpg`, is the unmodified-image 960px Wikimedia scan of a blank page from the 1902 *Armide*, applied at 10% CSS opacity and imported through Vite. `src/assets/paper/README.md` records the Commons file record, Internet Archive source, downloaded thumbnail, retrieval date, and “no known copyright restrictions” rights statement; the JPEG comment also records its source and intended use. It supplies paper material, not illustrated game content. Forced-colors mode suppresses the texture.

Finish-review evidence reported for this final state: widths 320, 390, 768, 1280, and 1440px without horizontal overflow or clipped commands, with both actions early; production site-check, Go build/vet/test/lint, and formatting pass. Nine type-size detector findings are advisory and nonblocking. These execution results are recorded from the implementation/finish-review handoff; this documentation pass inspected source and captures rather than rerunning those gates.

Read-only comparison with `origin/main` again confirms `DESIGN.md`, `.impeccable/design.json`, `PRODUCT.md`, and global CSS are unchanged. This user-directed local extension introduces no global token changes. Preserve the earlier-noted front-page description drift and condensed sidecar narrative; their repair remains outside this scope. Only this surface brief was edited in this documentation pass.
