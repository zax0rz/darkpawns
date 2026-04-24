# C → Go Port Plan — Dark Pawns

> **Goal:** 100% faithful C-to-Go port of all ~68K lines of Dark Pawns MUD source.
> **Strategy:** 13 waves over ~5 days. Each wave = build → QA → fix → push.
> **Build note:** Go improvements over C are *documented*, not implemented. Keep the port faithful.
> **Models:** GLM-5.1 for mechanical Go porting, Kimi K2.6 for creative/interpretive, Sonnet for architecture/QA, Opus for security final. **No DeepSeek Chat.**

---

## Current State

```
C source:          ~68,792 lines across 59 files
Already in Go:     ~34,509 lines across 150 files (partial equivalents)
Genuinely unported: ~29,155 lines across 28 C files
```

The "already ported" C files mostly have thin Go wrappers that inherit the heavy lifting from C or Lua. The C logic itself hasn't been translated to Go — the Go side has struct definitions and shell command registrations, but the actual business logic lives in C.

---

## Wave Plan

### Wave 1 — Skill Commands (`new_cmds.c`, ~2792 lines)
**What:** bash, backstab, kick, trip, rescue, sneak, hide, steal, pick lock, berserk, charge, parry, headbutt, spike
**Model:** K2.6 build → GLM-5.1 fix
**Go target:** `pkg/command/skill_commands.go` (expanded)
**Status:** Has a Go shell already. Needs faithful translation of C logic.

### Wave 2 — Misc Player Commands (`new_cmds2.c`, ~1027 lines)
**What:** scrounge, first_aid, disarm, mindlink, detect, serpent_kick, dig, turn
**Model:** K2.6 build → GLM-5.1 fix
**Go target:** New file `pkg/session/new_cmds2.go`

### Wave 3 — Display + Map + Tattoo (`act.display.c`, `mapcode.c`, `tattoo.c`, ~1129 lines)
**What:** lines, infobar, map command, tattoo/eq protection, `act.social.c` (if needed)
**Model:** GLM-5.1
**Go targets:** `pkg/session/display_cmds.go`, `pkg/session/map_cmds.go`, `pkg/session/tattoo.go`

### Wave 4 — Spec Assign + Spec Procs Batch 1 (`spec_assign.c`, `spec_procs.c` first half, ~1800 lines)
**What:** The assignment table that maps zone numbers → special procedures. First 20 spec procs (bank, mayor, guild, dragon_breath, elevator, janitor, pet_shops, etc.)
**Model:** K2.6 build → GLM-5.1 fix
**Go target:** `pkg/game/spec_procs.go`, `pkg/game/spec_assign.go`

### Wave 5 — Spec Procs Batch 2 (`spec_procs.c` rest, `spec_procs2.c` first half, ~2500 lines)
**What:** Remaining spec_procs.c specials + first 20 from spec_procs2.c (assassin, backstabber, shop_keeper, teleporter, medusa, bat, etc.)
**Model:** K2.6 build → GLM-5.1 fix
**Go target:** `pkg/game/spec_procs.go` (appended)

### Wave 6 — Spec Procs Batch 3 (`spec_procs2.c` rest, `spec_procs3.c` all, ~2000 lines)
**What:** Remaining spec_procs2.c + all of spec_procs3.c (butler, conjured, werewolf, mirror, turn_undead, recruiter, etc.)
**Model:** K2.6 build → GLM-5.1 fix
**Go target:** `pkg/game/spec_procs.go` (appended)

### Wave 7 — World Interactivity (`boards.c`, `objsave.c`, `mobprog.c`, ~2447 lines)
**What:** Bulletin boards, cryogenic storage + receptionist, mob programs (triggers on enter/speech/kill/give)
**Model:** GLM-5.1
**Go targets:** `pkg/game/boards.go`, `pkg/game/objsave.go`, `pkg/game/mobprogs.go`

### Wave 8 — OLC Framework (`improved-edit.c`, `olc.c`, `poof.c`, `tedit.c`, `luaedit.c`, `file-edit.c`, ~1608 lines)
**What:** Text editor engine, OLC framework, poof messages, trigger editor, Lua editor, file editor
**Model:** GLM-5.1 (mechanical)
**Go targets:** `pkg/olc/editor.go`, `pkg/olc/olc.go`, `pkg/olc/poof.go`, `pkg/olc/triggers.go`, `pkg/olc/files.go`

### Wave 9 — OLC Room + Object Editors (`redit.c`, `oedit.c`, ~2642 lines)
**What:** Room editor (descriptions, exits, flags, progs). Object editor (values, flags, wear locations, affects)
**Model:** Sonnet (architecturally complex — needs to fit with existing parser/system)
**Go targets:** `pkg/olc/rooms.go`, `pkg/olc/objects.go`

### Wave 10 — OLC Mob + Shop + Zone Editors (`medit.c`, `sedit.c`, `zedit.c`, ~3580 lines)
**What:** Mob editor, shop editor, zone editor (zone flags, reset commands, commands table)
**Model:** Sonnet
**Go targets:** `pkg/olc/mobs.go`, `pkg/olc/shops.go`, `pkg/olc/zones.go`

### Wave 11 — Systems (`clan.c`, `house.c`, `whod.c`, ~2850 lines)
**What:** Clan system (create/disband/invite/kick, rankings, halls). Player housing (rent, decorate, lock/unlock, visitors). External WHO daemon.
**Model:** K2.6 build → GLM-5.1 fix
**Go targets:** `pkg/game/clans.go`, `pkg/game/houses.go`, `pkg/game/whod.go`

### Wave 12 — Sonnet QA Audit
**What:** Review the full Go codebase for faithfulness to C original, compilation, logical correctness, proper error handling, logging.
**Model:** Sonnet
**Output:** Issues list + fix recommendations

### Wave 13 — Opus Security Audit
**What:** Security review (command injection, Lua sandbox bypass, privilege escalation, DoS vectors, memory safety)
**Model:** Opus
**Output:** Security report + fixes

---

## Model Routing Rules

| Model | Role | Rules |
|-------|------|-------|
| `moonshot/kimi-k2.6` | **Build** | Run the translation. Gets first pass at interpreting C → Go. |
| `zai/glm-5.1` | **Fix** | Get the translated output compiling and correct. Mechanical work. |
| `anthropic/claude-sonnet-4-6` | **QA + complex builds** | Architectural review. Room/object/zone editors. |
| `anthropic/claude-opus-4-6` | **Security final** | Final pass after QA. One shot, expensive. |
| `litellm/deepseek-chat` | **NEVER** | Prohibited for this project. Creates bugs. |

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
1. Read `PORT-PLAN.md` — this file
2. Read `RESEARCH-LOG.md` — recent entries for context
3. Read `SWARM-LEARNINGS.md` — lessons from previous waves (if it exists)
4. Check `git log --oneline -5` — latest commits
5. Check what wave is next or in progress (look for `WAVE-N-INPROGRESS` tag or uncommitted changes)
6. Proceed
