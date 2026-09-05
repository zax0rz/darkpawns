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
clone, Go source world, and their player databases are not modified. The
`--seed` value (default `1`) is applied to both processes so scenarios can prove
RNG draw parity across several seeds. `--show-oracle` prints normalized C blocks
even when the result is green; use it to ensure a queued command actually ran.

When `DP_ORACLE_BIN` is unset, the command prints `SKIP` and exits successfully,
so the default build and test path does not require the oracle.

## Scenario format

Scenario files live in [`scenarios`](scenarios). Setup is server-specific because
the C and Go login flows can require different keystrokes; setup output is
drained unless `[creation:oracle]` and `[creation:port]` are used. Probe commands
are shared and diffed block by block. Empty/comment lines are ignored, and
`<ENTER>` sends an empty line.

The supported sections are:

- `[setup:oracle]` and `[setup:port]` for the primary client
- `[setup:oracle:name]` and `[setup:port:name]` for passive audience clients
- `[warmup]` for shared commands whose output is discarded
- `[probe]` or `[probe:name]` for the shared, diffed command stream
- `[fixture]` for disposable world changes such as quieting mobs, spawning
  objects/mobs, replacing room exits, and toggling room flags

Read `ParseScenario` in `internal/oraclediff/scenario.go` for the authoritative
fixture grammar. Fixtures patch only throwaway C and Go world copies.

For command-depth work, annotate scenarios with `# depth-case: <case-id>` and
record the case in `docs/fidelity/depth/<command>.tsv`. Run `make fidelity-depth`
to reject missing scenario or unit-test proof references. See
`docs/fidelity/DEPTH_TESTING.md` for the complete workflow.
