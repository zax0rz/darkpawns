# Depth-fidelity handoff — `reallyquit`

Date: 2026-09-01

## Queue position

This round began from clean `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md` plus the
latest `redit` handoff. The special-procedure inventory remains exhausted, the
one blocked row `objmagic.sleep-entry-gates` remains queued after its single
cast-sleep vehicle, and the interpreter sweep advanced from `redit` to
`reallyquit`.

The source-order audit did not repick `retreat` (owned by `escape`),
`retrieve` (owned by `spec-procs`), or the other shared/registered families
already claimed in this interval. The next unclaimed interpreter-table family
is `review` at `src/interpreter.c:658`.

Frontier before this slice: 2,986 total; 2,909 proven/delegated; 26 blocked;
51 excluded.

Frontier after this slice: 2,994 total; 2,917 proven/delegated; 26 blocked;
51 excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:657 */
{ "reallyquit",POS_DEAD    , do_quit     , 0, SCMD_REALLY_QUIT },
```

`src/act.other.c:72-181` shares `do_quit` with `quit`, but the
`SCMD_REALLY_QUIT` subcommand bypasses the unsafe-room refusal and still takes
the common fighting and incapacitation gates. A successful call emits the
canonical visible leave act to the room, `Goodbye, friend.. Come back soon!\r\n`
to the actor, turns off the infobar, closes duplicate descriptors, and extracts
the player. In a safe room the existing rent-save path retains equipment; in
an unsafe room C skips `Crash_rentsave`, so the final saved record loses
equipment. The handler does not parse arguments.

## Evidence and integration

Scenario: `cmd/dp-oracle-diff/scenarios/reallyquit-depth.txt`

Manifest: `docs/fidelity/depth/reallyquit.tsv` (eight rows)

The implementation was already faithful. Seed 1 with `--show-oracle` and seeds
1, 2, 3, 5, and 8 all reported no normalized divergence. The live forced-peer
vehicle captured the primary's `Okay.` force acknowledgement, the peer's exact
goodbye, and one observer leave act; focused existing tests cover the separate
subcommand registration, unsafe equipment loss, safe equipment retention, and
shared fighting behavior. Shared entry/fighting gates are delegated to the
`quit` manifest under R5b/R5c. No Go, `src/`, or C-oracle files were changed.

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

Feature branch: `glm/depth-reallyquit`

Feature commit: `60b92044c` (`test: prove reallyquit depth fidelity (R1/R2/R3/R5e)`)

Feature PR: #1117 — the initial check report was empty, so the one permitted
exact workflow retry was used; hosted lint, security, and test checks then
passed. Build-and-push and deploy were skipped by workflow conditions. It was
self-merged as main commit `fcdce50d4` only after the required hosted checks
were green.

The earlier open PRs for `plot`, `purge`, and `qecho` remain open because their
checks did not fire after their one permitted exact workflow retry; none was
merged.

## Fidelity rules

This pure-coverage slice follows R1 (player-facing bytes), R2 (command
surface), R3 (determinism), R4 (no invention), R5 (process discipline), and
R5e (verify the actual C call path). Shared `do_quit` ownership is maintained
under R5b/R5c.
