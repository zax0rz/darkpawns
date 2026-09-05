# Movement activity-surface audit — 2026-09-04

## Scope

This slice audits the 110 literal `act()`/`send_to_char()` call sites in
`src/act.movement.c`, covering `do_move`, `do_gen_door`, `do_enter`,
`do_leave`, `do_stand`, `do_sit`, `do_rest`, `do_sleep`, `do_wake`, and
`do_follow`.

## Existing ownership

The complete ordinary movement and position surface is already represented by
`movement.tsv`, `door.tsv`, `enter.tsv`, `leave.tsv`, `position.tsv`,
`follow.tsv`, `sleeper.tsv`, `mount.tsv`, `shadow.tsv`, `escape.tsv`,
`flee.tsv`, `doh.tsv`, and `embrace.tsv`. Room and mobile special-procedure
call paths that enter through movement are owned by `spec-procs.tsv`; shared
communication and audience mechanics remain delegated to their canonical
rows. The existing manifests include the ordinary exit, door, landing,
position, follower, mount, death-trap, and special-hook branches, including
their reachable exclusions.

## Protocol and decision

The slice used the standard two-seed protocol with a 300-second per-scenario
timeout. `movement-directions`, `door-basic`, `enter-depth`, `position-basic`,
`follow-depth`, and `movement-mounted` all reported `no normalized divergence`
for seeds 1 and 2.

The inventory row is promoted to `proven-already` as a call-site ownership
result: all 110 literals map to existing focused rows, and the descriptorless
NPC movement path remains the explicit `movement.mob-entry-prog` exclusion.
No C or oracle source is modified, and no movement branch was silently
absorbed into a broad green claim.
