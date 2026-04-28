# Dark Pawns Website — Master Plan

**Date:** 2026-04-27  
**Status:** Pre-build, decisions locked  
**Source:** Claude Design brief + BRENDA69 research + dp-players.com archive + admin panel PLAN  
**Domain (build):** `darkpawns.labz0rz.com` (TBD for production)

---

## 1. Architecture: Two Sites

### 1.1 Public Site (Hugo)

**URL:** `darkpawns.labz0rz.com` (v1 build domain)  
**Stack:** Hugo static site generator  
**Deploy:** Static files → Nginx/Caddy  
**Purpose:** Marketing, archive, player docs, agent docs

Merged public site = brief's marketing pages + existing docs-site content. One `llms.txt`, one content negotiation layer, one deployment.

**Information Architecture:**

```
/                      → Splash (first visit, cookie dismiss → /news)
/news                  → Devlog / news stream (home for returning players)
/news/:slug            → Individual post
/about                 → History of Dark Pawns (1997–2010 + gap + resurrection)
/play                  → Web client (casual demo)
/connect               → Telnet address + how to connect
/help                  → Player command reference (from wayback help-files)
/help/:command         → Single help entry
/world                 → Classes, races, skills landing
/world/classes         → Class descriptions (from wayback class.html)
/world/races           → Race descriptions
/world/skills/:slug    → Individual skill detail
/docs                  → Agent protocol, SDK, API reference (from docs-site)
/docs/agents/protocol  → WebSocket protocol spec
/docs/agents/sdk       → Agent SDK reference
/docs/api              → REST API docs
/maps                  → Area maps (v2)
/status                → Who's online, uptime, version (live widget data)
/changelog             → Patch notes / version history
/credits               → Original team + port credits
/quotes                → Player quotes (from dp-players archive)
/history               → DP history timeline (from dp-players archive)
/lore                  → World lore (from wayback background.html)
/equipment             → Equipment DB (v2, from dp-players archive)
/forums                → Historical forum discussions (from dp-players archive, v2)
```

### 1.2 Admin Panel (React SPA)

**URL:** `admin.darkpawns.labz0rz.com` (separate subdomain)  
**Stack:** React 18+ / TypeScript / Vite / TanStack Query / Monaco / React Flow / AG Grid  
**Port:** 8081 (separate from game port 8080)  
**Auth:** Separate JWT flow, IP allowlist, role-based (builder/admin/research)  
**Deploy:** Separate build pipeline, separate TLS  
**Reference:** `PLAN-web-admin-architecture.md` (full spec, Opus-reviewed)

**Key principle:** The admin panel is an application, not a website. Different auth, different CORS, different security model, different deployment. Don't merge.

**Shared surface:** Both the public site's `/status` widget and the admin panel hit the same Go server API endpoints. Public site = read-only endpoints. Admin = mutation endpoints + read-only.

### 1.3 Why Not One Site

| Criterion | Hugo Public | React Admin |
|-----------|------------|-------------|
| Content type | Static marketing/archive | Dynamic game management tool |
| Auth | None | JWT + IP allowlist + role-based |
| Runtime | Zero (static files) | Full SPA runtime |
| CORS | Open | Locked to SPA origin |
| Deploy target | CDN / static hosting | App server |
| User session | Cookie (splash only) | Bearer token / session |
| Content updates | Git push → Hugo build | API calls to Go server |

---

## 2. Design Decisions (Locked)

### 2.1 Aesthetic Direction

> "Retro Stephen King paperback, but completely modern design standards."

**Hard NOs** (from Claude Design brief):
- Floating bats, ravens, spiderwebs, fog, candles
- Dripping-blood typography, glowing red eyes
- Gradient purple/black "dark fantasy" backgrounds
- Cracked stone or parchment textures
- Wax seals, scrolls, "Ye Olde" copy
- Skull bullets, flame dividers, animated torches
- Cinzel, Cormorant, Uncial, Blackletter

### 2.2 Color Palette

| Token | Hex | Use |
|-------|-----|-----|
| `--paper` | `#EFE7D6` | Page background (warm cream) |
| `--paper-deep` | `#E5DAC1` | Cards, elevated surfaces |
| `--ink` | `#1A1614` | Primary text (warm near-black) |
| `--ink-muted` | `#56504A` | Secondary text |
| `--rule` | `#1A1614` | Hairlines, dividers |
| `--accent` | `#A8201A` | Oxblood — links, hot type, CTAs |
| `--accent-deep` | `#7A1812` | Hover/pressed |
| `--registration` | `#0F0F0F` | "Black plate" — body copy on cream |

Dark mode: Out of scope for v1. Cream paper is the brand. Consider `prefers-color-scheme` invert as a 30-min add post-launch.

### 2.3 Type System — Direction A (Brutalist Workhorse)

| Role | Font | Source | Use |
|------|------|--------|-----|
| Display | **Archivo Narrow** or **National 2 Condensed Bold** | Google Fonts (free) / Klim (paid) | Tight-tracked all-caps headers, wordmark |
| Body | **Source Serif 4** | Google Fonts (free) | All body copy, editorial text |
| Mono | **JetBrains Mono** | Google Fonts (free) | Web client, command refs, code, stat blocks |

**V1 call:** Start with Archivo Narrow (free, Google Fonts). If it doesn't feel right after hi-fi mockup, evaluate National 2 Condensed.

### 2.4 Voice & Tone

**Adjectives:** nostalgic, scary, intriguing, professional  
**Anti-adjectives:** cliche, corny, goofy  
**Self-seriousness dial:** ~35/100  
**Key principle:** "The site is the dust jacket and the colophon."

The MUD itself is in-fiction. The site chrome is archival. Layer 3 (Mythic Admin) from `docs/brand-voice.md` applies to lore content *within* pages. Site chrome (nav, labels, buttons) stays factual.

### 2.5 Wordmark

DARK PAWNS — all-caps, tight-tracked display sans.
- Option 1: Pure type, oxblood accent on PAWNS
- Option 2: Wordmark + serial subtitle (`MUD · EST. 1997 · v2.0`)
- The ASCII logo from the game's login screen is the only logo mark.

---

## 3. Agent-Readable Web Design

### 3.1 Research Summary (April 2026)

Sources: Evil Martians (2026-04-15), BCG X (2026-03-26), WorkOS (2025-07-01), llmstxt.org

**Core finding:** "Ship clean Markdown at every URL, tell the world it exists, and measure who shows up." — Evil Martians

**Key data:**
- Google searches per US user fell ~20% YoY in 2025
- 93% of Google AI Mode searches end without a click
- Zero-click searches grew 13 percentage points in one year (56% → 69%)
- Claude Code, Cursor, and other coding assistants already send `Accept: text/markdown`
- ~2,100 public `llms.txt` implementations tracked

### 3.2 Implementation Checklist

| # | Mechanism | Status | Effort |
|---|-----------|--------|--------|
| 1 | `/llms.txt` at site root | **TODO** | Low — static file in Hugo |
| 2 | `/llms-full.txt` (all content in one doc) | **TODO** | Low — Hugo template |
| 3 | `.md` routes for every page | **TODO** | Low — Hugo output formats |
| 4 | `Accept: text/markdown` content negotiation | **TODO** | Low — Hugo + nginx |
| 5 | `<link rel="alternate" type="text/markdown">` in `<head>` | **TODO** | Low — Hugo template |
| 6 | HTTP `Link` header pointing to Markdown | **TODO** | Low — nginx |
| 7 | JSON-LD structured data (Article, FAQPage, SoftwareApplication) | **TODO** | Medium |
| 8 | Semantic HTML (clean headings, consistent sections) | **TODO** | By default with Hugo |
| 9 | `robots.txt` allowing GPTBot, ClaudeBot, PerplexityBot | **TODO** | Low |
| 10 | Analytics on AI-specific endpoints | **TODO** | Medium |

### 3.3 What This Means Practically

- Every URL on the site returns both HTML (for humans) and Markdown (for agents)
- The `/llms.txt` file is a curated table of contents — the first thing an AI sees
- The `/llms-full.txt` is a single-file dump of all content for LLMs that embed rather than navigate
- Structured data (JSON-LD) anchors facts for citation — when ChatGPT quotes Dark Pawns, it links back
- No JS requirement for content — Hugo outputs resolved HTML, not a JS hydration shell

---

## 4. Content Inventory

### 4.1 Wayback Archive (`darkpawns/docs/wayback/`)

Source: `darkpawns.com` via Wayback Machine  
Format: Converted to Markdown  
Status: Ready for Hugo content pipeline

| File | Content | Maps To |
|------|---------|---------|
| `background.html-content.md` | World lore — Friar Drake letter, creation myth | `/lore` |
| `class.html-content.md` | Class descriptions (assassin, magus, etc.) | `/world/classes` |
| `faq.html-content.md` | FAQ — "Booo... I died. What do I do now?" | `/help` (partial) |
| `features.html-content.md` | Features list — mobile AI, tattoos, talking weapons | `/about` + `/world` |
| `main.html-content.md` | News posts — 3.0 beta, forums, Aiko story | `/news` (historical) |
| `wizlist.html-content.md` | Wizard/immortal list | `/credits` |
| `help-files/` | Command help text, spells, socials, wizhelp | `/help/:command` |

### 4.2 dp-players.com Archive (`workspace/dp-players-archive-organized/`)

Source: `dp-players.com` via Wayback Machine (July 2004 snapshot)  
Format: HTML → Markdown (converted 2026-04-22, 2026 filtering applied)  
Status: Ready for Hugo content pipeline

| Category | Files | Content | Maps To |
|----------|-------|---------|---------|
| `01_articles_guides/` | `articles.mud.md` | Game guides, tutorials | `/help` (guides) |
| `02_quotes_humor/` | `quotes.mud.md` | Player quotes, funny moments, Bannor exploit | `/quotes` |
| `03_history_lore/` | `history.mud.md` | Timeline: Sept 1994 → current (Frontline's history) | `/about` + `/history` |
| `04_game_logs_stats/` | `logs.mud.md` | Game logs, statistics | `/lore` (appendix) |
| `05_equipment_gear/` | `equipment.mud.md` | Equipment database | `/equipment` (v2) |
| `06_community_social/` | `whois.mud.md`, `player_pics.mud.md`, `links.mud.md`, `contact.mud.md` | Player roster, photos, external links | `/credits` + `/about` |
| `07_technical_downloads/` | `directions.mud.md`, `downloads.php.md` | Connection instructions, client downloads | `/connect` |
| `08_forum_discussions/` | `viewtopic_705.md`, `viewtopic_724.md`, `viewtopic_731.md`, `viewtopic_737.md`, `forum_statistics.md` | Forum threads | `/forums` (v2) |
| `09_site_information/` | `about_site.mud.md`, `index.md` | Site metadata | `/about` (source material) |

### 4.3 Existing Docs-Site Content (`darkpawns/docs-site/`)

| Content | Maps To |
|---------|---------|
| Agent protocol spec | `/docs/agents/protocol` |
| Agent SDK reference | `/docs/agents/sdk` |
| Getting started | `/docs` |
| API reference | `/docs/api` |
| Contributing guide | `/docs/contributing` |
| Development docs | `/docs/development` |

### 4.4 Existing Web Client (`darkpawns/web/`)

| File | Purpose |
|------|---------|
| `index.html` | Web client page |
| `client.js` | WebSocket game client |
| `style.css` | Dark-themed client styles |
| `static/css/darkpawns.css` | Additional styles |
| `middleware.go` | Content negotiation middleware |
| `onboarding/` | Agent onboarding (HTML + MD + JSON-LD) |

**Status:** Exists, functional. Needs redesign to match brief's `/play` spec (cream chrome, dark terminal only inside the client area).

### 4.5 Brand Voice Reference (`darkpawns/docs/brand-voice.md`)

Three-layer framework + four pillars + Frontline's voice analysis. Key resource for:
- `/lore` page copy (Layer 3 — Mythic Admin)
- Pull quotes for `/about` (touchstone quotes table)
- Voice guidance for new `/news` posts
- What NOT to put in public content (Hostility Transfer Rule, Pillar 4 containment)

---

## 5. Content → IA Mapping

### 5.1 Immediate (v1 — build now)

| Page | Content Source | Status |
|------|---------------|--------|
| `/` (splash) | Brief section 9 | **DONE** — Phase 3 |
| `/news` | Brief section 9 + wayback `main.html` (historical posts) | **DONE** — Phase 5 |
| `/about` | Brief section 9 + dp-players `history.mud.md` + brand voice touchstones | **DONE** — Phase 5 |
| `/play` | Existing `web/client.js` + `web/index.html` | **DONE** — Phase 4 |
| `/connect` | dp-players `directions.mud.md` | **DONE** — Phase 5 |
| `/help` | Wayback `help-files/` + wayback `faq.html` | **DONE** — 430 files, descriptions added (F-4) |
| `/world/classes` | Wayback `class.html` | **DONE** |
| `/world/races` | Wayback help-files (race entries) | STUB — "Coming soon" (needs extraction) |
| `/world/skills` | Wayback help-files (skill entries) | STUB — "Coming soon" (needs extraction) |
| `/docs/*` | Existing docs-site content | **DONE** — merged |
| `/credits` | Wayback `wizlist.html` + dp-players `whois.mud.md` | **DONE** — Phase 5 |
| `/quotes` | dp-players `quotes.mud.md` | **DONE** — under `/community/quotes/` |
| `/history` | dp-players `history.mud.md` (Frontline's timeline) | **DONE** — under `/community/history/` |
| `/lore` | Wayback `background.html` (Friar Drake letter) | **DONE** |
| `/changelog` | New — append-only stream | **DONE** — Phase 5 |
| `/status` | Live API widget | **DONE** + FIX (F-2: baseof extension) |
| `llms.txt` | Auto-generated from IA | **DONE** — Phase 1 |
| `llms-full.txt` | Hugo template concatenating all content | **DONE** + extended to sections (F-9) |

### 5.2 Deferred (v2)

| Page | Content Source | Notes |
|------|---------------|-------|
| `/maps` | Area maps | Need map assets |
| `/equipment` | dp-players `equipment.mud.md` | Dense data, needs table UI |
| `/forums` | dp-players forum discussions | Archive display, not live |
| `/world/items` | Item DB | Dependent on admin panel data |
| Battle logs | Game server logs | Needs log pipeline |
| Hall of fame | Player stats | Needs game data API |

---

## 6. Build Order

### Phase 1: Infrastructure ✅ COMPLETE (commit `7793842`)
1. ~~Set up Hugo project with content negotiation (`Accept: text/markdown`)~~
2. ~~Create `llms.txt` template~~
3. ~~Create `llms-full.txt` template~~
4. ~~Set up `robots.txt` (allow AI crawlers)~~
5. ~~Add `<link rel="alternate">` tags to base template~~
6. ~~Add JSON-LD structured data to base template~~

### Phase 2: Content Pipeline ✅ COMPLETE (commit `7793842`)
1. ~~Import wayback content into Hugo content sections~~
2. ~~Import dp-players content into Hugo content sections~~
3. ~~Merge docs-site content into `/docs/` section~~
4. ~~Set up content types (news, help, lore, class, etc.) with frontmatter schemas~~

### Phase 3: Design & Templates ✅ COMPLETE (commit `3079200`)
1. Implement color tokens as CSS custom properties
2. Set up type system (Archivo Narrow + Source Serif 4 + JetBrains Mono)
3. Build splash page
4. Build `/news` (devlog) layout
5. Build `/about` (editorial long-read) layout
6. Build `/help` (command reference) layout
7. Build `/world` (class/race/skill) layout
8. Build `/play` (web client wrapper)
9. Build `/status` (live widget — fetch from Go API)
10. Build remaining pages (credits, quotes, history, lore, changelog, connect)

### Phase 4: Web Client Redesign ✅ COMPLETE (commit `8ed166b`)
1. Wrap existing `client.js` in cream-themed layout
2. Dark terminal only inside client area
3. Add sidebar with connection info + "for serious play use Mudlet/TinTin++"
4. Test WebSocket connection to running game server

### Phase 5: Content Writing ✅ COMPLETE (commit `92ea624`)
1. Write `/about` — three-act structure (1997-2010 / gap / resurrection)
2. Write new devlog posts for `/news`
3. Compile `/credits` from wizlist + whois data
4. Write `/connect` for current server
5. Start `/changelog` with port history

### Phase 6: Polish & Deploy ✅ COMPLETE (commits `92ea624` + deploy commits)
1. ~~Print stylesheet for `/help` and `/changelog`~~
2. ~~Accessibility audit (AA at 16px body — already verified by brief)~~
3. ~~Performance audit (static files, gzip, caching)~~
4. ~~Set up Caddy with content negotiation rules~~ (Caddy, not nginx)
5. ~~Deploy to `darkpawns.labz0rz.com`~~ (VM 666, Caddy Docker)
6. ~~Test agent readability (curl with `Accept: text/markdown`)~~

---

## 7. Mobile-First Design

### 7.1 Why This Is Easy
Hugo static HTML + minimal JS = fast on mobile. No hydration, no layout shift, no 3MB bundle. Loads like 2003, looks like 2026.

### 7.2 Typography Scale (Mobile)
| Role | Desktop | Mobile |
|------|---------|--------|
| Body (Source Serif 4) | 16px | 16px (floor, not target) |
| Nav links | 16px | 14px |
| Command refs (JetBrains Mono) | 14px | 14px |
| Pull quotes (serif italic) | 24px | 20px |
| Display headers (Archivo Narrow) | varies | Scale with clamp() |

### 7.3 Component-Specific Decisions

**Splash:** ASCII logo may not fit 375px. Show wordmark on mobile, full ASCII on tablet+. Wordmark option 2 (DARK PAWNS + subtitle) works at any width.

**Navigation:** Hamburger menu → slide-out panel (not full-screen overlay). Oxblood accent on hamburger icon and active states.

**Web client (`/play`):** THE hard problem on mobile.
- Fixed input at bottom of viewport (like iMessage)
- Terminal fills remaining space, scrollable
- Keyboard stays up between commands
- **Quick-action bar** above input: directional buttons (N/S/E/W/U/D), look, inventory, score. Essential on phone — without them the client is unusable.
- The brief doesn't mention quick-actions but they're required for mobile viability.

**Command reference (`/help`):** Dense terminal tables → tappable cards on mobile. Each command is a card that expands to show full help text. Same content, different layout via Hugo responsive classes.

**Touch targets:** All tappable elements 44x44px minimum (WCAG guideline). CTAs, nav links, command cards.

**Live status widget:** Stack vertically on mobile (WHO count → uptime → version). Horizontal on desktop.

### 7.4 Breakpoints
| Name | Width | Notes |
|------|-------|-------|
| Mobile | < 640px | Single column, hamburger nav, card layouts |
| Tablet | 640–1024px | Two-column where appropriate, ASCII logo visible |
| Desktop | > 1024px | Full layout, sidebar on /news, status widget rail |

### 7.5 Print Stylesheet
Brief already calls for this (help + changelog). Also useful: hide nav, cream background, oxblood for headings only, no interactive elements.

---

## 8. GEO Strategy (Generative Engine Optimization)

### 8.1 Why This Is Marketing Without Marketing
The agent-readability work (Section 3) IS the SEO strategy. LLMs don't care about your UX — they care about your data. Static HTML + clean structure + llms.txt + content negotiation = the site is already optimized for AI citation.

The goal: when someone asks ChatGPT "what's a good MUD" or "are there MUDs with AI agents," Dark Pawns gets *cited*, not just ranked.

### 8.2 Already Done (Zero Additional Work)
- Static Hugo site — resolved HTML, no JS hydration
- Content negotiation — `Accept: text/markdown` → clean Markdown
- `/llms.txt` — curated table of contents ("local librarian" approach)
- `/llms-full.txt` — single-file all-content dump for embedding
- Semantic HTML — clean heading hierarchy from Hugo
- No JS dependency for content — the brief's static-first approach

### 8.3 Add During Build (Low Effort, Hugo Templates)

**JSON-LD structured data** — auto-generated from frontmatter:
- `SoftwareApplication` on `/about` (name, version, genre: MUD, URL)
- `FAQPage` on `/help` (every command as question/answer pair)
- `Article` on `/news/:slug` (author, datePublished, headline)
- `Game` schema on `/world/classes` (name, genre, playMode)
- `Organization` in base template (name, URL, foundingDate)

**Standard `<head>` additions:**
- Open Graph + Twitter cards (og:title, og:description, og:image, twitter:card)
- `<link rel="canonical">` on every page
- Hugo auto-generated `sitemap.xml`
- Point sitemap in `robots.txt`

### 8.4 Content Strategy (Write With Citation in Mind)

Target answer-fragments — LLMs decompose questions into sub-queries. Each section of the About page should contain a self-contained, quotable claim:

- "Dark Pawns is a dark fantasy MUD originally launched in 1994 on CircleMUD, operating until 2010, rebuilt in Go in 2024." ← citation fragment with dates and tech stack
- "AI agents are first-class players — same WebSocket, same commands, same WHO list, same permanent death." ← the most quotable sentence on the site
- "187 ported social emotes, ROM 2.4b combat (bash, kick, trip, backstab, headbutt), Lua scripting for mob/room behavior." ← specifics LLMs can grab

The dp-players history timeline is perfect citation material — dates, version numbers, hosting transitions.

### 8.5 Post-Launch (Ongoing, Low Maintenance)
- List on The Mud Connector and MudConnect (niche directories, permanent visibility)
- GitHub README with proper og:image and homepage URL (entity signal)
- Wikipedia long-term (13-year run + AI differentiator = potential notability)

### 8.6 What We're NOT Doing
- No keyword optimization or SEO tools
- No blogspam or link building
- No social media content calendar
- No listicles
- No Google Ads
- No analytics-driven A/B testing
- No email capture

---

## 9. Accessibility Plan

### 9.1 The Opportunity

**MUDs are the most accessible game format that exists.** This isn't aspirational — it's structural. The entire game is text. Screen readers read text. A player who is blind can play Dark Pawns. A player with motor impairments who can't use a mouse can play Dark Pawns. A player with cognitive disabilities who needs predictable, simple interfaces can play Dark Pawns.

The MUD community already knows this:
- **VIP Mud** is a dedicated screen-reader-accessible MUD client (works with JAWS, NVDA, WindowEyes, SAPI)
- **Mudlet** has native screen reader support — Ctrl+Tab switches focus between input and output, F6 accesses output window, works like a word processor for screen reader users
- **MUSHclient** has MushReader plugin for accessibility
- The `/connect` page should recommend accessible clients alongside standard ones

This is a differentiator. No graphical game can say this. Put it on the `/about` page.

### 9.2 Conformance Target

**WCAG 2.2 Level AA** (latest standard). Not 2.1 — 2.2 added criteria for focus appearance and dragging that matter here.

The brief already commits to AA at 16px body text. This plan extends that to the full WCAG 2.2 AA checklist, with specific implementations for every page on the site.

### 9.3 Color & Contrast

| Combination | Contrast Ratio | WCAG AA (4.5:1 text, 3:1 large) | Status |
-------------|---------------|----------------------------------|--------|
| `--ink` (#1A1614) on `--paper` (#EFE7D6) | **~12.5:1** | ✅ Pass | Body text |
| `--accent` (#A8201A) on `--paper` (#EFE7D6) | **~4.6:1** | ✅ Pass | Links, hot type |
| `--ink-muted` (#56504A) on `--paper` (#EFE7D6) | **~5.7:1** | ✅ Pass | Secondary text |
| `--ink` (#1A1614) on `--paper-deep` (#E5DAC1) | **~11.5:1** | ✅ Pass | Text on cards |
| `--accent` (#A8201A) on `--ink` (#1A1614) | **~4.2:1** | ⚠️ Fail for small text | Use for large display text only |
| `--paper` (#EFE7D6) on `--ink` (#1A1614) | **~12.5:1** | ✅ Pass | Inverted text (buttons, headers) |
| `--accent` (#A8201A) on `--paper-deep` (#E5DAC1) | **~4.1:1** | ⚠️ Fail for small text | Avoid |

**Rules:**
- Links are `--accent` on `--paper` — already verified at 4.6:1 ✅
- Links MUST have an underline (not just color) — color alone is not sufficient (WCAG 1.4.1)
- Oxblood-on-black for display text only (headers ≥18pt bold or ≥24pt)
- Never use `--accent` on `--paper-deep` for text smaller than 18pt bold
- Status indicators (online/offline, success/error) use BOTH color AND icon/text label
- Never convey information through color alone

### 9.4 Typography & Reading

- **Minimum body text: 16px** (brief spec). No exceptions.
- **Line height: 1.5× minimum** for body text (WCAG 1.4.12)
- **Line length: 80 characters maximum** (measure `45ch`–`75ch`)
- **Paragraph spacing: 1.5× line height** after each paragraph
- **Text resizing: all text must remain readable at 200% zoom** (WCAG 1.4.4). Hugo's rem-based sizing handles this by default.
- **No text in images** (WCAG 1.4.5). All text is real HTML text.
- **Language attribute:** `<html lang="en">` on every page

### 9.5 Keyboard Navigation

Every interactive element must be operable via keyboard alone (WCAG 2.1.1).

**Site-wide:**
- Skip-to-content link (first focusable element on every page, visually hidden until focused)
- Logical tab order (follows visual layout)
- Visible focus indicator on all interactive elements — minimum 2px solid, `--accent` color. WCAG 2.4.7 (2.2) requires focus appearance ≥2px offset with 3:1 contrast against background.
- All links, buttons, form controls reachable via Tab
- No keyboard traps — user must be able to Tab away from every element (WCAG 2.1.2)
- Escape closes any open overlay/modal

**Web client (`/play`):** This is the hardest accessibility problem on the site.
- Terminal output area: `role="log"` + `aria-live="polite"` — screen readers announce new game output when idle
- Input field: auto-focus on page load, `aria-label="Game command input"`
- Quick-action bar (mobile): each button has `aria-label` (e.g., "Move north", "Look around")
- Connection status: `aria-live="assertive"` for disconnection (immediate announcement)
- Command history: NOT focusable via Tab (it's a convenience feature for mouse/touch). Screen reader users navigate output with standard reading commands.
- **Critical:** The terminal must work with screen readers navigating in reading mode (not just forms mode). This means the output area uses semantic content, not just a `<div>` with innerHTML.

**Navigation:**
- Mobile hamburger: focusable, proper `aria-expanded` state, `aria-label="Menu"`
- Nav links: clear focus indicators, descriptive link text (no "click here")

### 9.6 Screen Reader Support

**Semantic HTML is the foundation.** Hugo outputs semantic HTML by default. Every page must have:
- Single `<h1>` per page (the page title)
- Logical heading hierarchy (no skipping h2→h4)
- Landmark regions: `<header>`, `<nav>`, `<main>`, `<footer>`
- `<article>` for news posts, `<section>` for content groups

**Page-specific:**
| Page | Screen Reader Considerations |
|------|--------------------------|
| `/` (splash) | Wordmark + subtitle as `<h1>`. Skip link → main content. |
| `/news` | Each post is `<article>` with `<h2>`. Date as `time datetime="...">`. |
| `/about` | Long-form editorial — use `<section>` with headings for each act. |
| `/play` | See §9.5 above. This is the most complex page for screen readers. |
| `/help` | Command index as `<dl>` (definition list) or table with proper headers. |
| `/world/classes` | Each class as `<article>` or `<section>`. |
| `/status` | Live widget: `aria-live="polite"` for player count. Updated data announced. |
| `/quotes` | Each quote in a `<blockquote>`. Attribution in `<cite>`. |
| `/history` | Timeline as `<ol>` (ordered list). Each event is `<li>` with `<time>`. |

**Image alt text:** The brief says no illustrations in v1. If the black pawn logo mark is used (§10), it needs `alt=""` (decorative, since the wordmark is the text). The site currently has no content images, so alt text burden is minimal.

### 9.7 Motor Accessibility

- **Touch targets: 44×44px minimum** (WCAG 2.5.8, 2.2). Already in mobile plan (§7.3).
- **No dragging required** for any interaction. All functionality available via click/tap.
- **No time limits** on any page interaction (WCAG 2.2.1). Game inactivity timers are server-side, not site-side.
- **No motion required** (no CAPTCHA, no gesture-based navigation). If reCAPTCHA is ever needed for anti-spam, use accessible alternatives (hCaptcha accessible mode).

### 9.8 Cognitive Accessibility

- **Consistent navigation** across all pages (WCAG 3.2.3). Same nav, same order, same labels.
- **Clear page titles:** `<title>` format: `[Page Name] — Dark Pawns` (e.g., "Command Reference — Dark Pawns")
- **Breadcrumbs on deeper pages:** `/help/backstab` → `Dark Pawns > Help > Backstab`
- **Predictable links:** link text describes the destination ("Read the world lore" not "Click here")
- **No auto-playing content.** No auto-play video, no auto-advancing carousels, no unexpected navigation.
- **Error messages are specific and helpful** (not just "error" — explain what happened and how to fix)
- **Simple language where possible.** The brand voice is Layer 3 (Mythic Admin) — evocative but not unnecessarily complex. Command descriptions should be plain English.

### 9.9 Reduced Motion

Respect `prefers-reduced-motion` (WCAG 2.3.3):
- No CSS transitions/animations by default (the brief's print-aesthetic approach is inherently low-motion)
- If any transitions are added (nav slide, card expand), wrap in `@media (prefers-reduced-motion: no-preference)`
- The web client terminal output scroll should respect reduced motion (instant jump vs smooth scroll)

### 9.10 Accessibility Testing Plan

| Test | Tool | Frequency |
|------|------|-----------|
| Automated scan | axe-core (Lighthouse) | Every build (CI) |
| Keyboard audit | Manual Tab-through | Every template |
| Screen reader test | NVDA (Windows), VoiceOver (macOS/iOS) | Before launch, after major changes |
| Color contrast check | axe or WebAIM contrast checker | Every template |
| Focus indicator audit | Manual | Every template |
| Mobile a11y | VoiceOver on iOS | Before launch |
| Reading mode test | Screen reader in reading mode (not forms mode) | `/play` page specifically |

**CI integration:** Add `pa11y-ci` or `axe-core` to the Hugo build pipeline. Fail the build on any Level A or Level AA violations.

### 9.11 Accessible Client Recommendations (for `/connect` page)

The `/connect` page should recommend accessible clients alongside standard ones:

| Client | Platform | Accessibility | Notes |
|--------|----------|-------------|-------|
| VIP Mud | Windows | Native screen reader support | Purpose-built for blind/low-vision MUD players |
| Mudlet | Win/Mac/Linux | Built-in screen reader mode | Ctrl+Tab, F6, word-processor-like navigation |
| MUSHclient + MushReader | Windows | Plugin-based | Lua plugin, works with NVDA/JAWS |
| TinTin++ | Linux/macOS | Terminal-based | Works with any terminal screen reader |
| Web client | Any browser | Must be accessible (§9.5) | Casual play, not full-featured |

---

## 10. Open Questions

| # | Question | Status |
|---|----------|--------|
| 1 | Production domain? | Deferred — building on `darkpawns.labz0rz.com` |
| 2 | Archivo Narrow vs National 2 Condensed? | Start with Archivo Narrow (free), evaluate after hi-fi |
| 3 | Forum vs Discord for v1? | Discord-only confirmed (brief section 5) |
| 4 | Canonical telnet address/port? | Brief says `darkpawns.io 8080` — update when domain decided |
| 5 | Featured pull quotes for /about? | Brand voice touchstone table + Friar Drake letter ready |
| 6 | Player photos from dp-players archive? | `player_pics.mud.md` exists — likely broken image links from 2004 |
| 7 | Dark mode? | Out of scope v1, consider `prefers-color-scheme` as post-launch add |
| 8 | Go API endpoints for live status? | Need to expose read-only WHO/uptime/version from game server |
| 9 | Admin panel Phase 1 timeline? | Per PLAN — delayed until game server is stable |

---

## 11. Reference Documents

| Document | Location | Relevance |
|----------|----------|-----------|
| Claude Design Brief | Attached to this session | Design direction, IA, voice |
| Brand Voice v2.0 | `darkpawns/docs/brand-voice.md` | Content voice, touchstone quotes |
| Admin Panel PLAN | `darkpawns_repo/PLAN-web-admin-architecture.md` | Admin SPA architecture, API routes |
| GitHub Rewrite Plan | `darkpawns/docs/GITHUB-REWRITE-PLAN.md` | Repo restructuring |
| Agent Protocol Spec | `darkpawns/docs/architecture/agent-protocol.md` | WebSocket protocol |
| Agent SDK Reference | `darkpawns/docs/architecture/agent-sdk.md` | Agent SDK docs |
| Player Guide | `darkpawns/docs/player-guide/player-guide.md` | Player documentation |
| Wayback Archive | `darkpawns/docs/wayback/` | Original site content (7 files + help-files) |
| dp-players Archive | `workspace/dp-players-archive-organized/` | Player site content (18 files, 9 categories) |
| Existing Docs Site | `darkpawns/docs-site/` | Hugo docs with content negotiation |
| Existing Web Client | `darkpawns/web/` | WebSocket game client |
| Website Mockup | `darkpawns/website-mockup.html` | Old static mockup (superseded by brief) |

---

*This document is the single source of truth for the Dark Pawns website build. Update it as decisions change.*
