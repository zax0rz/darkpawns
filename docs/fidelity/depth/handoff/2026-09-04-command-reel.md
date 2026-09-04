# Depth-fidelity handoff — `reel`

Date: 2026-09-04

Feature branch: `glm/depth-reel`

## Queue position and scope

This slice starts from merged main at `9f2f8e549` after the `zlist` depth
slice. The special-procedure inventory remains exhausted; the previously
blocked `objmagic.sleep-entry-gates` vehicle and the Phase 2 red/blocked
families remain queued for their designated later passes. The social sweep is
now the active Phase 1 queue. `raise` is the last completed social handoff;
the next genuinely unmanifested `do_action` row in `src/interpreter.c` is
`reel` at line 637. No `reel` manifest, scenario, or focused registration
test exists on this starting main.

The governing evidence workflow is `docs/fidelity/DEPTH_TESTING.md`. The
player-facing contract follows R1/R2/R3/R4/R5e: verify the actual C call path,
copy no invented bytes, and prove all reachable branches before claiming the
social complete. Shared command-position, `PLR_NOSHOUT`, visible lookup, and
Act-audience behavior remains delegated to the existing social vehicles under
R5b/R5c.

## C call path and authored data

The registered C row is:

```c
/* src/interpreter.c:637 */
{ "reel"      , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` resolves the social, rejects
`PLR_NOSHOUT`, conditionally consumes the first target token with
`one_argument`, handles no-argument and missing-target branches, checks the
self-target and victim-position cases, then dispatches actor, observer, and
victim messages. The authored record at `lib/misc/socials:1088-1096` is:

```text
reel 1 0
You reel around the room drunkenly.
$n reels around like a drunken sot.
You reel up to $N and breathe alcohol into $S face.
$n reels up to $N and tries to steady $mself, and they both fall down.
$n reels up to you and grabs on to you for support, knocking you over.
Who? *laugh* Not here, buddy.
You reel around, feeling quite suave in your lack of sobriety.
$n reels around and looks like it's time to visit the porcelain god.
```

The C hide flag is `1`, the victim-position minimum is `0`, and all eight
message slots are authored. The reachable slice therefore includes no
argument, visible player target and three audiences, first-token parsing,
visible NPC target, self target, missing target, and sleeping target. Because
the victim minimum is zero, a sleeping target reaches the success dispatch;
shared delivery filtering is not reimplemented here.

## Planned proof vehicle

Add `cmd/dp-oracle-diff/scenarios/reel-depth.txt` with `# depth-case:` tags
for each branch and the standard actor/observer/target plus trainee-mob
fixture. Add `pkg/session/reel_depth_test.go` to pin the C command gate and
all eight parsed social messages. Run the scenario with the oracle at seeds
1, 2, 3, 5, and 8, using `--show-oracle` for seed 1. If the existing Go
handler and data are faithful, this is a pure-coverage slice; only confirmed
player-visible divergence may change Go behavior.

## Starting frontier

The merged `zlist` handoff reports 4,288 total cases: 4,183
proven/delegated, 54 blocked, and 51 excluded. The `reel` slice will add its
durable branch rows and update this note with pre-fix/final evidence, local
and hosted gates, merge identity, and the next social queue position.

