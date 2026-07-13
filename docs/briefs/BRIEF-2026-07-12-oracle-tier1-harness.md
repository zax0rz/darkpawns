# BRIEF 2026-07-12 — Tier-1 differential-test harness (walking skeleton)

**Executor:** codex (gpt-5.5/5.6). This is **net-new test tooling**, not a fidelity fix to
game logic — codex's greenfield-infra strength, and outside the "keep codex off structural
port fixes" caution. Hold tightly to this brief; Claude reviews the output.
**Branch:** `feat/oracle-diff-harness` off current `main`. **One PR.**
**Read first:** `docs/research/drafts/2026-07-12-c-oracle-differential-testing.md` (the why +
two-tier plan) and `docs/research/drafts/2026-07-12-c-oracle-build-notes.md` (how the C oracle
builds/boots, port, start/stop commands).

---

## Goal

Build a **differential-test harness** that drives the **C oracle** and the **Go port** with the
*same* scripted input, captures both transcripts, **normalizes** away acceptable differences,
**diffs** them, and emits a **divergence report**. Surviving diffs become `[Fidelity]` Linear
issues (same pipeline as the audit). This is **Tier 1**: structural/text parity only — NO
RNG-dependent comparison (see §Determinism).

## ⛔ Guardrails (do not violate)

1. **Both servers are black boxes.** Do NOT modify game logic, messages, tables, or the login
   flow in *either* codebase to make things match. The harness observes; it never "fixes"
   divergences. Divergences are **reported**, full stop.
2. **Do NOT commit the C oracle source or its `circle` binary** into the Go repo. Reference the
   binary by a path from an env var (§Layout).
3. **Do NOT break CI / the default build gate.** The harness must **skip cleanly** when the C
   oracle binary isn't present (CI can't build it). It must NOT be part of the default
   `go test ./...` critical path — gate it (build tag `//go:build oracle` and/or `t.Skip()` when
   `DP_ORACLE_BIN` is unset).
4. Do NOT touch `src/*.c` in either tree, or the `DP_SEED` patch (already applied in the C clone).

## Scope: a WALKING SKELETON, not the whole harness

Deliver a **minimal end-to-end vertical slice** that actually diffs **one real scenario** across
both live servers and produces a normalized report. Prove the pipe works end-to-end before
anyone adds more scenarios. Breadth (more scenarios, richer normalization) is explicit follow-up.

**The one scenario for the skeleton:** create a character, enter the game, and **`look` at the
starting room + `look`/`examine` a fixed object or mob** — then `quit`. Rationale: room/object
descriptions are **pure static world data with no RNG**, so they're the cleanest possible
signal for text/world-data fidelity. Creation steps are included (they're structural and
recently audited), but the *assertion* focus is the static `look` output.

## Architecture

### Transport — use telnet/TCP for BOTH (the Go port speaks telnet too)
The C oracle speaks **telnet over TCP** (see build-notes: `nc 127.0.0.1 <port>` / `expect`). The
Go port exposes **both WebSocket and telnet** — so **drive both servers over telnet/TCP.** This
is the simplest and most apples-to-apples path: one adapter, identical framing (CRLF), no
WebSocket payload reconciliation. Ignore the WS transport entirely for this harness.
```go
type Conn interface {
    Send(line string) error                              // write one command
    ReadUntilQuiescent(d time.Duration) (string, error)  // read until output settles
    Close() error
}
```
A single TCP/telnet adapter satisfies `Conn` for both servers. (Reuse the existing E2E test
scaffolding — CON_MENU E2E tests, commits around DP-1067 — only for **how it boots the Go
server**, not for transport; connect to the Go server's telnet port, not its WS endpoint.)

### Driver
Plays a scenario against one `Conn`: for each step, `Send` then `ReadUntilQuiescent` (read until
output stops for ~N ms, i.e. server is waiting for input), accumulate a raw transcript. Keep the
quiescence timeout configurable; MUD output is bursty.

### Scenario format
Simple and reviewable — an ordered list of send-steps (a `.txt`/`.jsonl`/YAML file, your call,
keep it dead simple). Each step = a line to send. No per-step "expect" assertions in the skeleton;
we diff the *whole* normalized transcript at the end. Store scenarios as data files under the
harness dir so non-coders can add them.

### Normalizer (the crux — get the starter rules right, documented, ordered)
Apply an **ordered, documented** list of canonicalizations to each raw transcript before diffing.
Starter rule set for the skeleton:
- Strip ANSI/color escape sequences.
- Normalize line endings (both are telnet/CRLF now, but canonicalize to `\n` anyway) and
  trailing whitespace.
- Replace the vitals prompt `\d+H \d+M \d+V >` → `<PROMPT>`.
- **Tier-1 RNG masking:** replace RNG-derived numerics with placeholders — rolled stat blocks,
  and the H/M/V numbers — so RNG differences (C vs Go use different generators) don't create
  false diffs. We're comparing **text/structure**, not roll values, in Tier 1.
- Strip volatile lines: timestamps, uptime, "players online" counts, MOTD version/date.
Keep every rule in ONE commented list with a rationale each; this list WILL grow — make it easy
to extend. Bias toward **under-normalizing** (a false diff is a cheap human glance; an
over-normalized real bug is invisible).

### Diff + report
Unified diff of the two normalized transcripts, plus a short report: scenario name, the two
servers/ports/seed, and each surviving hunk with a little context. Human triages the report into
`[Fidelity]` issues.

## Determinism
- Start the C oracle with **`DP_SEED=<fixed>`** (the patch is applied) so it's reproducible.
- Tier-1 scenarios **avoid RNG-branching actions** (no combat, no skill rolls); where RNG numerics
  still appear (stat rolls, vitals), the normalizer masks them. Do NOT try to seed-match Go to C
  in this brief — that's Tier 2 (porting `random.c` into Go), explicitly out of scope.

## Layout & lifecycle
- Put it under `cmd/dp-oracle-diff/` (driver) + a supporting package if needed; scenarios in a
  sibling `scenarios/` dir. Match repo conventions (see other `cmd/` tools).
- **Server lifecycle:** harness starts both servers on ephemeral/free ports, waits for readiness,
  runs the scenario, tears both down. C oracle binary path from **`DP_ORACLE_BIN`** env var
  (e.g. `/Users/zach/.openclaw/workspace/darkpawns-c-oracle/bin/circle`); the C server must run
  from the dir containing its `lib/`. Use a throwaway player dir so runs are clean/repeatable.
- Boot the Go server the way the existing E2E tests do — reuse, don't reinvent.

## Success criteria (the skeleton PR must show ALL)
1. `DP_ORACLE_BIN=... go run ./cmd/dp-oracle-diff --scenario look-start-room` (or a gated
   `go test`) **starts both servers, plays the scenario, and emits a normalized diff report** —
   demonstrated with **real captured output pasted into the PR** (proof-of-life).
2. When `DP_ORACLE_BIN` is unset, the harness/test **skips cleanly** — `go build ./... && go vet
   ./... && go test ./...` stays green, unaffected. Show this.
3. The normalizer rule list is present, ordered, and each rule has a one-line rationale.
4. Neither game codebase's logic/messages were modified (⛔). C oracle source/binary NOT committed.

Whatever divergences the report shows are **fine and expected** — that's the point. Do NOT chase
them in this PR; capture them for triage.

## Out of scope (follow-up briefs)
- More scenarios (full creation matrix, movement/sector costs, score, shops).
- Tier-2 RNG parity (porting `random.c` into Go for byte-identical combat rolls).
- CI integration of the oracle job (needs a C-build step in CI first).

## Wrap-up
`go build ./... && go vet ./... && go test ./...` green; commit; push; open PR with the
proof-of-life report inline; then STOP — Claude reviews against `origin/main` (esp. the ⛔ items
and CI-skip behavior) and merges.
