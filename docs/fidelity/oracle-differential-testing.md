---
type: concept
title: Oracle Differential Testing
last_updated: '2026-07-17T00:00:00.000Z'
ingested_via: 'mcp:put_page'
ingested_at: '2026-07-17T21:07:51.444Z'
source_kind: 'mcp:put_page'
tags:
  - darkpawns
  - fidelity
  - methodology
  - oracle
  - testing
---

# Oracle Differential Testing

**The methodology:** Drive a live C CircleMUD server and the Go port with the *same scripted input*, capture both transcripts, normalize away acceptable differences, diff them, and emit a divergence report. Surviving diffs become confirmed fidelity bugs with the exact expected behavior already captured from the C server.

## Why This Matters

This is not static analysis. Not type checking. Not "read the C source and guess." This is **run both implementations and compare transcripts**. The C server is a live oracle — a running game we can ask questions to. Every scenario encodes the correct C behavior. The harness is the judge, not the model.

## Architecture

- **Harness:** `cmd/dp-oracle-diff/` — drives both servers over telnet/TCP, normalizes transcripts, diffs them
- **Scenarios:** `cmd/dp-oracle-diff/scenarios/` — 32 scenario files covering combat, movement, items, socials, skills, creation, command gating
- **Normalizer:** `internal/oraclediff/normalize.go` — strips ANSI, masks RNG numerics, normalizes volatile lines
- **C Oracle:** `/Users/zach/.openclaw/workspace/darkpawns-c-oracle/bin/circle` — the original C Dark Pawns, built on macOS Apple Silicon

## The North Star

The Go port must be **1:1 with the original C Dark Pawns on the entire player-facing surface** — every prompt, message, menu, ordering, and byte a player can see. *"The game is the game."* Modern additions are allowed ONLY where they are not player-facing.

The endgame: Dark Pawns becomes an **open-source "2026 CircleMUD written in Go"** — a reference MUD game server where Dark Pawns is the faithful base game.

## Key Discoveries

1. **DP_CLOCK seam (2026-07-17):** Freezing real-time pulses + deterministic settle-pump enables reproducible combat scenarios. Without it, two independently-clocked processes interleave pulses nondeterministically → RNG stream drift.

2. **Zone-reset draw parity (2026-07-17):** Two Go zone-reset bugs caused a +2 draw offset: (a) R command sets lastCmd=1 unconditionally vs C only-on-success; (b) Go runs percentLoad BEFORE initRare vs C's reverse order.

3. **Character-creation 1:1 (2026-07-17):** The port heavily re-skinned C's nanny() — invented bracket menus, fabricated MOTD, leaked engine artifacts. DP-1173 stripped everything back to byte-for-byte C.

4. **Recall port (2026-07-17):** doRecall was a re-skin, not a port: wrong gate messages, fabricated self-messages, hardcoded room target. Rewritten to match C byte-for-byte.

5. **Fight message fidelity (2026-07-15):** The lib/misc/messages file is byte-identical C↔Go. Go's skill_message and dam_message draw counts must match C exactly or everything downstream desyncs.

## Pipeline

```
Scenario file → Harness drives both servers → Normalize transcripts → Diff → 
  Zero divergence = GREEN (faithful)
  Surviving diff = RED (confirmed bug with exact expected behavior)
  → Fix Go code → Re-run harness → GREEN
```

## Status (2026-07-17)

32 scenarios committed. 29/30 green after DP_CLOCK + zone-reset + character-creation + recall fixes. The Codex Goal (`docs/briefs/GOAL-2026-07-17-oracle-suite-green.md`) is driving remaining reds to green autonomously.

## Related

- [[darkpawns/c-oracle]] — the C oracle build/boot
- [[darkpawns/north-star-1to1]] — the governing directive
- [[darkpawns/oracle-proof-gate]] — fixes must be PROVEN by oracle run
- [[darkpawns/mobact-draw-parity]] — the draw-archaeology that led to DP_CLOCK
