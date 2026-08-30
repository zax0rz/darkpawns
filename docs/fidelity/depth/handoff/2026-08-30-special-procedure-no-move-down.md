# Depth handoff — no_move_down — 2026-08-30

## Frontier and queue position

Started from freshly pulled `main` at `3ad6ef0b9` after the merged
`con_seller` slice.  `make fidelity-depth` reported 1,269 cases: 1,221
proven/delegated, 14 blocked, and 34 excluded (98.9% actionable).  The full
`docs/fidelity/DEPTH_TESTING.md` guide and the newest con-seller handoff were
read before this audit.

The source-order inventory selected `no_move_down` immediately after the
registered `con_seller` definition in `src/spec_procs3.c`.  The next active
registered procedure is `troll`; no previously claimed procedure was
repicked.

## C-first path and reachability ruling

- The body is `SPECIAL(no_move_down)` at `src/spec_procs3.c:316-345`, and the
  name is declared in `src/spec_assign.c:151`.
- A complete search of the active C assignment tables found no
  `ASSIGNMOB(..., no_move_down)`, `ASSIGNOBJ(..., no_move_down)`, or
  `ASSIGNROOM(..., no_move_down)`.  The declaration is not a registration;
  it only makes the symbol available to the assignment translation unit.
- The only real player-command and autonomous mobile entry paths are
  `special()` in `src/interpreter.c:1407-1456` and `mobile_activity()` in
  `src/mobact.c:68-93`; both call a special through a registered object,
  room, or mobile pointer.  With no active pointer, this procedure cannot
  receive its `IS_MOVE`, `AWAKE`, `HUNTING`, level, or `down` branches in the
  C game.
- The latent body would block mortal `down` movement and emit its three C
  acts, but creating a vnum or assignment solely to reach it would invent a
  C command surface.  The Go `RegisterSpec("no_move_down", ...)` entry has no
  corresponding C registration or `MobSpecAssign` mapping and is therefore
  not a valid player-visible proof target.

## Scope and evidence

This is a documentation-only inventory slice. No Go behavior was changed;
`src/` and `darkpawns-c-oracle/` were not edited.  The manifest now contains
`mob.no-move-down-unassigned` as D5 `excluded`, with no synthetic oracle
scenario.  This preserves the C registration surface under R2, avoids
invented player-facing behavior under R4, and records the verified inactive
call path under R5e.  The excluded row is not a claim that the latent body is
faithfully ported; it is a reachability ruling for the actual game surface.

## Gates and counts

The pre-edit checkpoint passed `make fidelity-depth` with:

```text
1269 total; 1221 proven/delegated, 14 blocked, 34 excluded
Actionable completion: 1221/1235 = 98.9%
```

After adding the explicit exclusion row, the expected report is:

```text
1270 total; 1221 proven/delegated, 14 blocked, 35 excluded
Actionable completion: 1221/1235 = 98.9%
```

Because this slice changes only the manifest and handoff, the local depth
gate is the required proof gate; the full repository gates remain clean from
the immediately preceding merged slice and will be rerun before committing.

Rules applied: R2 (registered C command surface), R4 (no synthetic vnum or
invented player-visible behavior), and R5e (active assignment tables and both
real dispatchers checked).  R5b/R5c keep this latent body distinct from the
already proven registered direction blockers and prevent relabeling an
unreachable definition as live coverage.

## Next action

Run the depth gate and full repository gates, commit the documentation-only
slice on `glm/spec-no-move-down`, open one PR, and merge only after all GitHub
checks are green.  If CI does not fire, retry once with
`gh workflow run "Dark Pawns CI/CD" --ref glm/spec-no-move-down`; if it remains
not-green, leave the PR open and advance.  After merge, reset to `main`, pull,
rerun the frontier, reread the guide and newest handoff, and begin the next
active source-order procedure `troll` at mob 10029.
