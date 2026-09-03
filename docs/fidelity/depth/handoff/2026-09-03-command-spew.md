# Depth-fidelity handoff — `spew`

Date: 2026-09-03

Branch: `glm/depth-spew` (feature merged); handoff branch:
`handoff/2026-09-03-command-spew`

Feature PR: #1279 (merged green)

Feature commit: `66f208747`

Main merge: `1959d5870`

## Queue position and result

This round checked out `main`, pulled with `git pull --ff-only`, ran
`make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md` and the newest
dated handoff, and audited the interpreter table in source order. The special-
procedure inventory remains exhausted. The one blocked row,
`objmagic.sleep-entry-gates`, remains blocked after its one allowed cast-sleep
outlaw/reagent vehicle and was not repicked.

The next unclaimed interpreter-table family after `spank` was the `do_action`
social `spew` at `src/interpreter.c:729`. Its depth vehicle and manifest are
complete; the feature PR was self-merged only after all applicable GitHub
checks were green. The following source-order family, `spike` at
`src/interpreter.c:730`, has since been completed in feature PR #1280. The
next fresh queue boundary must therefore claim `spit` at
`src/interpreter.c:731` and must not repick `spew` or `spike`.

Pre-spew frontier: 3,848 total, 3,745 proven/delegated, 48 blocked, and 55
excluded. The spew manifest adds 11 proven/delegated cases. Post-spew
frontier: 3,859 total, 3,756 proven/delegated, 48 blocked, and 55 excluded;
actionable completion is 3,756/3,804 = 98.7%.

## C call path and observable contract

The registered C row is:

```c
/* src/interpreter.c:729 */
{ "spew"     , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` resolves the social record, checks
`PLR_NOSHOUT`, consumes only the first target token with `one_argument` when a
target is present, and then selects no-argument, missing-target, self-target,
proper-position refusal, or target-success branches. The `spew` record at
`lib/misc/socials:814-822` is `spew 0 0`, with no command minimum, no hidden
flag, the default victim-position minimum, and eight authored message slots.

The reachable player-visible branches are:

- no argument: the authored actor and room thin-air lines;
- visible target: authored actor, non-victim observer, and victim lines;
- leading fill word and trailing words: only the first target token is used;
- self target: the authored self-target actor and room lines; and
- missing visible target: the direct-send refusal with no room act.

The command POS_RESTING gate, `PLR_NOSHOUT` refusal, visible lookup, and shared
audience/visibility mechanics are delegated to the existing social vehicles
under R5b/R5c.

## Evidence and implementation boundary

The clean-main proof was GREEN at seed 1 with `--show-oracle`. Final
`spew-depth` runs at seeds 1, 2, 3, 5, and 8 all reported
`result: no normalized divergence`. No Go product behavior changed; this was
a pure depth-coverage slice.

Durable evidence:

- `cmd/dp-oracle-diff/scenarios/spew-depth.txt`;
- `docs/fidelity/depth/spew.tsv`; and
- `pkg/session/spew_depth_test.go`.

No file under `src/` or `darkpawns-c-oracle/` was edited.

## Gates and review

The final local gates passed:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...`
- `gofumpt -l .` clean
- `git diff --check`

PR #1279's hosted lint, security, and full test checks completed green;
conditional build-and-push and deploy were skipped. CI fired normally, so no
workflow retry was used. The PR was self-merged only after all applicable
checks were green, per the 2026-08-27 amendment.

This slice follows R1 (player-facing bytes), R2 (registered command surface),
R3 (deterministic seed and audience parity), R4 (no invented behavior),
R5/R5e (verify the actual C path and let C win), and R5b/R5c (shared social
gate, lookup, and audience ownership).

## Continuation

The next session must checkout `main`, pull with `--ff-only`, rerun
`make fidelity-depth`, reread the guide and newest handoff, and audit/claim
`spit` at `src/interpreter.c:731` before touching any implementation. Do not
repick `spew`, `spike`, or any earlier claimed family.
