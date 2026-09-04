# Item activity-surface audit — 2026-09-04

## Scope

This slice audits the 169 literal `act()`/`send_to_char()` call sites in
`src/act.item.c`, in source-file order. The handlers are `do_put`, `do_get`,
`do_drop`, `do_give`, `do_drink`, `do_eat`, `do_pour`/`do_fill`, `do_wear`,
`do_wield`, `do_grab`, and `do_remove`.

## Existing ownership

The ordinary command, transfer, disposal, container, liquid, equipment, and
audience branches are already represented by `put.tsv`, `get.tsv`,
`drop.tsv`, `disposal.tsv`, `give.tsv`, `drink.tsv`, `eat.tsv`, `pour.tsv`,
`fill.tsv`, `wear.tsv`, `wield.tsv`, `grab.tsv`, `hold.tsv`, and `remove.tsv`.
Shared transfer and equipment paths are additionally owned by the existing
`comm.tsv`, `flesh-alter.tsv`, and `spec-procs.tsv` rows where their call path
is outside the registered player command.

Two reachable state branches are deliberately not treated as exclusions:
`drink.vampire`, `eat.vampire`, and `eat.werewolf-corpse` remain explicit
blocked rows in their command manifests. The `give.mob-ongive` and
`get.palm-quiet-path` rows retain their separately documented ownership and
reachability decisions.

## Protocol and decision

The slice used the standard two-seed depth protocol with a 300-second
per-scenario timeout. `put-basic`, `get-room`, `get-container`, `drop-basic`,
`give-basic`, `drink-basic`, `eat-basic`, `pour-basic`, `fill-depth`,
`wear-basic`, `wield-basic`, `grab-depth`, `hold-depth`, `remove-basic`, and
`donate-routing` all reported `no normalized divergence` for seeds 1 and 2.
One parallel `donate-routing` seed-2 process failed before readiness because
the local harness ports collided; the isolated rerun was green and is the
usable result.

The inventory row is promoted to `proven-already` as a call-site ownership
result: all 169 literals map to existing focused rows, and the reachable
non-green vampire/werewolf paths remain explicit blocked cases rather than
hidden exclusions. No C or oracle source is modified. Any newly discovered
reachable path will be classified as proven, blocked, or a separately
justified exclusion; it will not be silently absorbed into a broad green
claim.
