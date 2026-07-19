---
visual_theme: "Retro Stephen King paperback, modern brutalist standards"
colors:
  paper: "#EFE7D6"          # Page background (warm cream)
  paper_deep: "#E5DAC1"     # Elevated surfaces, cards, console chassis
  ink: "#1A1614"            # Primary text, borders (warm near-black)
  ink_muted: "#56504A"      # Secondary/muted text
  rule: "#1A1614"           # Hairlines, grid borders
  accent: "#A8201A"         # Oxblood - links, primary CTAs, active highlights
  accent_deep: "#7A1812"    # Hover/pressed state for oxblood elements
  registration: "#0F0F0F"   # Body copy "black plate" on cream
  online: "#3A8A3A"         # Forest green - online/success status only
typography:
  font_display: "'DM Serif Display', 'Georgia', serif"
  font_body: "'Source Serif 4', 'Georgia', serif"
  font_mono: "'JetBrains Mono', 'Fira Code', monospace"
spacing:
  space_xs: "0.25rem"       # 4px (tight padding, micro-spacing)
  space_sm: "0.5rem"        # 8px (paragraphs, list items, card gaps)
  space_md: "1rem"          # 16px (standard gap, inner card padding)
  space_lg: "2rem"          # 32px (sections, main layouts, sidebar padding)
  space_xl: "4rem"          # 64px (hero elements, splash top-padding)
layout:
  max_width: "72rem"         # 1152px (global container width)
  content_width: "42rem"     # 672px (optimal editorial reading width)
  breakpoints:
    mobile: "640px"          # Single column, card stack, scrolling nav bar
    tablet: "1024px"         # Two column list grids
---

# Dark Pawns — Visual Design System Manifest

This document is the authoritative design system manifest and visual spec for the **Dark Pawns Website**. It bridges design intent and code implementation, serving as a machine-readable token contract for AI agents and a visual guide for human developers.

---

## 1. Core Philosophy & Design Voice

The design philosophy is defined as:
> **"Retro Stephen King paperback, but with modern brutalist standards."**

*   **Warmth & Texture:** The primary background is a warm, tactile cream paper (`--paper`), rather than generic white. All body text is printed in a dark, organic charcoal ink (`--ink`), creating a high-contrast but soft, bookish reading experience.
*   **Oxblood Accent:** Primary action states, navigation links, and crucial highlights use a rich, deep oxblood red (`--accent`).
*   **Brutalist Structuring:** The layouts are stark and logical, bounded by thin black borders (`--rule`) and structured list cards. There are no gradients, shadows, or rounded corners (except for micro-radius borders of 2px).
*   **Archival Seriousness:** The site chrome (menus, footers, buttons) stays clean, professional, and factual. The rich narrative "mythic" descriptions are reserved entirely for actual game content (lore, classes, races).

---

## 2. Typography

*   **Display (wordmark, h1–h3):** **DM Serif Display** (Georgia fallback). Used for the DARK PAWNS wordmark, page headings, and section titles — the paperback-cover register.
*   **Body Copy:** **Source Serif 4** (Georgia fallback). Generous line-height (`1.6`), optimal reading width (`45ch` to `75ch`), explicit spacing between paragraphs.
*   **Monospace:** **JetBrains Mono** (Fira Code fallback). Reserved for game chrome: terminal clients, command logs, in-game stats, commit hashes, and technical instructions.

---

## 3. Component Specifications

### 3.1 Buttons
*   **Primary CTA (`.btn-primary`):** Oxblood red (`--accent`) background with cream (`--paper`) text. Sharp edges, all-caps display font, high letter-spacing. On hover, transitions to a deeper oxblood red (`--accent-deep`).
*   **Secondary CTA (`.btn-secondary`):** Transparent background with ink-dark (`--ink`) text and a thin `--rule` border. On hover, swaps to an ink background with cream text.

### 3.2 Cards & Elevated Surfaces (`.card`)
*   Used for list items, guides, and section portals.
*   Uses a slightly darker cream background (`--paper-deep`), bounded by a thin 1px `--rule` border. Sharp corners, with no box-shadows.

### 3.3 The Web Client (`/play`)
*   Designed like a physical CRT command console embedded inside a retro paper housing:
    *   **The Chassis:** The outer panel (`.terminal-frame`) uses the warm `--paper-deep` cream background with thin ink-dark panel separators, integrating it seamlessly with the rest of the site chrome.
    *   **The CRT Screen:** The inner `#terminal` output area is locked to a dark background (`#0a0908`), framed by a crisp 2px `--rule` border, isolating the command line CRT display inside the paper console.
    *   **Controls:** The reconnect button and connectivity indicator dots are styled in oxblood (`--accent`) and ink rather than arbitrary grays.

### 3.4 The Interactive Map (`/map`)
*   Designed to resemble a vintage ink-and-parchment cartographic drawing:
    *   **Parchment Canvas:** The container uses the `--paper-deep` cream background with thin black ink borders.
    *   **Ink Coordinates:** All room nodes, grid lines, and connection lines are drawn using `--ink` (near-black) or `--accent` (oxblood) line strokes.
    *   **Cartography Controls:** Action buttons use the classic brutalist `.btn-secondary` outline styles. Inputs use warm cream backgrounds, ink borders, and oxblood focus outlines. Details boxes render inside standard cream `.card` panels.

---

## 4. Navigation System

The site uses one shared navigation system. Future agents **MUST** extend these shared patterns rather than invent new ones — no seventh section-index layout, no bespoke breadcrumbs, no page-local menus.

*   **Persistent top section bar:** A ruled bar (hairline top and bottom, newspaper style) listing the seven primary sections: **Play · World · Help · Database · Map · Dispatch · About**. All-caps, letter-spaced, equal spacing. The active section is marked in oxblood (`--accent`). On mobile (< 640px) the bar becomes a single horizontally scrollable row — no hamburger, no JS.
*   **Shared breadcrumbs:** All nested pages render `Section » Subsection » Page` via one shared partial. Ancestors are oxblood links; the current page is plain text. Not shown on section index pages or top-level leaves.
*   **Shared card grid:** All section indexes (`/world`, `/help`, `/community`, `/connect`, etc.) use one shared card-grid pattern — title, description, meta count, oxblood hover — from a single partial and a single CSS block.

---

## 5. AI Agent Visual Guardrails

When generating pages or components for the Dark Pawns website, agents **MUST NEVER** do the following:

1.  **NO Generic White/Black Backgrounds:** Never use pure `#FFFFFF` or `#000000` (except inside the CRT terminal canvas itself). Always use `--paper` (`#EFE7D6`) or `--paper-deep` (`#E5DAC1`).
2.  **NO Modern Gradients or Box Shadows:** The site uses flat, brutalist print aesthetics. Never use linear/radial gradients or blurred dropshadows.
3.  **NO Gimmicky "Spooky" Assets:** Avoid flying bats, spiderwebs, dripping blood text, or flaming dividers. Cohesiveness comes from typographical precision and editorial layouts, not corny visual tricks.
4.  **NO Border Radii > 2px:** All buttons, cards, and input frames must maintain crisp, sharp edges. If an overlay needs a border radius, it must not exceed `2px`.
5.  **NO Colors Outside the Palette:** Do not introduce arbitrary blues, greens, or purples. Status indicators must be mapped to oxblood (offline/error), forest green `#3a8a3a` (online/success), and `--ink-muted` (unknown/loading).

---

## 6. Posting News

*   **Dispatch posts are manual:** a news post is a markdown file in `content/news/`, created from the `archetypes/news.md` archetype. A `make new-post TITLE="..."` target in the repo-root Makefile wraps `hugo new` so posting is one command.
*   **The changelog is automatic:** `/changelog` is generated from the repo's conventional commits at build time. Do not hand-edit the auto-generated section — the hand-written v2.x release history at the top of the page is archival and is the only part maintained by hand.
