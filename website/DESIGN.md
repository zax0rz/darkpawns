---
name: Dark Pawns
description: A dark-fantasy MUD from 1994, presented as a haunted paperback archiving a game that refuses to die.
colors:
  paper: "#EFE7D6"
  paper-deep: "#E5DAC1"
  ink: "#1A1614"
  ink-muted: "#56504A"
  oxblood: "#A8201A"
  oxblood-deep: "#7A1812"
  oxblood-on-dark: "#CF4B42"
  registration: "#0F0F0F"
  online: "#3A8A3A"
typography:
  display:
    fontFamily: "DM Serif Display, Georgia, serif"
    fontSize: "clamp(2.5rem, 5vw + 1rem, 3rem)"
    fontWeight: 400
    lineHeight: 1.1
  headline:
    fontFamily: "DM Serif Display, Georgia, serif"
    fontSize: "clamp(1.75rem, 3vw + 0.5rem, 2rem)"
    fontWeight: 400
    lineHeight: 1.15
  title:
    fontFamily: "DM Serif Display, Georgia, serif"
    fontSize: "clamp(1.25rem, 2vw + 0.5rem, 1.5rem)"
    fontWeight: 400
    lineHeight: 1.2
  body:
    fontFamily: "Source Serif 4, Georgia, serif"
    fontSize: "1rem"
    fontWeight: 400
    lineHeight: 1.6
  label:
    fontFamily: "DM Serif Display, Georgia, serif"
    fontSize: "0.8rem"
    fontWeight: 400
    letterSpacing: "0.08em"
  mono:
    fontFamily: "JetBrains Mono, Fira Code, monospace"
    fontSize: "0.9rem"
    fontWeight: 400
    lineHeight: 1.5
rounded:
  sharp: "0"
  micro: "2px"
spacing:
  xs: "0.25rem"
  sm: "0.5rem"
  md: "1rem"
  lg: "2rem"
  xl: "4rem"
components:
  button-primary:
    backgroundColor: "{colors.oxblood}"
    textColor: "{colors.paper}"
    rounded: "{rounded.sharp}"
  button-primary-hover:
    backgroundColor: "{colors.oxblood-deep}"
    textColor: "{colors.paper}"
  button-secondary:
    backgroundColor: "transparent"
    textColor: "{colors.ink}"
    rounded: "{rounded.sharp}"
  button-secondary-hover:
    backgroundColor: "{colors.ink}"
    textColor: "{colors.paper}"
  card:
    backgroundColor: "{colors.paper-deep}"
    textColor: "{colors.ink}"
    rounded: "{rounded.micro}"
---

# Design System: Dark Pawns

## Overview

**Creative North Star: "The Haunted Paperback"**

Dark Pawns is a dark-fantasy world first built in 1994 and left for dead around 2010. This site is what you find when you pull that world off a dusty shelf: a warm, yellowed mass-market paperback with a newspaper folded inside it, still faintly warm, still running. The whole system points at one feeling — an archive of a game that refuses to die. Every surface should read like print that outlived its era, not like software chasing this one.

The look is cream paper and charcoal ink, structured with the stark hairline logic of a broadsheet and marked with a single oxblood accent used like a librarian's stamp. There are no gradients, no shadows, no soft glows. Depth comes from the weight of a rule and the faint tonal shift between two creams, the way it would on a printed page. Warmth carries the reading; brutalist structure carries the navigation. The literary, "mythic" voice is reserved for actual game content (lore, classes, rooms) — the site's own chrome stays factual and dry.

The primary visitor is a **human player** (see `PRODUCT.md`): a returning veteran or a MUD-curious newcomer. Agents and researchers are first-class in the game but are not the headline of the site. Persuasion, when it comes, is the depth of the world itself, never a product pitch.

**Key Characteristics:**
- Warm cream paper, charcoal ink, one oxblood stamp.
- Flat by physical law: hairline rules and tonal cream layering, never shadow.
- Serif for reading and headlines; monospace only for game and machine chrome. No sans-serif anywhere.
- Sharp corners (≤2px). The page is printed, not rounded.
- Editorial/archival register. Never SaaS.
- Sourced or silent: nothing asserted on a surface without a primary source.

## Colors

A warm print palette: two creams, two inks, and one rare oxblood stamp. Green appears only as a live-status signal.

### Primary
- **Oxblood** (`#A8201A`): the single accent. Links, primary CTAs, active nav, the pawn mark, and error/offline status. On dark surfaces (the CRT client) it lightens to **Oxblood-on-Dark** (`#CF4B42`) for contrast; pressed/hover state deepens to **Oxblood-Deep** (`#7A1812`).

### Neutral
- **Paper** (`#EFE7D6`): the default page background. Warm cream, never white.
- **Paper-Deep** (`#E5DAC1`): elevated surfaces — cards, the terminal chassis, sidebars. The only "elevation" the system has.
- **Ink** (`#1A1614`): body text, headings, and every hairline rule and border (`--rule` is Ink).
- **Ink-Muted** (`#56504A`): secondary text, captions, metadata, and unknown/loading status.
- **Registration** (`#0F0F0F`): the "black plate" for the densest body copy on cream.

### Functional
- **Online** (`#3A8A3A`): forest green, used *only* for online/success status. It is a signal color, not a palette color — never decorative.

### Named Rules
**The One Oxblood Rule.** Oxblood is the only accent and appears on roughly ≤10% of any screen. Its rarity is the entire point; the moment a second accent shows up, or oxblood becomes a fill instead of a mark, the system is broken.

**The Paper-Not-White Rule.** Pure `#FFFFFF` and `#000000` never appear, with one exception: the CRT canvas at `/play` (`#0a0908`). Every other surface is Paper or Paper-Deep; every "black" is Ink.

**The Paper Theme Lock.** The site is one theme, chosen once: warm paper, light. No per-section theme flips, no dark-mode toggle. The only dark surface is the CRT canvas at `/play`, and it is a component, not a theme. (Consistency lock, adapted from tasteskill.dev, alongside The One Oxblood Rule for color and The 2px Rule for shape.)

## Typography

**Display Font:** DM Serif Display (Georgia fallback)
**Body Font:** Source Serif 4 (Georgia fallback)
**Mono Font:** JetBrains Mono (Fira Code fallback)

**Character:** A paperback cover meeting a wire-service dispatch. The display serif has the high-contrast drama of a genre-fiction title; the body serif is a comfortable, bookish read; the mono is the machine speaking in its own voice. Nothing here is sans-serif.

### Hierarchy
- **Display** (400, `clamp(2.5rem, 5vw + 1rem, 3rem)`, 1.1): the DARK PAWNS wordmark, page titles, splash. The paperback-cover register (the splash wordmark amplifies this to `clamp(3.5rem, 8vw + 1rem, 5rem)`).
- **Headline** (400, `clamp(1.75rem, 3vw + 0.5rem, 2rem)`, 1.15): section titles (h2).
- **Title** (400, `clamp(1.25rem, 2vw + 0.5rem, 1.5rem)`, 1.2): sub-sections (h3).
- **Body** (400, `1rem`, 1.6): reading copy. Optimal measure 45–75ch.
- **Label** (400, `~0.8rem`, letter-spacing `0.08em`, uppercase): the section bar and editorial eyebrows, in the display serif.
- **Mono** (400, `~0.9rem`, 1.5): game chrome only — terminal output, command logs, in-game stats, install commands, commit hashes.

### Named Rules
**The Two-Serif-and-a-Terminal Rule.** Display serif for headings and the wordmark, body serif for reading, monospace *only* for game and machine chrome. No sans-serif font enters this system, and no font is loaded that isn't used (the `Archivo Narrow` link is dead weight — remove it).

## Layout

A centered broadsheet. The global container is `--max-width: 72rem`; editorial reading columns narrow to `--content-width: 42rem` (≈45–75ch). The home page is a two-column broadsheet grid (main dispatch column + sidebar). Spacing follows a fixed scale: `xs 0.25rem`, `sm 0.5rem`, `md 1rem`, `lg 2rem`, `xl 4rem` — density is tight and print-like, not airy SaaS whitespace. The persistent section bar sits at `--nav-height: 3.5rem`.

**Breakpoints:** mobile `640px` (single column; the section bar becomes one horizontally-scrolling row, no hamburger, no JS), tablet `1024px` (two-column list grids).

### Hero discipline (adapted from tasteskill.dev)
For any landing or hero surface: headline **two lines maximum** on desktop; subtext **≤20 words and 4 lines**; the primary action is **visible without scrolling**; the section bar stays a **single line, ≤80px tall**. A hero that needs a scroll cue has failed, and there are no scroll cues here.

## Elevation & Depth

**The system is flat.** There are no box-shadows, no blurs, no glows anywhere. Depth is conveyed two ways, both drawn from print: **hairline rules** (1–2px Ink borders) and **tonal cream layering** (Paper-Deep sits "above" Paper). That is the entire elevation vocabulary.

### Named Rules
**The Flat-Ink Rule.** Surfaces are flat at rest and flat on interaction. If something needs to feel raised, it gets a rule or a shift to Paper-Deep — never a shadow. State changes (hover, active) are conveyed by color and border, not by lifting.

## Shapes

Sharp, printed geometry. Corners are square by default; the maximum radius anywhere is `2px` (`--rounded-micro`), used on cards, inputs, and small frames to soften a hairline join, never to look "rounded." The pawn mark and the brutalist rules reinforce a stamped, letterpress silhouette.

### Named Rules
**The 2px Rule.** Nothing on this site is rounder than 2px, with one exception: a true circle (`50%`) for a genuine dot (status indicator) or avatar. There are currently two violations to fix — a `9999px` pill and a `20px` radius in `style.css`; both must come down to `2px` or become sharp.

## Components

### Buttons
- **Shape:** sharp (radius `0`), all-caps display serif, generous letter-spacing.
- **Primary (`.btn-primary`):** Oxblood background, Paper text. Hover deepens to Oxblood-Deep.
- **Secondary (`.btn-secondary`):** transparent background, Ink text, 1px Ink border. Hover inverts to an Ink fill with Paper text.

### Cards / Containers (`.card`)
- **Corner:** micro (`2px`). **Background:** Paper-Deep. **Border:** 1px Ink hairline. **Shadow:** none (see Elevation). Used for list items, section portals, guides.

### Inputs / Fields
- Warm Paper background, 1px Ink border, `2px` corners. Focus is an **oxblood outline**, not a glow or a shift to blue.

### Navigation
- **Section bar:** one shared, ruled bar (hairline top and bottom, newspaper style) listing the seven sections — Play · World · Help · Database · Map · Dispatch · About. All-caps, letter-spaced, equal spacing; active section marked in Oxblood. Mobile: one horizontally-scrolling row, no hamburger.
- **Breadcrumbs:** one shared partial, `Section » Subsection » Page`; ancestors are oxblood links, current page plain. Not shown on section indexes or top-level leaves.
- **Section card grid:** one shared card-grid pattern for all section indexes (title, description, meta count, oxblood hover) from a single partial and CSS block. **MUST** extend these shared patterns — no bespoke per-section nav, breadcrumbs, or index layouts.

### The Web Client (`/play`) — signature component
A physical CRT console in a paper housing. Chassis `.terminal-frame` in Paper-Deep with Ink separators; inner `#terminal` locked to a dark CRT canvas (`#0a0908`) framed by a crisp 2px Ink border. Controls and connectivity dots in Oxblood and Ink, never arbitrary grays. This is the ONE legitimate "terminal UI" on the site (see Don'ts) because it is the actual game.

### The Interactive Map (`/map`) — signature component
Vintage ink-and-parchment cartography. Paper-Deep canvas, room nodes and connection lines stroked in Ink or Oxblood, brutalist `.btn-secondary` controls, cream inputs with oxblood focus.

### The Codex Database (`/database`) — signature component
A full-viewport, monospace codex browser for the game's ~1,319 mobs and ~1,672 items (data from `/data/database.json`). A split layout (`#db-page`) sits under an ink top-rule; it opens with a "skyline" strip of Paper-Deep stat panels (`.skyline-panel`), then a searchable, filterable index. Monospace throughout: this is machine-readable game data on display, not editorial prose. Along with `/play` and `/map`, it is one of the three app-like surfaces where structure outranks expression. NOTE: `.skyline-panel` currently animates a `box-shadow` on hover, a Flat-Ink Rule violation to remove.

## Do's and Don'ts

Concrete guardrails. These are enforceable by `impeccable detect` and by review; the anti-slop bans below are as binding as the palette.

### Do:
- **Do** lead with the specificity of the world. The archive is the pitch (see `PRODUCT.md`).
- **Do** keep Oxblood rare (The One Oxblood Rule) and every "black" as Ink (The Paper-Not-White Rule).
- **Do** convey depth with rules and cream layering only (The Flat-Ink Rule).
- **Do** write all public copy in `docs/brand-voice.md` Layer 3 (Frontline's Mythic Admin register).
- **Do** treat the human player as the primary visitor in every layout and hierarchy decision.
- **Do** keep monospace for game and machine chrome exclusively.

### Don't:
- **Don't** use em-dashes in UI copy — headlines, buttons, labels, nav, captions. (Sentence-level prose in long-form articles may, sparingly; chrome may not.)
- **Don't** use section-numbering eyebrows ("01 · Features", decorative "PART I" counters) or decorative colophons and filler tags. The `FILED UNDER … SEE ALSO` line is banned.
- **Don't** build three-equal-card feature rows. Use asymmetric or two-column editorial layouts instead.
- **Don't** build fake terminal or dashboard UIs. The only real terminal is the game client at `/play`.
- **Don't** use gradients, box-shadows, glows, or any border radius > 2px (kill the `9999px` and `20px` offenders).
- **Don't** use pure `#FFFFFF`/`#000000` (except the CRT canvas), sans-serif fonts, or any color outside this palette. Status is Oxblood (offline/error), Online green (success), Ink-Muted (unknown).
- **Don't** use gimmicky "spooky" assets (bats, cobwebs, dripping-blood text, flaming dividers). Cohesion comes from typography and editorial layout, not costume.
- **Don't** adopt SaaS marketing patterns: hero-with-gradient, "Get Started free" funnels, pills-on-images, mesh-blob backgrounds, testimonial carousels.
- **Don't** assert any fact, stat, or lore without a primary source (The Sourced-or-Silent Rule; see `PRODUCT.md`).
- **Don't** put version labels ("v2.0"), locale/time/weather strips, scroll cues, or down-arrow icons on a hero or marketing surface (tasteskill.dev).
- **Don't** float labels, tags, or photo-credit captions on top of images.
- **Don't** hand-roll `window.addEventListener('scroll')`; use `IntersectionObserver` or CSS scroll-driven animation. Most of this site needs no scroll JS at all.
- **Don't** double-rule list rows (border-top *and* border-bottom on every item); one hairline between items is the newspaper way.
- **Don't** use status dots decoratively; they appear only for genuine live status (server online/offline).

### Named Rules
**The Sourced-or-Silent Rule.** No claim, number, date, or piece of lore appears on any surface unless it traces to a primary source. When there is no source, the surface stays silent rather than plausible. (This is why the hallucinated "First Age" and the drifting room counts were slop.)

**The Archive-Is-The-Pitch Rule.** Marketing surfaces persuade through the depth, age, and specificity of the world, never through product-launch conventions. A 30-year-old world does not need a gradient hero.

## Posting News

- **Dispatch posts are manual:** a news post is a Markdown file in `content/news/`, from the `archetypes/news.md` archetype; `make new-post TITLE="..."` wraps `hugo new`.

## History facts

All dates, founders, and lineage are governed by `data/history.toml` (the single source of truth) and `content/community/history/timeline.md` (the primary source). Never hardcode a founding fact anywhere else. Canonical: founded **September 1994**, ran to ~2010, Go rewrite 2026.
