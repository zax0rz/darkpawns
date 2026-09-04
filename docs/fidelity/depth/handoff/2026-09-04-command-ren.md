# Depth-fidelity handoff — `ren`

Date: 2026-09-04

Feature branch: `glm/depth-ren`

## Queue position and scope

This slice starts from merged main at `3189eb3a1` after the `reel` depth
slice. The special-procedure inventory remains exhausted; the designated
Phase 2 red/blocked families and the one blocked `objmagic.sleep-entry-gates`
vehicle remain queued for their later passes. Phase 1 is continuing through
the remaining socials. `reel` is the last completed social handoff; the next
genuinely unmanifested `do_action` row in `src/interpreter.c` is `ren` at
line 645. No `ren` manifest, scenario, or focused registration test exists on
this starting main.

The governing workflow is `docs/fidelity/DEPTH_TESTING.md`. The player-facing
contract follows R1/R2/R3/R4/R5e: verify the actual C call path, preserve the
authored bytes, and prove reachable branches before claiming this social.
Shared command-position, `PLR_NOSHOUT`, visible lookup, and Act-audience
behavior remains delegated to established social vehicles under R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:645 */
{ "ren"        , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` resolves the social, rejects
`PLR_NOSHOUT`, conditionally consumes the first target token with
`one_argument`, handles no-argument and missing-target branches, checks the
self-target and victim-position cases, then dispatches actor, observer, and
victim messages. The authored record at `lib/misc/socials:630-638` is
`ren 0 0` (with C's trailing header whitespace) followed by eight message
slots:

```text
Oh! Happy Happy, Joy Joy!
$n jumps up and shouts: "Oh, Happy Happy, Joy Joy!!"
You turn to $M and shout: "You eeediot!!"
$n turns to $N and shouts: "You eeediot!!"
$n turns to you and shouts: "You eeediot!!"
You eeediot!!!
Oh! Happy Happy, Joy Joy!
$n sniffs $mself and says: "Sttteeenky!!!"
```

The C hide flag is `0`, the victim-position minimum is `0`, and all eight
message slots are authored. The reachable slice therefore includes no
argument, visible player target and three audiences, first-token parsing,
visible NPC target, self target, missing target, and sleeping target. The
zero victim-position minimum admits a sleeping target; shared delivery
filtering remains owned by the social matrix.

## Planned proof vehicle

Add `cmd/dp-oracle-diff/scenarios/ren-depth.txt` with `# depth-case:` tags
for each branch and the standard actor/observer/target plus trainee-mob
fixture. Add `pkg/session/ren_depth_test.go` to pin the C command gate and
all eight parsed social messages. Run the scenario with the oracle at seeds
1, 2, 3, 5, and 8, using `--show-oracle` for seed 1. If the existing Go
handler and data are faithful, this is a pure-coverage slice; only confirmed
player-visible divergence may change Go behavior.

## Starting frontier

The merged `reel` handoff reports 4,299 total cases: 4,194
proven/delegated, 54 blocked, and 51 excluded. The `ren` slice will add its
durable branch rows and update this note with final oracle evidence, local
and hosted gates, merge identity, and the next social queue position.

