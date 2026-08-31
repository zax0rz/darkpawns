# Depth-fidelity handoff: `immlist`

Date: 2026-08-31

## Queue position and frontier

This session continued the source-order `src/interpreter.c` command-family
sweep after the special-procedure inventory and the one-time
`objmagic.sleep-entry-gates` attempt. The pre-slice frontier was 2,269 total
cases: 2,209 proven/delegated, 16 blocked, and 44 excluded (2,209 of 2,225
actionable cases, 99.3%). The three new `immlist` manifest rows bring the
frontier to 2,272 total: 2,212 proven/delegated, 16 blocked, and 44 excluded
(2,212 of 2,228 actionable cases, 99.3%).

The special-procedure inventory remains exhausted and the explicitly blocked
`objmagic.sleep-entry-gates` row remains blocked and was not repicked. The
`immlist` feature slice was PR #966 on `glm/depth-immlist`, merged to `main` as
`ad397dd7b` after all hosted checks were green. No non-green PR was merged.

The next source-order unclaimed command is `inactive` at
`src/interpreter.c:515`.

## C call path and branch inventory

The registration is:

```text
src/interpreter.c:514: { "immlist", POS_DEAD, do_gen_ps, 0, SCMD_IMMLIST }
```

`src/act.informative.c:2117-2144` dispatches `SCMD_IMMLIST` to
`page_string(ch->desc, immlist, 0)` without inspecting the argument. During
`boot_db`, `src/db.c:323` loads `IMMLIST_FILE`, defined as `text/immlist` in
`src/db.h:57`; the same pointer is refreshed by the C reboot paths at
`src/db.c:175-176` and `187-203`. The authored `lib/text/immlist` is a
two-line, 90-byte file, so it fits one pager page and its ordinary success
path sends the exact bytes without a pager prompt. Every level is admitted by
the command row, subject only to `POS_DEAD`.

The audited player-visible cases are:

- the all-level `POS_DEAD` entry gate;
- the exact static page and its actor audience; and
- ignored trailing command arguments.

No separate pager branch is reachable from this fixed short file. R1/R2/R3/R4
and the C-first call-path rule R5e apply. Shared static-file loading and pager
behavior remain owned by the existing `do_gen_ps` and help/page evidence under
R5b/R5c; this slice does not duplicate those shared branches.

## RED/GREEN result

This was a pure-coverage round. The branch started from clean `main`, and the
Go implementation diff was empty before the proof run. The unchanged path was
therefore GREEN on main-equivalent code:

- `immlist-depth --show-oracle --seed 1`: no normalized divergence; bare and
  trailing-argument probes match the exact C page including leading spaces.
- `immlist-depth` is also green at seeds 2, 3, 5, and 8.
- `TestImmlistRegistrationUsesCEntryGate` passes.

The durable proof is `cmd/dp-oracle-diff/scenarios/immlist-depth.txt`,
`pkg/session/immlist_test.go`, and `docs/fidelity/depth/immlist.tsv`. No file
under `src/` or `darkpawns-c-oracle/` was edited.

## Verification

The complete local gates passed: `make fidelity-depth`, `go build ./...`,
`go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and a clean
`gofumpt -l .` check. The hosted `test`, `lint`, and `security` checks for
PR #966 were green before merge; build and deploy were skipped by repository
workflow policy.

## Next session

Return to clean `main`, pull, rerun `make fidelity-depth`, reread the depth
testing guide and this newest handoff, then take only the unclaimed `inactive`
family at `src/interpreter.c:515`. Continue the command-table sweep in source
order with one slice and one PR.
