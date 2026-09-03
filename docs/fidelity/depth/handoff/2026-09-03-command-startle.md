# Depth-fidelity handoff — `startle`

Date: 2026-09-03

Branch: `glm/depth-startle` (feature merged); handoff branch:
`handoff/2026-09-03-command-startle`

Feature PR: #1289 (merged green)

Feature commit: `bb3501e74`

Feature merge commit: `93f92f024`

## Queue position and result

This round checked out `main`, pulled with `git pull --ff-only`, ran
`make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md` and the newest
dated handoff, and audited the interpreter table in source order. The special-
procedure inventory remains exhausted. The one blocked row,
`objmagic.sleep-entry-gates`, remains blocked after its one allowed cast-sleep
outlaw/reagent vehicle and was not repicked.

The interpreter sweep consumed the next genuinely un-manifested `do_action`
social after `stare`: `startle` at `src/interpreter.c:737`. Its depth vehicle
and manifest are complete; the feature PR was self-merged only after all
applicable GitHub checks were green. Several other depth slices were merged to
`main` concurrently while this feature was in review; they were preserved and
not repicked. The newest handoff identifies `smackheads` at
`src/interpreter.c:711` as the next dedicated unclaimed family.

The startle slice advanced its own frontier from 3,911 total, 3,807
proven/delegated, 48 blocked, and 56 excluded to 3,922 total, 3,818
proven/delegated, 48 blocked, and 56 excluded. After the concurrent merged
slices already present on current `main`, the live frontier is 3,989 total,
3,885 proven/delegated, 48 blocked, and 56 excluded; actionable completion is
3,885/3,933 = 98.8%.

## C call path and observable contract

The registered C row is:

```c
/* src/interpreter.c:737 */
{ "startle"  , POS_RESTING , do_action   , 0, 0 },
```

`do_action` in `src/act.social.c:102-151` resolves the social record, checks
`PLR_NOSHOUT`, consumes only the first target token with `one_argument` when a
target is present, and then selects no-argument, missing-target, self-target,
proper-position refusal, or target-success branches. The `startle` record at
`lib/misc/socials:1496-1504` is `startle 0 0`, with no command minimum, no
hidden flag, the default zero victim-position minimum, and eight authored
message slots.

The reachable player-visible branches are:

- no argument: the authored actor and room flinch lines;
- visible player target: authored actor, non-victim observer, and victim lines;
- visible NPC target: the same target audiences with NPC substitution;
- leading fill word and trailing words: only the first target token is used;
- self target: the authored actor and room flinch lines;
- missing visible target: the direct-send refusal with no room act; and
- sleeping visible target: success remains allowed because the victim-position
  minimum is zero, while the sleeping victim's `TO_VICT` bytes are suppressed.

The command POS_RESTING gate, `PLR_NOSHOUT` refusal, visible lookup, and shared
audience/visibility mechanics are delegated to existing social vehicles under
R5b/R5c.

## Evidence and implementation boundary

The source-equivalent main proof was GREEN at seed 1 with `--show-oracle`; the
normalized oracle blocks exposed every annotated branch, including the NPC and
sleeping-target cases. Final `startle-depth` runs at seeds 1, 2, 3, 5, and 8
all reported `result: no normalized divergence`. No Go product behavior
changed; this was a pure depth-coverage slice.

Durable evidence:

- `cmd/dp-oracle-diff/scenarios/startle-depth.txt`;
- `docs/fidelity/depth/startle.tsv`; and
- `pkg/session/startle_depth_test.go`.

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

PR #1289's hosted lint, security, and full test checks completed green;
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
`smackheads` at `src/interpreter.c:711` before touching any implementation.
Do not repick `startle` or any earlier claimed family.
