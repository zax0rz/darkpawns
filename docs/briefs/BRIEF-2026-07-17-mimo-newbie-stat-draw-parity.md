# BRIEF (mimo) — deep-dive: newbie stat-roll draw parity (+ observation co-location)

**Mode:** open-ended measured investigation → **written report**. Hammer it. Same shape as your boot-draw-parity win: instrument both engines, measure `drawsBefore`, bisect, pin the exact divergence, propose a faithful fix. Do NOT merge — make the fix mechanical for a follow-up.
**Owner:** mimo. **Gate:** Claude runs the oracle red→green after a fix lands. **You have `DP_ORACLE_BIN` / the C oracle; workers like codex/k3 do not.**

## Mission (PRIMARY): why do newbie rolled stats diverge C↔Go?
A fresh **Human Warrior** ends up with **different CON and WIS** in Go vs C for *some* characters, surfacing as two committed-suite reds:
- `hunger-thirst` (char "Oracletst"): **max HP 21 [C] vs 24 [Go]**, and **practice sessions 4 [C] vs 3 [Go]**.
- `guild-practice`: **practices 4 [C] vs 3 [Go]**.
- But `combat-death` (char "Cfighter") is **GREEN** — so the divergence is character-dependent (aligns for some names, drifts for others).

HP derives from CON (`conApp[con].Hitp`), practices from WIS (`wisApp[wis].Bonus`). Working the practice math backward: C's newbie WIS ≥ 12 (bonus 2 → +2 practices), Go's ≤ 11 (bonus 0 → +1). So Go rolls a **different WIS (and CON)** than C.

## Already ruled out — do NOT redo (all verified byte-identical to C)
- **Roll algorithm:** Go `rollStatTable()` (`pkg/game/character.go`) = 6×(4× `number(1,6)` = 24 draws), drop-lowest, descending insertion-sort — identical to C `roll_real_abils` (`src/class.c:389-404`).
- **Warrior stat assignment:** Go (`character.go` RollRealAbils warrior case) str=table[0], dex=table[1], con=table[2], wis=table[3], int=table[4], cha=table[5] — identical to C `class.c:446-458`.
- **Tables:** Go `conApp[].Hitp` and `wisApp[].Bonus` (`pkg/game/level.go`) are byte-identical to C `con_app`/`wis_app` (`src/constants.c`).
- **Race mod:** RACE_HUMAN `+1 CHA` — identical both sides (`class.c:461`).
- **Level-up HP/practices:** Go `AdvanceLevel` warrior (`level.go`: `addHP = conApp[con].Hitp + number(11,14)`; practices `MIN(2, MAX(1, wisApp[wis].Bonus))`) + `do_start` base `max_hit=10` and `+2` practices — all match C `class.c:539/590/669-673`.

**So the algorithm and data are perfect.** The ONLY remaining explanation: the **24 stat-roll `number(1,6)` draws land at a different STREAM POSITION** in Go vs C for these characters — a **creation-flow draw-parity drift** BEFORE `roll_real_abils`. This is the exact class of bug as the prior +2 hunts (mobact per-tick #374, per-command hide-clear #375, boot-reset R-command/init_rare #388/#389, parry-draw #389). **It has been MASKED all along** by the `<ROLLED_STATS>` normalization in the `character-creation` oracle scenario (same masking as the false-green sweep) — and it silently affects everything stat-derived (mana, hitroll, saving throws, HP).

## Environment & reproduction
- C oracle (build: `cd ~/.openclaw/workspace/darkpawns-c-oracle/src && make` → `bin/circle`). Both engines honor `DP_SEED` and `DP_CLOCK` (getenv seams, byte-identical when unset).
- Run: `DP_ORACLE_BIN="$HOME/.openclaw/workspace/darkpawns-c-oracle/bin/circle" go run ./cmd/dp-oracle-diff --scenario hunger-thirst` (also `guild-practice`, `character-view`). Harness sets `DP_SEED=1` + `DP_CLOCK=1`, builds the Go port from the current tree, drives both over telnet, diffs normalized output.
- **Green-check correctly on the OVERALL result line** (`grep 'result: no normalized divergence'`) — NOT the per-block `(no normalized divergence)` text (that false-greens multi-block scenarios).

## Method (the one that worked before)
Differential draw-counter:
1. **C:** global `unsigned long dp_draw_count` incremented in `prng_next()` (`src/random.c`); `fprintf(stderr, ...)` the count at `roll_real_abils` entry (`class.c:389`, before the roll loop). Captured into the harness per-process log buffer.
2. **Go:** atomic counter in `dprng.Generator.Next()` (`pkg/dprng/cmwc.go`) + a `DrawCount()` getter; print at `RollRealAbils`/`rollStatTable` entry.
3. Run `hunger-thirst` (Oracletst — diverges) AND `combat-death` (Cfighter — aligns). Compare `drawsBefore` at the stat roll. The delta on the diverging char (and zero on the aligning one) localizes it. Then **bisect backward** through the creation/nanny/login flow (name confirm, password, ANSI question, sex/race/class prompts, the settle-pump) to find the exact missing/extra draw. Compare against C's nanny (`src/interpreter.c` CON_* states) and `src/comm.c` per-command path.
4. Candidate suspects to check first (per prior hunts): a per-command `number(0,3)` AFF_HIDE clear (interpreter.c:889) count mismatch during creation; a draw during password/ANSI handling; a `str_add` `number(0,100)` fired asymmetrically (only when str==18); the settle-pump interacting with creation.

**Hard rules:** temp C instrumentation is allowed only if reverted + `make` rebuilt clean afterward (verify no `dp_draw_count`/`DP_DRAW` residue). Do NOT revert `comm.c`'s pre-existing `DP_SEED`/`DP_CLOCK` seams — edit out only your temp hunks. **Never** paper over with a compensating draw-burn or a normalization hack — the fix must be the genuine missing/extra draw.

## Deliverable — the report
1. **Root cause, pinned:** where in the creation flow Go over/under-draws vs C before `roll_real_abils` (with the measured `drawsBefore` for Oracletst vs Cfighter), and why it's character-dependent.
2. **Which side is wrong** vs the C original, with the C source that defines correct behavior.
3. **Proposed fix**, minimal and faithful, files/lines to touch.
4. **Blast radius:** it should be systemic to any scenario exposing stat-derived values; confirm the fix keeps the currently-green scenarios green.
5. Confirmation all temp instrumentation reverted + C rebuilds clean.

## SECONDARY (if budget remains): `observation` co-location
Scenario `observation` runs two telnet connections (primary "Obsactor" + peer "Obstarget"). C: the actor's room `look` shows "Obstarget the Warrior is standing here." and `diagnose Obstarget` → "Obstarget is in excellent condition." **Go shows neither** — `diagnose` returns "No-one by that name here." So in Go the two players are not co-located/perceivable. Known facts: `w.players` is `map[string]*Player` keyed by name (world.go:55; distinct names, no collision); `GetPlayersInRoom` filters `w.players` by `RoomVNum` (world.go:772); the harness sends the `[warmup]` `recall` to the **primary only** (`RunAudienceProbe`, internal/oraclediff/scenario.go:263-290), so the actor recalls to 8004 but the peer does not — yet C still shows them together. **Question to pin:** where is each player actually located in Go after setup+warmup (instrument/inspect `RoomVNum` for both), and why does C have them co-located while Go doesn't? Report the exact positioning divergence + proposed fix. (This is a positioning/registration bug, distinct from the stat-roll hunt — keep it a separate section.)
