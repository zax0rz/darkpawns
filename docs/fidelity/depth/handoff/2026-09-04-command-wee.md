# Depth-fidelity handoff — `wee`

Date: 2026-09-04

Feature branch: `glm/depth-wee`

## Queue position and scope

This slice starts from merged main at `42d9deac9` after the `wedgie` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families, blocked clinic vehicles, and the later
off-command-table phases remain queued. Phase 1 continues through the
remaining socials. The next genuinely unmanifested reachable `do_action` row
in `src/interpreter.c` is `wee` at line 812. No `wee` manifest, scenario, or
focused registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, and Act-audience behavior remain
delegated to established social vehicles under R5b/R5c. This is a self-only
record, so typed target, missing target, and named self arguments remain on
the no-argument path by the C `char_found` condition.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:812 */
{ "wee"      , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-149` resolves the social, rejects
`PLR_NOSHOUT`, and, when the record has no target branch, emits the self-only
actor and room slots. The authored record at `lib/misc/socials:974-977` is:

```text
wee 0 0
Weeeee!!!!!!
Flapping $s hands up and down like a bird, $n runs around screaming: "Weeee!!!"
#
```

The command row requires `POS_RESTING`; the C social hide field is 0, minimum
victim position is 0, and the explicit Go override remains 0. The record
terminates after the `#` room slot, so its parsed target/self slots are the
exact empty string. Shared command position, `PLR_NOSHOUT`, and room
visibility mechanics are not duplicated beyond the slice's differential
probes.

## Planned proof vehicle

Add a focused registration test pinning the C command gate, social metadata,
and all three parsed message slots. Add a self-only oracle scenario with a
named actor and peer fixture. Annotate no-argument, typed-target ignored,
missing-target ignored, and named-self ignored cases. Run the standard
deterministic seed matrix (1, 2, 3, 5, and 8), with seed 1 using
`--show-oracle`, then run the repository build, vet, test, lint, formatting,
security, and diff gates.

## Starting frontier

The merged `wedgie` handoff reported 4,650 total cases: 4,545
proven/delegated, 54 blocked, and 51 excluded. This slice is expected to add
eight proven/delegated rows. The next fresh social after this slice is
`weep` at `src/interpreter.c:813`.
