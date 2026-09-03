# Depth handoff — shutdown

Date: 2026-09-02  
Branch: `glm/depth-shutdown`  
Feature PR: #1239 (merged green)  
Feature commit: `f17246d67`  
Main merge: `ecec67268`

## Queue position

The special-procedure inventory remains exhausted. The single blocked row,
`objmagic.sleep-entry-gates`, was attempted once through the cast-sleep
outlaw/reagent vehicle and remains blocked for the unreachable entry gates.
After refreshing `main`, pulling with `--ff-only`, running `make
fidelity-depth`, reading `DEPTH_TESTING.md`, and reviewing the newest dated
handoff (`2026-09-02-command-sharpen.md`), the next unclaimed interpreter
family was the `do_shutdown` pair at `src/interpreter.c:698-699`. Earlier
entries were already manifested or belonged to shared `do_action`,
`do_not_here`, and kuji-kiri classes and were delegated under R5b/R5c.

Pre-slice frontier: 3,673 total, 3,572 proven/delegated, 48 blocked, 53
excluded.

The shutdown slice adds 9 manifest cases. Post-slice frontier: 3,682 total,
3,581 proven/delegated, 48 blocked, 53 excluded; actionable completion is
3,581/3,629 = 98.7%.

The next session must refresh `main`, rerun the frontier, reread the guide and
newest handoff, then continue after `shutdow`/`shutdown` in interpreter-table
order. The fresh source-order audit identifies `skillset` at
`src/interpreter.c:704` as the next genuinely unclaimed family. Confirm that
claim and its actual C path before working it; do not repick existing
manifests or shared-class claims.

## C call path and observable contract

The command table registers:

```c
{ "shutdow"  , POS_DEAD    , do_shutdown , LVL_IMPL-1, 0 },
{ "shutdown" , POS_DEAD    , do_shutdown , LVL_IMPL-1, SCMD_SHUTDOWN }
```

at `src/interpreter.c:698-699`. `do_shutdown` in
`src/act.wizard.c:1074-1118` first checks the subcommand. The abbreviation's
subcommand is not `SCMD_SHUTDOWN`, so it returns
`If you want to shut something down, say so!\r\n` before parsing arguments;
trailing words cannot select a shutdown arm.

The full command parses one option case-insensitively. No option broadcasts
`Shutting down.\r\n`; `reboot` broadcasts
`Rebooting.. come back in a minute or two.\r\n` and touches `../.fastboot`;
`die` and `pause` broadcast `Shutting down for maintenance.\r\n` and touch
their respective `../.killscript` and `../pause` markers. Each accepted arm
sets `circle_shutdown`, with reboot also setting `circle_reboot`. An unknown
option sends only `Unknown shutdown option.\r\n` and does not enter the
save/shutdown continuation.

For each accepted arm C calls `do_force(ch, "all save")`: the actor receives
`Okay.\r\n`, lower-level playing descriptors receive the force notice and the
normal `do_save` output, and `House_save_all()` follows. Go now emits the
authoritative global bytes and save acknowledgement, routes lower-level
players through the existing save path, writes the option marker in the
server lifecycle, and suppresses its generic external-shutdown notice when
the command already supplied C's text.

No `src/` or `darkpawns-c-oracle/` file was edited.

## Proof artifacts

Scenario: `cmd/dp-oracle-diff/scenarios/shutdown-depth.txt`. It exercises the
abbreviation guard with and without trailing arguments, unknown full-command
options, and mixed-case `ReBoOt`. The accepted option arms terminate the
isolated process, so their exact message/marker mappings and lifecycle state
are covered by the focused test rather than included as intermediate live
probe steps.

Manifest: `docs/fidelity/depth/shutdown.tsv` (9 rows).

Focused test: `pkg/session/shutdown_depth_test.go`
(`TestShutdownRegistrationUsesCEntryGates`,
`TestShutdownAbbreviationKeepsCGuardBytes`,
`TestShutdownPlanUsesCGlobalMessages`, and
`TestShutdownCommandBroadcastsCTextAndRequestsLifecycle`).

With `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle`, the scenario
produced `result: no normalized divergence` for seeds 1, 2, 3, 5, and 8;
seed 1 was inspected with `--show-oracle`. The first seed-1 run was RED on
main because recognized mixed-case reboot input emitted invented Go text
instead of C's global reboot notice and `Okay.` acknowledgement. The Go
implementation now matches the confirmed messages, ordering, option markers,
save continuation, and lifecycle request.

## Gates and review

All local gates passed on the final feature commit:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...`
- `gofumpt -l .` (clean)
- `git diff --check`

The full test gate briefly exposed the repository's existing U1000 suppression
ratchet after the first pass removed a file-level annotation; restoring that
annotation kept the ratchet at its expected count, and the final full gate
passed. PR #1239's hosted lint, security, and test checks were green;
build/deploy were skipped by the workflow. It was self-merged only after the
applicable checks were green.

This slice follows R1 (player-facing bytes), R2 (registered command surface),
R3 (ordering and lifecycle state), R4 (no invented behavior), R5 and R5e
(verify the actual C path and let C win), with shared behavior audited under
R5b/R5c.
