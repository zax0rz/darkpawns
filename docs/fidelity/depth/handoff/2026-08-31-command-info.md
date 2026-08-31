# Depth-fidelity handoff: `info`

Date: 2026-08-31

## Queue position and frontier

This session continued the source-order `src/interpreter.c` command-family
sweep after the special-procedure inventory and the one-time
`objmagic.sleep-entry-gates` attempt. The pre-slice frontier was 2,277 total
cases: 2,217 proven/delegated, 16 blocked, and 44 excluded (2,217 of 2,233
actionable cases, 99.3%). The six new `info` manifest rows bring the frontier
to 2,283 total: 2,223 proven/delegated, 16 blocked, and 44 excluded (2,223 of
2,239 actionable cases, 99.3%).

The special-procedure inventory remains exhausted and the explicitly blocked
`objmagic.sleep-entry-gates` row remains blocked and was not repicked. The
`info` feature slice was PR #970 from collision-safe branch
`glm/depth-info-20260831`, merged to `main` as `9456d9e0d` after all hosted
checks were green. No non-green PR was merged.

The next source-order unclaimed command is `infobar` at
`src/interpreter.c:517`.

## C call path and branch inventory

The registration is:

```text
src/interpreter.c:516: { "info", POS_SLEEPING, do_gen_ps, 0, SCMD_INFO }
```

`src/act.informative.c:2117-2133` dispatches `SCMD_INFO` to
`page_string(ch->desc, info, 0)`. The `info` buffer is loaded during boot at
`src/db.c:321` from `INFO_FILE`, defined as `text/info` in `src/db.h:55`.
The existing reload paths are `src/db.c:194` (reload all text) and
`src/db.c:215` (reload the specific info text). The authoritative
`lib/text/info` is 46 lines and 2,054 bytes, producing three pages with the
current `PAGE_LENGTH=22` path.

The audited player-visible cases are:

- the all-level `POS_SLEEPING` entry gate;
- the exact static three-page `text/info` output;
- ignored trailing arguments;
- RETURN-driven pager navigation and page boundaries; and
- shared pager quit and post-pager command behavior, delegated to
  `help.pager-quit` and `help.post-pager-command`.

`do_gen_ps` ignores the command arguments. The paging call path is
`src/modify.c:355-448,454-527`, with prompt and descriptor handling in
`src/comm.c:617-618,1042-1056`. R1/R2/R3/R4 and C-first call-path rule R5e
apply; shared pager behavior is delegated rather than duplicated under this
caller per R5b/R5c.

## RED/GREEN result

This was a pure-coverage round. The branch started from clean `main`, and the
Go behavior diff was empty before the proof run. The unchanged path was
therefore GREEN on main-equivalent code:

- `info-depth --show-oracle --seed 1`: no normalized divergence; the intended
  `SCMD_INFO` block executed and the exact three-page output and pager prompt
  boundaries matched.
- `info-depth` is also green at seeds 2, 3, 5, and 8.
- `TestInfoRegistrationUsesCEntryGate` passes.

The durable proof is
`cmd/dp-oracle-diff/scenarios/info-depth.txt`,
`pkg/session/info_test.go`, and `docs/fidelity/depth/info.tsv`. No file under
`src/` or `darkpawns-c-oracle/` was edited.

## Verification

The complete local gates passed: `make fidelity-depth`, `go build ./...`,
`go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and a clean
`gofumpt -l .` check. The hosted `test`, `lint`, and `security` checks for
PR #970 were green before merge; build and deploy were skipped by repository
workflow policy.

## Next session

Return to clean `main`, pull, rerun `make fidelity-depth`, reread the depth
testing guide and this newest handoff, then take only the unclaimed `infobar`
family at `src/interpreter.c:517`. Continue the command-table sweep in source
order with one slice and one PR.
