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

The slice will use the standard two-attempt depth protocol with per-scenario
timeouts of at least 240 seconds, and will promote the surface inventory only
when every call-site family has an existing proof or an explicit, sharpened
blocked row. No C or oracle source is modified. Any newly discovered
reachable path will be classified as proven, blocked, or a separately
justified exclusion; it will not be silently absorbed into a broad green
claim.
