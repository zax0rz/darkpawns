# Website Fix Phase — Post-Audit

**Date:** 2026-04-28
**Based on:** `AUDIT-BASELINE.md` (Opus structural audit)
**Status:** Planning

---

## Priority 1 — Do Now (structural correctness)

### F-1: Remove `public/` from git
- Add `public/` to `.gitignore`
- `git rm -r --cached public/`
- Commit: `chore: remove build artifacts from git tracking`
- **Risk:** Anyone pulling needs Hugo to rebuild. We're the only ones working this repo.
- **Model:** GLM-5-Turbo (mechanical)

### F-2: Fix `status.html` — extend `baseof.html`
- Move status logic into `layouts/_default/single.html` with a conditional, OR create a proper `{{ define "main" }}` template that works with baseof
- Move inline `<style>` block (M-2) into `assets/css/style.css`
- Ensure status page gets nav, footer, meta, CSS like every other page
- **Model:** Sonnet (template architecture judgment)

### F-3: Delete `scripts/generate-markdown.sh`
- Hugo already outputs `.md` files via `[outputFormats.Markdown]`
- Verify Hugo's native output produces the same files the script did (spot-check a few)
- Delete the script
- Remove any references in deploy docs
- **Model:** DeepSeek V4 Flash (verify + delete)

---

## Priority 2 — Next Sprint (consistency & security)

### F-4: Normalize frontmatter across 480 content files
- Standard field: `description` (not `summary`, not both inconsistently)
- Help files: add `description` (currently missing — cards render blank)
- Templates: audit `.Params.summary` usage → change to `.Params.description` everywhere
- Templates affected: `help.html`, `news.html`, `play.html`
- This is a bulk find-replace across content files + 3 template fixes
- **Model:** DeepSeek V4 Flash (bulk content ops) + Sonnet (template verification)

### F-5: Add SRI hashes to CDN scripts
- `layouts/section/play.html` loads xterm.js and FitAddon from jsdelivr — no `integrity`
- `layouts/partials/head.html` loads Google Fonts — no SRI (lower priority, fonts)
- Generate SRI hashes, add `integrity` and `crossorigin="anonymous"` attributes
- Alternative: vendor xterm.js into `static/` (it's a core game component)
- **Model:** DeepSeek V4 Flash (mechanical)

### F-6: Add favicon
- Generate a simple favicon (pawn icon? terminal cursor?)  
- Add `<link rel="icon">` to `layouts/partials/head.html`
- **Model:** DeepSeek V4 Flash + image generation

---

## Priority 3 — When Convenient (polish)

### F-7: Move inline styles from splash to CSS
- `layouts/index.html` has 5 inline `style` attributes on footer links
- Extract to CSS classes in `assets/css/style.css`
- **Model:** DeepSeek V4 Flash

### F-8: Add RSS feed discovery links
- `layouts/partials/head.html` — add `<link rel="alternate" type="application/rss+xml">` 
- Wire for home and section RSS feeds
- **Model:** DeepSeek V4 Flash

### F-9: Wire LlmsFull for section pages (or document root-only)
- Currently only homepage exports to `_llms/`
- Decision: extend to sections OR document intentional limitation
- **Model:** Sonnet (architectural decision)

### F-10: Flesh out world section stubs
- `world/` only has `classes/` — `races/` and `skills/` are empty
- Add content or improve "Coming soon" UX
- **Model:** Deferred — content task, not structural

---

## Execution Order

```
F-1 (git hygiene) ─── can run immediately, mechanical
F-2 (status page) ─── depends on nothing, Sonnet
F-3 (delete script) ─── depends on F-1 (verify native output first)
    ↓
F-4 (frontmatter) ─── bulk operation, independent of F-1/F-2/F-3
F-5 (SRI hashes) ─── independent
F-6 (favicon) ─── independent
    ↓
F-7 through F-10 ─── polish, no urgency
```

**Parallelism:** F-1, F-2, F-4, F-5, F-6 can all run in parallel.
F-3 depends on F-1 completing first (need clean `public/` to compare outputs).

## Estimated Cost

| Task | Model | Est. Tokens |
|------|-------|-------------|
| F-1 | GLM-5-Turbo | ~5K |
| F-2 | Sonnet | ~30K |
| F-3 | V4 Flash | ~10K |
| F-4 | V4 Flash + Sonnet | ~50K |
| F-5 | V4 Flash | ~10K |
| F-6 | V4 Flash | ~15K |
| F-7-F-10 | V4 Flash | ~40K |
| **Total** | | **~160K** |

Rough cost: ~$0.30-0.50 total. Cheap for a clean codebase.
