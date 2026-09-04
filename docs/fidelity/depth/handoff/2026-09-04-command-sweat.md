# Depth-fidelity handoff — `sweat`

Date: 2026-09-04

Feature branch: `glm/depth-sweat`

## Queue position and scope

This slice starts from merged main at `b0315ab98` after the `sulk` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families and the blocked clinic vehicles remain queued
for their later passes. Phase 1 is continuing through the remaining socials.
The next genuinely unmanifested reachable `do_action` row in
`src/interpreter.c` is `sweat` at line 750. No `sweat` manifest, scenario, or
focused registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, and Act-audience behavior remains
delegated to established social vehicles under R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:750 */
{ "sweat"     , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-127` resolves the social, rejects
`PLR_NOSHOUT`, checks the record's `char_found` slot, and because it is `#`
follows the self-only/no-target path. Typed targets, including a self alias
and an unresolved name, are ignored rather than looked up. The authored
record at `lib/misc/socials:1272-1275` is:

```text
sweat 0 5
You break out in a cold sweat.
$n starts to sweat profusely.
#
```

The command row requires `POS_RESTING` while the C social level is 0 and hide
flag is 5; the victim-position minimum is the default 0. The reachable slice
is therefore the resting-gated no-argument actor/room pair and typed-target,
self-target, and missing-target variants that all remain on that same path.
Shared command position, `PLR_NOSHOUT`, and room visibility mechanics are not
duplicated.

## Planned proof vehicle

Add `cmd/dp-oracle-diff/scenarios/sweat-depth.txt` with the standard actor and
peer fixture, `pkg/session/sweat_depth_test.go` to pin the C command gate,
the social level/hide metadata, and all three parsed message slots, and
`docs/fidelity/depth/sweat.tsv` with the durable unit, delegated, and oracle
rows. The existing Go handler and data are expected to be faithful; any
mismatch must be resolved from the C call path under R5e.

## Starting frontier

The merged `sulk` handoff reported 4,434 total cases: 4,329
proven/delegated, 54 blocked, and 51 excluded. Continue the remaining Phase 1
social sweep before the later red/blocked and off-command-table phases in the
objective. The next fresh social after this slice is `tackle` at
`src/interpreter.c:756`.
