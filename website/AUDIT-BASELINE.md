# Structural Baseline Audit — Dark Pawns Website

**Date:** 2026-04-28
**Auditor:** Opus (automated)
**Scope:** Template architecture, CSS, Hugo config, content negotiation, repo hygiene

## Executive Summary

The site is structurally competent — the template hierarchy is clean, the CSS design system is well-organized, and content negotiation is wired end-to-end from Hugo output formats through Caddy. However, there are several real problems: **1,254 build artifacts are committed to git** (bloating the repo), the post-build content-negotiation script duplicates work Hugo can already do natively, the `status` page uses an ad-hoc layout that bypasses `baseof.html` entirely, and frontmatter consistency across 480+ content files is sloppy. Nothing is broken on the live site, but these issues will compound.

---

## Critical Issues

### C-1: `public/` directory committed to git (1,254 files, 11MB)

Hugo's `public/` is a build artifact. It should never be tracked. It was committed in Phase 1 (`7793842`) and every subsequent website phase has added to it. This bloats the repo history permanently and causes merge conflicts on any branch that touches website content.

**Fix:** Add `public/` to `.gitignore`, then `git rm -r --cached public/ && git commit`. Requires coordination — anyone who pulls after this will need `hugo` to rebuild.

### C-2: Post-build markdown script duplicates Hugo's native capability

`scripts/generate-markdown.sh` manually strips frontmatter from `.md` files and copies them into `public/`. But Hugo already outputs Markdown format files via the `[outputFormats.Markdown]` definition in `hugo.toml`. The script is redundant — it produces the same files Hugo's `hugo` build already generates.

**Impact:** Double work in CI, potential for stale `.md` files if the script runs against old content, and the script's `awk` frontmatter stripping is fragile (breaks on `---` inside content bodies).

**Fix:** Delete `scripts/generate-markdown.sh`. If Hugo's native Markdown output isn't producing the desired files, debug that instead.

### C-3: `status.html` layout doesn't extend `baseof.html`

`layouts/status.html` defines `{{ define "main" }}` but **is a standalone template** (not under `_default/` and not extending `baseof.html` via Hugo's template lookup). The content page `content/status.md` sets `layout: status`, which Hugo resolves by looking for `layouts/status.html`. This means the status page renders with **no `<head>`, no `<meta>`, no nav, no footer** — just a raw `<main>` block inside an unstyled HTML shell.

Wait — actually, Hugo resolves `layout: status` as `layouts/_default/status.html` or `layouts/status.html`. If it finds `layouts/status.html`, it treats it as a **base template** (not a block override), meaning it replaces the entire page rendering. Hugo does *not* automatically wrap it with `baseof.html` unless the template explicitly uses `{{ define "baseof" }}` patterns.

**Result:** The status page has no CSS, no meta tags, no navigation, no footer, and no structured data (JSON-LD). It's a bare HTML fragment.

**Fix:** Move status logic into a proper template that extends baseof, or restructure as a single.html variant under `_default/` with a conditional.

---

## Structural Concerns

### S-1: Frontmatter inconsistency across 480 content files

Three distinct patterns exist:

| Pattern | Used by | Fields |
|---------|---------|--------|
| **Full** | docs, about, connect, credits, news | `title`, `description`, `date`, `draft`, `section` |
| **Minimal** | lore, play, world, community subsections | `title`, `description` only (no `date`, no `draft`) |
| **Help-style** | help/commands, help/spells, help/info, help/wizhelp | `title`, `date`, `draft`, `section`, `aliases` |

The `section` frontmatter field is set on docs pages and help pages but **not on about, community, credits, or news pages**. This is fine for rendering but means the JSON-LD FAQ schema in `meta.html` won't trigger for community sections (which might actually want it for forum FAQ content).

The `aliases` field is used only on help files. This is correct — help commands are the primary use case for URL aliases. No issue, just noting.

### S-2: Help section uses `.Params.summary` while community/lore use `.Params.description`

- `layouts/section/help.html` renders `.Params.summary` for card descriptions
- `layouts/section/community.html`, `connect.html`, `world.html` render `.Params.description`
- `layouts/section/news.html` uses `.Params.description` for the lead, `.Params.summary` is unused
- `layouts/section/play.html` uses `.Params.summary` for lore entries

Frontmatter spot-check shows help files don't consistently have a `summary` field — they have `section` and `aliases` but no `summary`. The `description` field is missing from help command files entirely. This means the help grid cards render with **no description text**.

**Fix:** Either add `description` to help command frontmatter or change the help section template to use `description` instead of `summary`.

### S-3: No `news/` section layout — relies on default `_default/list.html`

`content/news/` has no corresponding `layouts/section/news.html`... wait, it does exist. But `news.html` uses `{{ .Params.summary }}` for entry descriptions while the actual news content uses `description` in frontmatter. Let me verify:

News entries have `summary` in frontmatter (`website-launch.md`, `resurrection.md`), so this works. But the pattern is inconsistent with the rest of the site where `description` is the standard field.

### S-4: `goldmark.renderer.unsafe = true` is intentional but worth flagging

This enables raw HTML in markdown content files. For a site with 475+ help files imported from historical data, this is probably intentional (legacy content may contain HTML). But it means **any content file can inject arbitrary HTML/scripts**. If user-submitted content ever flows through this pipeline, it's an XSS vector.

**Risk:** Low — this is a static site with trusted authors only. But worth documenting.

### S-5: CDN dependencies loaded without SRI hashes

`layouts/section/play.html` loads xterm.js from `cdn.jsdelivr.net` with no `integrity` attribute. If the CDN is compromised, the web client loads untrusted code. Google Fonts in `head.html` also lack SRI, but fonts are lower risk.

**Fix:** Add `integrity` and `crossorigin="anonymous"` attributes. Alternatively, vendor xterm.js into `static/` (it's a core game component, not a library that changes often).

### S-6: `LlmsFull` output only wired for home page

`hugo.toml` defines `outputs` for home, section, page, taxonomy, and term — but only `home` includes `LlmsFull`. The template `layouts/index.llms-full.txt` exists, but there are no section-level or page-level LLMs templates. This means the `_llms/` path only works at the root.

**Impact:** Probably fine — the LLMs-full output at root already dumps all content recursively. But if someone requests `_llms` on a section URL, they'll get a 404.

---

## Minor Issues

### M-1: Inline styles in `index.html` splash page

The footer links below the splash actions use inline `style` attributes (`font-family`, `text-transform`, `letter-spacing`, `font-size`, `color`). These should be CSS classes. Five inline styles on the homepage — not catastrophic but ugly.

### M-2: Inline `<style>` block in `status.html`

The status page defines its own `<style>` block for status-specific styles (~40 lines). These should be in `assets/css/style.css` alongside the play page styles.

### M-3: `community/` subsection _index files are bare stubs

`community/equipment/_index.md`, `community/forums/_index.md`, `community/history/_index.md`, `community/quotes/_index.md` all have just `title` and no `description`, `date`, or content. They render as cards with a title and nothing else. Not broken, but empty.

### M-4: No `section` layout for `changelog` or `about`

These sections fall through to `_default/list.html`. The about section uses `.Params.description` (which it has), so it works. Changelog has a description too. Both render correctly — they just use the generic list template instead of a custom one. Fine for now.

### M-5: `content/world/` only has one content file

`world/_index.md` (section page) and `world/classes/classes.md` (a single file). The world section layout expects sub-sections (`classes`, `races`, `skills`) but only `classes` exists. The `world.html` template handles this gracefully with the "Coming soon" fallback, so no breakage.

### M-6: CSS `!important` usage is reasonable

Six instances, all in `@media (prefers-reduced-motion)` and `@media print` — these are appropriate uses. No `!important` abuse in the main stylesheet.

### M-7: Print styles are minimal but functional

Hides nav, footer, buttons. Forces white background. Doesn't customize much else. Acceptable for a MUD website.

### M-8: No favicon

No `favicon.ico` or `<link rel="icon">` in `head.html`. Browser will 404 on `/favicon.ico`. Minor but noticeable in browser tabs.

### M-9: No RSS feed icon or feed discovery link

RSS is generated for sections (`outputs` includes "RSS") but `head.html` doesn't include `<link rel="alternate" type="application/rss+xml">` for feed discovery.

### M-10: `content/status.md` uses non-standard `url` frontmatter field

The `url: /status` field is Hugo's built-in URL override, which is correct. But `layout: status` combined with the template issue (C-3 above) means this page is broken.

---

## Positive Findings

### P-1: Template hierarchy is clean

Every section layout, single page, and list page correctly uses `{{ define "main" }}` blocks that slot into `baseof.html`. The skeleton is solid — partial chain is `head → meta → jsonld`, and `header → nav`, `footer`. No broken block overrides.

### P-2: CSS design system is well-structured

909 lines is appropriate for this scope. CSS variables are defined in `:root` and used consistently throughout. Responsive breakpoint at 639px handles mobile nav. `clamp()` for fluid typography. No framework dependency — all hand-rolled, which is correct for a retro-themed site.

### P-3: Content negotiation is properly wired end-to-end

Hugo outputs Markdown format → Caddy serves `.md` files for `Accept: text/markdown` requests → `Link` headers advertise the alternate. The `head.html` template also adds `<link rel="alternate">` tags. This is complete.

### P-4: Accessibility foundations are in place

Skip link, `:focus-visible`, `sr-only` utility, `prefers-reduced-motion` support, semantic HTML (`<nav>`, `<main>`, `<article>`, `<time>`), `aria-label` on nav, `aria-live` on status display.

### P-5: No orphaned or unused partials

All 6 partials (`head`, `meta`, `jsonld`, `header`, `nav`, `footer`) are called from `baseof.html` and nowhere else. All are used. None are orphaned.

### P-6: The play page is well-engineered

WebSocket client with reconnect, fallback for non-WS browsers, xterm.js integration, and the sidebar gracefully hides on mobile. The `page-play` body class correctly hides the footer. This was done by someone who understood what they were building.

### P-7: `static/js/client.js` is correctly placed in `static/`

Not a build artifact — it's source code for the web client. Correctly lives in `static/js/` and gets copied to `public/js/` by Hugo.

### P-8: Deploy setup is clean

Caddyfile is well-organized with proper cache headers, content negotiation, WebSocket proxy, and health/metrics endpoints. Docker Compose is minimal and correct.

---

## Recommendations

**Priority 1 (Do Now)**
1. Remove `public/` from git tracking (`git rm -r --cached public/`, add to `.gitignore`)
2. Fix `status.html` to extend `baseof.html` — move status-specific CSS/JS into a proper block template
3. Delete `scripts/generate-markdown.sh` — rely on Hugo's native Markdown output

**Priority 2 (Next Sprint)**
4. Normalize frontmatter: add `description` to help command files, pick one field (`description`) as the standard summary field across all sections
5. Add SRI hashes to CDN-loaded scripts (xterm.js, FitAddon)
6. Add a favicon

**Priority 3 (When Convenient)**
7. Move inline styles from `index.html` splash to CSS classes
8. Add RSS feed discovery `<link>` tags to `head.html`
9. Wire `LlmsFull` output for section pages (or document that it's root-only)
10. Add `world/races` and `world/skills` content to flesh out the world section
