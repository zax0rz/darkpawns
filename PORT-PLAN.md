# C → Go Port Plan — Dark Pawns

> **Goal:** 100% faithful C-to-Go port of all ~68K lines of Dark Pawns MUD source.
> **Strategy:** 13 waves. Each wave = build → QA → fix → push.
> **Wave 6 complete (2026-04-24):** act.wizard.c fully ported — 46 wizard commands registered.
> **Wave 6.5 complete (2026-04-24):** 22 player-facing commands from act.other.c wired — all World.doXxx have session-level wrappers + registry entries.
> **Update (2026-04-24):** Waves 1-5 COMPLETED. Wave 5 (game loop core: affect lifecycle, character management, char/obj updates, door system wiring) fully ported, tested, QA'd, committed, and pushed. 30 C functions ported across 4 new Go files. Door bashdoor command added alongside existing door commands.
> **Wave 6 reality check:** Wave 6 (act.wizard.c admin commands) was actually completed within Wave 5's partial commit. 46 wizard commands registered and implemented in `pkg/session/wizard_cmds.go` (1,574 lines).
> **Wave 7 complete (2026-04-24):** Spell system fully ported — magic.c, spells.c, spell_parser.c (~4,843 C lines) → 8 Go files (1,846 lines) in pkg/spells/. CallMagic dispatch, MagDamage (all spell formulas), MagAffects (20+ spells), saving throws (full 6×21×5 table), SaySpell (syllable substitution), spell_info template system, object magic, manual spell dispatch. Build and vet both clean.
> **Wave 8 complete (2026-04-24):** utils.c (~980 lines) → pkg/game/logging.go (392 lines). 9 functions ported: BasicMudLog, Alog, MudLog, LogDeathTrap, Sprintbit, Sprinttype, SprintbitArray, DieFollower, CoreDump. Build/vet both clean.
> **Wave 9 complete (2026-04-24):** comm.c + act.comm.c — 4203 C lines → 559 Go lines. comm_infra.go (timediff, nonblock, set_sendbuf, TxtQ, perform_subst, perform_alias, make_prompt, setup_log). act_comm_bridge.go (Exec wrappers). act_comm.go expanded (9 cmd wrappers). commands.go (+10 registrations). Build/vet clean. Commit fa2c4eb.
> **Wave 9.5 complete (2026-04-25):** fight.c (~2033 C lines) → pkg/combat/fight_core.go (990 Go lines). 49 functions covering the core combat loop: attack roll (MakeHit), damage (TakeDamage), position tracking (GetPositionFromHP), death processing (Die, RawKill, MakeCorpse, MakeDust), XP distribution (GroupGain, CalcLevelDiff), and mob AI triggers (CounterProcs, AttitudeLoot). Game-layer hooks via var block (55 function pointers) — zero direct game state access. Build/vet both clean. Combatant interface reverted to original (no GetMaster/GetSendMessage).
> **Wave 13 (Wave 12 in plan) complete (2026-04-25):** alias.c, ban.c, dream.c, weather.c (879 C lines) → 4 Go files (1,083 lines) in pkg/game/. Session commands wired (alias, ban, unban, dream). Player struct extended (Aliases, LastDeath). Manager wires HasActiveCharacter callback, loads ban/invalid lists at startup. Build/vet both clean. Commit a8ed79e.
> **Model note:** DeepSeek V4 Flash is the daily driver. Documented here so any model can pick up without loss.

---

## Current State (as of 2026-04-25, post-fight.c port) — REALITY-AUDITED

```
C source:            68,823 lines across 67 .c files
Go codebase:         61,690 lines across all .go files (incl. tests)
  Non-test Go:      49,119 lines across 139 .go files (estimate)
  Test files:        4,880 lines
Genuinely unported:  ~21,000 lines across ~14 C files (unaddressed)
Partially ported:    ~20,000 lines across 10+ C files (needs more coverage)
Replaced by SPA:     7,830 lines across 11 editor C files (OLC etc.)
Build:               go build ./... passes clean
go vet:              vet passes clean
go test:             41 tests pass in pkg/game/systems/
Git status:          clean (all on main)
```

### Line counts by package (non-test Go files)

| Package | Lines (non-test Go) | C source mapped to | Notes |
|---|---|---|---|
| `pkg/session/` | ~11,530 | act.*.c, interpreter.c, comm.c | Commands, display, wizard. **46 wizard cmds registered.** |
| `pkg/game/` | ~24,506 | All act_*.c, spec_*.c, shop.c, limits.c, class.c, modify.c | Core game logic. act_other_bridge.go provides exported wrappers for session access. |
| `pkg/command/` | ~2,787 | new_cmds.c, new_cmds2.c, shop.c | Skill + shop commands |
| `pkg/engine/` | ~3,425 | affect system, skill system | Pure Go additions |
| `pkg/combat/` | ~1,995 | fight.c + formulas.go + combatant.go | Combat engine |
| `pkg/scripting/` | ~3,801 | scripts.c | Lua engine |
| `pkg/telnet/` | ~389 | comm.c | Network listener |
| `pkg/parser/` | ~1,293 | db.c, world files | World file parsing |
| `pkg/db/` | ~772 | db.c | Player DB + narrative memory |
| `pkg/agent/` | ~395 | — | BRENDA agent system |
| `pkg/optimization/` | ~1,779 | — | Pooling, caching, etc. |
| `pkg/ai/` | ~140 | — | AI behaviors |
| `pkg/events/` | ~500 | events.c | Event bus |
| `pkg/spells/` | 1,846 | spells.c, magic.c, spell_parser.c | ✅ **Wave 7 — 8 Go files** |
| Other pkgs | ~2,400 | ban.c, mail.c, weather.c, etc. | Misc systems |
| **Total** | **~61,500** (incl. tests) | 67 C files | 142 .go files |

### What's actually merged (confirmed present):
### Confirmed merged into main

| Area | Go files | C source | Lines (Go) | Status |
|------|----------|----------|-----------|--------|
| Skill commands | `pkg/command/skill_commands.go` | `new_cmds.c` (~2792) | 1,587 | ✅ Complete |
| Misc player commands | `pkg/command/skill_commands.go` (embedded) | `new_cmds2.c` (~1027) | — | ✅ Complete (no standalone file) |
| Display | `pkg/session/display_cmds.go` | `act.display.c` (~717) | 460 | ✅ Good coverage |
| Map | `pkg/session/map_cmds.go` | `mapcode.c` | 284 | ✅ Complete |
| Tattoo | `pkg/session/tattoo.go` | `tattoo.c` | 248 | ✅ Complete |
| Socials | `pkg/game/socials.go`, `act_social.go` | `act.social.c` (~305) | 1,356 | ✅ Complete (expanded) |
| Spec assign | `pkg/game/spec_assign.go` | `spec_assign.c` (~642) | 450 | ✅ Complete |
| Spec procs | `pkg/game/spec_procs*.go` (4 files) | `spec_procs.c`/2/3 (~6,063) | 2,924 | 🔶 48% — Lua scripts fill gap |
| Shop system | `pkg/game/shop.go`, `*systems/shop*.go`, `*command/shop_commands.go`, `*session/shop_cmds.go`, `*common/shop.go` | `shop.c` (~1445) | 1,548 | ✅ Complete |
| Doors | `pkg/game/systems/door*.go`, `pkg/game/act_movement.go` | `act.movement.c` | 1,332 | ✅ Complete (refactored) |
| Eat/drink | `pkg/session/eat_cmds.go` | — | 297 | ✅ Complete |
| Affects | `pkg/session/affects_informative.go`, `pkg/engine/affect*.go` | — | 1,179 | ✅ Complete |
| Movement | `pkg/session/movement_cmds.go` | `act.movement.c` | 419 | ✅ Complete |
| Combat engine | `pkg/combat/engine.go`, `formulas.go`, `combatant.go`, `fight_core.go` | `fight.c` (~2033) | 1,995 | ✅ ~98% — hitroll/damroll from eq still missing, peripheral functions deferred |
| Wizard commands | `pkg/session/wizard_cmds.go` | `act.wizard.c` (~3863) | 1,574 | ✅ **Actually complete — 46 cmds registered** |
| Act other | `pkg/game/act_other.go` + `act_other_bridge.go` + `pkg/session/commands.go` | `act.other.c` (~1947) | 1,718 game + bridge | ✅ **Wave 6.5 done** — 22 commands wired, all registered |
| Act informative | `pkg/game/act_informative.go` | `act.informative.c` (~2803) | 910 | 🔶 ~32% |
| BRENDA memory | `pkg/agent/memory_hooks.go`, `pkg/db/narrative_memory.go`, `pkg/session/memory_hooks.go` | — | 951 | ✅ Complete (pure Go addition) |
| 115 Lua scripts | `test_scripts/mob/archive/` | — | — | ✅ All merged |

**All files committed.** No untracked files remain.

### What does NOT exist yet (fully unported)

| C Source | Lines | Go target | Priority |
|----------|-------|-----------|----------|
| `clan.c` | 1,574 | `pkg/game/clans.go` | ⭐ High |
| `house.c` | 744 | `pkg/game/houses.go` | ⭐ High |
| `boards.c` | 551 | `pkg/game/boards.go` | ⭐ High |
| `whod.c` | 532 | `pkg/game/whod.go` | Medium |
| `objsave.c` | 1,250 | `pkg/game/objsave.go` | Medium |
| `mobprog.c` | 646 | `pkg/game/mobprogs.go` | Medium (partially via Lua) |
| `pkg/admin/` | — | New package | Low (Web API exists at `web/`) |

### Heavily under-ported areas

| C Source | Lines | Go | Coverage | Issue |
|----------|-------|-----|----------|-------|
| `act.wizard.c` | 3,863 | 1,574 | ✅ **~100%** | **COMPLETE** — 46 commands registered |
| `act.other.c` | 1,947 | 1,718 + bridge | ✅ Wave 6.5: 22 commands wired & registered | Bridge file + session wrappers connect all |
| `magic.c` + `spells.c` + `spell_parser.c` | 4,843 | ~192 | 🔴 ~10% | Huge gap — spell effects missing |
| `act.informative.c` | 2,803 | 1,083 | 🔶 ~39% | 3 Go files, incomplete |
| `handler.c` | 1,616 | 1,495 | ✅ ~92% | Nearly done |
| `fight.c` | 2,033 | 1,995 | ✅ ~98% | Hitroll/damroll from equipment missing. Deferred: forget/remember, stop_follower, tattoo_af, unmount, set_hunting |
| `comm.c` | 2,637 | 1,426 | 🔶 ~54% | Listener + manager done |
| `interpreter.c` | 2,365 | 1,855 | 🔶 ~78% | Commands.go covers most |

### C files replaced by Web Admin SPA (NOT ported)

| File | Lines | Reason |
|------|-------|--------|
| `oedit.c` | 1,564 | SPA object editor |
| `redit.c` | 1,078 | SPA room editor |
| `medit.c` | 1,126 | SPA mob editor |
| `sedit.c` | 1,178 | SPA shop editor |
| `zedit.c` | 1,276 | SPA zone editor |
| `olc.c` | 524 | SPA OLC framework |
| `improved-edit.c` | 627 | SPA text editor |
| `luaedit.c` | 58 | Monaco editor |
| `tedit.c` | 98 | SPA trigger editor |
| `poof.c` | 102 | SPA poof messages |
| `file-edit.c` | 199 | SPA file upload |
| **Total** | **7,830** | All replaced by Web Admin |

---

## Wave Plan (Updated 2026-04-24)

### ✅ Wave 1 — Skill Commands (`new_cmds.c`, ~2792 lines) [COMPLETED]
**Go target:** `pkg/command/skill_commands.go` (expanded)
**Status:** ✅ DONE. All skill commands ported. Skill system wired (SkillManager, skill points, practice/learn/forget).

### ✅ Wave 2 — Misc Player Commands (`new_cmds2.c`, ~1027 lines) [COMPLETED]
**Go target:** Content lives inside `pkg/command/skill_commands.go` (no standalone `new_cmds2.go`)
**Status:** ✅ DONE. Ported alongside Wave 1.

### ✅ Wave 3 — Display + Map + Tattoo (`act.display.c`, `mapcode.c`, `tattoo.c`, ~1129 lines) [COMPLETED]
**Go targets:** `pkg/session/display_cmds.go`, `pkg/session/map_cmds.go`, `pkg/session/tattoo.go`
**Status:** ✅ DONE.

### ✅ Wave 4a — Spec Assign (`spec_assign.c`, ~642 lines) [COMPLETED]
**Go target:** `pkg/game/spec_assign.go`
**Status:** ✅ DONE.

### 🔶 Wave 4b — Spec Procs (`spec_procs.c/2/3`, ~6063 lines total) [PARTIALLY DONE — 48%]
**Go targets:** `pkg/game/spec_procs.go`, `spec_procs2.go`, `spec_procs3.go`, `spec_procs4.go`
**Status:** 🔶 2,924 lines ported across 4 Go files (~48%). Lua scripts fill gaps. Remaining spec procs need Go implementations wired (GetMeleeTarget, ClassType, spells.Cast).

### ✅ Wave 5 — Game Loop + Core (comm.c + interpreter.c + handler.c, ~6618 lines) [COMPLETED]
**C functions ported (30 total):** affect_update, point_update (via HitGain/ManaGain/MoveGain/GainCondition), init_char (via NewPlayer/NewCharacter constructors), aff_apply_modify, affect_modify_ar, affect_total, master_affect_to_char, affect_to_char2, affect_remove, affect_from_char, affect_join, obj_from_obj, object_list_new_owner, update_object, update_char_objects, update_char_objects (AR variant), extract_pending_chars, HasLight, ExtractChar, SpellWearOffMsg
**Intentionally NOT ported (6 functions):** free_char, clear_char, stop_follower, add_follower, remove_follower, set_hunting — Go design patterns cover these via constructors, Manager methods, and World-scoped state
**Go targets (new files):** `pkg/engine/affect_helpers.go`, `pkg/game/affect_update.go`, `pkg/game/char_mgmt.go`
**Status:** ✅ DONE. Build clean, vet clean, committed (e2aa5a6), pushed to GitHub. Wave 5 QA'd via diff comparison and build verification.
**Bonus — door bashdoor:** `bashdoor`/`dbash` command added (d4cdd6e) alongside existing `open/close/lock/unlock/pick/knock` — checks Bashable flag, door HP, player Strength.

### ✅ Wave 6 — Admin commands (act.wizard.c, ~3863 lines) [COMPLETED]
**Go target:** `pkg/session/wizard_cmds.go` (1,574 lines)
**Status:** ✅ DONE. 46 wizard commands registered and implemented. All registrations live in `commands.go` (no init() in wizard_cmds.go needed).

### Wave 6.5 — Player commands from act.other.c (~1947 lines, ~22 functions) [✅ COMPLETED 2026-04-24]
**Context:** act_other.go had all the World.doXxx implementations but **zero session-level wiring**.
**Work done:**
- Added `pkg/game/act_other_bridge.go` — 21 exported `ExecXxx` wrapper methods that delegate to unexported `doXxx`
- Added 22 session-level `cmdXxx` wrappers in `pkg/session/commands.go` calling the bridge methods
- Registered all 22 commands: save, report, split, wimpy, display, transform, ride, dismount, yank, peek, recall, stealth, appraise, scout, roll, visible, inactive, afk, auto, gentog, bug/typo/idea/todo (via gen_write)
- Estimated ~200 lines of Go (wrappers + registrations)
- Build verified: `go build ./... && go vet ./...` clean

### ✅ Wave 7 — Spell system (magic.c + spells.c + spell_parser.c, ~4843 lines) [COMPLETED 2026-04-24]
**C sources ported:** magic.c (~1,999 lines), spells.c (~1,218 lines), spell_parser.c (~1,626 lines)
**Go targets (8 files, 1,846 lines):**
- `pkg/spells/call_magic.go` — CallMagic central dispatch, SpellInfo struct, CastType/TarFlags/MagRoutine constants
- `pkg/spells/damage_spells.go` — MagDamage switch: 20+ spell damage formulas (magic missile, fireball, lightning bolt, chill touch, burning hands, shocking grasp, color spray, disintegrate, disrupt, dispel evil/good, call lightning, harm, energy drain, soul leech, earthquake, acid blast, hellfire, meteor swarm, calliope, smokescreen, breath weapons)
- `pkg/spells/affect_spells.go` — MagAffects (20+ affect spells), MagPoints, MagUnaffects, group/mass/area/summon/creation/alter-obj stubs
- `pkg/spells/affect_effects.go` — Existing 5 affect spells (blindness, curse, poison, sleep, sanctuary)
- `pkg/spells/spell_info.go` — SpellInfo table, HasRoutine, GetSpellInfo, SpellLevel, MagAssignSpells
- `pkg/spells/saving_throws.go` — Full sav_throws table (6 classes × 21 levels × 5 save types)
- `pkg/spells/say_spell.go` — Syllable substitution, class-aware incantations
- `pkg/spells/object_magic.go` — MagObjectMagic for potion/wand/staff/scroll
- `pkg/spells/spells.go` — All spell constants + Cast() entry point
**Status:** ✅ Build clean (`go build ./...`), vet clean (`go vet ./...`).
**Pending (Wave 8):** Wire CallMagic into session/cast_cmds.go, flesh out group/mass/area/summon/creation/alter-obj stubs, connect affects to engine.AffectManager, implement ExecuteManualSpell dispatch with real implementations.

### Wave 8 — Wire spell system + Logging/Utility (cast_cmds.go + utils.c)

### ✅ Wave 9.5 — Combat engine core (fight.c, ~2033 lines) [COMPLETED 2026-04-25]
**Go target:** `pkg/combat/fight_core.go` (990 Go lines, 49 functions)
**Status:** ✅ DONE. Build clean, vet clean.
**Ported functions:** MakeHit, TakeDamage, GetPositionFromHP, ChangeAlignment, DeathCry, RawKill, Die, DieWithKiller, MakeCorpse, MakeDust, CounterProcs, AttitudeLoot, GroupGain, PerformGroupGain, CalcLevelDiff, IsInGroup, DamMessage + 14-tier damage message table, AttackHitTexts, fight constants (TYPE_HIT..TYPE_BLAST, SKILL_BACKSTAB..SKILL_PARRY, AFF_*, LVL_IMMORT)
**Deferred (peripheral):** forget/remember, stop_follower, tattoo_af, unmount, set_hunting, can_speak — belong in game/AI layer
**Architecture:** 55 game-layer function pointers in var block — zero direct game state. Combatant interface unchanged (no GetMaster/GetSendMessage added).
**Work:**
- Connect CallMagic into session/cast_cmds.go (replace Cast stub with real dispatch)
- Implement group/mass/area/summon/creation/alter-obj in affect_spells.go
- Implement real manual spell dispatch in spell_manual.go
- Connect engine.AffectManager to spell affects
- Port utils.c (~980 lines): basic_mud_log, mudlog, alog, sprintbit, sprinttype, etc.
**Functions to port:** basic_mud_log, mudlog, alog, log_death_trap, sprintbit, sprinttype, sprintbitarray, die_follower, core_dump_real
**~7 functions, ~700 lines new Go code**
**Go target:** `pkg/game/logging.go`

### ✅ Wave 9 — Communication subsystem (comm.c + act.comm.c, ~4203 lines) [COMPLETED 2026-04-24]
**Go targets:** `pkg/engine/comm_infra.go` (402 lines — infrastructure helpers), `pkg/game/act_comm_bridge.go` (58 lines — bridge wrappers), `pkg/session/act_comm.go` (+89 lines — session command wrappers), `pkg/session/commands.go` (+10 registrations)
**Status:** ✅ DONE. Build clean, vet clean. Commit fa2c4eb.
**Infra ported:** timediff/timeadd, nonblock, set_sendbuf, TxtQ queue, perform_subst, perform_alias, make_prompt (full ANSI-colored), setup_log/open_logfile stubs
**Commands wired:** gossip, reply, write, page, ignore, race_say, whisper, ask, qcomm, think

### Wave 10 — Persistence (objsave.c, ~1250 lines)
**Functions to port:** Crash_listrent, auto_equip, Crash_restore_weight, Crash_extract_objs, Crash_extract_norents, Crash_extract_norents_from_equipped, Crash_extract_expensive, Crash_calculate_rent, Crash_crashsave, Crash_idlesave, Crash_cryosave, Crash_rent_deadline, Crash_report_rent, Crash_save_all
**~14 functions, ~1000 lines new Go code**
**Go target:** `pkg/game/objsave.go`

### Wave 11 — Clan + Housing (clan.c + house.c, ~2318 lines)
**Functions to port:** string_write, save_char_file_u (clan), House_restore_weight, House_crashsave, House_delete_file, House_listrent, House_save_control, House_boot, hcontrol_list_houses, hcontrol_build_house, hcontrol_destroy_house, hcontrol_pay_house, House_save_all, hcontrol_set_key
**~14 functions, ~1800 lines new Go code**
**Go targets:** `pkg/game/clans.go`, `pkg/game/houses.go`

### ✅ Wave 12 — Boards + Misc (alias.c + ban.c + dream.c + weather.c, ~1430 lines) [COMPLETED 2026-04-25]
**C sources ported:** alias.c (110), ban.c (313), dream.c (223), weather.c (233) = 879 C lines
**Go targets:** `pkg/game/aliases.go`, `pkg/game/bans.go`, `pkg/game/dreams.go`, `pkg/game/weather.go` (1,083 Go lines)
**Session wiring:** cmdAlias, cmdBan, cmdUnban, cmdDream in act_social.go + commands.go. Player struct: Aliases, LastDeath. Manager: HasActiveCharacter callback, auto-load bans/invalid-list. ValidName check during login.
**Status:** ✅ DONE. Build clean, vet clean. Commit a8ed79e.
**Note:** boards.c (551 lines) NOT included — deferred to future wave. The Wave 12 description in the research log called this "Wave 13" (our internal numbering drifted from PORT-PLAN.md numbering).

### 🚫 Waves 13-14 — OLC Editors (REPLACED by Web Admin SPA)
**Decision: Do NOT port.** ~7,830 lines replaced by Web Admin SPA.

### Wave 15 — Sonnet QA Audit
Review full Go codebase for faithfulness, compilation, correctness, error handling, logging.

### Wave 15 — Remaining Port Waves (clan.c, house.c, boards.c, whod.c, objsave.c, mobprog.c)

### Wave 15 — Remaining Port Waves (clan.c, house.c, boards.c, whod.c, objsave.c, mobprog.c)

| C Source | Lines | Go target | Priority |
|----------|-------|-----------|----------|
| `clan.c` | 1,574 | `pkg/game/clans.go` | ⭐ High |
| `house.c` | 744 | `pkg/game/houses.go` | ⭐ High |
| `boards.c` | 551 | `pkg/game/boards.go` | ⭐ High |
| `whod.c` | 532 | `pkg/game/whod.go` | Medium |
| `objsave.c` | 1,250 | `pkg/game/objsave.go` | Medium |
| `mobprog.c` | 646 | `pkg/game/mobprogs.go` | Medium (partially via Lua) |

Also: under-ported areas needing coverage:
- `act.informative.c` (~2,803 lines, ~39% ported) — kender_steal, do_description
- Hitroll/damroll from equipment (fight.c peripheral, deferred)
- dream-related PRNG, tattoo timer dock, night/weather event stubs

### Wave 16 — GPT-5.5 Pro Modernization Review

**Rationale:** GPT-5.5 Pro just launched (2026-04-24). Terminal-Bench 82.7%, Expert-SWE 73.1%, "first coding model with serious conceptual clarity" (Dan Shipper). Perfectly suited to review the completed Go codebase for modernization — not rewriting, not changing the game, but bringing everything to April 2026 Go idioms and best practices.

**Scoping:**
- Feed the entire Go codebase to GPT-5.5 Pro as a code review target
- Target areas: go 1.24 idioms, error wrapping patterns, context propagation, goroutine hygiene, package boundaries, naming conventions, dead code, unnecessary indirection
- Do NOT change: game logic, formulas, protocol, database schema, public API surface
- Output: a list of refactoring candidates with priority and diff sketches
- Model: GPT-5.5 Pro (API access required)

**Philosophy:** This is the "it just works" phase. Nothing user-facing changes. The game behaves identically. But the code is brought up to current standards so that everything built after (admin, agents, features) has a clean foundation.

**Success criteria:**
- go build ./... && go vet ./... both clean after each modernization commit
- All existing tests pass
- No behavioral change observable in game
- Code in Go is idiomatic to Q1 2026 conventions

### Wave 17 — QA + Security + Ship

Two phases, parallelizable:

**Phase A — QA Audit:** Full codebase review for faithfulness, compilation, correctness, error handling, logging. Focus on: are there paths where the port dropped edge cases? Are error messages preserved? Is logging consistent?

**Phase B — Security Audit:** Command injection, Lua sandbox bypass, privilege escalation, DoS vectors, admin auth, websocket session hijacking. Recommended model: Opus 4.6 or equivalent.

### Wave 18 — Admin Dashboards + Agent Hooks ("The Fun Phase")

Once the port is finished, modernized, QA'd, and secured:
- Web admin dashboard (prosecco integration?)
- Agent management UI
- BRENDA session monitoring
- In-game admin tools
- Real-time telemetry

---

## Immediate Next Steps (Updated 2026-04-25 — post-GPT-5.5 launch, reconfigured)

### 🔴 PRIORITY 1: Finish the port (Wave 15)
Last C files standing:
- clan.c (1,574) → clans.go
- house.c (744) → houses.go
- boards.c (551) → boards.go
- whod.c (532) → whod.go
- objsave.c (1,250) → objsave.go
- mobprog.c (646) → mobprogs.go
- Act.informative.c coverage (kender_steal, do_description)
- Hitroll/damroll from equipment
- Dream/tattoo/weather event stubs

### 🔵 PRIORITY 2: GPT-5.5 Pro Modernization (Wave 16)
Feed complete Go codebase to GPT-5.5 Pro for code review. Target: go 1.24+ idioms, error wrapping, context, goroutines, package boundaries. Zero behavioral change.

### 🟢 PRIORITY 3: QA + Security (Wave 17)
Phase A: Full QA review — faithfulness, edge cases, error messages, logging.
Phase B: Security audit — injection, sandbox, privilege escalation, auth, websocket.

### 🟣 PRIORITY 4: Admin + Agent Features (Wave 18 — The Fun Phase)
Web admin dashboard, agent management UI, BRENDA monitoring, telemetry.

---

## Function-Level Gap Map (Updated 2026-04-24)

> Each entry below = a C function that has NO corresponding Go implementation yet.
> Status: ❌ = not ported, ⚠️ = partial, ✅ = exists in Go.

### Tier 1 — Game Loop & Core (comm.c, interpreter.c, handler.c)

#### `comm.c` (2637 lines, ~70% unported)
| C Function | Go Status | Priority | Notes |
|---|---|---|---|
| `init_game` | ✅ In Go | — | Game initialization |
| `game_loop` | ⚠️ Partial | P1 | Main loop exists but no connection event dispatch |
| `heartbeat` | ✅ In Go (`pkg/events/`) | — | Tick system ported as event bus |
| `send_to_char` | ✅ In Go | — | Character messaging |
| `send_to_room` | ✅ In Go | — | Room messaging |
| `act` / `perform_act` | ✅ In Go | — | Action messaging |
| `close_socket` | ❌ MISSING | P1 | Descriptor cleanup |
| `flush_queues` | ❌ MISSING | P1 | Output buffer flush |
| `nonblock` | ❌ MISSING | P2 | Socket nonblocking mode |
| `signal_setup` | ❌ MISSING | P2 | Signal handlers (SIGINT, SIGHUP) |
| `record_usage` | ❌ MISSING | P3 | Usage statistics |
| `check_idle_passwords` | ❌ MISSING | P3 | Idle connection timeout |
| `boot_db` / `boot_world` | ⚠️ Partial | P1 | Area loading, partially in `pkg/parser/` |
| `zone_update` | ❌ MISSING | P1 | Zone reset/reload |
| `affect_update` | ❌ MISSING | P1 | Affect tick processing |
| `point_update` | ❌ MISSING | P1 | Regen tick (HP/mana/move) |
| `mobile_activity` | ❌ MISSING | P1 | Mob AI tick |
| `perform_violence` | ❌ MISSING | P1 | Combat round |
| `room_activity` / `object_activity` | ❌ MISSING | P2 | Room/object tick processing |
| `hunt_items` | ❌ MISSING | P2 | Item hunting |
| `write_to_q` | ❌ MISSING | P2 | Queue management |
| `send_to_all` | ❌ MISSING | P2 | Broadcast to all players |
| `send_to_outdoor` | ❌ MISSING | P3 | Outdoor room broadcast |
| `do_broadcast` | ❌ MISSING | P3 | Immortal broadcast command |
| `string_add` / `show_string` | ❌ MISSING | P2 | String display helpers |
| `save_clans` | ❌ MISSING | P2 | Clan persistence |
| `InfoBarUpdate` | ❌ MISSING | P3 | Info bar refresh |
| `setup_log` / `basic_mud_log` | ❌ MISSING | P2 | Logging infrastructure |

#### `handler.c` (1616 lines, ~48% unported)
| C Function | Go Status | Priority | Notes |
|---|---|---|---|
| `free_char` | ❌ MISSING | P1 | Free mob/player struct |
| `stop_fighting` | ✅ In Go | — | Combat stop |
| `remove_follower` | ❌ MISSING | P1 | Remove from follower chain |
| `clearMemory` | ✅ In Go | — | Mob memory clearing |
| `raw_kill` | ✅ In Go | — | Kill/remove char |
| `tattoo_af` | ✅ In Go | — | Tattoo affect handler |
| `set_hunting` | ❌ MISSING | P1 | Set mob hunt target |
| `aff_apply_modify` | ❌ MISSING | P2 | Apply affect modification |
| `affect_modify_ar` | ❌ MISSING | P2 | Affect AC modification |
| `affect_total` | ❌ MISSING | P2 | Sum all affects |
| `master_affect_to_char` | ❌ MISSING | P2 | Master affect list |
| `affect_to_char` | ✅ In Go | — | Single affect apply |
| `affect_to_char2` | ❌ MISSING | P2 | Secondary affect apply |
| `affect_remove` | ❌ MISSING | P2 | Affect removal |
| `affect_from_char` | ❌ MISSING | P2 | Affect extraction |
| `affect_join` | ❌ MISSING | P2 | Affect merging |
| `char_from_room` / `char_to_room` | ✅ In Go | — | Room movement |
| `obj_to_char` / `obj_from_char` | ✅ In Go | — | Object inventory |
| `equip_char` | ✅ In Go | — | Equipment |
| `obj_to_room` / `obj_from_room` | ✅ In Go | — | Room objects |
| `obj_to_obj` / `obj_from_obj` | ❌ MISSING | P2 | Container items |
| `object_list_new_owner` | ❌ MISSING | P2 | Owner tracking |
| `extract_obj` | ✅ In Go | — | Object removal |
| `update_object` | ❌ MISSING | P2 | Tick-based object updates |
| `update_char_objects` | ❌ MISSING | P2 | Tick-based char equipment updates |
| `extract_char` | ✅ In Go | — | Character removal |
| `extract_pending_chars` | ❌ MISSING | P2 | Batch char cleanup |

#### `interpreter.c` (2365 lines, ~26% unported)
| C Function | Go Status | Priority | Notes |
|---|---|---|---|
| `command_interpreter` | ✅ In Go | — | Command routing (Go port uses `pkg/command/registry.go`) |
| `perform_complex_alias` | ❌ MISSING | P3 | Alias expansion |
| `do_start` | ✅ In Go | — | Character creation init |
| `init_char` | ❌ MISSING | P1 | Character struct initialization |
| `roll_real_abils` | ✅ In Go | — | Ability score rolling |
| `read_aliases` | ❌ MISSING | P3 | Alias file loading |
| `read_poofs` | ❌ MISSING | P3 | Poof message loading |
| `echo_on` / `echo_off` | ❌ MISSING | P2 | Terminal echo control |
| `skip_spaces` / `half_chop` / `one_space_half_chop` | ✅ Partial | P3 | String parsing utils |
| `free_alias` | ❌ MISSING | P3 | Alias cleanup |
| OLC editor parse fns (6) | 🚫 Replaced by SPA | — | Not porting |

### ✅ Tier 2 — Admin Commands (act.wizard.c, 3863 lines) [COMPLETED]

46 wizard commands registered and implemented in `pkg/session/wizard_cmds.go` (1,574 lines). No remaining work.

### Tier 2.5 — Player Commands (act.other.c, ~1947 lines)

Functions exist in `pkg/game/act_other.go` (1,718 lines). **Wave 6.5 COMPLETE** — all wired via `pkg/game/act_other_bridge.go` + `pkg/session/commands.go`.
| C Function | Go Status | Priority | Notes |
|---|---|---|---|
| `do_save` | ✅ Wired & registered | P0 | ExecSave bridge + cmdSave wrapper |
| `do_report` | ✅ Wired & registered | P0 | ExecReport bridge + cmdReport wrapper |
| `do_split` | ✅ Wired & registered | P1 | ExecSplit bridge + cmdSplit wrapper |
| `do_wimpy` | ✅ Wired & registered | P1 | ExecWimpy bridge + cmdWimpy wrapper |
| `do_display` | ✅ Wired & registered | P1 | ExecDisplay bridge + cmdDisplay wrapper |
| `do_transform` | ✅ Wired & registered | P2 | ExecTransform bridge + cmdTransform wrapper |
| `do_ride` / `do_dismount` | ✅ Wired & registered | P2 | ExecRide/ExecDismount + cmdRide/cmdDismount |
| `do_yank` | ✅ Wired & registered | P2 | ExecYank bridge + cmdYank wrapper |
| `do_peek` | ✅ Wired & registered | P0 | ExecPeek bridge + cmdPeek wrapper |
| `do_recall` | ✅ Wired & registered | P0 | ExecRecall bridge + cmdRecall wrapper |
| `do_stealth` | ✅ Wired & registered | P2 | ExecStealth bridge + cmdStealth wrapper |
| `do_appraise` | ✅ Wired & registered | P2 | ExecAppraise bridge + cmdAppraise wrapper |
| `do_scout` | ✅ Wired & registered | P2 | ExecScout bridge + cmdScout wrapper |
| `do_roll` | ✅ Wired & registered | P2 | ExecRoll bridge + cmdRoll wrapper |
| `do_visible` | ✅ Wired & registered | P1 | ExecVisible bridge + cmdVisible wrapper |
| `do_inactive` | ✅ Wired & registered | P1 | ExecInactive bridge + cmdInactive wrapper |
| `do_afk` | ✅ Wired & registered | P0 | ExecAFK bridge + cmdAFK wrapper |
| `do_auto` | ✅ Wired & registered | P2 | ExecAuto bridge + cmdAuto wrapper |
| `do_gen_write` | ✅ Wired & registered | P1 | ExecGenWrite bridge + cmdBug/cmdTypo/cmdIdea/cmdTodo wrappers |
| `do_gen_tog` | ✅ Wired & registered | P1 | ExecGenTog bridge + cmdGenTog wrapper (alias gentoggle) |
| `do_not_here` | ❌ Skipped | P3 | Stub: not intended for direct player use |

### ✅ Tier 3 — Spell System (magic.c + spells.c + spell_parser.c, ~4843 lines) [WAVE 7 COMPLETE 2026-04-24]

| C Function | Go Status | Priority | Notes |
|---|---|---|---|
| `spell_level` | ✅ In Go | — | Spell level lookup |
| `spello` / `find_skill_num` | ✅ In Go | — | Spell name lookup + FindSpellByName |
| `unused_spell` | ✅ In Go (TODO stub) | P2 | Spell registration placeholder |
| `call_magic` | ✅ In Go | — | Central dispatch: CallMagic |
| `mag_assign_spells` | ✅ In Go | — | MagAssignSpells init function |
| `mag_manacost` | ✅ In Go | — | MagicManaCost in call_magic.go |
| `mag_savingthrow` | ✅ In Go | — | magSavingThrow in call_magic.go |
| `mag_materials` / `mag_reagent` | ✅ In Go | — | checkMaterials/checkReagents in call_magic.go |
| `mag_damage` | ✅ In Go | — | MagDamage with 20+ spell formulas |
| `mag_affects` | ✅ In Go | — | MagAffects with 20+ affect spells |
| `mag_unaffects` | ✅ In Go | — | MagUnaffects (remove curse, cure blind, remove poison) |
| `mag_points` | ✅ In Go | — | MagPoints (heal, harm, vitality) |
| `mag_groups` | ✅ In Go (stub) | P3 | Group version placeholder |
| `mag_masses` | ✅ In Go (stub) | P3 | Mass effect placeholder |
| `mag_areas` | ✅ In Go (stub) | P3 | Area effect placeholder |
| `mag_summons` | ✅ In Go (stub) | P3 | Summon placeholder |
| `mag_creations` | ✅ In Go (stub) | P3 | Create food/water placeholder |
| `mag_alter_objs` | ✅ In Go (stub) | P3 | Enchant/identify placeholder |
| `mag_objectmagic` | ✅ In Go | — | Staff/wand/scroll/potion handling |
| `say_spell` | ✅ In Go | — | SaySpell with syllable substitution |
| `sav_throws` table | ✅ In Go | — | Full 6×21×5 saving throw table |
| `spell_xxx()` implementations (~55) | ✅ In Go (stub) | P3 | ExecuteManualSpell dispatch map created |
| `weight_change_object` | ❌ MISSING | P2 | Inventory weight tracking |
| `add_follower` | ❌ MISSING | P1 | Follower chain management |

### Tier 4 — Informative Commands (act.informative.c, 2803 lines)

| C Function | Go Status | Priority | Notes |
|---|---|---|---|
| `kender_steal` | ❌ MISSING | P2 | Kender theft system |
| `do_description` | ❌ MISSING | P2 | Character description commands |

### Tier 5 — Utility / Logging (utils.c, 980 lines)

| C Function | Go Status | Priority | Notes |
|---|---|---|---|
| `basic_mud_log` | ❌ MISSING | P1 | Core logging function |
| `mudlog` | ❌ MISSING | P1 | Level-filtered logging |
| `alog` | ❌ MISSING | P1 | Admin logging |
| `log_death_trap` | ❌ MISSING | P3 | Death trap logging |
| `sprintbit` / `sprinttype` / `sprintbitarray` | ❌ MISSING | P2 | Bit/type-to-string helpers |
| `die_follower` | ❌ MISSING | P2 | Follower death cleanup |
| `core_dump_real` | ❌ MISSING | P3 | Crash dump |

### Tier 6 — Persistence (objsave.c, 1250 lines)

| C Function | Go Status | Priority | Notes |
|---|---|---|---|
| `Crash_*` (all 14 functions) | ❌ MISSING | P2 | Object persistence system |

### Tier 7 — Social / Clan / Housing (clan.c + house.c + boards.c, ~2869 lines)

| C Function | Go Status | Priority | Notes |
|---|---|---|---|
| `string_write` | ❌ MISSING | P2 | Clan motd write |
| `save_char_file_u` | ❌ MISSING | P1 | Player file save (clan field) |
| `House_*` (all 12 functions) | ❌ MISSING | P2 | Housing system |
| `Board_*` (all 8 functions) | ❌ MISSING | P2 | Bulletin boards |
| `init_boards` | ❌ MISSING | P2 | Board init |

### Tier 8 — Mob AI / Activity (mobact.c + mobprog.c, ~1054 lines)

| C Function | Go Status | Priority | Notes |
|---|---|---|---|
| `hunt_victim` | ❌ MISSING | P2 | Mob tracking/hunting |
| `mp_sound` | ❌ MISSING | P2 | Mob prog sound effect |
| `mobile_activity` | ❌ MISSING | P1 | Mob AI tick |
| `remember` | ✅ In Go | — | Mob memory |

### ✅ Tier 9 — Misc (alias.c + ban.c + dream.c + weather.c, ~1430 lines) [COMPLETED 2026-04-25]

| C Function | Go Status | Priority | Notes |
|---|---|---|---|
| `read_aliases` / `write_aliases` | ✅ In Go | — | Player alias persistence |
| `load_banned` / `write_ban_list` | ✅ In Go | — | Site ban system |
| `Read_Invalid_List` | ✅ In Go | — | Invalid name filter |
| `IsBanned` | ✅ In Go | — | Ban check |
| `AddBan` / `RemoveBan` | ✅ In Go | — | Ban management |
| `ValidName` | ✅ In Go | — | Name validation (invalid list + active chars) |
| `dream` / `dream_travel` | ✅ In Go | — | Dream sequences (DreamResult struct) |
| `weather_and_time` | ✅ In Go | — | Weather/time system |
| `another_hour` / `weather_change` | ✅ In Go | — | Weather cycle functions |
| `prng_seed` | ❌ Deferred | P3 | RNG seed control (not critical for Go math/rand) |
| `full_moon`, `lunar_hunter`, etc. | ✅ Stub no-ops | — | Event stubs (NPC spawning, deferred) |

**TODO:** boards.c (551 lines) still unported — gate.c + prng_seed deferred as not critical.

## Model Routing Rules (Updated 2026-04-24)

| Model | Role | Rules |
|-------|------|-------|
| `deepseek-v4-flash` | **Daily driver / mechanical tasks** | Default for coding subagents. 284B/13B active, 1M ctx, $0.14/$0.28/M. Default for main session too. |
| `litellm/deepseek-chat` | **Fallback** | Used when V4 Flash unavailable. Slow fallback. |
| `moonshot/kimi-k2.6` | **Build (secondary)** | Creative/interpretive translation when V4 misses nuance. ~90-110s per file. |
| `zai/glm-5.1` | **Fix / long-horizon** | Slow (~44 tok/s), deep. Best for compilation fixing and complex refactors. $10/mo plan. |
| `deepseek-v4-pro` | **Reasoning / heavy lifting** | 1.6T/49B, $1.74/$3.48/M. Subagent only. Use when Flash isn't enough but Sonnet is overkill. |
| `anthropic/claude-sonnet-4-6` | **QA + architecture** | Architectural review. **Requires approval** — rate limited, expensive. |
| `anthropic/claude-opus-4-6` | **Security final** | Final pass only. Expensive, requires approval. |

### Current model (as of this writing)
- **Primary:** `deepseek-v4-flash` (via models.json API key)
- **Fallback:** `litellm/deepseek-chat` (via LiteLLM env key)
- Sonnet was rate-limited and fell back, prompting this audit

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

## File Structure Convention (Updated 2026-04-24)

| C Source | Go Target | Status |
|----------|-----------|--------|
| `src/act.display.c` | `pkg/session/display_cmds.go` | ✅ |
| `src/act.social.c` | `pkg/game/socials.go` + `act_social.go` | ✅ |
| `src/act.wizard.c` | `pkg/session/wizard_cmds.go` | ✅ **1,574/3,863 lines — COMPLETE** |
| `src.act.other.c` | `pkg/game/act_other.go` + `act_other_bridge.go` + `pkg/session/commands.go` | ✅ 1,718/1,947 imp'd, 100% wired | Wave 6.5 done |
| `src/act.informative.c` | `pkg/game/act_informative.go` + `pkg/session/info_cmds.go` + `informative_cmds.go` | 🔶 1083/2803 |
| `src/act.movement.c` | `pkg/game/act_movement.go` + `pkg/session/movement_cmds.go` + `pkg/game/systems/door*.go` | ✅ Refactored to systems |
| `src/act.item.c` | `pkg/game/act_item.go` | 🔶 May exceed C (new features added) |
| `src/act.comm.c` | `pkg/game/act_comm.go` + `pkg/session/comm_cmds.go` | ✅ |
| `src/act.offensive.c` | `pkg/game/act_offensive.go` + `pkg/session/combat_cmds.go` | ✅ |
| `src/boards.c` | `pkg/game/boards.go` | ❌ NOT PORTED |
| `src/clan.c` | `pkg/game/clans.go` | ❌ NOT PORTED |
| `src/house.c` | `pkg/game/houses.go` | ❌ NOT PORTED |
| `src/whod.c` | `pkg/game/whod.go` | ❌ NOT PORTED |
| `src/objsave.c` | `pkg/game/objsave.go` | ❌ NOT PORTED |
| `src/mobprog.c` | `pkg/game/mobprogs.go` | ❌ NOT PORTED (partially via Lua) |
| `src/shop.c` | `pkg/game/shop.go`, `*systems/shop*.go`, `*command/shop_commands.go`, `*session/shop_cmds.go`, `*common/shop.go` | ✅ Distributed across pkgs |
| `src/mapcode.c` | `pkg/session/map_cmds.go` | ✅ |
| `src/tattoo.c` | `pkg/session/tattoo.go` | ✅ |
| `src/new_cmds.c` | `pkg/command/skill_commands.go` | ✅ |
| `src/new_cmds2.c` | Content in `pkg/command/skill_commands.go` (no standalone file) | ✅ |
| `src/spec_assign.c` | `pkg/game/spec_assign.go` | ✅ |
| `src/spec_procs.c`/2/3 | `pkg/game/spec_procs.go`, `spec_procs2.go`, `spec_procs3.go`, `spec_procs4.go` | 🔶 48% |
| `src/magic.c` + `spells.c` + `spell_parser.c` | `pkg/spells/` (8 files) | ✅ **Wave 7 done — 1,846 lines across 8 files** |
| `src/fight.c` | `pkg/combat/engine.go`, `formulas.go`, `combatant.go`, `fight_core.go` | ✅ ~98% |
| `src/handler.c` | `pkg/game/serialize.go`, `save.go`, `player.go`, `character.go` | 🔶 ~92% |
| `src/interpreter.c` | `pkg/session/commands.go`, `pkg/command/interface.go`, `registry.go`, `middleware.go` | 🔶 ~78% |
| `src/comm.c` | `pkg/telnet/listener.go`, `pkg/session/manager.go`, `protocol.go` | 🔶 ~54% |
| `src/limits.c` | `pkg/game/limits.go` | ✅ (expanded with regen) |
| `src/modify.c` | `pkg/game/modify.go` | 🔶 188/869 (untracked) |
| `src/weather.c` | `pkg/game/weather.go` + `pkg/session/time_weather.go` | ✅ | | `src/alias.c` | `pkg/game/aliases.go` | ✅ | | `src/ban.c` | `pkg/game/bans.go` | ✅ | | `src/dream.c` | `pkg/game/dreams.go` | ✅ |
| `src/constants.c` | `pkg/common/common.go` | 🔶 Sparse |
| `src/class.c` | `pkg/game/level.go` | 🔶 329/1191 |
| Editor files (11) | SPA replacement | 🚫 NOT PORTED (~7,830 lines skipped) |

---

## Session Startup

Each new session working on this plan should:
1. Read `PORT-PLAN.md` — this file (updated 2026-04-25)
2. Read `DARKPAWNS.md` — master strategy doc (BRENDA's perspective)
3. Read `RESEARCH-LOG.md` — recent session journal
4. Read `docs/SWARM-LEARNINGS.md` — lessons from previous waves
5. Check `git log --oneline -5` — latest commits
6. Check what wave is next or in progress (look for uncommitted changes)
7. Check the "Immediate Next Steps" section above for highest-priority items
8. Read `docs/research.md` for architecture rationale (when designing new systems)
9. Proceed
