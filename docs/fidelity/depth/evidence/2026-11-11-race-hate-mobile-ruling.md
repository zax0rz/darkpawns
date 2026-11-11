# Race-hate mobile dispatch ruling request — 2026-11-11

## C path and fixture boundary

The existing case asks a player to wield stock dagger #5200, then expects a
race-18 mobile's pulse to select the wearer through `APPLY_RACE_HATE`.
`src/handler.c:205-238` adds/removes race-hate slots while equipment changes;
`src/mobact.c:236-258` scans room occupants for a matching slot and dispatches
the attack for any non-shopkeeper mobile. These C lines were grepped in this
session (R5g).

The original vehicle names mob #8552, which exists as Race 18 in
`lib/world/mob/85.mob:217-232`, and `lib/world/zon/85.zon:9` references it in an
M reset. But `src/db.c:549-594,653-665` opens the mob index and parses the files
it names; `lib/world/mob/index` lists `80.mob` followed by `91.mob` and has no
`85.mob` entry. Thus the C boot path does not load #8552, and the fixture's
`spawn-mob 8552` cannot create the claimed C mobile. The old
`wear-race-hate-pulse` green was hollow: its pulse block never reached the
race-hate mobile dispatch (R5e/R5h).

This does not make the shared C path dead. The wearer's race-hate slots match
the mobile's race, not its vnum. The checked-in index does include other
Race-18 prototypes, including #4213 in `42.mob` and #5502 in `55.mob`. A
scenario using one of them could exercise the same shared branch while
changing the named mobile from #8552.

## Decision requested

Should the row's claim remain tied to #8552, leaving it blocked because C does
not load that prototype, or may the witness substitute an indexed Race-18
mobile and treat the row as generic `APPLY_RACE_HATE` dispatch? No candidate
replacement was promoted without this scope decision, and no census was run
for one.
