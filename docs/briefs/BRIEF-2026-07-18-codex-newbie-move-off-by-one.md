# BRIEF (codex) — newbie move point off-by-one (Go 84 vs C 85)

**Owner:** codex. **Gate:** Claude runs `hunger-thirst` red→green (worker has no `DP_ORACLE_BIN`). **Branch off `main`, one PR.** Small, self-checkable from C source.

## The divergence (measured 2026-07-18, both engines now deterministic)
A fresh Human Warrior's movement points: **C = 85, Go = 84** (stable, reproducible on both sides). HP now matches exactly (21 = 21), so the PRNG stream is **aligned through the stat roll and the `advance_level` move draw** — this is **NOT a draw-parity bug**. It's a **base `max_move` init constant** that Go sets one lower than C.

## Why it's the base constant, not a draw
- C `do_start` (`class.c:501-576`) sets `max_hit = 10` and `max_mana = 100` explicitly, but **never sets `max_move`** — so a new char's base `max_move` is whatever char-initialization left it at *before* `do_start`.
- Then `advance_level` (`class.c:600+`, warrior case) adds `add_move = number(1, 4)`; Go mirrors this (`level.go:391 addMove = levelNumber(1,4)`), and the draw is aligned (HP matched ⇒ stream position matched).
- Finally `GET_MOVE = GET_MAX_MOVE` (`class.c:576`). So: `final_move = base_max_move + number(1,4)`. Same `number(1,4)` both engines ⇒ the −1 is entirely in `base_max_move`.

## The work
1. Find where **C** initializes a new character's base `max_move` (before `do_start`) — check `clear_char` / char creation in `src/db.c` (and any `CREATE`/reset of `ch->points.move`/`max_move`), plus the nanny creation path in `src/interpreter.c`. Identify the exact constant/formula C uses.
2. Find the **Go** equivalent (`pkg/game/character.go` `newCharacter`/creation, or `pkg/game/level.go`) that sets the newbie's base max move. Compare to C.
3. Reconcile the off-by-one **faithfully to C** — match C's base value exactly. Do NOT add a compensating +1 elsewhere; fix the actual base constant/formula so it equals C's.
4. Verify the fix doesn't perturb HP/mana (those are already correct) and doesn't shift any draw (base move is not a draw).

## Acceptance (Claude-gated)
1. `hunger-thirst` → `no normalized divergence` on the score `Movement points:` line (Go now 85). *(Note: the same scenario has a separate `quit`-block divergence being handled in another brief — your fix only needs to clear the move line; the quit line is out of scope.)*
2. Full committed sweep stays green — especially any scenario printing move points for a fresh char.
3. `go build ./... && go vet ./...` clean.
