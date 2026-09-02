# Depth-fidelity handoff — `shoot`

Date: 2026-09-02

## Queue position and result

This round began from clean `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md`, the
2026-08-27 brief amendment, and the newest handoff,
`2026-09-02-command-show.md`. The special-procedure inventory remains
exhausted. The one-time blocked `objmagic.sleep-entry-gates` row was attempted
through the cast-sleep outlaw/reagent vehicle and is not repicked. The
interpreter sweep consumed the next source-order family, `shoot`, at
`src/interpreter.c:695`.

The pre-slice frontier was 3,305 total cases, with 3,217 proven/delegated, 36
blocked, and 52 excluded. The shoot manifest contributes 26 cases: 17
proven/delegated cases, one unreachable handler branch, and eight blocked
target/state branches. The resulting frontier is:

- 3,331 total cases
- 3,234 proven/delegated
- 44 blocked
- 53 excluded

Actionable completion is 3,234/3,278 = 98.7%.

## C call path and behavior surface

The registered row is:

```c
/* src/interpreter.c:695 */
{ "shoot"    , POS_STANDING, do_shoot    , 0, 0 },
```

The handler is `src/act.offensive.c:746-980`. The actual path first applies
the known-skill and fighting gates, then parses projectile, direction, and
target; resolves the carried object and projectile type; resolves the exit,
door keyword, wielded fireweapon, peaceful source/destination, and destination
target; and finally enters the target/damage/relocation/retaliation state
machine. A target found in the destination room is resolved with the shooter
retained as the viewer. The sentinel target arm emits the aim-refusal line;
the ordinary target arm can emit projectile, damage, relocation, room-audience,
and synchronous combat bytes.

The scenario matrix proves the parser, missing and non-projectile object
gates, direction and exit gates, wielded-bow gate, closed-door keyword,
peaceful-room gate, empty-destination projectile drop, and sentinel target
arm at seeds 1, 2, 3, 5, and 8, with seed 1 run using `--show-oracle`.
`shoot-target-depth.txt` was also run against the real non-sentinel target
path. After the adjacent-room lookup correction, C and Go still diverged in
the deeper target state machine: C emitted the projectile preamble, damage
outcome, relocation, and synchronous retaliation, while Go emitted only
`You hear a roar of pain! Your shot hits!`.

## Go changes and proof boundary

Only confirmed entry and reachable-gate divergences were fixed in Go:

- `SkillShoot` now has C's standing position and exact unknown-skill bytes;
- `CmdShoot` now preserves C's parser prompts, object/type/direction/exit,
  door, wielded-fireweapon, peaceful, no-target, drop, and sentinel branches;
- `ResolveCharInRoomAt` preserves destination-room target lookup while
  keeping the origin-room shooter as the visibility viewer;
- focused registration tests and annotated oracle scenarios pin the C entry
  gate and all claimed live cases.

The eight ordinary-target, fallback, level-window, fighting-target,
hit/miss, wait, object-consumption, and audience/state rows remain `blocked`
after two honest full-target attempts. Their notes preserve the blocker and
do not invent replacement audience or combat output. Generic combat proof is
not substituted for shoot's C draw order and re-entrant path.

Durable evidence is `docs/fidelity/depth/shoot.tsv`, the annotated
`shoot-*-depth.txt` scenarios, and `pkg/session/shoot_depth_test.go`.

## Verification and integration

The required local gates passed on `glm/depth-shoot`:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature commit: `72e0cb617`.

Feature PR: #1171 (`glm/depth-shoot`). Hosted lint, security, and test checks
completed green; conditional build-and-push and deploy jobs were skipped.
Checks did not initially appear, so the permitted single retry was run with
`gh workflow run "Dark Pawns CI/CD" --ref glm/depth-shoot`; the PR was
self-merged only after the applicable checks were green. The resulting
`main` merge commit is `de239a127`.

This round follows R1/R2 (player-facing bytes and command surface), R3
(multi-seed determinism), R4 (no invented output), R5/R5e (the actual C call
path), and R5b/R5c (shared gate/lookup behavior and whole-class review).

## Continuation

The source-order audit finds 509 registered command rows and 386 manifest
command tokens; 191 interpreter tokens remain without token-level manifests.
The next unclaimed token is:

```c
/* src/interpreter.c:330 */
{ "accuse"   , POS_SITTING , do_action   , 0, 0 },
```

The next session must return to clean `main`, pull, rerun
`make fidelity-depth`, reread the guide and this handoff, then map and prove
`accuse` before advancing in interpreter-table order.
