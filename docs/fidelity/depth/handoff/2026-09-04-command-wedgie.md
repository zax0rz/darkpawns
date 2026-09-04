# Depth-fidelity handoff — `wedgie`

Date: 2026-09-04

Feature branch: `glm/depth-wedgie`

## Queue position and scope

This slice starts from merged main at `f805a493e` after the `wave` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families, blocked clinic vehicles, and the later
off-command-table phases remain queued. Phase 1 continues through the
remaining socials. The next genuinely unmanifested reachable `do_action` row
in `src/interpreter.c` is `wedgie` at line 811. No `wedgie` manifest, scenario,
or focused registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, target lookup, visibility, and
Act-audience behavior remain delegated to established social vehicles under
R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:811 */
{ "wedgie"   , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-149` resolves the social, rejects
`PLR_NOSHOUT`, parses the first target token, handles no-target, not-found,
self-target, and visible-target branches, and routes the authored audience
through `act`. The authored record at `lib/misc/socials:1312-1320` is:

```text
wedgie 0 5
You pull your underwear out of your crack!
You watch as $n un-wedgies $mself.
You grab $N's underwear and pull up HARD.
You see $n give a painful wedgie to $N.
$n almost rips your underwear off with an incredible wedgie.
Wedgie who?
You give yourself a wedgie.
You watch $n pull $s underwear nearly over $s head.
```

The command row requires `POS_RESTING`; the C social level is 0, hide flag is
5, and the victim-position minimum is the default 0. This target-capable
record reaches no-argument, visible player/NPC, named self, missing-target,
and first-token/trailing-argument branches. Shared command position,
`PLR_NOSHOUT`, target lookup, and room visibility mechanics are not duplicated
beyond the slice's differential probes.

## Planned proof vehicle

Add a focused registration test pinning the C command gate, social metadata,
and all eight authored message slots. Add a full-target oracle scenario with
named actor, observer, target, and generic-mob fixtures. Annotate no-argument,
target success/audiences, first-token parsing, mob target, self target,
not-found, visibility, and sleeping-target cases. Run the standard deterministic
seed matrix (1, 2, 3, 5, and 8), with seed 1 using `--show-oracle`, then run
the repository build, vet, test, lint, formatting, security, and diff gates.

## Starting frontier

The merged `wave` handoff reported 4,638 total cases: 4,533
proven/delegated, 54 blocked, and 51 excluded. This slice is expected to add
twelve proven/delegated rows. The next fresh social after this slice is
`wee` at `src/interpreter.c:812`.
