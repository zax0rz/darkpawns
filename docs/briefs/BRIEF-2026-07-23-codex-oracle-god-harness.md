# BRIEF (codex) — oracle harness: `empty-players` fixture + probe-on-peer (skill-gate enabler)

**Owner:** codex. **Gate:** the ENTIRE existing scenario suite stays green
(critical regression guard) + a smoke scenario proving the new plumbing; Claude
reconciles + wires the real God/skill scenarios. CI green.
**Git:** branch `codex/oracle-god-harness` off `main`. **You have had HEAD-flip
issues in this repo before** — before any commit, run
`git rev-parse --abbrev-ref HEAD` in the SAME shell as the commit and confirm it's
your branch; do NOT check out other branches mid-run. Edit → commit → push → open
a PR; do NOT merge.
**Context:** This is the harness half of a plan to oracle-gate the combat SKILL
layer. A first-player-God bootstrap (separate PR, `glm/firstplayer-god`, may land
before or after yours) lets a scenario mint an Implementor who runs the ported
`skillset` command to grant a fixture mortal a skill; the mortal then uses the
skill as an opener (`wait==0`) and we diff it. You build the two harness
capabilities that make that possible. **Read** `internal/oraclediff/scenario.go`
and `cmd/dp-oracle-diff/main.go` fully first.

You are editing the **oracle harness itself** — the instrument every fidelity
gate depends on. Correctness > cleverness. The #1 requirement is that **every
currently-green scenario stays byte-identically green.**

---

## Sub-piece A — `empty-players` fixture

Goal: a scenario can request that BOTH servers boot as a *fresh MUD* (no players),
so the first character created becomes an Implementor (God) on each — C via its
native `top_of_p_table == 0` bootstrap, Go via the `glm/firstplayer-god` bootstrap
which keys on the env var `DP_FRESH_MUD`.

1. **Parse:** in `internal/oraclediff/scenario.go`, accept `empty-players` as a
   token in the `[fixture]` section (alongside `quiet-mobs`), setting a new
   `Scenario.EmptyPlayers bool`. Mirror how `quiet-mobs` / `spawn-mob` are parsed.
2. **C oracle side** (`cmd/dp-oracle-diff/main.go`, where the C server launches
   with `oracleBin, "-d", oracleData, port`): when `EmptyPlayers`, boot the C
   server against an **isolated data dir with an empty (or absent) `etc/players`
   file**, WITHOUT modifying the real C lib (same discipline as the disposable
   world copies). Minimal isolation is fine — the C server needs `world`, `text`,
   `misc`, `etc`, etc. from the real `oracleData`, but with `etc/players`
   truncated/removed so `top_of_p_table` boots at 0. Prefer symlinking the large
   subtrees (world/text/misc) into a throwaway dir and providing only an empty
   `etc/players` (+ whatever `etc/` files boot requires) as real files, to avoid
   copying the whole tree. Verify the C server boots and the first created char is
   level 34+ (`LVL_IMPL`). When `EmptyPlayers` is false: pass `oracleData`
   unchanged (today's behavior).
3. **Go side** (where the Go server launches with `-db deadDBURL` and an env slice
   including `DP_SEED`/`DP_CLOCK`/`DP_FIXED_TIME`): when `EmptyPlayers`, append
   **`DP_FRESH_MUD=1`** to the Go server's env. When false: do not (today's
   behavior). (The Go DB stays the dead placeholder; `DP_FRESH_MUD` is the fresh
   signal the Go bootstrap consumes — see the contract below.)

**Interface contract (do not deviate):** `EmptyPlayers` ⇒ C gets an empty
`etc/players` AND Go gets `DP_FRESH_MUD=1`. Both together, or neither. Any
asymmetry desyncs the two servers.

## Sub-piece B — probe-on-peer (a peer is the diffed actor)

Today the `[probe]` runs on the **primary** connection and its output is diffed
(`oraclediff.RunAudienceProbe(oracleConn, oraclePeers, scenario.Probe, …)` in
main.go). For the skill-gate flow the **primary is the God configurator** and a
**peer is the mortal** who runs the probe — so the diffed actor must be able to be
a named peer.

1. **Parse:** let the probe section name its actor — e.g. a header
   `[probe:mortal]` (where `mortal` matches a peer name from `[setup:oracle:mortal]`
   / `[setup:port:mortal]`), defaulting to the primary when unnamed (`[probe]`,
   today's behavior). Store the actor name on the `Scenario`.
2. **Execute:** in the probe run, send the probe commands to the **actor
   connection** (primary or the named peer) and diff **that connection's** output
   block-by-block, exactly as today but with a selectable source connection. Peer
   connections already exist and are already read for audience output — you're
   making one of them the command sender + primary diff source. Keep the
   audience-capture behavior intact for the others.

## The end-to-end flow this enables (for your smoke test + Claude's real scenarios)

```
[fixture]
empty-players
quiet-mobs
spawn-mob 16303 1 8105 80

[setup:oracle] / [setup:port]        # PRIMARY = God (first char on a fresh MUD)
<creation keystrokes>

[setup:oracle:mortal] / [setup:port:mortal]   # PEER = ordinary L1 mortal, then walks to the mob
<creation keystrokes>
north
east
south
east

[warmup]                              # runs on the PRIMARY (God), after the peer exists
skillset Mortal 'kick' 75            # world-wide target lookup (get_char_vis → get_player_vis)

[probe:mortal]                        # DIFFED, run on the mortal peer
kick trainee
~dpclock pulse 20
```
`skillset` targets by name world-wide, so the God (in the temple) can grant the
mortal (at 8105) its skill. The God never enters the probe room → no pollution.

## Cross-dependency (important)

The Go-side God only appears once `glm/firstplayer-god` (the `DP_FRESH_MUD`
bootstrap) lands. So:
- **You can fully build + verify the C side now** (C has the native bootstrap):
  an `empty-players` smoke scenario should show the C primary as a God (e.g. it
  can run an immortal-only command that errors for a mortal).
- **The Go side of the smoke test may not crown a God until the GLM PR merges** —
  that's expected. Verify the *plumbing* (fixture parsed, `DP_FRESH_MUD` passed,
  probe routed to the named peer, existing scenarios unaffected). Claude does the
  full C-vs-Go God/skill integration once both land. Note this clearly in the PR.

## Gates (run ALL before pushing)

- **Every existing scenario stays green** — run the full `cmd/dp-oracle-diff`
  suite against `DP_ORACLE_BIN` and confirm `result: no normalized divergence`
  for each currently-green scenario (`combat-death`, `combat-backstab-opener`,
  `act-obj-sweep`, `act-informative-sweep`, the help/pager ones, …). This is the
  load-bearing guard: your C-lib isolation and any main.go refactor must not
  perturb them. If `DP_ORACLE_BIN` isn't set in your env, say so and hand the
  suite run to Claude — do NOT guess.
- `go build ./...`, `go vet ./...`, `go test ./... -race`.
- **run `golangci-lint`** and `gofumpt -w` on every file you touch.
- A committed smoke scenario exercising `empty-players` + `[probe:peer]`.

## Guardrails

- **Never** edit `src/`, `darkpawns-c-oracle/`, or the real C `lib/` — isolate via
  throwaway/symlink; the real trees are read-only reference.
- Don't stage `website/static/map/world-sphere.json` or `docs/reports/reek/*`.
- Keep the non-`empty-players` path byte-for-byte identical to today.

## Deliverable

`empty-players` fixture (C empty-playerfile isolation + Go `DP_FRESH_MUD` env) +
`[probe:peer]` actor selection + a smoke scenario + the full existing suite still
green. Claude reconciles and authors the real God/skill scenarios.
