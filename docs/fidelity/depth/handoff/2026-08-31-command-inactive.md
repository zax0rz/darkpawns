# Depth-fidelity handoff: `inactive`

Date: 2026-08-31

## Queue position and frontier

This session continued the source-order `src/interpreter.c` command-family
sweep after the special-procedure inventory and the one-time
`objmagic.sleep-entry-gates` attempt. The pre-slice frontier was 2,272 total
cases: 2,212 proven/delegated, 16 blocked, and 44 excluded (2,212 of 2,228
actionable cases, 99.3%). The five new `inactive` manifest rows bring the
frontier to 2,277 total: 2,217 proven/delegated, 16 blocked, and 44 excluded
(2,217 of 2,233 actionable cases, 99.3%).

The special-procedure inventory remains exhausted and the explicitly blocked
`objmagic.sleep-entry-gates` row remains blocked and was not repicked. The
`inactive` feature slice was PR #968 on `glm/depth-inactive`, merged to `main`
as `15502d384` after all hosted checks were green. No non-green PR was merged.

The next source-order unclaimed command is `info` at
`src/interpreter.c:516`.

## C call path and branch inventory

The registration is:

```text
src/interpreter.c:515: { "inactive", POS_SLEEPING, do_inactive, 0, 0 }
```

`src/act.other.c:1818-1824` contains the complete handler. It only tests and
flips `PRF_INACTIVE`; it emits no command text and does not read the argument.
The player-visible effect occurs in the command/prompt cycle:
`src/comm.c:1028-1157` renders `INACTIVE > ` when the preference is set and
returns to the ordinary `> ` prompt after it is cleared. The command is
admitted at `POS_SLEEPING` and above, with no level gate.

The audited player-visible cases are:

- the all-level `POS_SLEEPING` entry gate;
- enabling the preference and its inactive prompt framing;
- disabling the preference and restoration of the ordinary prompt; and
- ignored command arguments, including the state transition they do not alter.

The state bit is also consulted by the periodic limits path
(`src/limits.c:476-502`) for condition/resource updates, but that is a
heartbeat behavior owned by the broader status/limits evidence rather than a
new direct command branch. R1/R2/R3/R4 and the C-first call-path rule R5e
apply; shared prompt behavior is not duplicated under R5b/R5c.

## RED/GREEN result

This was a pure-coverage round. The branch started from clean `main`, and the
Go implementation diff was empty before the proof run. The unchanged path was
therefore GREEN on main-equivalent code:

- `inactive-depth --show-oracle --seed 1`: no normalized divergence; C emits
  no command text, exposes `INACTIVE > ` after enable, and returns to the
  ordinary prompt after the argument-bearing disable.
- `inactive-depth` is also green at seeds 2, 3, 5, and 8.
- `TestInactiveRegistrationUsesCEntryGate` passes.
- `TestDoInactiveToggleState` passes.

The durable proof is `cmd/dp-oracle-diff/scenarios/inactive-depth.txt`,
`pkg/session/inactive_test.go`, `pkg/game/inactive_test.go`, and
`docs/fidelity/depth/inactive.tsv`. No file under `src/` or
`darkpawns-c-oracle/` was edited.

## Verification

The complete local gates passed: `make fidelity-depth`, `go build ./...`,
`go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and a clean
`gofumpt -l .` check. The hosted `test`, `lint`, and `security` checks for
PR #968 were green before merge; build and deploy were skipped by repository
workflow policy.

## Next session

Return to clean `main`, pull, rerun `make fidelity-depth`, reread the depth
testing guide and this newest handoff, then take only the unclaimed `info`
family at `src/interpreter.c:516`. Continue the command-table sweep in source
order with one slice and one PR.
