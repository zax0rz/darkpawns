# C → Go Port Plan — Dark Pawns

> **Goal:** 100% faithful C-to-Go port of all ~68K lines of Dark Pawns MUD source.
> **Strategy:** 13 waves. Each wave = build → QA → fix → push.
> **Update (2026-04-24):** Waves 1-4a COMPLETED. See Current State below.
> **Model update:** DeepSeek V4 Flash is now the daily driver for mechanical tasks, replacing GLM-5.1 for most build work. See model routing section.

---

## Current State (as of 2026-04-24)

```
C source:          ~68,792 lines across 59 files
Already in Go:     ~34,509 lines across 150 files (partial equivalents)
Genuinely unported: ~29,155 lines across 28 C files
Build:             go build ./cmd/server passes clean
```

**What's actually merged into main:**
- **115 Lua scripts** in `test_scripts/mob/archive/` — combat AI, economy, environmental, crafting chains, newbie pipeline, specials
- **Engine stubs:** `create_event`, `tell`, `plr_flagged`, `cansee`, `isnpc`, `has_item`, `obj_in_room`, `objfrom`, `objto` — all registered as Lua globals
- **Combat AI:** `feat/combat-ai-1` and `feat/combat-ai-2` both merged
- **Wave 3:** Display, map, tattoo — `pkg/session/display_cmds.go`, `map_cmds.go`, `tattoo.go` exist
- **Wave 4a:** `spec_assign.go` exists
- **Waves 1-2:** All skill commands in `pkg/command/skill_commands.go` + `new_cmds.c`/`new_cmds2.c` equivalents
- **Shops, eat/drink, spell affects, socials** (20+ command categories) — Go implementations
- **BRENDA memory system:** `agent_narrative_memory` schema, kill/death hooks, bootstrap injection, salience decay, session consolidation crons — all active

**Unmerged but on zax0rz remote (real code):**
1. `feat/engine-stubs-2` — `has_item`, `obj_in_room`, `objfrom`, `objto` implementations (one commit behind main)
2. `feat/party-follow-group` — party/follow/group/gtell commands
3. `feat/social-commands` — score/who/tell/emote/shout/where/help
4. `fix/lua-script-bugs` — fight trigger arg fix, nil ch fix, bane state machine, breed_killer obj
5. `feat/regen-limits` — go test passes, combat_test.go fixed
6. `fix/ci-engine-tests` — PR #10 open, fixes mutex deadlocks + test failures (+40/-25 lines)

---

## Wave Plan

### ✅ Wave 1 — Skill Commands (`new_cmds.c`, ~2792 lines) [COMPLETED]
**What:** bash, backstab, kick, trip, rescue, sneak, hide, steal, pick lock, berserk, charge, parry, headbutt, spike
**Go target:** `pkg/command/skill_commands.go` (expanded)
**Status:** ✅ DONE. All skill commands ported to Go with faithful formulas. Skill system wired (`SkillManager`, skill points, practice/learn/forget).

### ✅ Wave 2 — Misc Player Commands (`new_cmds2.c`, ~1027 lines) [COMPLETED]
**What:** scrounge, first_aid, disarm, mindlink, detect, serpent_kick, dig, turn
**Go target:** `pkg/session/new_cmds2.go`
**Status:** ✅ DONE. Ported alongside Wave 1.

### ✅ Wave 3 — Display + Map + Tattoo (`act.display.c`, `mapcode.c`, `tattoo.c`, ~1129 lines) [COMPLETED]
**What:** lines, infobar, map command, tattoo/eq protection
**Go targets:** `pkg/session/display_cmds.go`, `pkg/session/map_cmds.go`, `pkg/session/tattoo.go`
**Status:** ✅ DONE. All three Go targets exist and build.

### ✅ Wave 4a — Spec Assign (`spec_assign.c`) [COMPLETED]
**What:** Assignment table mapping zone numbers → special procedures
**Go target:** `pkg/game/spec_assign.go`
**Status:** ✅ DONE. `spec_assign.go` exists.

### 🔶 Wave 4b — Spec Procs Batch 1 (`spec_procs.c` first half, ~1800 lines) [PARTIALLY DONE via Lua]
**What:** First 20 spec procs (bank, mayor, guild, dragon_breath, elevator, janitor, pet_shops, etc.)
**Go target:** `pkg/game/spec_procs.go`
**Status:** 🔶 Lua scripts ported for most of these. The Go `spec_procs.go` structure exists but references functions (`me.GetMeleeTarget()`, `engine.ClassType`, `spells.Cast()`) that need their Go implementations wired. The C spec_procs logic is mostly handled through the Lua script layer now.

### Wave 5 — Spec Procs Batch 2 (`spec_procs.c` rest, `spec_procs2.c` first half, ~2500 lines)
**What:** Remaining `spec_procs.c` specials + first 20 from `spec_procs2.c` (assassin, backstabber, shop_keeper, teleporter, medusa, bat, etc.)
**Go target:** `pkg/game/spec_procs.go` (appended)

### Wave 6 — Spec Procs Batch 3 (`spec_procs2.c` rest, `spec_procs3.c` all, ~2000 lines)
**What:** Remaining `spec_procs2.c` + all of `spec_procs3.c` (butler, conjured, werewolf, mirror, turn_undead, recruiter, etc.)
**Go target:** `pkg/game/spec_procs.go` (appended)

### Wave 7 — World Interactivity (`boards.c`, `objsave.c`, `mobprog.c`, ~2447 lines)
**What:** Bulletin boards, cryogenic storage + receptionist, mob programs (triggers on enter/speech/kill/give)
**Go targets:** `pkg/game/boards.go`, `pkg/game/objsave.go`, `pkg/game/mobprogs.go`

### Wave 8 — OLC Framework (`improved-edit.c`, `olc.c`, `poof.c`, `tedit.c`, `luaedit.c`, `file-edit.c`, ~1608 lines)
**What:** Text editor engine, OLC framework, poof messages, trigger editor, Lua editor, file editor
**Go targets:** `pkg/olc/editor.go`, `pkg/olc/olc.go`, `pkg/olc/poof.go`, `pkg/olc/triggers.go`, `pkg/olc/files.go`

### Wave 9 — OLC Room + Object Editors (`redit.c`, `oedit.c`, ~2642 lines)
**What:** Room editor (descriptions, exits, flags, progs). Object editor (values, flags, wear locations, affects)
**Go targets:** `pkg/olc/rooms.go`, `pkg/olc/objects.go`

### Wave 10 — OLC Mob + Shop + Zone Editors (`medit.c`, `sedit.c`, `zedit.c`, ~3580 lines)
**What:** Mob editor, shop editor, zone editor (zone flags, reset commands, commands table)
**Go targets:** `pkg/olc/mobs.go`, `pkg/olc/shops.go`, `pkg/olc/zones.go`

### Wave 11 — Systems (`clan.c`, `house.c`, `whod.c`, ~2850 lines)
**What:** Clan system (create/disband/invite/kick, rankings, halls). Player housing (rent, decorate, lock/unlock, visitors). External WHO daemon.
**Go targets:** `pkg/game/clans.go`, `pkg/game/houses.go`, `pkg/game/whod.go`

### Wave 12 — Sonnet QA Audit
**What:** Review the full Go codebase for faithfulness to C original, compilation, logical correctness, proper error handling, logging.
**Output:** Issues list + fix recommendations

### Wave 13 — Opus Security Audit
**What:** Security review (command injection, Lua sandbox bypass, privilege escalation, DoS vectors, memory safety)
**Output:** Security report + fixes

---

## Immediate Next Steps (from ROADMAP.md "In Progress")

These are the highest-priority items before continuing Waves 5+:

### 1. Door Commands
- Door data parsed from zone files (D commands in zone resets)
- `pkg/command/door_commands.go` was deleted — needs complete rewrite
- Port `act.movement.c` `do_gen_door()` (open/close/lock/unlock/pick)
- Pick lock skill needs real door interaction

### 2. Shop System
- 10 shop scripts ported (shopkeeper, shop_give, etc.)
- Engine buy/sell/list commands missing — port `shop.c` (~1,445 lines)
- Scripts fire triggers; engine needs the actual transaction commands

### 3. Rescue Skill
- `DoRescue()` exists but needs combat engine wiring (`StopCombat()` + `StartCombat()` swap)
- Needs combat engine interface method exposure

### 4. Hitroll/Damroll from Equipment
- `formulas.go` currently returns 0 for equipment hit/dam bonuses
- Wire `APPLY_HITROLL`/`APPLY_DAMROLL` from equipped items
- Affects all combat accuracy/damage calculations

### 5. Non-Damage Spell Effects
- `spell()` deals damage but doesn't apply affects (blindness, curse, poison, sleep, sanctuary, etc.)
- Affect system exists (`pkg/engine/affect.go`); wire spell → affect application

---

## Model Routing Rules (Updated 2026-04-24)

| Model | Role | Rules |
|-------|------|-------|
| `deepseek-v4-flash` | **Daily driver / mechanical tasks** | Default for coding subagents. 284B/13B active params, 1M context, $0.14/$0.28/M. Fast, cheap, good enough for most builds. |
| `moonshot/kimi-k2.6` | **Build (secondary)** | Good for creative/interpretive translation when V4 isn't getting the nuance. Slower (~90-110s per file). |
| `zai/glm-5.1` | **Fix / long-horizon** | Slow (~44 tok/s), deep. Best for compilation fixing and complex refactors. Still active on $10/mo plan. |
| `deepseek-v4-pro` | **Reasoning / heavy lifting** | 1.6T/49B, $1.74/$3.48/M. Use when Flash isn't enough but Sonnet is overkill. Subagent only. |
| `anthropic/claude-sonnet-4-6` | **QA + architecture** | Architectural review and complex builds. **Requires approval** — expensive. |
| `anthropic/claude-opus-4-6` | **Security final** | Final pass after QA. One shot, expensive. Requires approval. |

### Swarm discipline (from SWARM-LEARNINGS.md):
1. **Don't parallelize on same provider** — rate limits kill the whole batch
2. **Right-size scope per subagent** — ~600-line C files / ~50K tokens sweet spot
3. **Sequential > parallel for large files** — 1200+ line C files should be sequential sub-waves
4. **QA gate enforced** — build agents write files but don't commit; QA approves first
5. **Read the original source** before writing game logic. Port faithfully, deviate intentionally.

---

## Documentation: Go Improvements Over C

Every build wave must produce a section at the bottom of the Go file or a companion `IMPROVEMENTS.md` note documenting:
1. What Go does better than the C original
2. Potential modernization targets (when we're 100% ported)
3. Any code smells caught during translation

Do NOT implement these improvements. Just document them.

---

## File Structure Convention

| C Source | Go Target |
|----------|-----------|
| `src/act.display.c` | `pkg/session/display_cmds.go` |
| `src/act.social.c` | `pkg/game/socials.go` |
| `src/act.wizard.c` | `pkg/session/wizard_cmds.go` |
| `src/boards.c` | `pkg/game/boards.go` |
| `src/clan.c` | `pkg/game/clans.go` |
| `src/file-edit.c` | `pkg/olc/files.go` |
| `src/house.c` | `pkg/game/houses.go` |
| `src/improved-edit.c` | `pkg/olc/editor.go` |
| `src/luaedit.c` | `pkg/olc/lua_editor.go` |
| `src/mapcode.c` | `pkg/session/map_cmds.go` |
| `src/medit.c` | `pkg/olc/mobs.go` |
| `src/mobprog.c` | `pkg/game/mobprogs.go` |
| `src/new_cmds.c` | `pkg/command/skill_commands.go` |
| `src/new_cmds2.c` | `pkg/session/new_cmds2.go` |
| `src/objsave.c` | `pkg/game/objsave.go` |
| `src/oedit.c` | `pkg/olc/objects.go` |
| `src/olc.c` | `pkg/olc/olc.go` |
| `src/poof.c` | `pkg/olc/poof.go` |
| `src/redit.c` | `pkg/olc/rooms.go` |
| `src/sedit.c` | `pkg/olc/shops.go` |
| `src/spec_assign.c` | `pkg/game/spec_assign.go` |
| `src/spec_procs.c` | `pkg/game/spec_procs.go` |
| `src/spec_procs2.c` | `pkg/game/spec_procs.go` |
| `src/spec_procs3.c` | `pkg/game/spec_procs.go` |
| `src/tattoo.c` | `pkg/session/tattoo.go` |
| `src/tedit.c` | `pkg/olc/triggers.go` |
| `src/whod.c` | `pkg/game/whod.go` |
| `src/zedit.c` | `pkg/olc/zones.go` |

---

## Session Startup

Each new session working on this plan should:
1. Read `PORT-PLAN.md` — this file (updated as of 2026-04-24)
2. Read `RESEARCH-LOG.md` — recent session journal
3. Read `docs/SWARM-LEARNINGS.md` — lessons from previous waves
4. Check `git log --oneline -5` — latest commits
5. Check what wave is next or in progress (look for uncommitted changes)
6. Check the "Immediate Next Steps" section above for highest-priority items
7. Read `docs/research.md` for architecture rationale (when designing new systems)
8. Proceed
