# Depth-fidelity handoff — `thank`

Date: 2026-09-04

Feature branch: `glm/depth-thank`

## Queue position and scope

This slice starts from merged main at `f36378ef9` after the `taunt` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families and the blocked clinic vehicles remain queued
for their later passes. Phase 1 is continuing through the remaining socials.
The next genuinely unmanifested reachable `do_action` row in
`src/interpreter.c` is `thank` at line 765. No `thank` manifest, scenario, or
focused registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, and Act-audience behavior remains
delegated to established social vehicles under R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:765 */
{ "thank"    , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` resolves the social, rejects
`PLR_NOSHOUT`, parses the first target token, handles no-target, not-found,
self-target, and visible-target branches, and routes the authored audience
through `act`. The authored record at `lib/misc/socials:929-937` is:

```text
thank 0 5
Thank you too.
#
You thank $N heartily.
$n thanks $N heartily.
$n thanks you heartily.
No one answers to that name here.
You thank yourself since nobody else wants to!
$n thanks $mself since you won't.
```

The command row requires `POS_RESTING`; the C social level is 0, hide flag is
5, and the victim-position minimum is the default 0. The target-capable
record reaches no-argument, visible-player/NPC, named self, missing-target,
and first-token/trailing-argument branches. Shared command position,
`PLR_NOSHOUT`, target lookup, and room visibility mechanics are not duplicated
beyond the slice's differential probes.

## Planned proof vehicle

Add `cmd/dp-oracle-diff/scenarios/thank-depth.txt` with the standard actor,
observer, target, and generic-mob fixture, `pkg/session/thank_depth_test.go`
to pin the C command gate, social metadata, and all eight parsed message
slots, and `docs/fidelity/depth/thank.tsv` with the durable unit, delegated,
and oracle rows. The existing Go handler and data are expected to be faithful;
any mismatch must be resolved from the C call path under R5e.

## Starting frontier

The merged `taunt` handoff reported 4,490 total cases: 4,385
proven/delegated, 54 blocked, and 51 excluded. Continue the remaining Phase 1
social sweep before the later red/blocked and off-command-table phases in the
objective. The next fresh social after this slice is `thpbt` at
`src/interpreter.c:767`.
