# Depth-fidelity handoff — `taunt`

Date: 2026-09-04

Feature branch: `glm/depth-taunt`

## Queue position and scope

This slice starts from merged main at `4dcd2a449` after the `tap` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families and the blocked clinic vehicles remain queued
for their later passes. Phase 1 is continuing through the remaining socials.
The next genuinely unmanifested reachable `do_action` row in
`src/interpreter.c` is `taunt` at line 761. No `taunt` manifest, scenario, or
focused registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, and Act-audience behavior remains
delegated to established social vehicles under R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:761 */
{ "taunt"    , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` resolves the social, rejects
`PLR_NOSHOUT`, parses the first target token, handles no-target, not-found,
self-target, and visible-target branches, and routes the authored audience
through `act`. The authored record at `lib/misc/socials:919-927` is:

```text
taunt 0 0
You taunt the nothing in front of you. 
$n taunts something that seems to be right in front of $m.
You taunt $M, to your own delight.
$n taunts $N rather insultingly.  $n seems to enjoy it tremendously.
$n taunts you.  It really hurts your feelings.
Hmmmmmmm.....nope, no one by that name here.
You taunt yourself, almost making you cry...:(
$n taunts $mself to tears.
```

The command row requires `POS_RESTING`; the C social level and hide flag are
both 0, and the victim-position minimum is the default 0. The target-capable
record reaches no-argument, visible-player/NPC, named self, missing-target,
and first-token/trailing-argument branches. Shared command position,
`PLR_NOSHOUT`, target lookup, and room visibility mechanics are not duplicated
beyond the slice's differential probes.

## Planned proof vehicle

Add `cmd/dp-oracle-diff/scenarios/taunt-depth.txt` with the standard actor,
observer, target, and generic-mob fixture, `pkg/session/taunt_depth_test.go`
to pin the C command gate, social metadata, and all eight parsed message
slots, and `docs/fidelity/depth/taunt.tsv` with the durable unit, delegated,
and oracle rows. The existing Go handler and data are expected to be faithful;
any mismatch must be resolved from the C call path under R5e.

## Starting frontier

The merged `tap` handoff reported 4,478 total cases: 4,373
proven/delegated, 54 blocked, and 51 excluded. Continue the remaining Phase 1
social sweep before the later red/blocked and off-command-table phases in the
objective. The next fresh social after this slice is `thank` at
`src/interpreter.c:765`.
