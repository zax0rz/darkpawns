# Depth-fidelity handoff — `shrug`

Date: 2026-09-04

Feature branch: `glm/depth-shrug`

## Queue position and scope

This slice starts from merged main at `a1199e1fe` after the `shake` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families and the one blocked `objmagic.sleep-entry-gates`
vehicle remain queued for their later passes. Phase 1 is continuing through
the remaining socials. `shake` is the last completed social handoff; the next
genuinely unmanifested reachable `do_action` row in `src/interpreter.c` is
`shrug` at line 696. No `shrug` manifest, scenario, or focused registration
test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R4/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, and Act-audience behavior remains
delegated to established social vehicles under R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:696 */
{ "shrug"      , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-127` resolves the social, rejects
`PLR_NOSHOUT`, checks the record's `char_found` slot, and because it is `#`
follows the self-only/no-target path. Typed targets, including a self alias
and an unresolved name, are ignored rather than looked up. The authored
record at `lib/misc/socials:690-693` is:

```text
shrug 0 0
You shrug.
$n shrugs.
#
```

The C hide flag and victim-position minimum are both `0`; only the actor and
ordinary room message slots are authored. The reachable slice is therefore
the no-argument actor/room pair and the typed-target, self-target, and
missing-target variants that all remain on that same pair. Shared command
position, `PLR_NOSHOUT`, and room visibility mechanics are not duplicated.

## Planned proof vehicle

Add `cmd/dp-oracle-diff/scenarios/shrug-depth.txt` with `# depth-case:` tags
for the no-argument and ignored-argument branches, plus the standard actor
and peer fixture. Add `pkg/session/shrug_depth_test.go` to pin the C command
gate and all three parsed message slots. Run the scenario with the oracle at
seeds 1, 2, 3, 5, and 8, using `--show-oracle` for seed 1. If the existing Go
handler and data are faithful, this is a pure-coverage slice; only confirmed
player-visible divergence may change Go behavior.

## Starting frontier

The merged `shake` handoff reports 4,329 total cases: 4,224
proven/delegated, 54 blocked, and 51 excluded. The `shrug` slice will add its
durable branch rows and update this note with final oracle evidence, local
and hosted gates, merge identity, and the next social queue position.

