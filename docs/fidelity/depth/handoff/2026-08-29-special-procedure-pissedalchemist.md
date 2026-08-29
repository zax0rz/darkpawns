# 2026-08-29 — `pissedalchemist` inventory exclusion

The next source-order body after `rescuer` is `SPECIAL(pissedalchemist)` at
`src/spec_procs2.c:546-625`, assigned by `ASSIGNMOB(15814, pissedalchemist)`
at `src/spec_assign.c:422`. The authoritative prototype is
`lib/world/mob/158.mob:#15814` (`the Chamber Master`); its action flags are
`26650`. `src/structs.h` defines `MOB_SPEC` as bit 0, and 26650 has no bit 0.

The actual C call path in `src/mobact.c:68-93` first requires
`MOB_FLAGGED(ch, MOB_SPEC)` before looking up the assigned function. Thus this
assignment cannot reach `pissedalchemist` from the registered mobile surface,
despite the table entry. Under R5e/R2 it is recorded as the excluded manifest
case `mob.pissedalchemist-15814-no-spec`; no synthetic behavior vehicle or Go
implementation is justified. This is the same actual-registration audit used
for the prior `rescuer` vnum 15808 exclusion.

The next reachable unclaimed source-order special is `remorter`, defined at
`src/spec_procs2.c:682` and assigned to mob vnum 4 at `src/spec_assign.c:183`.
Continue the standing queue there; do not repick `pissedalchemist` or its
excluded vnum 15814 registration.

The frontier after adding this exclusion is 1011 total cases: 974
proven/delegated, 12 blocked, and 25 excluded; actionable completion is
974/986 (98.8%). No `src/` or `darkpawns-c-oracle/` file was edited.
