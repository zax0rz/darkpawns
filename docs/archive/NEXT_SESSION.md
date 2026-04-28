# Dark Pawns — Next Session Prompt (Website)

**Branch:** main  
**Working dir:** `~/darkpawns/website/`  
**Plan:** `docs/plans/WEBSITE-MASTER-PLAN.md` (Phases 1-6 complete, see status below)
**Audit:** `website/AUDIT-BASELINE.md` (Opus structural audit)
**Fixes:** `website/FIX-PHASE.md` (all 9 fixes complete)

## Context

The Dark Pawns Hugo website (v1) is complete and deployed to VM 666 behind Caddy at `darkpawns.labz0rz.com`. 508+ pages, content negotiation (HTML + Markdown + LLMs-full), zero JS in templates, clean build.

On 2026-04-28 we did an Opus structural audit and knocked out 9 fixes (F-1 through F-9). All shipped to `origin/main`.

**Run `git log --oneline -12` at session start.**

## Current State

- Hugo builds clean (1001 pages, 0 errors)
- 430 help files with `description` frontmatter (was 0)
- Content negotiation: HTML + Markdown + LLMs-full on all sections
- SVG favicon, RSS discovery, SRI hashes, no inline styles, proper .gitignore
- Caddy on VM 666 (192.168.1.125) serves the site

## What's Next (in priority order)

### Priority 1: Content depth (~2-3 hours)
1. **`/world/races`** — Extract race descriptions from wayback help-files. Content exists, needs formatting into Hugo markdown with proper frontmatter.
2. **`/world/skills`** — Same treatment for skills. Larger set, may need to scope to top N skills.
3. **Community section stubs** — `community/equipment/`, `community/forums/`, `community/history/`, `community/quotes/` subsections are bare stubs with just a title. Add descriptions and introductory content.

### Priority 2: Missing meta / SEO (~30 min)
4. **Open Graph / Twitter Cards** — Add `og:title`, `og:description`, `og:image`, `twitter:card` to `layouts/partials/meta.html`. Important for Discord/Telegram link previews.
5. **`<title>` template** — Verify it's not just showing the site title on every page. Should be "Page Title — Dark Pawns".

### Priority 3: Accessibility & Performance audit (~1 hour)
6. **Lighthouse / axe-core pass** — Verify WCAG 2.2 AA. Foundations are good (skip link, focus-visible, sr-only, reduced motion, semantic HTML) but haven't done a real automated pass.
7. **Performance check** — Verify Caddy caching headers, gzip/brotli, no render-blocking resources.

### Priority 4: Admin Panel (separate project)
8. **React SPA** — `admin.darkpawns.labz0rz.com`. Full spec at `PLAN-web-admin-architecture.md` (Opus-reviewed). React 18+/TypeScript/Vite/TanStack Query. JWT auth, IP allowlist, role-based access. This is a multi-day build — don't start it in a short session.
9. **Admin depends on:** stable game server API, Go REST endpoints for mutations (currently only read-only endpoints exist on the public site's `/status` widget).

## Key Files

| File | Purpose |
|------|---------|
| `docs/plans/WEBSITE-MASTER-PLAN.md` | Full site plan (Phases 1-6 marked complete) |
| `website/AUDIT-BASELINE.md` | Opus structural audit findings |
| `website/FIX-PHASE.md` | Post-audit fix log (all 9 complete) |
| `website/hugo.toml` | Hugo config (output formats, params) |
| `website/assets/css/style.css` | Design system (909 lines, CSS variables) |
| `website/layouts/_default/baseof.html` | Template skeleton |
| `docs/brand-voice.md` | Voice/tone guidance for content writing |

## Design Rules (from brief)
- Cream paper (`#EFE7D6`) is the brand. No dark mode in v1.
- Accent: oxblood (`#A8201A`)
- Fonts: Archivo Narrow (display), Source Serif 4 (body), JetBrains Mono (mono)
- Voice: nostalgic, scary, intriguing, professional. NOT cliche gothic.
- Zero JS in templates (web client aside).
- "Retro Stephen King paperback, but completely modern design standards."

## Subagent Learnings (from this build)
1. **Multi-agent Hugo work needs strict template ownership.** Multiple agents touching the same partial can create subtle conflicts. Audit before merging.
2. **Opus earns its keep on structural reviews.** Found the `public/` in git, the status page baseof bypass, the frontmatter inconsistency — none of these were obvious.
3. **Hugo's native Markdown output works once you add templates.** The post-build bash script was unnecessary — just needed `single.md`, `list.md`, `index.md` under `layouts/_default/`.
4. **DeepSeek V4 Flash handles bulk operations well** (428 file frontmatter update, SRI hashes, CSS extraction). Sonnet for template architecture decisions.
5. **Content negotiation is fully wired end-to-end.** Hugo outputs → Caddy serves `.md` for `Accept: text/markdown` → `Link` headers advertise alternates → `<link rel="alternate">` in head. Complete pipeline.
