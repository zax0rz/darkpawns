# Depth-fidelity handoff — `snowball`

Date: 2026-09-03

Branch: `glm/depth-snowball`

Feature PR: #1265 (merged green)

Feature commit: `ec6d68ec7`

Main merge: `264ca9900`

## Queue position and result

This round returned to `main`, ran `git pull --ff-only`, confirmed the
frontier with `make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md`
and the newest dated handoff, and then audited the interpreter table against
the depth manifests. The special-procedure inventory remains exhausted. The
one blocked row, `objmagic.sleep-entry-gates`, remains blocked after its one
allowed cast-sleep outlaw/reagent vehicle and was not repicked.

The next genuinely unclaimed source-order row after `snowball` is the
dedicated `snoop` handler at `src/interpreter.c:723`. The preceding `snore`
and `snowball` rows are now claimed by their manifests; do not repick them.

Pre-slice frontier: 3,774 total, 3,671 proven/delegated, 48 blocked, and 55
excluded. The snowball manifest adds 9 proven/delegated cases. Post-slice
frontier: 3,783 total, 3,680 proven/delegated, 48 blocked, and 55 excluded;
actionable completion is 3,680/3,728 = 98.7%.

## C call path and observable contract

The registered C row is:

```c
/* src/interpreter.c:722 */
{ "snowball" , POS_STANDING, do_action, LVL_IMMORT, 0 },
```

`do_action` in `src/act.social.c:102-151` resolves the social record,
enforces the shared `PLR_NOSHOUT` gate, parses the first target token, and
dispatches the actor, observer, victim, self, or missing-target branches. The
`snowball` record at `lib/misc/socials:784-794` is `snowball 0 0` and has no
additional hide or victim-position gate.

The reachable player-visible branches proven in this slice are:

- no argument: actor `Who do you want to throw a snowball at??`; the `#`
  room sentinel emits no room bytes;
- visible target: actor `You throw a snowball in $N's face.`, observer
  `$n conjures a snowball from thin air and throws it at $N.`, and victim
  `...throws it at you.`;
- trailing words: only the first target token is resolved;
- self target: actor `You conjure a snowball from thin air and throw it at
  yourself.` and the authored self room line;
- missing target: `You stand with the snowball in your hand because your
  victim is not here.`.

The command-level position, `PLR_NOSHOUT`, visibility, lookup, and audience
mechanics are shared with the existing `flip.position-gate`, `dance-noshout`,
and `socials-depth` evidence and are delegated under R5b/R5c.

## Evidence and implementation boundary

The durable evidence is:

- `cmd/dp-oracle-diff/scenarios/snowball-depth.txt`;
- `docs/fidelity/depth/snowball.tsv`; and
- `pkg/session/snowball_depth_test.go`.

The clean-main oracle vehicle was already GREEN across seeds 1, 2, 3, 5, and
8, with seed 1 inspected using `--show-oracle`. The Go social path and command
gate were already faithful, so this was a pure-coverage round: no source
behavior changed. Inventing a fix would violate R4.

The scenario proves the no-argument, target-success, target-audience,
argument-ignored, self-target, and target-not-found branches. It uses named
actor, target, and observer peers, and preserves the exact C `#` silent-room
sentinel. No file under `src/` or `darkpawns-c-oracle/` was edited.

## Gates and review

The final local gates passed on the feature branch:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...`
- `gofumpt -l .` clean
- `git diff --check`

PR #1265's hosted lint, security, and full test checks completed green;
conditional build-and-push and deploy were skipped. CI fired normally, so no
workflow retry was needed. The PR was self-merged only after the applicable
checks were green, per the 2026-08-27 amendment.

This slice follows R1 (player-facing bytes), R2 (registered command surface),
R3 (seed matrix and audience ordering), R4 (no invented behavior), and R5/R5e
(verify the actual C path and let C win), with R5b/R5c governing delegated
shared social behavior.

## Continuation

The next session must checkout `main`, pull with `--ff-only`, rerun
`make fidelity-depth`, reread the guide and newest handoff, and audit/claim
`snoop` at `src/interpreter.c:723` before touching implementation. Do not
repick `snowball` or its delegated shared cases.
