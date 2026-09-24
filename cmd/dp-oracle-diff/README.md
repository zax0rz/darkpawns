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
- `[relogin:oracle]` and `[relogin:port]` for the login lines a returning
  character needs, played by the `<RELOGIN>` probe step

## Leaving and coming back (`<RELOGIN>`)

A `<RELOGIN>` probe step settles the server on the actor's connection (C
extracts a quitting character on the next pulse), closes it, dials the same
server again, plays that server's `[relogin:*]` lines, and diffs the whole
returning-player transcript as one block; later steps run on the new
connection. Both servers then read the character back from their own
persistence: the C player and rent files in the disposable lib copy, and a
throwaway SQLite store the harness gives the port for these scenarios (seeded
with one placeholder player, so the new character is a mortal on both sides
unless the scenario empties the player file).

A dropped C connection leaves the character linkdead and a login reattaches to
it without reading the save, so a persistence scenario quits before it relogs.
A connection that closes on the step just before `<RELOGIN>` is accepted.
The `lifecycle-*` scenarios are the model vehicles: preferences and inventory
across a quit, the login room after an unsafe quit, and idling into the void.

Read `ParseScenario` in `internal/oraclediff/scenario.go` for the authoritative
fixture grammar. Fixtures patch only throwaway C and Go world copies.

## Raw ANSI proof mode (`keep-ansi`)

Normalization rule 1 strips ANSI CSI escapes, which hides any surface where C
embeds color bytes the port is missing (or invents color C never emits). A
scenario that declares the `keep-ansi` fixture compares its probe blocks with
escapes intact instead (`NormalizeKeepANSI`), certifying the raw color bytes
themselves. `scenarios/redit-menu-color-on.txt` and
`scenarios/redit-menu-color-off.txt` are the model vehicles: they answer the
creation ANSI question Y/N respectively and keep every probe step inside the
OLC menus, because the ordinary playing prompt and vitals masking expect
ANSI-stripped text and are outside what the mode certifies.

## Raw prompt proof mode (`keep-prompts`)

The ordinary normalizer removes prompt-only lines, trailing spaces, ANSI, and
line-ending distinctions. A focused scenario with `keep-prompts` compares the
captured text bytes exactly after telnet IAC negotiation is removed. Keep its
probe blocks limited to prompt states with deterministic output.
`prompt-pager-depth`, `prompt-pager-color-depth`, and `prompt-playing-depth`
cover the pager and playing prompts. For the first prompt after entering the
game, use `keep-prompts`, `no-settle`, and `entry-prompt` together with
`[creation:oracle]` and `[creation:port]`. This compares the trailing newline
run and `> ` before any clock pulse can supply a later prompt; the full entry
dialogue is covered by separate scenarios.

For command-depth work, annotate scenarios with `# depth-case: <case-id>` and
record the case in `docs/fidelity/depth/<command>.tsv`. Run `make fidelity-depth`
to reject missing scenario or unit-test proof references. See
`docs/fidelity/DEPTH_TESTING.md` for the complete workflow.

## Driving the browser client (`-go-transport ws`)

By default the Go port is driven over telnet, the same transport as the C
oracle. `-go-transport ws` (or `DP_ORACLE_GO_TRANSPORT=ws`, which
`make oracle-regression` passes through) drives it through `/play` instead:
`internal/oraclediff/wsdriver/driver.mjs` runs the real
`web/public/mud-client.js` headless under Node 22+, types each scenario line
into it, and the transcript is what the client writes to its terminal. The C
side is unchanged, so every divergence in this mode is something a browser
player sees and a telnet player does not.

Two things are removed from the browser transcript because a telnet client has
them too and the telnet transcript never holds them: the client's local echo
of the line just typed, and its own connection messages ("Connecting to…",
"Connected.", "--- Connection lost ---"). Everything else counts. The
`~dpclock pulse N` control is handled by the session, so it works over either
transport.

This mode is a probe, not a gate yet: the expected-divergence ledger and pins
describe the telnet transport.
