# Tier-1 oracle differential harness

`dp-oracle-diff` launches the original C server and the Go port as black boxes,
drives both telnet listeners with one shared line sequence, normalizes accepted
Tier-1 noise, and prints a unified divergence report.

Run the walking-skeleton scenario from the repository root:

```bash
DP_ORACLE_BIN=/path/to/darkpawns-c-oracle/bin/circle \
  go run ./cmd/dp-oracle-diff --scenario look-start-room
```

The harness builds the Go server, allocates free ports (including the C
server's adjacent WHOD port), copies the C `lib/` tree to a throwaway runtime
directory, starts both servers, and tears them down after the scenario. The C
clone and its player database are not modified. `DP_SEED=1` is applied only to
the C process; Tier 1 masks RNG-derived values instead of seed-matching Go.

When `DP_ORACLE_BIN` is unset, the command prints `SKIP` and exits successfully,
so the default build and test path does not require the oracle.

## Scenario format

Scenario files live in [`scenarios`](scenarios). Each non-comment line is one
line sent identically to both servers. Empty/comment lines are ignored, and
`<ENTER>` sends an empty line. There are deliberately no server-specific steps
or expected-prompt branches: creation-flow differences remain visible in the
resulting transcript.

The walking skeleton creates a Human Warrior, enters the game, runs `look`,
examines the fixed starter sword (object vnum 8037), and quits. Additions beyond
that single scenario belong in follow-up work.
