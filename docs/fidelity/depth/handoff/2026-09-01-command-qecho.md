# Depth-fidelity handoff — `qecho`

Date: 2026-09-01

## Queue position

This round began from clean `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md` plus the
prior `purge` handoff. The special-procedure inventory remains exhausted, the
one blocked row `objmagic.sleep-entry-gates` remains queued after its single
cast-sleep vehicle, and the interpreter sweep advanced from `purge` to
`qecho`.

`qecho` is now claimed by this handoff and must not be repicked. The next
unmanifested interpreter family is `qui` at `src/interpreter.c:629`.
`quaff` is already represented by its existing depth manifests, and `quest`
is covered by `gen-tog`; neither was repicked.

Frontier on `main` before this slice: 2,904 total; 2,833 proven/delegated; 22
blocked; 49 excluded.

Feature-branch frontier after this slice: 2,915 total; 2,844 proven/delegated;
22 blocked; 49 excluded. The main frontier remains at the prior count because
the feature PR is still open pending hosted checks.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:627 */
{ "qecho"    , POS_DEAD    , do_qcomm    , LVL_IMMORT, SCMD_QECHO },
```

`src/act.comm.c:1301-1360` first requires `PRF_QUEST`, then rejects
`PLR_NOSHOUT`, then skips leading argument whitespace. An empty qecho formats
`CMD_NAME` into the exact `Qecho?  Yes, fine, qecho we must, but WHAT??` line.
For non-empty input, C deletes ANSI markers, sends the actor either the raw
message through `act()` or `Okay.` when `PRF_NOREPEAT` is set, then sends the
capitalized message through `TO_VICT|TO_SLEEP` to every other questing,
non-writing descriptor. The recipient loop is global rather than room-bound;
sleeping quest participants still receive the message, while non-questing and
writing descriptors do not.

## Evidence and confirmed divergences

Scenario: `cmd/dp-oracle-diff/scenarios/qecho-depth.txt`

Manifest: `docs/fidelity/depth/qecho.tsv` (11 rows)

Focused tests: `pkg/session/qecho_depth_test.go`

Clean-main RED probes confirmed that Go collapsed internal/trailing spacing,
left ANSI markers in place, omitted C `act()` first-letter capitalization,
and echoed the actor's text despite `PRF_NOREPEAT`. The Go path now retains the
transport raw argument remainder, deletes C color markers, applies C
capitalization, and substitutes `Okay.` only for the actor's norepeat copy.
The shared qcomm recipient loop also excludes `PLR_WRITING`, as required by
the same C call path. No `src/` or C-oracle file was edited.

The vehicle matched C with `--show-oracle` and at seeds 1, 2, 3, 5, and 8,
with no normalized divergence. It proves the pre-quest and noshout gates, the
empty command, quest/non-quest global audience, raw spacing, ANSI deletion,
norepeat split, and sleeping quest-recipient path; the focused unit test proves
the writing-recipient filter.

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

Feature branch: `glm/depth-qecho`

Feature commit: `afe233625` (`fix: match qecho depth behavior (R1/R2/R3/R5e)`)

Feature PR: #1096 — OPEN and unmerged. No checks were reported, so the one
permitted exact retry was performed:

```text
gh workflow run "Dark Pawns CI/CD" --ref glm/depth-qecho
```

Per the standing procedure, this PR is treated as not-green and remains open;
it must not be merged without green hosted checks. The prior plot handoff PR
#1064 and purge handoff PR #1095 likewise remain open because their checks did
not fire after their permitted retries.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(determinism and ordering), R4 (no invention), R5 (process discipline), and
R5e (verify the actual C call path). The shared qcomm recipient behavior and
source-order claim are maintained under R5b/R5c.
