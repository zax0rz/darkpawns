# 2026-08-29 — `enter_circle` depth slice

## Frontier and queue

- This session started on `main`; the required local pull was already current
  at the recorded `origin/main` tip, with the prior pet-shop handoff still
  pending external publication because GitHub DNS was unavailable.
- Before this slice, `make fidelity-depth` reported 904 total cases, 882
  proven/delegated, 6 blocked, and 16 excluded: 882/888 actionable (99.3%).
- The next source-order special after the already-claimed `pet_shops` slice
  was `enter_circle`.
- This slice adds one explicit excluded row. The resulting frontier is 905
  total, 882 proven/delegated, 6 blocked, and 17 excluded: 882/888
  actionable (99.3%).

## C path and registration census

- `src/spec_procs.c:1920-1979` defines `SPECIAL(enter_circle)`. Its latent
  command branches accept `enter` and `look`; `enter circle/platform` checks
  the population of virtual room 5799, moves the actor, and calls `do_look`,
  while `look circle/platform` renders the occupants of that room. Other
  `look` arguments delegate to ordinary `do_look`.
- `src/spec_assign.c:573-603` declares the room-special function symbols,
  including `SPECIAL(enter_circle)`, but `src/spec_assign.c:605-635` contains
  no `ASSIGNROOM(..., enter_circle)` call. The mobile assignment table also
  contains no `ASSIGNMOB(..., enter_circle)` call.
- `src/spec_assign.c:68-74` shows that only an `ASSIGNROOM` call installs a
  room function pointer. `src/interpreter.c:1407-1415` invokes the room
  pointer before the ordinary command table, and its later mobile loop only
  invokes assigned mobile pointers. Therefore the C dispatch path for this
  procedure is empty: no configured room or mobile can reach any of its
  player-visible branches.
- The Go registry contains a latent `specEnterCircle` function, but
  `RoomSpecAssign` has no `enter_circle` entry and no C assignment exists to
  justify adding one. Elevating that unassigned definition into the Go room
  command surface would invent behavior under R2/R4 and contradict the actual
  C call path required by R5e.

## Manifest result

Added `mob.enter-circle-unassigned` to
`docs/fidelity/depth/spec-procs.tsv` as D5 `excluded`, with no proving
scenario. The row records the source definition and both assignment-table
checks; it does not claim a synthetic portal-room vehicle or any latent branch
as reachable evidence. No `src/` or `darkpawns-c-oracle/` file was edited.

## Verification and next handoff

`make fidelity-depth` passes with the frontier above. This is a
documentation-only reachability slice; no Go behavior changed, so no RED/GREEN
oracle scenario is legitimate and no C-visible divergence was fixed.

The next source-order special definition is `elevator` in
`src/spec_procs.c:1981` and must undergo the same assignment census before any
vehicle is built. Continue the special-procedure inventory without repicking
`enter_circle` or any prior claimed item. After the active inventory is
exhausted, attempt the single blocked `objmagic.sleep-entry-gates` row through
the cast-sleep outlaw/reagent vehicle, then sweep un-manifested interpreter
command families in table order.

This handoff applies R2 (registered command surface), R4 (no invented
behavior), and R5e (verify the actual C assignment/call path); R1 remains
unclaimed because no player-facing C branch is dispatchable.
