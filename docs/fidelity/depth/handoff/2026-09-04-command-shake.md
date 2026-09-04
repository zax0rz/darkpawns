# Depth-fidelity handoff — `shake`

Date: 2026-09-04

Feature branch: `glm/depth-shake`

## Queue position and scope

This slice starts from merged main at `0f051b36c` after the `scream` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families and the one blocked `objmagic.sleep-entry-gates`
vehicle remain queued for their later passes. Phase 1 is continuing through
the remaining socials. `scream` is the last completed social handoff; the
next genuinely unmanifested reachable `do_action` row in `src/interpreter.c`
is `shake` at line 691. No `shake` manifest, scenario, or focused
registration test exists on this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R4/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, visible lookup, and Act-audience
behavior remains delegated to established social vehicles under R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:691 */
{ "shake"      , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` resolves the social, rejects
`PLR_NOSHOUT`, conditionally consumes the first target token with
`one_argument`, handles no-argument and missing-target branches, checks the
self-target and victim-position cases, then dispatches actor, observer, and
victim messages. The authored record at `lib/misc/socials:665-673` is
`shake 0 5` followed by eight message slots:

```text
You shake your head.
$n shakes $s head.
You shake $S hand.
$n shakes $N's hand.
$n shakes your hand.
Sorry good buddy, but that person doesn't seem to be here.
You are shaken by yourself.
$n shakes and quivers like a bowlful of jelly.
```

The C hide flag is `0` and the victim-position minimum is `5` (`POS_RESTING`).
The reachable slice therefore includes no argument, visible player target and
three audiences, first-token parsing, visible NPC target, self target,
missing target, and sleeping-target rejection. The shared position failure,
lookup, and audience mechanisms are not duplicated here.

## Planned proof vehicle

Add `cmd/dp-oracle-diff/scenarios/shake-depth.txt` with `# depth-case:` tags
for each branch and the standard actor/observer/target plus trainee-mob
fixture. Add `pkg/session/shake_depth_test.go` to pin the C command gate and
all eight parsed social messages. Run the scenario with the oracle at seeds
1, 2, 3, 5, and 8, using `--show-oracle` for seed 1. If the existing Go
handler and data are faithful, this is a pure-coverage slice; only confirmed
player-visible divergence may change Go behavior.

## Starting frontier

The merged `scream` handoff reports 4,318 total cases: 4,213
proven/delegated, 54 blocked, and 51 excluded. The `shake` slice will add its
durable branch rows and update this note with final oracle evidence, local
and hosted gates, merge identity, and the next social queue position.

