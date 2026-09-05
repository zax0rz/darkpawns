# Oracle Differential Testing

The oracle harness drives the original C server and the Go port with equivalent
scripted input, captures player-visible transcripts, normalizes explicitly
accepted noise, and diffs the results block by block. A surviving difference is
a confirmed fidelity bug. A green result proves only the behavior exercised by
that scenario.

## Authority and Components

- `src/` and the external `darkpawns-c-oracle/` checkout are read-only ground truth.
- `cmd/dp-oracle-diff/` launches and drives both servers.
- `cmd/dp-oracle-diff/scenarios/` contains the scripts.
- `internal/oraclediff/` parses scenarios, captures blocks, normalizes output,
  and reports divergences.
- `docs/fidelity/RULEBOOK.md` governs every fidelity judgment.
- `docs/fidelity/DEPTH_TESTING.md` explains how scenarios become command-depth proof.

The harness copies both world trees into disposable runtime directories before
applying fixtures. Never modify either oracle source tree to make a test pass.

## Running a Scenario

```bash
DP_ORACLE_BIN=/path/to/darkpawns-c-oracle/bin/circle \
  go run ./cmd/dp-oracle-diff --scenario flee-audience-success --seed 1
```

Use `--show-oracle` while developing a scenario. A green diff can otherwise be
false comfort if timing caused the intended command output to be drained or
never emitted. Repeat RNG-sensitive probes with several explicit `--seed`
values and make the chosen draw player-visible where possible.

The harness skips successfully when `DP_ORACLE_BIN` is unset so normal CI does
not require the external C checkout. Such a skip is not oracle proof (R5a).

## Scenario Model

Scenarios can have different C/Go login setup, named passive peers for recipient
bytes, discarded warmup commands, a shared probe, and disposable fixtures. The
authoritative syntax is documented beside `ParseScenario` in
`internal/oraclediff/scenario.go` and summarized in the harness README.

The important boundary is:

1. Setup establishes equivalent state and is normally not diffed.
2. Warmup establishes timing or state and is discarded.
3. Probe commands are sent to both servers and diffed for every connected audience.
4. Fixtures alter only disposable copies and must preserve the behavior under test.

## What Green Means

Green means the normalized player-visible blocks matched for that exact state,
audience, topology, and seed. It does not prove unvisited branches, internal
state transitions, automatic/NPC call sites, or draw order hidden by equivalent
outcomes. Those require additional scenarios or focused state-level unit tests.

This is why breadth coverage and depth proof are tracked separately. See the
depth guide before selecting or closing the next command.
