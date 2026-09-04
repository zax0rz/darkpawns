# Depth-fidelity handoff — `thumbsup`

Date: 2026-09-04

Feature branch: `glm/depth-thumbsup`

## Queue position and scope

This slice starts from merged main at `115150bd9` after the `throttle` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families, blocked clinic vehicles, and the later
off-command-table phases remain queued. Phase 1 continues through the
remaining socials. The next genuinely unmanifested reachable `do_action` row
in `src/interpreter.c` is `thumbsup` at line 771. No `thumbsup` manifest,
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
/* src/interpreter.c:771 */
{ "thumbsup" , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` resolves the social, rejects
`PLR_NOSHOUT`, parses the first target token, handles no-target, not-found,
self-target, and visible-target branches, and routes the authored audience
through `act`. The authored record at `lib/misc/socials:939-947` is:

```text
thumbsup 0 0
You seem very happy today.
$n gives everyone a big thumbs up.
You give $N a big thumbs up.
$n gives $N a big thumbs up.
$n gives you a big thumbs up.
You don't see that person.
You feel extremely silly.
$n gives $mself a big thumbs up.  How silly.
```

The command row requires `POS_RESTING`; the C social level is 0, hide flag is
0, and the victim-position minimum is the default 0. This target-capable
record reaches no-argument, visible-player/NPC, named self, missing-target,
and first-token/trailing-argument branches. Shared command position,
`PLR_NOSHOUT`, target lookup, and room visibility mechanics are not duplicated
beyond the slice's differential probes.

## Planned proof vehicle

Add the focused registration test, a durable TSV manifest, and a
`thumbsup-depth` oracle scenario using the standard actor, observer, target,
and generic-mob fixture. Exercise no argument, a visible player target with
trailing words, a generic mob target, a named self target, a missing target,
and a sleeping target; retain the standard delegated rows for the shared
social boundary. Run the five-seed oracle matrix and all repository gates
before the final handoff.

## Starting frontier

The merged `throttle` handoff reported 4,534 total cases: 4,429
proven/delegated, 54 blocked, and 51 excluded. This slice is expected to add
twelve proven/delegated rows, bringing the frontier to 4,546 total, 4,441
proven/delegated, 54 blocked, and 51 excluded. The next fresh social after
this slice is `thunk` at `src/interpreter.c:772`.
