# Depth-fidelity handoff — `tickle`

Date: 2026-09-04

Feature branch: `glm/depth-tickle`

## Queue position and scope

This slice starts from merged main at `a41f9561b` after the `thunk` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families, blocked clinic vehicles, and the later
off-command-table phases remain queued. Phase 1 continues through the
remaining socials. The next genuinely unmanifested reachable `do_action` row
in `src/interpreter.c` is `tickle` at line 775. No `tickle` manifest,
scenario, or focused registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, target lookup, visibility, and
Act-audience behavior remain delegated to established social vehicles under
R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:775 */
{ "tickle"   , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` resolves the social, rejects
`PLR_NOSHOUT`, parses the first target token, handles no-target, not-found,
self-target, and visible-target branches, and routes the authored audience
through `act`. The authored record at `lib/misc/socials:949-957` is:

```text
tickle 0 0
Who do you want to tickle??
#
You tickle $N.
$n tickles $N.
$n tickles you - hee hee hee.
Who is that??
You tickle yourself, how funny!
$n tickles $mself.
```

The command row requires `POS_RESTING`; the C social level is 0, hide flag is
0, and the victim-position minimum is the default 0. This target-capable
record has an authored `#` room slot for no-argument use, and reaches
no-argument, visible-player/NPC, named self, missing-target, and
first-token/trailing-argument branches. Shared command position,
`PLR_NOSHOUT`, target lookup, and room visibility mechanics are not duplicated
beyond the slice's differential probes.

## Planned proof vehicle

Add the focused registration test, a durable TSV manifest, and a `tickle-depth`
oracle scenario using the standard actor, observer, target, and generic-mob
fixture. Exercise no argument, a visible player target with trailing words, a
generic mob target, a named self target, a missing target, and a sleeping
target; retain the standard delegated rows for the shared social boundary.
Run the five-seed oracle matrix and all repository gates before the final
handoff.

## Starting frontier

The merged `thunk` handoff reported 4,558 total cases: 4,453
proven/delegated, 54 blocked, and 51 excluded. This slice is expected to add
twelve proven/delegated rows, bringing the frontier to 4,570 total, 4,465
proven/delegated, 54 blocked, and 51 excluded. The next fresh social after
this slice is `tilt` at `src/interpreter.c:777`.
