# DP-1371 E2-attr implementation

Implements the approved design in #1733 from fresh `origin/main` at
`08d6811dada9ffd5a9d27dc609cfee27c826b320`. The explicit runtime effective
view is C's `aff_abils`; serialized `Player.Stats` stays `real_abils`.
No save field, RNG draw, oracle source/binary, governing document, manifest
status, expected-divergence pin, or seed change.

## Reader and writer audit (R5c/R5e)

[Per-site dispositions](../evidence/2026-09-30-e2-attr/go-site-dispositions.tsv)
classify the two approved raw inventories, including unrelated `Stats()`
methods and editor/prototype data. Source locations in that file are the
pre-change inventory locations. All typed effective consumers in combat,
spells, skills, regeneration, followers, shops, communication and stat
reports now receive the runtime view through their existing getters.

| Site family | Final disposition | C authority |
|---|---|---|
| PC and NPC typed getters | Cached effective STR/ADD/DEX/INT/WIS/CON/CHA; no read-time clamp | `src/utils.h` GET_* macros; `src/handler.c:337-372` |
| Level gains; carry limits | Read effective CON/WIS and STR/ADD/DEX under existing locks; capacity refresh after ability boundary | `src/class.c:607-699`; `src/utils.h` CAN_CARRY_* |
| Direct communication INT; observation state; Lua CHA; NPC alignment reports and consumers | Use effective getters | `src/act.comm.c:1370`; `src/scripts.c` char_to_table; `src/mobact.c:210-212`; `src/spec_procs.c:1465-1466,1504`; `src/act.wizard.c:748` |
| SQLite/JSON stats, orig_con, rolled-stat display, admin base-data view, parser/OLC prototypes | Retain real/base fields; no serialization change | `src/db.c:2441-2442,2587`; `src/class.c:380-495` |
| Creation, prototype spawn, guest roll, wizard reroll | Explicit unbounded base→effective copy | `src/class.c:495`; `src/db.c:1339`; `src/act.wizard.c:2103` |
| Wizard restore | Copy only inside the qualified immortal branch; restored PC 25 persists until next total | `src/act.wizard.c:1600-1612` |
| Wizard set | Total after C's max/current pools, alignment, abilities/ADD, AC, hit/damroll, and tattoo cases | `src/act.wizard.c:2735-2841,2999-3000,3027-3030` |
| PC equipment and world movement/extraction | Successful equip/unequip triggers total outside eq.mu; failed operations do not | `src/handler.c:748,792` |
| NPC equipment, spawning and Lua equipment paths | Total after actual map mutation; instance base stays unchanged | `src/handler.c:748,792` |
| Spell add/join/remove/expiry, Lua/wizard bulk clearing, follower charm removal | Rebuild from all modifiers, then bound; absent removal does not total. NPC records retain multiple locations of one spell | `src/handler.c:395,419,437,449-456`; `src/magic.c:431-457` |
| Tattoo | One table/implementation. Direct tattoo_af changes effective abilities without clipping; total adds current tattoo from base. Non-ability deltas retain existing ordering | `src/tattoo.c:104-195`; `src/handler.c:333-348` |
| Restore from saved record | Unbounded base copy plus tattoo ability modifiers; total only if restored spells/equipment cause that boundary | `src/db.c:2441-2481` |
| Remort and constitution seller | Preserve real_abils writes; total before advance_level / stunned position | `src/spec_procs2.c:942`; `src/spec_procs3.c:306-309` |
| Permanent death CON / combat callback | Decrement real CON; total afterward. Getter callback is used only by that real-CON decrement path | `src/fight.c:605-608` |
| Deprecated MasterAffects / engine.AffectTotal / GetStat / GetStrength | Existing base-mutation compatibility adapter retained separately; no use as the live repair. Remort continues unwinding that adapter before its final total | `pkg/engine/affect_helpers.go`; approved design exception |

The shared bounds function floors STR/DEX/INT/WIS/CON at zero, caps the
four non-STR abilities at PC 18/NPC 25, caps NPC STR at 25, converts PC STR
above 18 with **existing base ADD + excess×10**, saturating ADD at 100, and
leaves CHA unchanged. Alignment is bounded only at total. Rebuilding from
base avoids accumulating ADD or losing bonuses after removing clipped
modifiers. Effective-only tattoo application has the lifetime C specifies.

Tests assigning fixture base fields now explicitly perform the initial copy.
Tattoo assertions inspect effective getters and preserve base assertions.
Draw-order reference actors use the same effective INT/WIS as their actor
after equipping, so the tests retain their message/improve ordering predicate.

The writer audit was extended beyond the original reader inventories to all
`ActiveAffects` assignments, equipment map writes/deletes, and legacy clearing
helpers. This found `World.EquipChar`, Lua `Unaffect`, wizard unaffect,
`removeCharmAffect`, and both corpse/extraction paths. Successful removals
recompute; empty collections do not invent a total boundary. `stat file`
reconstructs its effective view after removing Crash_load equipment, as C
`store_to_char` does. NPC extraction recomputes after releasing the mob lock.

NPC affect records keep multiple locations and duplicate spell records, with
an unexported insertion sequence selecting C's newest matching record during
join (`src/handler.c:476-495`). Map iteration cannot choose which duration
or modifier survives. Removal retains flags supplied by remaining records;
record insertion/removal is protected by the mob lock. None of this state
is serialized or consumes randomness. Existing PC slice/callback order is
retained; it is not replaced with the deprecated master-affect engine.

## Validation and stop

The frozen implementation passed build, vet, all Go tests, lint, formatting,
fidelity depth, fidelity units, string census and diff checks. There are 37
retained assertion-based revert triples (0 → 1 → 0). Evidence is retained at
`~/Archives/darkpawns/oracle-runs/2026-09-30/dp-1371-p4-e2-attr-proofs/`.
The shipped vest 14425 vehicle now reports mortal DEX 18 before, while worn,
and after removal, matching C; fresh main reports 18/20/18. The complete
report's AC difference (C 110, Go 100 while worn) persists on fresh main and
is a proposed follow-up, along with the earlier report's STL and Poof fields.
The diagnostic scenario is retained as evidence, outside the shipped corpus.
`lua.bind-set-skill` remains blocked; this batch does not claim it proven.

The first implementation full census was CLEAN_AFTER_RECHECK (1039 rows,
1031 passed, 5 expected, no failures). A subsequent implementation full
census reported `shutdown-depth` FAIL, after that row had passed. Its C
transcript accepts mixed-case reboot and terminates, consistent with
`src/act.wizard.c:1074-1118`. Shutdown implementation was not changed.
A targeted fresh-main run passed, and the full fresh-main census also passed:
1039 rows, 1033 passed, 5 expected, no failures, one unstable, CLEAN.
Consequently the regression is not established as pre-existing.

Per the goal's explicit “Any PASS→FAIL is a stop,” work stops before commit,
claims census or PR. The current source includes a final deterministic NPC
join-order correction added after the failed implementation census; that
source still needs a full census and claims census once this stop is resolved.
No failing row was retried to obtain a green result. Retained run directories:

- `dp-1371-p4-e2-attr-full-after`
- `dp-1371-p4-e2-attr-full-final`
- `dp-1371-p4-e2-attr-shutdown-main`
- `dp-1371-p4-e2-attr-full-main`

All are under the same dated oracle-runs root. No census.log was read.

## Resumption after shutdown diagnosis

On 2026-10-01 Zach identified the shutdown failure as the pre-existing
case-insensitive reboot probe racing the comparison. PR #1736 removes that
terminating probe and corrects its depth row. The branch merged origin/main
550cf6da3 before resuming validation. The previous stop and its evidence are
retained above; fresh full and claims censuses will validate the final source.
