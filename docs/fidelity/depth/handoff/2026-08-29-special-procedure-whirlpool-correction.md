# 2026-08-29 — source-order inventory correction

The prior `whirlpool` handoff named `couch` as the next active special. A
fresh R5/R5e inventory check found that this was incorrect: `src/spec_procs2.c`
defines `SPECIAL(couch)` at lines 282-313 and `src/spec_assign.c` declares it
inside `assign_objects`, but there is no `ASSIGNOBJ(..., couch)` or
`ASSIGNMOB(..., couch)` registration. The C `special()` dispatcher can never
reach that body from the registered surface, so it is now recorded as the
excluded `obj.couch-unassigned` case; no synthetic object vehicle is valid.

The actual next active source-order item is `stableboy`, defined at
`src/spec_procs2.c:315` and assigned to mob vnum 8022 at
`src/spec_assign.c:282`. The stableboy slice continues on
`glm/spec-stableboy`.
