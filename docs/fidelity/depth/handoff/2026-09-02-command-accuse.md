# Depth-fidelity handoff — `accuse`

Date: 2026-09-02

## Queue position and result

This round began from clean `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md`, the
2026-08-27 brief amendment, and the newest handoff,
`2026-09-02-command-shoot.md`. The special-procedure inventory remains
exhausted. The one-time blocked `objmagic.sleep-entry-gates` row was already
attempted through the cast-sleep outlaw/reagent vehicle and was not repicked.
The interpreter sweep consumed the next source-order family, `accuse`, at
`src/interpreter.c:330`.

The pre-slice frontier was 3,331 total cases, with 3,234 proven/delegated, 44
blocked, and 53 excluded. The accuse manifest contributes 11 cases: seven
proven/delegated cases and four blocked cases. The resulting frontier is:

- 3,342 total cases
- 3,241 proven/delegated
- 48 blocked
- 53 excluded

Actionable completion is 3,241/3,289 = 98.5%.

## C call path and behavior surface

The registered row is:

```c
/* src/interpreter.c:330 */
{ "accuse"   , POS_SITTING , do_action   , 0, 0 },
```

The handler is `src/act.social.c:102-151`. `do_action` first resolves the
social record and PLR_NOSHOUT gate, then uses `one_argument` only when
`char_found` is non-NULL. The accuse record in
`lib/misc/socials:1-9` has hide=0, minimum victim position POS_RESTING, an
actor-only no-argument message with a `#` room sentinel, target actor/room/
victim templates, an exact not-found response, and self-target actor/room
templates. A non-self target below POS_RESTING gets the proper-position line
before any audience act. The shared visible-room lookup, Act audience rules,
and noshout gate are owned by existing social manifests.

## Evidence and proof boundary

The target vehicle uses three players so actor, target, and distinct observer
audiences are all live within the differential harness's reliable connection
envelope. The sleeping vehicle uses an actor and a sleeping player. The
sleeping-target branch is green at seeds 1, 2, 3, 5, and 8, with seed 1 run
using `--show-oracle`. Self-target and not-found blocks are also coherent in
the target vehicle across the retained seed matrix.

Two honest no-argument attempts, including isolated runs at seeds 1 and 2,
return non-text pointer-like bytes from the C actor block instead of the
source literal `Accuse who??`. Two ordinary-target attempts at seeds 1 and 2
consistently suppress C's actor `TO_CHAR` line while C still emits coherent
observer and victim lines. The source call path says those actor messages
should be delivered. These are unresolved C-oracle/runtime anomalies, not
confirmed Go divergences; the four affected no-argument, target, audience,
and first-token rows remain `blocked` after the two-attempt rule. Go does
not copy non-text or suppressed output because that would violate R4 and
R5e.

The durable evidence is:

- `cmd/dp-oracle-diff/scenarios/accuse-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/accuse-noarg-depth.txt`;
- `cmd/dp-oracle-diff/scenarios/accuse-sleeping-depth.txt`;
- `docs/fidelity/depth/accuse.tsv`; and
- `pkg/session/accuse_depth_test.go`.

No Go behavior change was confirmed or made. The focused test pins the
POS_SITTING command entry and the authored social metadata; shared position,
noshout, visibility, and Act behavior are delegated to their existing owners.

## Verification and integration

The required local gates passed on `glm/depth-accuse`:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature commit: `782948191`.

Feature PR: #1173 (`glm/depth-accuse`). Hosted lint, security, and test
checks completed green; conditional build-and-push and deploy jobs were
skipped. CI fired normally, so no workflow retry was used. The PR was
self-merged only after all applicable checks were green. The resulting
`main` merge commit is `c64a9afc6`.

This round follows R1/R2 (player-facing bytes and command surface), R3
(multi-seed evidence), R4 (no invented output), R5/R5e (the actual C call
path), and R5b/R5c (shared social gate/lookup ownership and whole-class
review).

## Continuation

The source-order audit now finds `agree` as the next unclaimed interpreter
token:

```c
/* src/interpreter.c:331 */
{ "agree"    , POS_RESTING , do_action   , 0, 0 },
```

The next session must return to clean `main`, pull, rerun
`make fidelity-depth`, reread the guide and this handoff, then map and prove
`agree` before advancing in interpreter-table order.
