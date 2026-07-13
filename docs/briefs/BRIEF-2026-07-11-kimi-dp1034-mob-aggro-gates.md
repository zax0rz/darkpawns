# BRIEF 2026-07-11 — Kimi: DP-1034 / F14 Mob aggro visibility/NOHASSLE/sneak/peaceful gates

**Executor:** Kimi. **Branch:** `kimi/dp1034-mob-aggro-gates` (fresh off current
`main`, your own clone/worktree — never share a HEAD with another executor).
**One PR.** Claude verified the C source and every Go anchor below against the
tree at `main` @ `c6794f2`. This is pure pattern-replication — the correct gates
already exist in the **race-hate block** of the same file; you are copying them
into three sibling blocks that lack them.

## The bug

`pkg/game/mobact.go` has four mob-aggression blocks. The **race-hate** block
(lines ~266-308) correctly gates its targets on visibility, NOHASSLE, and
protection. The other three do NOT:

- **Aggressive / alignment-aggro** block (~217-261): checks only wimpy and
  protect-evil/good. Missing: CAN_SEE, NOHASSLE, and the AFF_SNEAK 75% skip.
- **Memory** block (~314-330): no gates at all. Missing: CAN_SEE, NOHASSLE.
- **AGGR24** block (~368-379): no gates at all. Missing: CAN_SEE, NOHASSLE,
  peaceful-room.

Result: invisible / sneaking players get attacked by aggro mobs, NOHASSLE
immortals get mobbed, and AGGR24 fires in peaceful rooms.

## Authoritative C (src/mobact.c — Claude read this directly)

**Aggressive / AGGR_TO_ALIGN loop (mobact.c:205-215)** — gates checked per victim
before the alignment match:
```c
if (IS_NPC(vict) || !CAN_SEE(ch, vict) || PRF_FLAGGED(vict, PRF_NOHASSLE))
  continue;
if (MOB_FLAGGED(ch, MOB_WIMPY) && AWAKE(vict))                       continue;
if (IS_AFFECTED(vict, AFF_PROTECT_EVIL) && IS_EVIL(ch) && !number(0,5)) continue;
if (IS_AFFECTED(vict, AFF_PROTECT_GOOD) && IS_GOOD(ch) && !number(0,5)) continue;
if (IS_AFFECTED(vict, AFF_SNEAK) && !number(0,3))                    continue;  // 75% skip
```

**Memory loop (mobact.c:267-269):**
```c
if (IS_NPC(vict) || !CAN_SEE(ch, vict) || PRF_FLAGGED(vict, PRF_NOHASSLE))
  continue;
```
(Note: the peaceful-room flag in the memory block only changes which *message*
prints — the `hit()` fires regardless. Do NOT gate the memory attack on peaceful.)

**AGGR24 loop (mobact.c:307-309):**
```c
if(!IS_NPC(tmp_ch) && CAN_SEE(ch, tmp_ch) &&
   !PRF_FLAGGED(tmp_ch, PRF_NOHASSLE) &&
   !ROOM_FLAGGED(ch->in_room, ROOM_PEACEFUL))
  if (GET_LEVEL(tmp_ch) >= 24) vict = tmp_ch;
```
(Here peaceful-room DOES gate the attack.)

## Go helpers to use (all already in package `game`, all used by the race-hate block)

- `canSee(ch, vict)` — `pkg/game/act.go:124`, takes the `Actor` interface;
  `*MobInstance` and `*Player` both satisfy it (the race-hate block calls
  `canSee(ch, vict)` at mobact.go:282 with exactly these types).
- NOHASSLE: `vict.GetFlags()&(1<<PrfNohassle) != 0` — `PrfNohassle = 28`
  (`pkg/game/other_helpers.go:34`); pattern already at mobact.go:285.
- Sneak: `vict.IsAffected(affSneak)` — `affSneak = 18`
  (`pkg/game/affects_constants.go:26`).
- Peaceful room: `w.roomHasFlag(ch.RoomVNum, "peaceful")` — pattern at
  `pkg/game/skill_combat.go:124`.

## The AFF_SNEAK skip — get the probability right

C: `if (IS_AFFECTED(vict, AFF_SNEAK) && !number(0,3)) continue;`

`number(0,3)` returns 0,1,2,3 uniformly. In C, `!x` is true only when `x == 0`,
so `!number(0,3)` is true exactly when the draw is 0 — a **1-in-4 (25%) skip**.
(The audit's parenthetical "75% skip" is wrong; the guard fires 25% of the time.
Match the C code, not the audit gloss.) When the guard fires, `continue` skips
this victim, so a sneaking player is passed over 25% of the time and attacked the
other 75%.

Exact Go equivalent — `rand.IntN(4) == 0` reproduces `!number(0,3)`:
```go
// #nosec G404 — game RNG, not cryptographic
if vict.IsAffected(affSneak) && rand.IntN(4) == 0 {
    continue
}
```
Keep the `#nosec` comment (the file uses it on every RNG line). **Sneak-skip goes
ONLY in the aggressive block** — C does not gate memory or AGGR24 on sneak.

## Implementation — three surgical edits to pkg/game/mobact.go

### 1. Aggressive block (~line 219, right after `if vict.IsNPC() { continue }`)
Insert, in C order (CAN_SEE + NOHASSLE first, before the existing wimpy check):
```go
if !canSee(ch, vict) {
    continue
}
if vict.GetFlags()&(1<<PrfNohassle) != 0 {
    continue
}
```
Then, **after** the existing protect-good check (~line 235) add the sneak skip
shown above.

### 2. Memory block (~line 318, after `if vict.IsNPC() { continue }`)
Insert:
```go
if !canSee(ch, vict) {
    continue
}
if vict.GetFlags()&(1<<PrfNohassle) != 0 {
    continue
}
```

### 3. AGGR24 block (~line 369, inside the `for _, p := range ...` loop, before
the `p.GetLevel() >= 24` check)
Insert:
```go
if !canSee(ch, p) {
    continue
}
if p.GetFlags()&(1<<PrfNohassle) != 0 {
    continue
}
if w.roomHasFlag(ch.RoomVNum, "peaceful") {
    continue
}
```
(`w.roomHasFlag` is invariant across the loop, but placing it inside per-victim
keeps the edit trivial and matches C's per-iteration test; fine either way.)

## Tests (add to pkg/game — e.g. mobact_test.go if it exists, else a new file)

Use the existing pkg/game test scaffolding (see `newCombatTestWorld` /
`spawnTargetMob` in combat_test.go, and however existing mobact tests set mob
flags). For each of the three blocks, assert the mob does NOT initiate combat
when the sole candidate player is:
- not visible to the mob (`canSee` false — set the mob unable to see, or the
  player invisible without the mob having detect-invis; use whatever mechanism
  existing canSee tests use),
- NOHASSLE (`p.SetFlag(PrfNohassle, ...)` or the existing setter),
- (AGGR24 only) in a `peaceful` room.
And a positive control: a plain visible non-NOHASSLE player in a normal room
still gets attacked. For the sneak case, since it's probabilistic, either seed/
inject the RNG if the file supports it, or assert the weaker invariant that a
sneaking player is attacked strictly less often than a non-sneaking one over N
iterations — prefer a deterministic RNG hook if one exists.

If mob-aggro combat initiation is awkward to assert directly (StartCombat on a
nil/■ engine), assert via the mob's resulting FIGHTING/target state or the
combat engine's registered pairs — match whatever existing mobact tests do.

## Definition of done

- All three blocks gate on CAN_SEE + NOHASSLE; aggressive also on sneak; AGGR24
  also on peaceful-room.
- `go build ./...`, `go test ./pkg/game/...`, `-race` on pkg/game, `gofumpt -l`
  (empty), `go vet ./pkg/game/...` all clean.
- Commit trailer `Co-Authored-By: <your executor id>`; PR body notes DP-1034 / F14.

## Scope guard

Touch ONLY `pkg/game/mobact.go` and a pkg/game test file. Do not modify
`world.go` or `act_movement.go` (a parallel GLM task, DP-1029, owns those), and
do not touch combat/damage/death/spell files (a Claude task, DP-1022, owns
those). Staying in mobact.go keeps every in-flight branch conflict-free.
