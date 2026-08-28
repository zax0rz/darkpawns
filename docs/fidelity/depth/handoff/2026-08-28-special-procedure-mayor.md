# Depth-fidelity handoff: `mayor`

Date: 2026-08-28  
Branch: `glm/spec-mayor`  
Starting main: `55425e630` (`fix: deepen cityguard special procedure (#703)`)

## Frontier

After the merged cityguard slice, `make fidelity-depth` reported 586 total
cases, 569 proven/delegated, 5 blocked, and 12 excluded: 569/574 actionable,
or 99.1%.

This slice adds one explicit excluded row for the next source-order special.
The resulting frontier is 587 total, 569 proven/delegated, 5 blocked, and 13
excluded: 569/574 actionable, or 99.1%.

## Queue position and C inventory

`mayor` is the next source-defined special after `cityguard`, at
`src/spec_procs.c:823-924`. The procedure declares the daily open/close paths,
the process-static route state, and the commandless path-walking branches for
movement, speech, sleep/wake, and gate open/close operations.

The required registration census found only the prototype
`SPECIAL(mayor)` in `src/spec_assign.c:85-126`. A complete search of
`src/spec_assign.c` found no `ASSIGNMOB(..., mayor)` and no
`ASSIGNROOM(..., mayor)` entry. Therefore no C mobile instance receives this
function pointer through `assign_mobiles`, and no world mob can reach it via
`mobile_activity()` or the interpreter's registered special-procedure path.

The actual reachable-dispatch path is consequently empty:

- `src/mobact.c:68-93` invokes a special only from an assigned mob's
  commandless autonomous activity;
- `src/interpreter.c:1407-1456` invokes the special attached to a command's
  room/mobile target, but there is no mayor assignment to select;
- `src/spec_procs.c:823-924` is a defined but unregistered C procedure.

Per R5e, the C assignment table—not the existing Go registry entry or Go
implementation—determines the reachable proof surface. Per R2 and R4, an
invented mayor mob or a synthetic oracle vehicle would claim behavior that the
configured C game cannot dispatch. No `src/` or `darkpawns-c-oracle/` file was
edited.

## Manifest result

Added `mob.mayor-unassigned` to `docs/fidelity/depth/spec-procs.tsv` as D5
`excluded`, with no proof scenario. This follows the prior unassigned-special
precedent: the row records the source procedure and dispatch census without
promoting an unassigned Go registry entry to reachable evidence.

## Verification and next handoff

The manifest validator passes with the post-slice frontier above. This is a
documentation-only reachability slice; no Go behavior was changed and no
oracle scenario is claimed because there is no registered C vehicle.

The next source-order special procedure is `dragon_breath` at
`src/spec_procs.c:927` (subject to its registration census). After the special
inventory is exhausted, attempt the one blocked `objmagic.sleep-entry-gates`
row via the cast-sleep vehicle, then sweep unmanifested command families in
`src/interpreter.c` table order.

This handoff follows R1 (player-facing bytes remain unclaimed), R2 (registered
command/autonomous surface), R4 (no invention), and R5e (verify the actual C
call/assignment path). Full repository gates are required before the slice PR
is merged.
