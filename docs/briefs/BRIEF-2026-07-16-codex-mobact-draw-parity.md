# BRIEF (codex) — `mobile_activity` per-tick draw parity (DP-1170 root cause)

**Owner:** codex (frontier). **Gate:** Claude runs the differential oracle red→green (workers have no `DP_ORACLE_BIN`).
**Branch off `main`.** Sized to one PR.

## TL;DR
`mobile_activity` in the Go port draws a **different PRNG sequence per tick** than C
`src/mobact.c`. The headline bug: C draws the wander direction `door = number(0, 18)`
**unconditionally**, *before* the `MOB_SENTINEL` gate — so even a sentinel mob that can
never move still burns one draw every tick. Go hides that draw *inside* `wanderMob`,
which the caller only invokes for non-sentinel mobs. Result: every sentinel (and every
non-standing / can't-move) mob is **1 draw/tick short** of C, and the shared seeded stream
drifts a little on every `PULSE_MOBILE`. This is the same bug *class* as DP-1167 (the mob-HP
"average" shortcut that drew zero) — draw-count/order parity is law for recurring paths,
not just spawn.

This is **not** a steal bug. `DoSteal`/`stealCoins` are already C-faithful; the steal
oracle scenario just happened to be the first roll-sensitive probe sitting downstream of a
persistently-alive fixture mob, so it surfaced the drift. Do **not** touch the steal code.

## Root-cause evidence (already proven; here for context)
A `steal coins <mob>` oracle scenario diverges: C succeeds, Go fails. Isolation showed:
char-creation stats identical (byte-for-byte), movement draws 0, the mob *load* draws 3 on
both sides (matches C `read_mobile`). The only remaining variable was the fixture mob (vnum
16303, "a guard trainee") being **alive at its room from boot onward** and running
`mobile_activity` every `PULSE_MOBILE` during the harness's idle waits. Its action flags =
`10` = `MOB_ISNPC`(bit 3) + `MOB_SENTINEL`(bit 1) — a sentinel. C draws `number(0,18)` for
it each tick; Go draws nothing. The drift accumulated over the scenario's ticks and moved
Go's later steal roll off C's. Full write-up: Linear **DP-1170**.

## Read-only source of truth
C: `~/.openclaw/workspace/darkpawns-c-oracle/src/mobact.c`, function `mobile_activity`
(lines ~54–340). **Never edit anything under `darkpawns-c-oracle/`.**
Go: `pkg/game/mobact.go` (`mobileActivityForMob`) and `pkg/game/ai.go` (`wanderMob`).

## The canonical C draw order (per mob, per tick)
The outer gate (mobact.c:71): `if (!IS_MOB(ch) || FIGHTING(ch) || !AWAKE(ch)) continue;`
— a fighting or non-awake (sleeping/worse) mob draws **nothing**. Everything below is for a
mob that passes that gate. Draws happen in **this order**:

1. **Hunter** (mobact.c:75–80): `hunt_victim(ch)` called twice if `MOB_HUNTER` && standing.
   Verify whether Go's `huntVictim` draws any `number()`/`dice()`; if it does, its count &
   order must match. (For the trainee this is a no-op — hunter flag unset.)
2. Spec proc (83), wake sleepers (97) — no `number()` draws in the base path.
3. **Scavenger** (103–117): `if (MOB_FLAGGED(ch, MOB_SCAVENGER)) if (world[...].contents && !number(0, 10))`.
   The `number(0, 10)` is drawn **only if** `MOB_SCAVENGER` **and** the room has contents.
   (Go already matches this at mobact.go:199 — keep it.)
4. **Movement direction — THE FIX (120):** `door = number(0, 18);` drawn **UNCONDITIONALLY**
   here, *before* the sentinel/standing/CAN_GO checks at 121–129. The single draw is both the
   move gate and the direction (Go's `wanderMob` already reuses it correctly — the bug is
   purely *where* it's gated).
5. **Sounds (146):** `if (!number(0, 15))` — drawn **UNCONDITIONALLY**, right after movement,
   **before** aggressive/race-hate/memory. (Go currently draws this too late — see fix #2.)
6. Aggressive (199–231): conditional draws `number(0,5)` (210, AFF_PROTECT_EVIL+evil),
   `number(0,5)` (212, AFF_PROTECT_GOOD+good), `number(0,3)` (214, AFF_SNEAK) — only when
   `MOB_AGGRESSIVE || MOB_AGGR_TO_ALIGN` and the victim has the matching affect.
7. Race-hate (237–258): `number(0,5)` (249, protect-evil bypass) and `number(0,5)` (251,
   speak-before-attack) — only on a race-hate match against a visible PC.
8. Memory (261), helper (285), aggr24 (304, 325): no `number()` draws in these blocks.

## Fix #1 (required) — draw `number(0,18)` unconditionally
Hoist the `door = number(0, 18)` draw out of `wanderMob` so it fires for **every** mob that
passes the outer gate, then pass the drawn value into the movement logic. Sketch:

```go
// -- Mob Movement (C mobact.c:120) — the direction draw is UNCONDITIONAL, before
// the sentinel/standing/CAN_GO gate. Even sentinel or non-standing mobs burn it.
// #nosec G404 — game RNG, not cryptographic
door := dprng.Number(0, 18)
if !hasMobFlag(ch, "sentinel") && ch.GetPosition() >= combat.PosStanding {
    w.wanderMobWithDoor(ch, door) // wander logic, NO draw of its own
}
```

`wanderMob` must no longer draw — split the draw out and have it accept `door` (rename to
`wanderMobWithDoor` or pass the value; your call). Keep every downstream gate/skip in
`wanderMob` exactly as-is (the `door >= len(dirs)` early-out, `CAN_GO`, `STAY_ZONE`, sector
checks). Update the ai.go doc comment that currently says "SENTINEL is checked by the
caller" — that comment described the bug.

Watch the position gate: C's condition is `GET_POS(ch) == POS_STANDING`. Go uses
`>= combat.PosStanding`. The draw itself is unconditional regardless; only the *move* is
position-gated. Match C: draw always, move only when standing & not sentinel & `CAN_GO`.

## Fix #2 (required) — move the `number(0,15)` sounds draw to C's position
C draws `number(0,15)` at :146, **immediately after** the movement block and **before**
aggressive/race-hate/memory. Go draws it at `mobact.go:~477`, *after* those blocks. For the
sentinel trainee (no aggro/memory draws in between) this still corrupts the value because
Go currently skips the `number(0,18)` slot; but for aggressive/race-hate mobs the ordering
is independently wrong. Relocate Go's `if dprng.Number(0, 15) == 0 { w.MpSound(ch) ... }` to
run right after the movement step (step 5 above), before the aggressive block.

## Fix #3 (verify, likely fine) — conditional draws already present
The scavenger `number(0,10)` and the aggressive `number(0,5)/(0,5)/(0,3)` and race-hate
`number(0,5)/(0,5)` draws already exist in Go. **Confirm each is gated by the exact same
condition as C and fires in the same relative order** (steps 3, 6, 7). Do not add or drop
any. If Go's hunter path (`huntVictim`, step 1) draws PRNG, confirm it matches C's
`hunt_victim` and that Go calls it in the C position (before spec) — but do not refactor
hunt logic beyond draw parity in this PR.

## Out of scope (do NOT change)
- Steal, hide, sneak, or any `act.other.c` skill code.
- The C oracle tree, and `website/static/map/world-sphere.json`, and `docs/reports/reek/*`.
- Mob movement *behavior* beyond the draw hoist (don't "improve" wander/aggro logic).
- The `MpSound`/lua sound side effects — only the *draw position* moves, not the effect.

## Tests you own (golden, deterministic — no oracle needed on your side)
Add to `pkg/game/mobact_test.go`, using `dprng.ResetStream(seed)` (NOT `Seed`) for a
reproducible stream and a bare `&World{}` where possible (avoid `NewWorld` tickers racing
the global stream — see the pattern in `combat_helpers_test.go`):

1. **Sentinel mob burns exactly one movement draw/tick.** Build a sentinel, non-aggressive,
   non-scavenger, non-hunter mob (like 16303). `ResetStream(1)`; record `dprng`-equivalent
   count before/after one `mobileActivityForMob`; assert **exactly two draws** consumed in
   order `number(0,18)` then `number(0,15)`, and that the mob did **not** move. (A
   `DrawCount()` test seam on the process stream is a reasonable small addition if it helps —
   but keep any such seam minimal and clearly test-only.)
2. **Non-sentinel standing mob:** same two draws in the same order; movement may or may not
   occur depending on the rolled door, but the *draw count/order* is identical to the
   sentinel case (the movement draw is unconditional).
3. **Sounds ordering:** assert the `number(0,15)` draw precedes any aggressive-block draw for
   an aggressive mob (e.g. by constructing a scenario where an aggressive draw would fire and
   checking the stream sequence).

## Acceptance / how Claude gates it
Claude will run the live oracle: the existing `steal coins <mob>` scenario (fixture mob
16303 at 8105, warmup `n/e/s/e`, probe `steal coins trainee`) is **RED on `main`** and must
go **GREEN** on your branch — that is the red→green proof that the per-tick draw sequence now
matches C. Claude will also re-run any existing green mob scenarios to confirm no regression.
Put the scenario name and expected-green in the PR description; Claude owns the actual run.

## PR hygiene
- Commit messages end with: `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`
- PR body ends with: `🤖 Generated with [Claude Code](https://claude.com/claude-code)`
