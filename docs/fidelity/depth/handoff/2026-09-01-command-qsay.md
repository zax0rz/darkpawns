# Depth-fidelity handoff — `qsay`

Date: 2026-09-01

## Queue position

This round began from clean `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md` plus
the latest `quit` handoff. The special-procedure inventory remains exhausted,
the one blocked row `objmagic.sleep-entry-gates` remains queued after its
single cast-sleep vehicle, and the interpreter sweep advanced from `quit` to
`qsay`.

The next unmanifested interpreter family is `report` at
`src/interpreter.c:634`. The claimed `qecho` family and its open feature and
handoff PRs must not be repicked. `quaff` and `quest` are already represented
by existing depth manifests; `rest` is covered by the position manifest and
the social rows are owned by the shared social manifest.

Frontier before this slice: 2,926 total; 2,853 proven/delegated; 22 blocked;
51 excluded.

Frontier after this slice: 2,937 total; 2,864 proven/delegated; 22 blocked;
51 excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:631 */
{ "qsay"     , POS_SLEEPING, do_qcomm    , 0, SCMD_QSAY },
```

`src/act.comm.c:1301-1360` first checks `PRF_QUEST`, then `PLR_NOSHOUT`,
before skipping leading argument whitespace. Empty input reaches the
capitalized `Qsay?  Yes, fine, qsay we must, but WHAT??` response. Non-empty
input runs `delete_ansi_controls` on the argument, builds the distinct
SCMD_QSAY actor and recipient templates with the `&W`/`&n` markers, sends the
actor copy unless `PRF_NOREPEAT` changes it to `Okay.`, and sends the recipient
copy to every other questing descriptor. The recipient loop includes sleeping
players, excludes `PLR_WRITING`, and does not apply a room-soundproof filter.

## Evidence and confirmed divergences

Scenario: `cmd/dp-oracle-diff/scenarios/qsay-depth.txt`

Manifest: `docs/fidelity/depth/qsay.tsv` (11 rows)

Focused tests: `pkg/session/qsay_depth_test.go`, supplemented by the existing
communication and preference tests.

The clean-main RED exposed four confirmed divergences: Go collapsed internal
and trailing argument spacing, omitted C's qsay `&W`/`&n` template bytes,
retained the argument's ampersand color markers, and repeated the actor copy
despite `PRF_NOREPEAT`. The fix adds a raw-argument qsay path, C's argument
cleanup and templates, and the shared writing-recipient gate. The corrected
vehicle was GREEN at seeds 1, 2, 3, 5, and 8; seed 1 was run with
`--show-oracle` to confirm the intended C block. No `src/` or C-oracle file
was edited.

## Verification and integration

All local gates passed on the feature branch:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
```

Feature branch: `glm/depth-qsay`

Feature commit: `c1525887a` (`fix: prove qsay depth fidelity (R1/R2/R3/R5e)`)

Feature PR: #1102 — merged as `9421251d9`. Hosted security, lint, and race/e2e
test checks were green in run `33571509876`; build/deploy were skipped by
workflow conditions. The PR was merged only after the hosted required checks
were green.

The `qecho` feature PR #1096 and handoff PR #1097, purge handoff PR #1095,
and plot handoff PR #1064 remain open because their checks did not fire after
their one permitted exact workflow retries; none was merged.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(determinism and ordering), R4 (no invention), R5 (process discipline), R5b
and R5c (shared behavior ownership), and R5e (verify the actual C call path).
The source-order claim is maintained for the next `report` slice.
