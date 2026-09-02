# Depth-fidelity handoff — `rlist`

Date: 2026-09-01

## Queue position

This round began from clean `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md` plus the
latest `rin` handoff. The special-procedure inventory remains exhausted, the
single blocked row `objmagic.sleep-entry-gates` remains queued after its one
cast-sleep vehicle, and the interpreter sweep advanced from `rin` through the
shared `ride` row. The source-order audit confirms that `ride` is already
owned by the `mount` manifest and `roomflags` is already owned by `gen-tog`.
The next unclaimed interpreter-table family is `roar` at
`src/interpreter.c:662`.

Frontier before this slice: 3,015 total; 2,938 proven/delegated; 26 blocked;
51 excluded.

Frontier after this slice: 3,023 total; 2,946 proven/delegated; 26 blocked;
51 excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:661 */
{ "rlist"     , POS_DEAD, do_rlist, LVL_BUILDER, 0 },
```

`src/act.wizard.c:3336-3366` implements `do_rlist`. It consumes only the
first `one_argument()` token, converts it with C `atoi` (including a decimal
prefix), scans the world table in order for rooms whose zone number matches,
and builds rows in the shape `%3d. [%5d] %s\r\n`. An absent zone sends
`The desired zone does not exist.\r\n`; an overlong buffer sends
`Truncating room list due to size.\r\n` and retains the prefix. The final
buffer routes through `page_string`, so output is actor-only and uses the
shared pager. The command is available at the C builder level and `POS_DEAD`.

## Evidence and confirmed divergence

Scenario: `cmd/dp-oracle-diff/scenarios/rlist-depth.txt`

Manifest: `docs/fidelity/depth/rlist.tsv` (8 rows)

Focused tests: `pkg/session/rlist_depth_test.go`

The clean-main RED scenario found that Go had invented a required keyword,
used substring matching and a 50-row cap, emitted `No rooms found.`, and
formatted rows without C's ordinal. The corrected implementation uses the
numeric zone argument, C's decimal-prefix conversion, world order, exact
row/error/warning bytes, the 8192-byte C buffer limit, and the existing
pager. No `src/` or C-oracle file was edited.

The corrected scenario is GREEN with `--show-oracle` and across seeds 1, 2,
3, 5, and 8. It covers absent zones, `999abc`, ignored trailing words, a
populated zone, and bare `rlist` (zone 0).

## Verification and integration

All required local gates passed on the feature branch:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature branch: `glm/depth-rlist`

Feature commit: `36405afc5` (`fix: match C rlist zone output`)

Feature PR: #1123 — hosted lint, security, and test checks were green; the
workflow's build-and-push and deploy jobs were skipped by conditions. The
automatic workflow did not initially report checks, so the one permitted
exact retry `gh workflow run "Dark Pawns CI/CD" --ref glm/depth-rlist` was
used. The PR was self-merged as main commit `9d005d2deb50` only after all
required hosted checks were green.

The earlier open PRs for `plot`, `purge`, and `qecho` remain open because
their checks did not fire after their one permitted exact workflow retry; none
was merged.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(determinism across the oracle matrix), R4 (no invention), R5 (process
discipline), and R5e (verify the actual C call path). Numeric argument
translation, world-order rendering, and pager ownership remain under
R5b/R5c.
