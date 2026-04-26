# Dark Pawns — Full Port Completion Plan

**Created:** 2026-04-26  
**Goal:** Zero TODOs, zero stubs. Complete C-to-Go port ready for production.  
**Post-completion:** Sonnet QA review → Opus security review

---

## Current State

- **Go codebase:** 72,644 lines (non-test, non-bak)
- **C codebase:** 68,823 lines across 63 files
- **Remaining TODOs:** 73
- **Commits today:** 3 (equipment regen, veteran/kill-counter/autosplit)

---

## TIER 1 — Port Mechanics (C has it, Go must have it)

### Batch 1: Core Game Loop Gaps (~800 lines)
- [ ] **KK_JIN/KK_ZHEN** — `affected_by_spell` infrastructure + wire into HitGain/MoveGain
  - C ref: `src/limits.c:144-146` (KK_JIN), `src/limits.c:212-214` (KK_ZHEN)
  - Skills: KK_JIN=162, KK_ZHEN=165 (src/spells.h:210,213)
  - Need: spell-affect tracking on Player (beyond bitmask flags)
  - TODOs: `pkg/game/limits.go` lines 227, 306

- [ ] **PRF_INACTIVE** check in PointUpdate
  - C ref: `src/limits.c:460` — skips condition decay + regen for inactive players
  - TODO: `pkg/game/limits.go` line 453

- [ ] **Dream system** — port `src/dream.c`, wire into PointUpdate
  - C ref: `src/dream.c` (~125 lines), called at `src/limits.c:476`
  - Already have `pkg/game/dreams.go` with AFF_DREAM_BIT check
  - TODO: `pkg/game/limits.go` line 466

- [ ] **AFF_FLESH_ALTER** in level-up (GainExp + GainExpRegardless)
  - C ref: `src/class.c` — prevents level gain if AFF_FLESH_ALTER active
  - TODOs: `pkg/game/limits.go` lines 834, 869

- [ ] **Full idle handling** — void pull, disconnect, timer tracking
  - C ref: `src/limits.c:654-686` — idle timer, char_to_void, close_socket
  - TODO: `pkg/game/limits.go` line 904

### Batch 2: Player Systems (~1000 lines)
- [ ] **Gender field + pronouns**
  - C ref: Player sex is in char_data, used throughout act.comm.c/act.movement.c
  - TODOs: `pkg/session/char_creation.go:100`, `pkg/session/movement_cmds.go:417`

- [ ] **Character creation full state machine**
  - C ref: `src/interpreter.c` — nanny() function, multi-stage character creation
  - TODOs: `pkg/session/char_creation.go` lines 23, 103

- [ ] **Tattoo mob spawn + follow**
  - C ref: `src/tattoo.c` — mob creation + charm effect
  - TODO: `pkg/session/tattoo.go:66`

- [ ] **Ghost ship / night gate time events**
  - C ref: `src/time_weather.c` — time-based world events
  - TODO: `pkg/session/time_weather.go:316`

### Batch 3: Scripting Engine (~600 lines)
- [ ] **Zone broadcast** — send message to all players in a zone
  - C ref: `src/comm.c` send_to_zone
  - TODOs: `pkg/scripting/engine.go` lines 404, 2233

- [ ] **Mob command execution** — mobs can execute game commands
  - C ref: `src/scripts.c` — script command interpreter
  - TODO: `pkg/scripting/engine.go:982`

- [ ] **Soul leech heal** — caster heals dam/3 on soul leech
  - C ref: spell effect in magic.c
  - TODO: `pkg/scripting/engine.go:1398`

- [ ] **Move points restoration**
  - TODO: `pkg/scripting/engine.go:1303`

- [ ] **Death handling in scripts** — handle HP=0 from scripted damage
  - TODO: `pkg/scripting/engine.go:1509`

- [ ] **Object placement** — add to room/char inventory based on location type
  - TODO: `pkg/scripting/engine.go:1028`

- [ ] **Full can_see implementation** — PLR_INVISIBLE, room DARK, AFF_BLIND
  - TODO: `pkg/scripting/engine.go:1964`

- [ ] **Global channel broadcast** — send to all online players
  - TODO: `pkg/scripting/engine.go:915`

### Batch 4: Wizard Commands (~800 lines)
- [ ] **cmdLoad** — load mob/object from database
  - C ref: `src/act.wizard.c` do_load()
  - TODO: `pkg/session/wizard_cmds.go:103`

- [ ] **cmdPurge** — remove mobs/objects from room
  - C ref: `src/act.wizard.c` do_purge()
  - TODO: `pkg/session/wizard_cmds.go:121`

- [ ] **Character switch** — immortal switches into another char
  - C ref: `src/act.wizard.c` do_switch()
  - TODO: `pkg/session/wizard_cmds.go:311`

- [ ] **Character return** — return from switched char
  - C ref: `src/act.wizard.c` do_return()
  - TODO: `pkg/session/wizard_cmds.go:332`

- [ ] **cmdReload** — reload game config/area data
  - C ref: `src/act.wizard.c` do_reboot()
  - TODO: `pkg/session/wizard_cmds.go:585,593`

- [ ] **Object stat** — display object prototype info
  - C ref: `src/act.wizard.c` do_stat_object()
  - TODO: `pkg/session/wizard_cmds.go:671`

- [ ] **Wizlist / last login**
  - TODOs: `pkg/session/wizard_cmds.go:885`, `pkg/game/limits.go:730`

- [ ] **Combat-stop broadcast** — send freeze to room
  - TODO: `pkg/session/wizard_cmds.go:1021`

### Batch 5: Spell System Cleanup (~400 lines)
- [ ] **Damage spells default case** — Cast() should handle all spell types
  - TODO: `pkg/spells/spells.go` lines 8, 40, 190

- [ ] **Room iteration for zone messages** — iterate world[room].people
  - TODOs: `pkg/spells/say_spell.go:327`, `pkg/spells/damage_spells.go:316`

- [ ] **Death attack type constants**
  - TODO: `pkg/game/death.go:368`

- [ ] **Combat backstab function** — actual implementation not TODO stub
  - TODO: `pkg/combat/formulas.go:565`

---

## TIER 2 — Go Modernization (~800 lines)

- [ ] **Database error handling + retry** — `pkg/optimization/database.go:244`, `pkg/optimization/websocket.go:209`
- [ ] **Board room echo** — `pkg/game/boards.go` lines 335, 489
- [ ] **House storage** — `pkg/game/houses.go` lines 64, 69, 333, 494
- [ ] **Clan string_write** — `pkg/game/clans.go:1202`
- [ ] **Weather graph penalty** — `pkg/game/graph.go:179`
- [ ] **Zone mob AI dispatcher** — `pkg/game/zone_dispatcher.go:126`
- [ ] **World target resolution** — `pkg/game/world.go` lines 1019, 1341
- [ ] **World death handling** — `pkg/game/world.go:1007`
- [ ] **Death reconnection/resurrection** — `pkg/game/death.go` lines 14, 311
- [ ] **Other settings bug/typo/idea write** — `pkg/game/other_settings.go:104`

---

## TIER 3 — Reviews (post-completion)

- [ ] **Sonnet QA review** — fidelity to C source, correctness, edge cases
  - Split into 3-4 passes pairing Go subsystems with corresponding C source:
    1. Combat + death + XP (formulas.go, fight_core.go, death.go ↔ fight.c, class.c)
    2. Regen + limits + level-up (limits.go, level.go ↔ limits.c, class.c)
    3. Spells + affects (spells/, affect.go ↔ magic.c, spells.c, handler.c)
    4. Equipment + objects + world (equipment.go, world.go ↔ handler.c, db.c)

- [ ] **Opus security review** — input validation, overflow, race conditions, auth
  - Focus: combat input, death handling, group gain, equipment manipulation
  - Session handling, wizard command authorization
  - Script injection via Lua engine

---

## Post-Review: Production Readiness

- [ ] Address all review findings
- [ ] Remove .bak files
- [ ] Final `go vet ./...` + `go test ./...`
- [ ] Update NEXT_SESSION.md with "PORT COMPLETE" status
- [ ] GBrain page update: darkpawns/port-status

---

## Execution Strategy

- Batch 1-2: Direct implementation (small, interconnected)
- Batch 3-4: Subagent dispatch (larger, self-contained)
- Batch 5: Direct (spell cleanup)
- Tier 2: Subagent (infrastructure polish)
- Tier 3: Parallel Opus/Sonnet review subagents
- Build + test after every commit

---

## Session Log

### Session 1 — 2026-04-26 5:16 PM EDT
**Commits:** fc613b7, 11c1558, aaac0b3, 361d4aa, be13e59
**TODOs killed:** 11 (73 → 62, verified 45 remaining after session)

Completed:
- ✅ Equipment regen bonuses (APPLY_MANA/HIT/MOVE_REGEN)
- ✅ is_veteran wired to PlayedDuration + Kills
- ✅ Kill counter rewards (CounterProcs with C fallthrough bug)
- ✅ Auto-split (PRF_AUTOSPLIT gold division)
- ✅ affected_by_spell infrastructure (SpellID on Affect, HasSpellAffect)
- ✅ KK_JIN/KK_ZHEN skill bonuses in regen
- ✅ PRF_INACTIVE check in PointUpdate
- ✅ AFF_FLESH_ALTER handling in level-up
- ✅ CheckIdling full implementation (void pull + disconnect)
- ✅ Dream system wired into PointUpdate (PlayerDreamAdapter)
- ✅ CheckAutowiz (log instead of shell exec)
- ✅ Character creation state machine (sex→race→class→confirm)
- ✅ Gender pronouns in movement
- ✅ Tattoo skull mob spawn + charm + follow

### Session 2 — 2026-04-26 5:48 PM EDT
**Commits:** dfa6c7c, 0ad9293
**TODOs killed:** 19 (45 → 26)

Completed:
- ✅ Batch 3: All 8 scripting engine TODOs (SendToAll, SendToZone, can_see, ExecuteMobCommand, etc.)
- ✅ Batch 4: All 9 wizard command TODOs (cmdLoad, cmdPurge, cmdSwitch, cmdReturn, cmdReload, obj stat, wizlist, combat-stop, idlist)
- ✅ Stale cleanup: backstab TODO (already implemented), SPELL_PARALYSE verified

Subagent learnings:
- GLM-5.1 crashes on large files (wizard_cmds.go 1574 + act.wizard.c 3863 = too much)
- Batch 4 done manually after 2 failed subagent attempts
- DeepSeek V4 Pro model ID is `deepseek-v4/deepseek-v4-pro`, not `zai/deepseek-v4-pro`
- Unexported→exported method renames cascade across files
- Parser.World uses `Objs` not `Objects`

Next up: Batch 5 (spell system — Cast() routing, room iteration stubs), then Tier 2 (Go modernization).
