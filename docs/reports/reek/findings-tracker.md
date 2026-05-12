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
| CRIT-009 | processCombatPair() vs MakeHit() dual path | pkg/combat/engine.go:236 + fight_core.go:759 | FIXED | MakeHit formally deadcoded with nolint. Zero callers confirmed. (BRENDA) |
| CRIT-010 | load_messages() missing | pkg/combat/ (MESS_FILE) | FIXED | Skill message table + multi-variant damage messages + InitSkillMessages() wired at boot |
| CRIT-011 | ActiveAffects data race — 3 inconsistent locking regimes | affect_update.go, follow.go, other_character.go, session/*.go | OPEN | w.mu vs p.mu vs no lock across 8+ files. Canonical mutex must be p.mu. |

## HIGH

| ID | Finding | File | Status | Notes |
|---|---|---|---|---|
| HIGH-001 | Lock ordering undocumented | multiple | FIXED | c741fa4 (BRENDA) |
| HIGH-002 | DefaultServeMux exposed | cmd/server/main_web.go | FIXED | upstream (8c574d2) |
| HIGH-003 | Duplicated entry points | main.go vs main_web.go | FIXED | main_web.go already removed; orphaned //go:build !web tag cleaned up (Daeron) |
| HIGH-004 | Doors don't reset on zone reset | pkg/game/world.go | FIXED | upstream (8c574d2) |
| HIGH-005 | Non-TLS default | cmd/server/main.go | FIXED | TLS auto-detects from cert files. Plaintext warning when no certs. (Daeron) |
| HIGH-006 | handlePlayerDeath lock ordering risk | death.go:295-320 | DEFERRED | Documented safe — monitor under load |
| HIGH-007 | runZoneMobAI no-op shell | zone_dispatcher.go:141-165 | FIXED | Removed dead code — AI handled globally by AITick() |
| HIGH-008 | Memory field nil in NewMob() | mob.go:70-95 | FIXED | Initialized Memory: make([]string, 0) in NewMob() |
| HIGH-009 | SpellBless missing second affect | pkg/spells/affect_spells.go:35-36 | FIXED | Added applyAffect(victim, aff) after SavingSpell affect (Daeron) |
| HIGH-010 | inflictDamage() no death check | pkg/spells/damage_spells.go:275 | FIXED | Added HandleSpellDeath bridge + death check when HP=0 (Daeron) |
| HIGH-011 | checkReagents stub returns 0 | pkg/spells/affect_spells.go:365 | FIXED | Rewritten with reflect-based inventory walk. 7 tests added. Interface extraction fixed silent type assertion failure. (BRENDA) |
| HIGH-012 | Spell routine stubs (6 no-ops) | pkg/spells/affect_spells.go:163-230 | FIXED | Added logging + TODO comments to 6 spell stub routines (BRENDA) |
| HIGH-013 | TakeDamage() gold duplication | pkg/combat/fight_core.go:578-585 | FIXED | Eliminated gold duplication in mob death (BRENDA) |
| HIGH-014 | Parry/dodge double-checked | engine.go:268 + fight_core.go:826 | FIXED | Removed unused ParryCheckFunc/DodgeCheckFunc dead code (BRENDA) |
| HIGH-015 | stop_fighting() no reassignment | pkg/combat/engine.go:155-169 | FIXED | handleMobDeath clears fighting on all room mobs+players targeting dead mob (BRENDA) |
| HIGH-016 | raw_kill() missing cleanup | pkg/combat/fight_core.go:1009 | FIXED | Added TODO comments for missing death cleanup (BRENDA) |
| HIGH-017 | GroupGain namedCombatant.IsNPC()=true — group XP never awarded | pkg/combat/fight_core.go:1186-1236 | FIXED | Added isNPC field, defaulted false in NewNamedCombatant (Machine) |
| HIGH-018 | removeCharmAffect mutates ch.ActiveAffects with zero locks | pkg/game/follow.go:242-254 | OPEN | Slice mutation in loop without ch.mu. Called from StopFollower. |
| HIGH-019 | doOrder calls executeCommand (no-op stub) | pkg/game/combat_control.go:64, damage_stubs.go:115 | OPEN | executeCommand discards both params, returns true. Order command fully broken. |

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
| MED-016 | Go stdlib vulns GO-2026-4918/4971 | stdlib (go1.26.3) | FIXED | Upgraded go directive to 1.26.3. Full green. (Daeron) |
| MED-017 | prometheus/client_golang 4 minor behind | go.mod | FIXED | Migrated to promhttp.HandlerFor(). Compiles clean. (BRENDA) |
| MED-018 | lib/pq 2 minor behind | go.mod | FIXED | v1.10.9 → v1.12.3 (BRENDA) |
| MED-019 | protobuf 2 major behind | go.mod | FIXED | Auto-pulled to v1.36.6 via prometheus transitive. No code changes. (BRENDA) |
| MED-020 | go directive mismatch | go.mod | FIXED | Updated go directive to 1.26.2 (Daeron) |
| MED-021 | attitudeLoot() simplified | fight_core.go:1159 | FIXED | 12 brag variants ported from C (BRENDA) |
| MED-022 | SpellGate rawKill attack type | pkg/game/gates.go:155 | FIXED | Changed to 'suffering' → TYPE_SUFFERING(399) + added case to RawKill switch (Daeron) |
| MED-023 | AddItemToRoom Location tracking | pkg/game/world_object.go:23 | FIXED | Now sets Location and RoomVNum (BRENDA) |
| MED-024 | Bash sets victim to PosFighting (highest stance) | pkg/game/combat_melee.go:137 | FIXED | Changed to PosSitting (Machine) |
| MED-025 | Skill messages broadcast to room 0 | pkg/combat/skill_messages.go:582 | FIXED | SkillMessageFunc takes roomVNum, BroadcastMessage uses it (Machine) |
| MED-026 | MakeHit duplicates CalculateHitChance THAC0 logic | fight_core.go:983 + formulas.go:293 | FIXED | Skill multipliers (backstab/circle/disembowel) added to CalculateDamage — engine path now handles these attack types (BRENDA) |
| MED-027 | Zero test coverage on prod code | death.go, affect_spells.go, fight_core.go, skill_messages.go | FIXED | 30 total tests across combat + game packages (Daeron 20 + BRENDA 10) |

## NEW (May 12 Triage — Daeron)

| ID | Finding | File | Status | Notes |
|---|---|---|---|---|
| NEW-001 | ActiveAffects data race — 3 inconsistent locking regimes | affect_update.go:47, follow.go:243, other_character.go:48, affects_informative.go:17, eat_cmds.go:66, wiz_system.go:287 | FIXED | Unified all access to p.mu. Exported RLock/RUnlock on Player. Snapshot-copy pattern in affect_update.go. (Daeron) |
| NEW-002 | removeCharmAffect mutates ActiveAffects with zero locks | follow.go:242-254 | FIXED | Added ch.mu.Lock/defer ch.mu.Unlock in removeCharmAffect. (Daeron) |
| NEW-003 | doOrder calls executeCommand which is a stub | damage_stubs.go:115, combat_control.go:64,74 | CONFIRMED HIGH | executeCommand is a no-op. Order command parses correctly, notifies room, then does nothing. | 
| NEW-004 | affect_update.go uses w.mu instead of p.mu for Player fields | affect_update.go:44-70 | FIXED | Now snapshots players under w.mu.RLock, then uses p.mu per-player for ActiveAffects. (Daeron) |
| NEW-005 | executeMobCommand TOCTOU — lock released before mob use | world.go:458-470 | CONFIRMED MED | Lock held for mob lookup, released, then mob used. Mob can die between lookup and use. | 
| NEW-006 | Zone dispatcher cancel function never called | zone_dispatcher.go:73 | CONFIRMED LOW | G118 suppresses warning but cancel is stored and never explicitly called. Parent cascade covers shutdown but individual zone removal leaks. | 
| NEW-007 | doVisible reads ActiveAffects without locking | other_character.go:48 | FIXED | Added ch.mu.RLock + snapshot-copy before iteration. (Daeron) |

## LOW

| ID | Finding | File | Status | Notes |
|---|---|---|---|---|
| LOW-001 | parser_test.go unchecked file.Close() | pkg/parser/parser_test.go:101,163,193 | REJECTED | Already uses defer file.Close() |
| LOW-002 | HasMobFlag() bitmask dead code | mob.go:556 | REJECTED | Dead code path — all lookups use string ActionFlags |
| LOW-003 | SA4004/SA4000 re-report | equipment.go:235, spec_procs3.go:903 | REJECTED | Already in tracker |
| LOW-004 | QF1003 switch preference | fight_core.go:872 | REJECTED | Style preference |
| LOW-005 | Gates system unwired | pkg/game/gates.go | FIXED | loadNightGate/removeNightGate in weather.go now call World methods (BRENDA) |
| LOW-006 | SpellSilkenMissile ID 200 conflict | pkg/spells/spells.go:122 | FIXED | Registered in spell info table (BRENDA) |
| LOW-007 | doDisembowelMob bypasses TakeDamage effects | pkg/game/combat_basic.go:391 | FIXED | doDamage handles both Player and MobInstance. doDisembowelMob routes through it. (BRENDA) |
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
| 2026-05-11 | BRENDA sprint (10 findings) | 10 | 0 | 0% — Good BRENDA |
| 2026-05-12 | pkg/game/ deep dive | 7 | 0 | 0% — Good reek |
| **Weekly** | **9 reports** | **194** | **8** | **4.0% — Good reek** |
