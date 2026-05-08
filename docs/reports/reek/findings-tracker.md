# Reek Findings Tracker

Maintained by Daeron. Updated per triage cycle.

## CRITICAL

| ID | Finding | File | Status | Fixed In |
|---|---|---|---|---|
| CRIT-001 | Graceless shutdown | cmd/server/main.go, main_web.go | FIXED | upstream (8c574d2) |
| CRIT-002 | HandleNonCombatDeath stub | pkg/game/limits_condition.go | FIXED | 052945a |
| CRIT-003 | Spawner.StartZoneResets no-op | pkg/game/spawner.go:57 | DOWNCLOSED | not a bug (dead code, not broken path) |
| CRIT-004 | Memory slice concurrent read/write — no lock | mobact.go:215, deferred_fight_fns.go:215-233 | OPEN | AI tick reads, combat writes, zero sync |
| CRIT-005 | Hunting/HuntingID unlocked writes | deferred_fight_fns.go:196-206 | FIXED | Added m.mu.Lock/Unlock to SetHunting, Remember, Forget |
| CRIT-006 | aiCombatEngine global — no sync | ai.go:28-32 | OPEN | Package-level var, no mutex/atomic |
| CRIT-007 | executeMobCommand dangling pointer | world.go:458-465 | OPEN | RUnlock then use pointer |
| CRIT-008 | HasMobFlag() bitmask dead code | mob.go:556, mob_flags_bits.go | REJECTED | Downgraded to LOW — bit path never called |

## HIGH

| ID | Finding | File | Status | Notes |
|---|---|---|---|---|
| HIGH-001 | Lock ordering undocumented | multiple | FIXED | c741fa4 (BRENDA) |
| HIGH-002 | DefaultServeMux exposed | cmd/server/main_web.go | FIXED | upstream (8c574d2) |
| HIGH-003 | Duplicated entry points | main.go vs main_web.go | OPEN | Architecture refactor |
| HIGH-004 | Doors don't reset on zone reset | pkg/game/world.go | FIXED | upstream (8c574d2) |
| HIGH-005 | Non-TLS default | cmd/server/main_web.go | OPEN | Configuration decision |
| HIGH-006 | handlePlayerDeath lock ordering risk | death.go:295-320 | OPEN | player.mu → Inventory/Equipment sub-locks, documented safe today |
| HIGH-007 | runZoneMobAI no-op shell | zone_dispatcher.go:141-165 | OPEN | Full function body is comments, mobs stand there |
| HIGH-008 | Memory field nil in NewMob() | mob.go:70-95 | FIXED | Initialized Memory: make([]string, 0) in NewMob() |

## MEDIUM

| ID | Finding | File | Status |
|---|---|---|---|
| MED-001 | SA4004 unconditionally terminated loop | equipment.go:235 | REJECTED (intentional) |
| MED-002 | SA4000 identical && expressions | spec_procs3.go:903 | REJECTED (intentional) |
| MED-003 | 268 U1000 unused code | pkg/game/*, pkg/session/* | FIXED | c741fa4 (BRENDA) |
| MED-004 | SA1019 deprecated strings.Title | multiple | OPEN |
| MED-005 | SA6003 range over []rune | multiple | OPEN |
| MED-006 | SA6005 strings.EqualFold | multiple | OPEN |
| MED-007 | S1039 unnecessary fmt.Sprintf | multiple | OPEN |
| MED-008 | SA4006 assigned and not used | multiple | OPEN |
| MED-009 | MobileActivity() no mob-level locking | mobact.go:90-290 | OPEN | State changes between accessor calls |
| MED-010 | wanderMob() stale snapshot mutation | ai.go:109-170 | OPEN | Reads snapshot, mutates via world methods |
| MED-011 | handleMobDeath pointer race | death.go:62-99 | OPEN | Two combat rounds could target same mob |
| MED-012 | Deserialized objects not tracked | serialize.go:39-50, 75-90 | OPEN | Player save items bypass spawner tracking |
| MED-013 | GetExtraFlags() zero-value comparison | object.go:427 | OPEN | Brittle but functionally correct |
| MED-014 | NewMob() Flags bitmask uninitialized | mob.go:70-95 | OPEN | Same root cause as CRIT-008, dead code path |
| MED-015 | CanSpawn() VNum collision | spawner.go:361-362 | REJECTED | Mob/obj VNums in separate namespaces |

## LOW

| ID | Finding | File | Status |
|---|---|---|---|
| LOW-001 | parser_test.go unchecked file.Close() | pkg/parser/parser_test.go:101,163,193 | OPEN | Test code only |
| LOW-002 | HasMobFlag() bitmask dead code | mob.go:556 | OPEN | Reclassified from CRIT-008 |
| LOW-003 | SA4004/SA4000 re-report | equipment.go:235, spec_procs3.go:903 | REJECTED | Already in tracker |
| LOW-004 | QF1003 switch preference | fight_core.go:872 | REJECTED | Style preference |

## Cycle History

| Date | Reek Report | Confirmed | Rejected | False Positive Rate |
|---|---|---|---|---|
| 2026-05-07 | Deep dive (server/) | 122 | 2 | 1.6% — Good reek |
| 2026-05-08 | Deep dive (mob/object/zone) | 19 | 2 | 9.5% — Good reek |
