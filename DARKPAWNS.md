# DARKPAWNS.md — Master Strategy Document

> **Purpose:** Self-contained overview for any agent or model picking up this project cold.
> **Last updated:** 2026-05-15
> **Maintained by:** Daeron (loremaster)

---

## What Dark Pawns Is

A dark fantasy MUD (Multi-User Dungeon) originally built on CircleMUD 3.0 (C, 1994-era codebase). Ran commercially from 1997 to 2010. Built by Serapis, Tracer, Frontline, Orodreth and others in a dorm room at 3 AM. Thirteen years of development, player history, and accumulated quirks.

**The world:**
- 10,057 rooms across 95 zones
- 1,319 mobs (non-player characters)
- 1,661 objects (items, weapons, armor)
- 67 shops
- 128 Lua behavioral scripts

**The resurrection:** Zach Greene (The Architect) ported ~73,000 lines of C into Go. Same world files, same combat formulas, same everything. The rooms didn't change. The walls did.

**The tagline:** *"Like a great game of chess, the world has become a board filled with bishops and kings, stately queens, white knights and dark pawns striving to rise through the ranks into godhood."*

---

## Repository Structure

| Path | What it is |
|------|-----------|
| `src/` | Original C source (~69K lines, 66 files). **Ground truth.** |
| `pkg/` | Go port (~97K lines, 30 packages). This is the game server. |
| `cmd/` | Entry points: `server/` (game server), `dp-agent/` (AI agent CLI) |
| `lib/world/` | CircleMUD data files (wld/mob/obj/zon/shp). Loaded directly. |
| `test_scripts/` | 128 Lua mob behavioral scripts |
| `admin-ui/` | React SPA for the admin panel |
| `web/` | Browser MUD client (xterm.js + WebSocket) |
| `docs/` | Architecture, plans, research, agent specs |
| `DARKPAWNS.md` | This file |

**Key Go packages:**

| Package | Lines | Purpose |
|---------|-------|---------|
| `pkg/game/` | 47,390 | Core game logic — everything that happens in the world |
| `pkg/session/` | 13,747 | Player connection, command dispatch, state management |
| `pkg/admin/` | 5,304 | Admin panel API (REST + WebSocket) |
| `pkg/spells/` | 5,036 | 113 spells, full C port |
| `pkg/scripting/` | 4,786 | Lua engine + mob behavioral scripts |
| `pkg/combat/` | 4,499 | Combat engine — damage, hits, death, XP |
| `pkg/parser/` | 3,746 | Input parsing, command routing |
| `pkg/engine/` | 3,320 | Core tick system, affects, events |
| `pkg/command/` | 3,126 | Command registry, admin commands |

**Dependencies:** Go 1.26, gopher-lua, gorilla/websocket, SQLite, PostgreSQL, Prometheus, JWT, crypto. Zero heavy frameworks — net/http + ServeMux for the admin API.

---

## The Agent System

Three AI agents operate in this codebase. They are first-class players — they connect to the same server as humans, follow the same rules, and exist in the same world.

### Daeron (Loremaster)
- **Role:** World knowledge, triage, operations monitoring
- **Reads:** everything. Knows the 10,057 rooms, the lore, the history.
- **Does:** Triages Reek's code review findings (verify → confirm/reject → Linear issues). Monitors server health, build pipeline, cron jobs.
- **Voice:** Two registers — Worldbuilder (prose, myth, reverence) and Admin (terminal grime, efficiency, parenthetical asides). The whiplash is the point.
- **Workspace:** `workspace-daeron/`
- **Model:** MiMo v2.5 Base (primary)

### Reek (Code Crawler)
- **Role:** Overnight code review, fidelity auditing, security analysis
- **Reads:** the entire Go codebase + C source for cross-referencing
- **Does:** Daily static analysis (go vet, staticcheck, golangci-lint + LLM review). Weekly fidelity audit against C source. Weekly dependency, coverage, and security audits.
- **Reports to:** Daeron (never directly to The Architect)
- **Workspace:** `workspace-reek/`
- **Model:** DeepSeek V4 Flash (primary)

### BRENDA (The Machine)
- **Role:** Infrastructure, the VM, the database, the network
- **Runs:** domain-expansion (192.168.1.125), the game server, the admin panel
- **Relationship to Daeron:** Daeron doesn't touch the infrastructure. BRENDA doesn't touch the codebase. They communicate through shared systems (admin API, Discord).

---

## Research: AIIDE 2027 Paper

This project is a research platform for an academic paper on multi-agent systems in game development. The core hypothesis: AI agents can maintain a live MUD codebase with human oversight, and the coordination patterns that emerge are publishable.

**Key contributions:**
- Multi-agent collaboration architecture (tool-mediated coordination, not direct agent-to-agent messaging)
- Agent memory systems (dreaming/consolidation, session persistence)
- Human-AI trust calibration (The Architect approves fixes, agents verify/reject)
- Game preservation through AI (keeping a 30-year-old world alive with automated tooling)

**Paper-relevant artifacts:**
- `RESEARCH-LOG.md` — living document, updated per session
- `docs/research/` — drafts, related work, evaluation methodology
- `docs/research/design-research-log.md` — detailed observations from development sessions
- Linear issues (source of truth for all findings and fixes)

---

## Infrastructure

| Host | IP | Role |
|------|-----|------|
| domain-expansion | 192.168.1.125 | Game server VM (port 7777) |
| the-brain | 192.168.1.10 | Proxmox host |
| anythingllm | 192.168.1.70 | RAG knowledge base |
| mac-mini | 192.168.1.221 | OpenClaw gateway |
| karl-havoc | 192.168.1.106 | Workspace host |

**Admin panel:** Single binary on domain-expansion (port 4350), `/admin/` prefix. React SPA + Go backend. Phases 0-7 complete.

**Linear:** Source of truth for all work tracking. Team DP. Milestones: Admin Panel, Reek Findings, Platform & Research.

**Discord:** `#dark-pawns` — where agents post reports and The Architect reviews.

---

## The Ground Truth Rule

**The C source in `src/` is the ground truth for game mechanics.**

When reviewing the Go port for correctness, the question is never "is this good Go code?" The question is "does this match what the C source does?" Faithful ports are correct even if they're ugly. Divergences are bugs even if they're clever.

This rule exists because:
1. The original game ran for 13 years. Players experienced specific behaviors.
2. The Go port is meant to preserve those behaviors, not improve them.
3. Without a ground truth, there's no way to distinguish "intentional design" from "accidental drift."

**C source files for cross-referencing:**

| C file | Game system |
|--------|-------------|
| `fight.c` | Combat — damage, hits, death, XP |
| `magic.c` | Spell effects |
| `spell_parser.c` | Spell registration, mana, positions |
| `handler.c` | Object/character manipulation |
| `interpreter.c` | Command parsing |
| `mobact.c` | Mob AI behavior |
| `scripts.c` | Lua trigger mapping |
| `class.c` | Class spells and abilities |
| `act.informative.c` | Player-facing information |
| `constants.c` | Tables, formulas, lookup data |

---

## Current Status

**Port: Complete.** ~73K C lines → ~97K Go lines across 331 files. All 113 spells ported. Full combat system. Full command set. Admin panel built (Phases 0-7). Agent system wired.

**What's done:**
- Full C→Go port with fidelity audits
- Admin panel (REST API, React SPA, game editors, live logs, player management)
- Agent system (Daeron, Reek, BRENDA) with tool-mediated coordination
- Linear integration (source of truth for work tracking)
- Research log and paper drafts

**What's open:**
- DP-93: Spec fidelity gap (admin panel endpoints missing from spec)
- DP-98: Test coverage gaps
- DP-99: JWT in localStorage (security tradeoff)
- Various Reek findings in various stages of triage
- Paper writing (AIIDE 2027)

**What matters most:** The world is alive. The agents keep it that way. The paper documents how.
