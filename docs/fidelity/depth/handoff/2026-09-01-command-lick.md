# Depth-fidelity handoff — `lick` — 2026-09-01

## Queue position

This session started from clean `main` at the post-`listen` handoff frontier,
ran `git pull --ff-only`, confirmed `make fidelity-depth`, and reread
`docs/fidelity/DEPTH_TESTING.md`, the standing brief, and
`docs/fidelity/depth/handoff/2026-09-01-command-listen.md`.

The special-procedure inventory remains exhausted and the one blocked
`objmagic.sleep-entry-gates` row remains deferred; neither was repicked. The
source-order sweep reached `lick` at `src/interpreter.c:541`, immediately after
the already claimed `listen` row at line 540. The next unmanifested command is
`lines` at `src/interpreter.c:542`.

## C call path and branch inventory

The C registration is `{ "lick", POS_RESTING, do_action, 0, 0 }` in
`src/interpreter.c:541`. `comm.c:910-947` applies the POS_RESTING and zero
level gates before invoking `ACMD(do_action)` in `src/act.social.c:102-151`.
The record at `lib/misc/socials:460-468` has zero hide and victim-position
metadata and these eight authored slots:

- no argument: `You lick your mouth and smile.` / room `$n licks $s mouth and smiles.`;
- visible target: `You lick $M.` / room `$n licks $N.` / victim `$n licks you.`;
- not found: `Lick away, nobody's here with that name.`;
- self target: `You lick yourself.` / room `$n licks $mself -- YUCK.`.

The reachable proof branches are no argument, visible target, first-token
argument parsing with trailing input ignored, self target, missing target, and
a sleeping visible target admitted by the zero victim-position minimum. Shared
noshout, visibility, and command-position behavior remain delegated to their
existing manifests. A first fixture attempt used the command token in a
character name and hit the C invalid-name list; the vehicle was corrected to
the safe names `Mordecai` and `Sable` before any fidelity result was recorded.

## Proof and implementation

Added `cmd/dp-oracle-diff/scenarios/lick-depth.txt` and
`lick-sleeping-depth.txt`, with `# depth-case:` annotations for the actor,
room, victim, self, not-found, argument parsing, and sleeping-target paths.
The focused `pkg/session/lick_depth_test.go` pins the C entry gate, social
metadata, and all eight message slots. Added 11 manifest rows in
`docs/fidelity/depth/lick.tsv`.

This was a pure-coverage round: the existing Go social implementation was
already byte-faithful, so no Go behavior change was made. Both vehicles were
run with `--show-oracle` and reached the intended C blocks. The depth and
sleeping-target vehicles were oracle-green at seeds 1, 2, 3, 5, and 8.
No `src/` or `darkpawns-c-oracle/` file was edited.

## Durable proof and verification

Post-slice frontier: `2467 total, 2406 proven/delegated, 17 blocked, 44
excluded`; actionable completion is `2406/2423 = 99.3%`, and `do_action` is
`471/471`.

The feature branch passed all local gates:

- `make fidelity-depth`;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` (`0 issues`);
- `gofumpt -l .` clean and `git diff --check` clean.

Feature PR #1005 (`glm/depth-lick-20260901`) was initially missing checks.
The one permitted exact retry, `gh workflow run "Dark Pawns CI/CD" --ref
glm/depth-lick-20260901`, started run `33470329821`; required `lint`,
`security`, and `test` checks finished green, with optional build/deploy
skipped by workflow policy. PR #1005 was then self-merged at main commit
`f68e230e4`. No merge was performed while a required check was pending.

This note is the required dated handoff for the session. The slice follows
R1/R2/R3/R4/R5e; shared social behavior is bounded under R5b/R5c.
