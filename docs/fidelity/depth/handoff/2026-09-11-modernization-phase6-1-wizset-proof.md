# Modernization Phase 6.1 — `wiz_set` proof closure

Date: 2026-09-11
Base: `origin/main` at `b1ff6a62c`
Scope: the six direct binary table entries and the `nohassle`/`frozen`
authority/self-target exceptions named by the 2026-09-11 Phase 6.1/6.2 audit.

## Case matrix written before implementation

The live path is C `do_set` (`src/act.wizard.c:2523-3069`) from the `set`
interpreter row (`src/interpreter.c:682`), through target resolution, the
generic target-level check, field-table level/PC-NPC check, binary parsing,
the per-field branch, and the final actor acknowledgement. Go is
`ExecuteCommand` → registered `set` wrapper → `cmdSetText`
(`pkg/session/commands.go:547-563`; `pkg/session/wiz_set.go:91-192`) →
`findSetField`/field checks → `applySetField`
(`pkg/session/wiz_set.go:198-243,423-442`) → the direct table setter or the
explicit exception switch. C sends only the final acknowledgement to the
actor for these cases; no target or observer transcript is expected.

| field | C site / Go path | actor, target, and restrictions | on/off state and bytes | existing proof / missing case |
|---|---|---|---|---|
| `invstart` | C table `act.wizard.c:2543-2545`, assignment `:2724-2726`; Go `setFields`/`setBinaryFieldTable` and `setCPlayerFlag` at `wiz_set.go:21-52,394-421` | `LVL_GOD` (34), PC only; target is a player; generic target-level check still applies for non-self targets | C `PLR_INVSTART` bit 14; both transitions; actor gets `Invstart ON/OFF for <target>.\r\n`; no target/observer bytes | `set.binary-field` proved generic parsing/ack only; no named live `invstart` state case |
| `roomflag` | C table `act.wizard.c:2579-2580`, assignment `:2913-2915`; Go direct table `wiz_set.go:399` → preference setter | `LVL_GRGOD` (38), PC only; target is a player | C `PRF_ROOMFLAGS` bit 21; both transitions; actor gets `Roomflag ON/OFF for <target>.\r\n`; no target/observer bytes | No named live `roomflag` `do_set` case; `roomflags` is a different self-only `do_gen_tog` surface |
| `siteok` | C table `act.wizard.c:2579-2581`, assignment `:2916-2918`; Go direct table `wiz_set.go:400` → player setter | `LVL_GRGOD` (38), PC only; target is a player | C `PLR_SITEOK` bit 7; both transitions; actor gets `Siteok ON/OFF for <target>.\r\n`; no target/observer bytes | No named live `siteok` `do_set` state case |
| `deleted` | C table `act.wizard.c:2580-2582`, assignment and C clan/crash/alias side branch `:2919-2933`; Go direct table `wiz_set.go:401` → player setter | `LVL_GRGOD` (38), PC only; target is a disposable live player; no clan/file target is used | C `PLR_DELETED` bit 10; both transitions; actor gets `Deleted ON/OFF for <target>.\r\n`; C's live visible bytes are only the ack; Go must not invent deletion or persistence side effects | No named live `deleted` case; exact bit and unchanged neighbors are missing |
| `nowizlist` | C table `act.wizard.c:2582-2584`, assignment `:2934-2943`; Go direct table `wiz_set.go:402` → player setter | `LVL_GOD` (34), PC only; target is a player | C `PLR_NOWIZLIST` bit 12; both transitions; actor gets `Nowizlist ON/OFF for <target>.\r\n`; no target/observer bytes | No named live `nowizlist` `do_set` state case; `nowiz` is a different self-only `do_gen_tog` surface |
| `quest` | C table `act.wizard.c:2583-2585`, assignment `:2944-2946`; Go direct table `wiz_set.go:403` → preference setter | `LVL_GOD` (34), PC only; target is a player | C `PRF_QUEST` bit 9; both transitions; actor gets `Quest ON/OFF for <target>.\r\n`; no target/observer bytes | No named live `quest` `do_set` state case; quest channel coverage does not prove this setter |
| `nohassle` | C table `act.wizard.c:2567-2570`, explicit branch `:2850-2856`; Go remains explicit at `wiz_set.go:560-565` | `LVL_GRGOD` (38), PC only; after generic checks, non-Implementor may self-target, but a non-Implementor targeting another player reaches C's extra `LVL_IMPL` rejection | C `PRF_NOHASSLE` bit 8; self on/off succeeds for level 38; other-player on/off rejects with `You aren't godly enough for that!\r\n`; successful actor ack is `Nohassle ON/OFF for <target>.\r\n`; no target/observer bytes | Existing `set-extended-depth` exercises the field but does not isolate self success versus non-Implementor other-target rejection |
| `frozen` | C table `act.wizard.c:2568-2571`, explicit branch `:2857-2863`; Go remains explicit at `wiz_set.go:565-570` | `LVL_FREEZE` = `LVL_GRGOD` (38), PC only; a level-38 actor can target a lower-level player; self target is rejected after field checks | C `PLR_FROZEN` bit 2; other-player on/off succeeds; self on/off rejects with `Better not -- could be a long winter!\r\n`; successful actor ack is `Frozen ON/OFF for <target>.\r\n`; no target/observer bytes | Existing `set-extended-depth` exercises `frozen off` only as a broad sequence; it does not isolate successful other-target transitions or the self refusal |

Shared checks: C's generic safety runs before field lookup (`act.wizard.c:2674-2681`),
then the field level and PC/NPC checks (`:2682-2700`), then exact lower-case
`on|yes|off|no` parsing (`:2701-2712`). The Go path keeps those checks before
the direct table or explicit exception. Unit expectations below use the C bit
numbers and storage family above, not the Go table under test.

## Proof additions

The direct vehicle covers each named field with on and off commands against a
disposable live player. The exception vehicle first lowers the Implementor to
level 38, then separately reaches `nohassle` self success, `nohassle` other
target authority rejection, `frozen` other-target success, and `frozen`
self-target refusal. `loadroom` remains explicit and outside this milestone;
equipment maps, Phase 6.2 inventory work, and Phase 6.3 remain outside scope.

## Final disposition

To be completed after the proof runs: record per-field oracle and state-test
results, the focused scenario/seed tally, full-corpus census, exact tested
commit, changed files, and any residual gaps. This handoff does not claim
whole-Phase 6.1 completion or close the equipment-map debt.
