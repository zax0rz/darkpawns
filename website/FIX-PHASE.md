# Website Fix Phase - Post-Audit

**Date:** 2026-04-28
**Based on:** `AUDIT-BASELINE.md` (Opus structural audit)
**Status:** COMPLETE — all 9 fixes shipped
**Commits:** c506d0b, 612a1c7, 4680638, 6096025, 297e7da, 12ac927, cc81a5d, c867df0 (F-9 bundled with cc81a5d)

---

## Priority 1 - Do Now (structural correctness)

### F-1: Remove `public/` from git ✅ `c506d0b`
- Added `.gitignore` (public/, resources/_gen/, editor files)
- Removed 1,255 build artifact files (11MB) from tracking

### F-2: Fix `status.html` — extend `baseof.html` ✅ `4680638`
- Deleted standalone `layouts/status.html` (was replacing baseof)
- Created `layouts/partials/status.html` with card + JS
- Added conditional in `single.html` for `.Params.status`
- Moved inline styles to `style.css` under `.page-status` namespace
- Ensure status page gets nav, footer, meta, CSS like every other page
- **Model:** Sonnet (template architecture judgment)

### F-3: Delete redundant post-build script ✅ `612a1c7`
- Added `layouts/_default/single.md`, `list.md`, `index.md` for native Markdown output
- Hugo now generates 490 `.md` files natively
- Deleted `scripts/generate-markdown.sh` (fragile awk frontmatter stripping)

---

## Priority 2 - Next Sprint (consistency & security)

### F-4: Normalize frontmatter ✅ `6096025`
- Fixed templates: `help.html` and `news.html` → `.Params.description`
- Fixed news frontmatter: `summary:` → `description:` (2 files)
- Added `description` to 428 help files via heuristic script

### F-5: Add SRI hashes to CDN scripts ✅ `297e7da`
- Computed SHA-384 hashes for xterm.css, xterm.min.js, addon-fit.min.js
- Added `integrity` + `crossorigin="anonymous"` to all 3 CDN resources

### F-6: Add favicon ✅ `12ac927`
- Created SVG favicon (pawn silhouette, accent red on ink-dark)
- Added `<link rel="icon">` to head.html

---

## Priority 3 - When Convenient (polish)

### F-7: Move inline styles from splash to CSS ✅ `cc81a5d`
- Extracted 5 inline styles → `.splash-divider`, `.splash-links`, `.splash-link`

### F-8: Add RSS feed discovery links ✅ `c867df0`
- Added `<link rel="alternate" type="application/rss+xml">` via Hugo `.OutputFormats.Get`

### F-9: Extend LlmsFull to all sections ✅ (bundled in `cc81a5d`)
- Added `LlmsFull` to section outputs in `hugo.toml`
- Created `layouts/_default/section.llms-full.txt`
- 21 section-level `_llms` endpoints now working

### F-10: Flesh out world section stubs
- `world/` only has `classes/` - `races/` and `skills/` are empty
- Add content or improve "Coming soon" UX
- **Model:** Deferred - content task, not structural

---

### F-10: Flesh out world section stubs — DEFERRED
- `world/` only has `classes/` — `races/` and `skills/` are empty
- Content extraction from wayback help files needed
- This is creative work, not structural
