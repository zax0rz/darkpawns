# Depth-fidelity handoff — `stare`

Date: 2026-09-03

Branch: `glm/depth-stare` (feature merged); handoff branch:
`handoff/2026-09-03-command-stare`

Feature PR: #1287 (merged green)

Feature commit: `f7bab0f7a`

Feature merge commit: `7820031e8`

## Queue position and result

This round checked out `main`, pulled with `git pull --ff-only`, ran
`make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md` and the newest
dated handoff, and audited the interpreter table in source order. The special-
procedure inventory remains exhausted. The one blocked row,
`objmagic.sleep-entry-gates`, remains blocked after its one allowed cast-sleep
outlaw/reagent vehicle and was not repicked.

The interpreter sweep consumed the next genuinely un-manifested `do_action`
social after `squeeze`: `stare` at `src/interpreter.c:736`. Its depth vehicle
and manifest are complete; the feature PR was self-merged only after all
applicable GitHub checks were green. `stand`, `stake`, and `stable` are already
represented by existing depth manifests. The next genuinely un-manifested
source-order family is `startle` at `src/interpreter.c:737`.

Pre-stare frontier: 3,900 total, 3,796 proven/delegated, 48 blocked, and 56
excluded. The stare manifest adds 11 proven/delegated cases. Post-stare
frontier: 3,911 total, 3,807 proven/delegated, 48 blocked, and 56 excluded;
actionable completion is 3,807/3,855 = 98.8%.

## C call path and observable contract

The registered C row is:

```c
/* src/interpreter.c:736 */
{ "stare"    , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` resolves the social record, checks
`PLR_NOSHOUT`, consumes only the first target token with `one_argument` when a
target is present, and then selects no-argument, missing-target, self-target,
proper-position refusal, or target-success branches. The `stare` record at
`lib/misc/socials:844-852` is `stare 0 5`: no command minimum, no hidden flag,
and a POS_RESTING victim minimum, followed by eight authored message slots.

The reachable player-visible branches are:

- no argument: the authored actor and room sky-stare lines;
- visible player at or above resting: authored actor, non-victim observer, and
  victim lines;
- visible NPC target: the same target audiences with NPC substitution;
- leading fill word and trailing words: only the first target token is used;
- self target: the authored actor and room self-target lines;
- missing visible target: the direct-send refusal with no room act; and
- sleeping visible target: the POS_RESTING victim minimum returns the exact
  proper-position actor message and no room/victim act.

The command POS_RESTING gate and `PLR_NOSHOUT` refusal are shared social
vehicles owned by existing manifests under R5b/R5c; the sleeping-target probe
also proves this social's distinct victim-position gate.

## Evidence and implementation boundary

The source-equivalent main proof was GREEN at seed 1 with `--show-oracle`; the
normalized oracle blocks exposed every annotated branch, including the NPC and
sleeping-target cases. Final `stare-depth` runs at seeds 1, 2, 3, 5, and 8 all
reported `result: no normalized divergence`. No Go product behavior changed;
this was a pure depth-coverage slice.

Durable evidence:

- `cmd/dp-oracle-diff/scenarios/stare-depth.txt`;
- `docs/fidelity/depth/stare.tsv`; and
- `pkg/session/stare_depth_test.go`.

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

PR #1287's hosted lint, security, and full test checks completed green;
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
`startle` at `src/interpreter.c:737` before touching any implementation. Do
not repick `stare` or any earlier claimed family.
