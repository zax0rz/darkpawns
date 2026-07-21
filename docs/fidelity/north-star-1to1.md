---
type: project
title: 'North Star: 1:1 Player-Facing Fidelity'
last_updated: '2026-07-16T00:00:00.000Z'
ingested_via: 'mcp:put_page'
ingested_at: '2026-07-17T21:07:51.532Z'
source_kind: 'mcp:put_page'
tags:
  - darkpawns
  - fidelity
  - north-star
  - policy
---

# North Star: 1:1 Player-Facing Fidelity

**The governing directive for the whole port (Zach, 2026-07-16):** the Go port must be **1:1 with the original C Dark Pawns on the entire player-facing surface** — every prompt, message, menu, ordering, and byte a player can see. *"The game is the game."*

## What This Means

- No player-facing "modern" UX: no invented bracket menus (`[M] Male/[F] Female`), no fabricated MOTD/rules screens, no re-worded prompts, no leaked engine artifacts
- When a scenario shows the port doing something "nicer" than C, that niceness is a **bug**, not a feature — remove it
- Modern additions are allowed ONLY where they are not player-facing — architecture, Go concurrency, DB/persistence, the oracle harness, internal APIs

## Why This Matters

Dark Pawns is meant to become an **open-source "2026 CircleMUD written in Go"** — a reference MUD game server where Dark Pawns is the faithful base game. Player-facing fidelity is not pedantry; it's the product thesis.

"If someone needs a MUD server in Go in 2026, here you go, and it runs the real game 1:1."

## Practical Implications

- The oracle gate's job is to prove the player-facing surface is byte-identical to C
- Do NOT normalize away real player-facing differences to make a scenario pass
- The normalizer may only canonicalize transport noise: ANSI codes, prompt framing, RNG masks — never legitimate wording/menu/ordering differences
- Any port UX that C lacks = remove it; any C text the port lacks = add it

## First Major Application: Character Creation (DP-1173)

The creation flow was one of the MOST diverged surfaces. The port heavily re-skinned C's nanny():
- Invented bracket menus (`[M] Male/[F] Female`, `[Y] Yes/[N] No`)
- Fabricated `Rules of the Realm` MOTD that does NOT exist in C
- Leaked `No database connection. Create new character?` line
- Duplicated start-room display on entry

DP-1173 stripped everything back to byte-for-byte C. The creation scenario went from fully-red to green.
