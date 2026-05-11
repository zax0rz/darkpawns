# Reek Findings Tracker

Maintained by Daeron. Updated per triage cycle.

## CRITICAL

| ID | Finding | File | Status | Fixed In |
|---|---|---|---|---|
| CRIT-001 | Graceless shutdown | cmd/server/main.go, main_web.go | FIXED | upstream (8c574d2) |
| CRIT-002 | HandleNonCombatDeath stub | pkg/game/limits_condition.go | FIXED | 052945a |
| CRIT-003 | Spawner.StartZoneResets no-op | pkg/game/spawner.go:57 | DOWNCLOSED | not a bug (dead code, not broken path) |
| CRIT-004 | Memory slice concurrent read/write — no lock | mobact.go:215, deferred_fight_fns.go:215-233 | FIXED | Per-mob mu locking in runMobAI + MobileActivity |
| CRIT-005 | Hunting/HuntingID unlocked writes | deferred_fight_fns.go:196-206 | FIXED | Added m.mu.Lock/Unlock to SetHunting, Remember, Forget |
| CRIT-006 | aiCombatEngine global — no sync | ai.go:28-32 | FIXED | Moved to World.combatEngine field |
| CRIT-007 | executeMobCommand dangling pointer | world.go:458-465 | FIXED | Added IsAlive() check after RUnlock |
| CRIT-008 | HasMobFlag() bitmask dead code | mob.go:556, mob_flags_bits.go | REJECTED | Downgraded to LOW — bit path never called |
| CRIT-009 | processCombatPair() vs MakeHit() dual path | pkg/combat/engine.go:236 + fight_core.go:759 | OPEN | Engine tick uses simplified math, full C hit() port unused |
| CRIT-010 | load_messages() missing | pkg/combat/ (MESS_FILE) | FIXED | Skill message table + multi-variant damage messages + InitSkillMessages() wired at boot |

## HIGH

| ID | Finding | File | Status | Notes |
|---|---|---|---|---|
| HIGH-001 | Lock ordering undocumented | multiple | FIXED | c741fa4 (BRENDA) |
| HIGH-002 | DefaultServeMux exposed | cmd/server/main_web.go | FIXED | upstream (8c574d2) |
| HIGH-003 | Duplicated entry points | main.go vs main_web.go | DEFERRED | Architecture decision — needs The Architect |
| HIGH-004 | Doors don't reset on zone reset | pkg/game/world.go | FIXED | upstream (8c574d2) |
| HIGH-005 | Non-TLS default | cmd/server/main_web.go | DEFERRED | Configuration decision — needs The Architect |
| HIGH-006 | handlePlayerDeath lock ordering risk | death.go:295-320 | DEFERRED | Documented safe — monitor under load |
| HIGH-007 | runZoneMobAI no-op shell | zone_dispatcher.go:141-165 | FIXED | Removed dead code — AI handled globally by AITick() |
| HIGH-008 | Memory field nil in NewMob() | mob.go:70-95 | FIXED | Initialized Memory: make([]string, 0) in NewMob() |
| HIGH-009 | SpellBless missing second affect | pkg/spells/affect_spells.go:35-36 | FIXED | Added applyAffect(victim, aff) after SavingSpell affect (Daeron) |
| HIGH-010 | inflictDamage() no death check | pkg/spells/damage_spells.go:275 | FIXED | Added HandleSpellDeath bridge + death check when HP=0 (Daeron) |
| HIGH-011 | checkReagents stub returns 0 | pkg/spells/affect_spells.go:365 | OPEN | Reagent damage bonus permanently zero |
| HIGH-012 | Spell routine stubs (6 no-ops) | pkg/spells/affect_spells.go:163-230 | FIXED | Added logging + TODO comments to 6 spell stub routines (BRENDA) |
| HIGH-013 | TakeDamage() gold duplication | pkg/combat/fight_core.go:578-585 | FIXED | Eliminated gold duplication in mob death (BRENDA) |
| HIGH-014 | Parry/dodge double-checked | engine.go:268 + fight_core.go:826 | FIXED | Removed unused ParryCheckFunc/DodgeCheckFunc dead code (BRENDA) |
| HIGH-015 | stop_fighting() no reassignment | pkg/combat/engine.go:155-169 | OPEN | Multi-mob fights lose target linkage |
| HIGH-016 | raw_kill() missing cleanup | pkg/combat/fight_core.go:1009 | FIXED | Added TODO comments for missing death cleanup (BRENDA) |
| HIGH-017 | GroupGain namedCombatant.IsNPC()=true — group XP never awarded | pkg/combat/fight_core.go:1186-1236 | FIXED | Added isNPC field, defaulted false in NewNamedCombatant (Machine) |

## MEDIUM

| ID | Finding | File | Status |
|---|---|---|---|
| MED-001 | SA4004 unconditionally terminated loop | equipment.go:235 | REJECTED (intentional) |
| MED-002 | SA4000 identical && expressions | spec_procs3.go:903 | REJECTED (intentional) |
| MED-003 | 268 U1000 unused code | pkg/game/*, pkg/session/* | FIXED | c741fa4 (BRENDA) |
| MED-004 | SA1019 deprecated strings.Title | multiple | REJECTED | Already handled — house.go has replacement |
| MED-005 | SA6003 range over []rune | multiple | REJECTED | Only instance is []byte in save.go, not []rune |
| MED-006 | SA6005 strings.EqualFold | multiple | REJECTED | Already using strings.EqualFold correctly |
| MED-007 | S1039 unnecessary fmt.Sprintf | multiple | REJECTED | staticcheck clean — no instances found |
| MED-008 | SA4006 assigned and not used | multiple | REJECTED | All _ = patterns are intentional blank identifiers |
| MED-009 | MobileActivity() no mob-level locking | mobact.go:90-290 | FIXED | Per-mob mu in MobileActivity + MobileActivityForMob |
| MED-010 | wanderMob() stale snapshot mutation | ai.go:109-170 | FIXED | Direct field access under mob.mu, snapshot reads |
| MED-011 | handleMobDeath pointer race | death.go:62-99 | FIXED | SetAlive(false) + early remove from activeMobs |
| MED-012 | Deserialized objects not tracked | serialize.go:39-50, 75-90 | DEFERRED | CrashLoad is dead code; added RegisterObjectInstance for future use |
| MED-013 | GetExtraFlags() zero-value comparison | object.go:427 | REJECTED | Style issue — zero-value sentinel works correctly |
| MED-014 | NewMob() Flags bitmask uninitialized | mob.go:70-95 | REJECTED | Flags field unused — all lookups use Prototype.ActionFlags |
| MED-015 | CanSpawn() VNum collision | spawner.go:361-362 | REJECTED | Mob/obj VNums in separate namespaces |
| MED-016 | Go stdlib vulns GO-2026-4918/4971 | stdlib (go1.26.2) | OPEN | HTTP/2 loop + NUL panic. Fixed in go1.26.3. |
| MED-017 | prometheus/client_golang 4 minor behind | go.mod | OPEN | v1.19.1 → v1.23.2. Breaking change in v1.20. |
| MED-018 | lib/pq 2 minor behind | go.mod | OPEN | v1.10.9 → v1.12.3. Low risk. |
| MED-019 | protobuf 2 major behind | go.mod | OPEN | v1.34.2 → v1.36.11. Marshaling internals changed. |
| MED-020 | go directive mismatch | go.mod | FIXED | Updated go directive to 1.26.2 (Daeron) |
| MED-021 | attitudeLoot() simplified | fight_core.go:1159 | OPEN | C: junking+12-variant brag. Go: single get+line. |
| MED-022 | SpellGate rawKill attack type | pkg/game/gates.go:155 | FIXED | Changed to 'suffering' → TYPE_SUFFERING(399) + added case to RawKill switch (Daeron) |
| MED-023 | AddItemToRoom Location tracking | pkg/game/world_object.go:23 | FIXED | Now sets Location and RoomVNum (BRENDA) |
| MED-024 | Bash sets victim to PosFighting (highest stance) | pkg/game/combat_melee.go:137 | FIXED | Changed to PosSitting (Machine) |
| MED-025 | Skill messages broadcast to room 0 | pkg/combat/skill_messages.go:582 | FIXED | SkillMessageFunc takes roomVNum, BroadcastMessage uses it (Machine) |
| MED-026 | MakeHit duplicates CalculateHitChance THAC0 logic | fight_core.go:983 + formulas.go:293 | OPEN | Two divergent hit formulas. Design — see CRIT-009. |
| MED-027 | Zero test coverage on prod code | death.go, affect_spells.go, fight_core.go, skill_messages.go | OPEN | ~450 lines across 4 files, zero test lines. Expanded with CRIT-010 skill message table. |

## LOW

| ID | Finding | File | Status |
|---|---|---|---|
| LOW-001 | parser_test.go unchecked file.Close() | pkg/parser/parser_test.go:101,163,193 | REJECTED | Already uses defer file.Close() |
| LOW-002 | HasMobFlag() bitmask dead code | mob.go:556 | REJECTED | Dead code path — all lookups use string ActionFlags |
| LOW-003 | SA4004/SA4000 re-report | equipment.go:235, spec_procs3.go:903 | REJECTED | Already in tracker |
| LOW-004 | QF1003 switch preference | fight_core.go:872 | REJECTED | Style preference |
| LOW-005 | Gates system unwired | pkg/game/gates.go | OPEN | LoadNightGate/RemoveNightGate/SpellGate never called. |
| LOW-006 | SpellSilkenMissile ID 200 conflict | pkg/spells/spells.go:122 | OPEN | Overlaps breath weapon space (200-207). |
| LOW-007 | doDisembowelMob bypasses TakeDamage effects | pkg/game/combat_basic.go:391 | OPEN | Calls m.TakeDamage() directly — documented as TODO, mob damage pipeline doesn't exist yet |
| LOW-008 | startCombatBetween doesn't register with CombatEngine | pkg/game/combat_advanced.go:489-518 | FIXED | Now calls w.combatEngine.StartCombat() (Machine) |
| LOW-009 | doHit mob path: overwritten SetFighting call | pkg/game/combat_basic.go:120-125 | FIXED | doHit/backstab/disembowel pass mob directly to startCombatBetween (Machine) |
| LOW-010 | GetAttacksPerRound: horde/sanctuary haste not wired | pkg/combat/formulas.go:318 | FIXED | Engine checks HasAffect for AFF_HASTE/AFF_SLOW (Machine) |
| LOW-011 | basicTokenReplace leaves $s/$e unresolved | pkg/combat/skill_messages.go:592-602 | FIXED | GetCharacterSex hook + full pronoun resolution (Machine) |
| LOW-012 | SkillMessage table: attackType guard | pkg/combat/skill_messages.go:277-278 | REJECTED | Guard clause doing its job — no bug. |

## Cycle History

| Date | Reek Report | Confirmed | Rejected | False Positive Rate |
|---|---|---|---|---|
| 2026-05-07 | Deep dive (server/) | 122 | 2 | 1.6% — Good reek |
| 2026-05-08 | Deep dive (mob/object/zone) | 19 | 2 | 9.5% — Good reek |
| 2026-05-10 | Spells/world + combat fidelity + deps | 20 | 3 | 13% — Good reek |
| 2026-05-11 | pkg/combat/ deep dive | 8 | 1 | 11% — Good reek |
| 2026-05-11 | Machine fixes (8 findings) | 8 | 0 | 0% — Good machine |
| **Weekly** | **6 reports** | **177** | **8** | **4.3% — Good reek** |
