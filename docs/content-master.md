# Dark Pawns — Content Master Doc

**Created:** 2026-04-27  
**Status:** Living document — update as decisions are made

---

## The Big Picture

Dark Pawns has a story to tell, content to surface, and players to serve. This doc tracks every thread of content work across all outputs: GitHub, website, in-game, research, and development storytelling.

---

## Content Sources (What We Have)

### Archived / External
- [ ] **darkpawns.com** — Wayback Machine scrape (in progress)
- [ ] **dp-players.com** — Wayback Machine archive (already harvested, needs inventory)
- [ ] Original C source comments (`src/`) — authoritative reference for game mechanics
- [ ] World files (`lib/`) — rooms, mobs, objects, zones, shops, socials
- [ ] Original help files (in-game `.hlp` format, ported to Go)

### Generated / Internal
- [x] `docs/architecture.md` — needs refresh (references Redis/React that don't exist)
- [x] `docs/agent-protocol.md` — WebSocket agent protocol spec
- [x] `docs/agent-sdk.md` — Agent SDK documentation
- [x] `docs/skill-system.md` — Skills reference
- [x] `docs/shops.md` — Shop system docs
- [x] `docs/doors.md` — Door/exit system docs
- [x] `docs/lore/history.md` — Game lore/history (40 lines — thin)
- [x] `docs/lore/quotes.md` — Memorable quotes (120 lines)
- [x] `docs/player-guide.md` — Player guide (181 lines)
- [x] `docs/SOURCE_ACCURACY_AUDIT.md` — Opus audit findings
- [x] `docs/code-review-v4-pro.md` — V4 Pro review notes
- [x] `docs/MONITORING.md` + `docs/QUICKSTART-MONITORING.md`
- [x] `docs/SECURITY_HARDENING_GUIDE.md`
- [x] `docs/MODERATION.md`
- [x] `docs/research.md` — AIIDE research direction
- [x] `docs/script-inventory.md` — Lua script inventory

### In Code
- [ ] In-game help files — need audit against current code state
- [ ] 854 parsed objects with full stat blocks (extractable to DB)
- [ ] 1,313 mobs with descriptions and stats
- [ ] 10,057 rooms with descriptions
- [ ] 95 zones
- [ ] 187 social emotes with pronoun substitution

---

## Content Outputs (Where It Goes)

### 1. GitHub Repository (`zax0rz/darkpawns`)
Audience: Developers, contributors, researchers

**Needs:**
- [ ] `CONTRIBUTING.md` — How to contribute, coding standards, PR process
- [ ] `docs/architecture.md` refresh — Remove Redis/React references, reflect actual stack
- [ ] Setup guide — one-command "get this running" doc
- [ ] README.md update — Post-Opus status, current features, link to website
- [ ] Consolidate duplicate docs — agent-protocol.md and agent-sdk.md overlap

### 2. Website (darkpawns.com)
Audience: Former players, new players, curious devs, AIIDE reviewers

**Needs:**
- [ ] Lore and history section
- [ ] Player guides / getting started
- [ ] Item database (from world files — 854 objects ready to extract)
- [ ] Skills & spells reference
- [ ] Zone maps
- [ ] AI agent documentation (public-facing version)
- [ ] Development blog / dev story
- [ ] Community links (Discord, GitHub)
- [ ] Hugo site or simpler static site (currently stubs with `draft: true`)

### 3. In-Game Content
Audience: Active players

**Needs:**
- [ ] Help file audit — compare current help files to actual implemented commands
- [ ] New player experience — intro text, tutorial hints, first-hour guidance
- [ ] Skill descriptions — accurate to current implementation
- [ ] Item descriptions — rich, flavorful, consistent voice
- [ ] Room descriptions — already exist in world files, but some may need polish

### 4. Research Paper (AIIDE 2027)
Audience: Academic reviewers

**Needs:**
- [ ] Narrative memory architecture writeup
- [ ] AI agent protocol documentation (exists, needs academic framing)
- [ ] Experimental design
- [ ] Evaluation methodology
- [ ] Related work (Deep Research already gathered much of this)

### 5. Development Story
Audience: Dev community, blog readers, conference attendees

**Needs:**
- [ ] Token usage stats (by model, by phase)
- [ ] Commit history analysis (37 commits in Opus fix session alone)
- [ ] Lines of code ported (73K C → 211 Go files)
- [ ] Personal narrative — the human side of building this with AI
- [ ] "AI as labor force" angle — what worked, what didn't, what it cost

---

## Open Design Decisions

### Brand Voice Guidelines
**Status:** Needs creation  
**Why:** If AI generates most content, we need a document that says "this is how Dark Pawns talks."  
**Source material:** Original website, old help files, in-game text from world files, dp-players.com archives.  
**Deep Research:** Running — `brand_voice_guidelines` query active.

The Dark Pawns voice is:
- Dark, snarky, late-90s internet — not try-hard edgy, just naturally that way
- The help files had personality. The immortal announcements had personality.
- Not a corporate game studio. Not a modern indie dev. A guy and his MUD in the late 90s.
- AI-generated content needs to match this, not sound like a tech blog.

### Item Database — Spoiler Policy
**Status:** Undecided  
**Why:** Do we show everything or encourage discovery?  
**Deep Research:** Running — `item_database_design` query active.

Options:
1. **Full disclosure** — Everything public, all stats, all locations. Classic MUD wiki style.
2. **Tiered** — Basic stats for all, hide rare spawn conditions / hidden mechanics behind spoiler warnings.
3. **Progressive** — Only show items you've encountered in-game. (Complex to implement, may not be worth it.)
4. **In-game only** — No external item DB. Players explore. (Fights nostalgia audience expectations.)

Leaning toward tiered. ROM 2.4b is 30 years old — the info isn't secret. But there's value in preserving some mystery for new players.

### Knowledge Architecture
**Status:** Undecided  
**Why:** What goes in a wiki vs a searchable DB vs in-game help?  
**Deep Research:** Running — `knowledge_architecture` query active.

Current thinking:
- **Searchable DB:** Items, mobs, zones (structured data, filterable)
- **Wiki/narrative:** Lore, guides, mechanics explanations (narrative, linked)
- **In-game:** Quick reference, command help, current status (immediate context)

### Hugo Site vs Simpler
**Status:** Undecided  
**Why:** Current Hugo site is empty stubs. Is Hugo the right tool?

Options:
- Hugo (current) — full framework, theming, but empty
- Simple static site (markdown → HTML) — less overhead, easier to maintain
- Gallery-style (like Hardware Hunter) — Hugo + thin API backend

Leaning toward keeping Hugo but simplifying. The theme already exists.

---

## Deep Research Jobs (Running)

| Query | ID | Status |
|-------|-----|--------|
| Knowledge architecture | `v1_Chd6OVR2...` | in_progress |
| Brand voice guidelines | `v1_ChcwTlR2...` | in_progress |
| Content strategy revival | `v1_ChcwZFR2...` | in_progress |
| Item database design | `v1_ChcwdFR2...` | in_progress |

Results will be saved to `research/dp_docs_{query_name}.json`.

---

## Priority Stack

### This Week
1. [ ] Harvest darkpawns.com from Wayback Machine
2. [ ] Inventory dp-players.com archive
3. [ ] Collect Deep Research results → synthesize into decisions
4. [ ] Draft brand voice guidelines from archived content

### Next Two Weeks
5. [ ] Extract item database from world files → JSON/SQL
6. [ ] Refresh architecture.md for GitHub
7. [ ] Write CONTRIBUTING.md
8. [ ] Audit in-game help files

### Month+
9. [ ] Populate website with content
10. [ ] Write development story (token stats, personal narrative)
11. [ ] Item DB on website with spoiler tiers
12. [ ] Dev blog first post

---

## Numbers (By The Numbers)

These are the stats that tell the story. Gather from git, model logs, and memory.

- **Original C source:** ~73,000 lines
- **Go codebase:** 211 files
- **Objects parsed:** 854
- **Mobs parsed:** 1,313
- **Rooms parsed:** 10,057
- **Zones:** 95
- **Social emotes:** 187
- **Opus review session:** 37 commits, 4,019 insertions, 433 deletions, 145 files, 75/86 findings fixed (87%)
- **golangci-lint baseline:** 437 findings (50 errcheck, 50 unused, 32 staticcheck, 12 ineffassign)
- [ ] **Total tokens burned across all sessions** — needs calculation
- [ ] **Total sessions / hours** — needs calculation
- [ ] **Cost by model** — needs calculation

---

_This doc lives at `docs/content-master.md`. Update it as decisions are made and work is completed._
