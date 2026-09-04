# Depth-fidelity handoff — `whap`

Date: 2026-09-04

Feature branch: `glm/depth-whap`

## Queue position and scope

This slice starts from merged main at `ffd2fe766` after the `weep` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families, blocked clinic vehicles, and the later
off-command-table phases remain queued. Phase 1 continues through the
remaining socials. The next genuinely unmanifested reachable `do_action` row
in `src/interpreter.c` is `whap` at line 815. No `whap` manifest, scenario, or
focused registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, target lookup, visibility, and
Act-audience behavior remain delegated to established social vehicles under
R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:815 */
{ "whap"     , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-149` resolves the social, rejects
`PLR_NOSHOUT`, parses the first target token, handles no-target, not-found,
self-target, and visible-target branches, and routes the authored audience
through `act`. The authored record at `lib/misc/socials:1168-1176` is:

```text
whap 0 5
You whap the ground.
$n whaps the ground.
You whap $N upside the head.
$n whaps $N upside the head.
$n whaps you upside the head.
Whap who? They aren't here.
You whap yourself upside the head.
$n tries to knock some sense into $mself.
```

The command row requires `POS_RESTING`; the C social hide field is 0 and its
minimum victim position is 5 (represented by the legacy Go `HideFlag` field),
while the explicit Go override remains 0. This target-capable record reaches
no-argument, visible player/NPC, named self, missing-target, first-token/
trailing-argument, and sleeping-target branches. Shared command position,
`PLR_NOSHOUT`, target lookup, and room visibility mechanics are not duplicated
beyond the slice's differential probes.

## Planned proof vehicle

Add a focused registration test pinning the C command gate, social metadata,
and all eight authored message slots. Add a full-target oracle scenario with
named actor, observer, target, and generic-mob fixtures. Annotate no-argument,
target success/audiences, first-token parsing, mob target, self target,
not-found, visibility, and sleeping-target position-gate cases. Run the
standard deterministic seed matrix (1, 2, 3, 5, and 8), with seed 1 using
`--show-oracle`, then run the repository build, vet, test, lint, formatting,
security, and diff gates.

## Starting frontier

The merged `weep` handoff reported 4,666 total cases: 4,561
proven/delegated, 54 blocked, and 51 excluded. This slice is expected to add
twelve proven/delegated rows. The next fresh social after this slice is
`whimper` at `src/interpreter.c:816`.
