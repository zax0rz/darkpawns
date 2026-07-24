# BRIEF (glm) — DP-1212 ROOT CAUSE: God char-creation consumes 2 phantom AdvanceLevel draws

**Owner:** glm-5.2. **Gate:** a draw-count regression test now (the `levelNumber`
seam pattern); Claude runs the opener sweep after merge. CI green.
**Git:** branch off `main` as `glm/god-advancelevel-draws`. Edit → commit → push →
open a PR. Do NOT merge. Sized to one PR (S/M). **⚠️ Before branching, `git fetch
origin main && git rebase onto origin/main` — recent PRs have been based on a stale
local main; confirm `git merge-base HEAD origin/main` == origin/main tip.**
**Finding:** DP-1212 (the outcome-parity class). Root cause found via a Fable
consult, **call-path verified** (R5e) against both trees.
**Cite:** Go `pkg/session/char_creation.go:482,497-524`, `pkg/game/player.go:309-374`
(`newCharacter`/`NewCharacterWithStats`, `AdvanceLevel()` at :374),
`pkg/game/character.go` (`BootstrapFirstPlayerGod`), `pkg/game/level.go:125-137`
(`AdvanceLevel`, the `levelNumber` seam), `pkg/game/level_draw_order_test.go`;
C oracle `src/interpreter.c:2214-2216`, `src/class.c:572` (`advance_level`),
`src/db.c:3057` (`init_char` first-player block). Rules **R3/R3a** (draw parity).

---

## The bug — a wasted RNG side-effect on the God path

The oracle's combat-skill openers seed both servers from `DP_SEED=1`; the shared
CMWC stream must stay lock-step (every `number()`/`dice()` same order+count). It
does NOT: after the first-player-God fixture creates the God (primary) + a mortal
peer, **Go's stream is exactly 2 draws AHEAD of C's**, so every subsequent roll —
including the opener's hit roll — diverges. (Instrumented proof: trip opener, Go
`number(1,121)`=117 vs C ≤75; the constant +2 offset + CMWC's low autocorrelation
makes the two values effectively unrelated.) This is why **bash/trip/headbutt
openers flip C-hits/Go-misses**, and why kick/backstab are green only by outcome
*coincidence* (their single roll lands the same side despite different values).

### Root cause (verified both sides)
- **C:** `init_char` sets the first player to `LVL_IMPL` (40) **before** the
  creation menu (`db.c:3057`). At `CON_MENU` choice `'1'`, C runs
  `do_start()`→`advance_level()` **only `if (!GET_LEVEL(d->character))`**
  (`interpreter.c:2214-2216`). The God is already level 40 → **`do_start` is
  skipped → 0 draws**. The mortal is level 0 → `do_start` runs →
  `advance_level` (`class.c:572`) draws (warrior: `number(11,14)` HP +
  `number(1,4)` move = **2 draws**).
- **Go:** `completeCharCreation` (`char_creation.go:482`) calls
  `NewCharacterWithStats` → `newCharacter` → **`p.AdvanceLevel()` unconditionally**
  (`player.go:374`) — for the God too — **before** `isGod` is even computed
  (`char_creation.go:497`). `AdvanceLevel` (warrior) draws the same 2 values via
  the `levelNumber` seam. `BootstrapFirstPlayerGod` then overwrites the God's
  stat *fields* (`SetMaxHP(500)`, …) but **cannot un-consume the 2 draws**.

So Go draws 2 for the God that C never draws. (Weight/height draws match — both
sides draw them unconditionally for God and mortal alike; the *only* divergence is
`AdvanceLevel`.) Note: `GiveStartingSkills` two lines below is **already**
`!isGod`-gated (`char_creation.go:517,522`) — the author had the right pattern;
this one RNG-consuming call was just missed because it hides inside the constructor
before `isGod` is known.

## The fix — mirror C: run AdvanceLevel only for non-God characters

Restructure so the level-up (and its draws) happens **after** God-ness is decided
and **only for non-God** characters — exactly C's `if (!GET_LEVEL(ch)) do_start()`:

1. **Remove the unconditional `p.AdvanceLevel()` from `newCharacter`**
   (`player.go:374`) so the shared constructor no longer has the RNG side effect.
2. **Call `AdvanceLevel()` explicitly where a real leveled character is finalized,
   gated `!isGod`** — in `completeCharCreation`, right next to the existing
   `GiveStartingSkills` `!isGod` branches (so God ⇒ neither AdvanceLevel nor
   starting skills; mortal ⇒ both, in the C order: advance_level then class skills).
   The God path calls only `BootstrapFirstPlayerGod` (which sets level 40 + stats,
   drawing nothing).
3. **Audit every other caller of `newCharacter`/`NewCharacterWithStats`/
   `NewCharacter`** (player.go:310, :316; and any test/programmatic callers — grep
   them). Each that represents a normal level-1 character and previously relied on
   the constructor's `AdvanceLevel` must now call it explicitly, preserving today's
   HP/move for non-God characters byte-for-byte. **`NewPlayer` (player.go:249) is a
   separate constructor** — confirm whether it independently calls `AdvanceLevel`
   and leave its behavior unchanged. Do not change any non-God character's
   resulting stats or draw count.

**Faithfulness check:** after the fix, creating a God draws 0 level-draws (matches
C's skipped `do_start`); creating a mortal warrior draws exactly 2 (`Number(11,14)`,
`Number(1,4)`) in that order (matches `class.c` `advance_level`) — unchanged from
today for mortals.

## Tests (extend the existing `levelNumber` seam pattern)
`pkg/game/level_draw_order_test.go` already swaps `dprng.Number` for a counting
stub. Add a **char-creation draw-count** test in that style:
- **God creation consumes ZERO level-draws:** with the first-player-God path
  (`BootstrapFirstPlayerGod`/`isGod=true`), assert the counting stub records **0**
  `AdvanceLevel` draws, and the God still ends at level 40 / HP 500 / move 82.
- **Mortal creation consumes exactly 2 (warrior):** `isGod=false`, assert the stub
  records `Number(11,14)` then `Number(1,4)` (mirror `TestAdvanceLevelDrawOrderByClass`),
  and the mortal's HP/move are unchanged from today.
- **Regression guard:** a God-then-mortal sequence leaves the shared stream at the
  same position C would (God 0 + mortal 2 = 2 total), not 4.

## Oracle gate (Claude, after merge — informational)
This is the highest-leverage fix in the sweep. I re-run all openers: **trip +
headbutt go GREEN** (already rerouted in #467 — they only needed roll parity);
**bash's outcome aligns** (resolves the DP-1210 outcome half; bash still needs its
own message reroute separately); **kick + backstab stay green but now
*truly* aligned** (not by luck) — I'll spot-check actual roll values, not just
red/green. Until then the draw-count unit test is the gate.

## Guardrails
- **Never** edit `src/`, `darkpawns-c-oracle/`, or `lib/` — read-only reference.
- All gates (AGENTS.md §Build & Verify): build, vet, `test ./... -race`,
  `golangci-lint run`, `gofumpt -l .` empty, `make reachability`. Watch for tests
  that assert new-character HP/move — they must still pass unchanged for mortals.
- Don't stage `.zcode/`, generated reachability reports,
  `website/static/map/world-sphere.json`, or `docs/reports/reek/*`.

## Deliverable
`AdvanceLevel` gated out of the God creation path (mirroring C's `!GET_LEVEL`
`do_start` skip), non-God characters byte/draw-identical to today, the God→mortal
sequence 2 draws total (not 4), and the draw-count regression tests. Claude greens
the trip/headbutt/bash openers and re-validates the whole combat RNG stream.
