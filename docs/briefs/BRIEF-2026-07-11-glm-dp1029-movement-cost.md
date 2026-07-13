# BRIEF 2026-07-11 — GLM: DP-1029 / F9 Movement cost table + immortal exemption

**Executor:** GLM. **Branch:** `glm/dp1029-movement-cost` (fresh off current `main`,
your own clone/worktree — never share a HEAD with another executor). **One PR.**
Claude verified the C source and every Go anchor below against the tree at
`main` @ `c6794f2`. Implement to match exactly.

## The bug (three parts)

1. **Live movement-cost table is wrong for almost every sector.** The live move
   path is `pkg/game/world.go` `sectorMoveCost` (a `switch`, ~line 969), used by
   `MovePlayer` at world.go:941. Its values (1,1,2,3,4,6,4,4,4,1 + `default:1`)
   match C on almost nothing, and sectors 10–15 (desert / the four elemental
   planes / swamp) all fall through `default` to **1**.
2. **Immortals pay movement.** `MovePlayer` (world.go:941-945) charges everyone.
   C exempts immortals **and** NPCs (src/act.movement.c:210-211:
   `if (GET_LEVEL(ch) < LVL_IMMORT && !IS_NPC(ch)) GET_MOVE(ch) -= need_movement;`).
   `MovePlayer` is the player path, so the fix is: skip the cost when
   `p.Level >= LVL_IMMORT`.
3. **The secondary table has a comment-vs-value swap bug at indices 8/9.**
   `pkg/game/act_movement.go` `movementLoss` (line 74) is correct on every entry
   **except** UNDERWATER and FLYING, which are backwards — see the trap below.

## THE TRAP — read this before touching the table

C's `movement_loss[]` (src/constants.c:1345-1363) is indexed by the `SECT_*`
enum. But the **inline comments in the C array are shifted/swapped** relative to
the actual enum, specifically at indices 8 and 9:

```c
const int movement_loss[] = {
  2,  /* Inside */      // idx 0
  2,  /* City */        // idx 1
  3,  /* Field */       // idx 2
  4,  /* Forest */      // idx 3
  5,  /* Hills */       // idx 4
  7,  /* Mountains */   // idx 5
  5,  /* Swimming */    // idx 6  (SECT_WATER_SWIM)
  6,  /* Unswimable */  // idx 7  (SECT_WATER_NOSWIM)
  2,  /* Flying */      // idx 8  <-- comment LIES: enum SECT_UNDERWATER=8, so cost is 2
  6,  /* Underwater */  // idx 9  <-- comment LIES: enum SECT_FLYING=9, so cost is 6
  8,  /* Desert */      // idx 10 (SECT_DESERT)
  6,  /* Fire Plane */  // idx 11
  6,  /* Eart Plane */  // idx 12
  6,  /* Wind Plane */  // idx 13
  6,  /* Water Plane */ // idx 14
  4,  /* Swamp */       // idx 15 (SECT_SWAMP)
};
```

`structs.h`: `SECT_UNDERWATER=8`, `SECT_FLYING=9`. **The array is indexed by the
enum, and the runtime uses `movement_loss[SECT(room)]` — so the VALUES win, the
comments are noise.** Runtime behavior: **UNDERWATER (sector 8) = 2, FLYING
(sector 9) = 6.** (Underwater is cheap; flying costs like a plane.)

`act_movement.go` `movementLoss` copied the *comments* — it has UNDERWATER(8)=6
and FLYING(9)=2, i.e. **swapped**. Fix those two to UNDERWATER=2, FLYING=6.

## The one true table (index = SECT_* enum value)

```
2, 2, 3, 4, 5, 7, 5, 6, 2, 6, 8, 6, 6, 6, 6, 4
```

## Implementation plan

### 1. Single shared table (kill the duplicate)

There must be exactly ONE `movementLoss` table. `act_movement.go` already
declares `var movementLoss = []int{...}` (line 74) — keep that one, correct
indices 8 and 9 to `2` (UNDERWATER) and `6` (FLYING). Leave the other 14 entries
as-is (they're already right). Add a short comment noting the C comment/value
swap so nobody "fixes" it back.

### 2. Rewrite `world.go` `sectorMoveCost` to read the shared table

Replace the entire `switch` (world.go:967-994) with a bounds-checked lookup:

```go
func sectorMoveCost(sector int) int {
    if sector < 0 || sector >= len(movementLoss) {
        return movementLoss[SECT_INSIDE] // sane default; C never indexes OOB
    }
    return movementLoss[sector]
}
```

(`movementLoss` and `SECT_INSIDE` are both already in package `game`.)

### 3. Immortal exemption in `MovePlayer`

At world.go:941-945, wrap the cost so immortals move free (mirrors
act.movement.c:210). `LVL_IMMORT` is already available in package `game`:

```go
moveCost := (sectorMoveCost(currentRoom.Sector) + sectorMoveCost(newRoom.Sector)) / 2
if p.Level < LVL_IMMORT && !p.SpendMove(moveCost) {
    errMsg = "You are too exhausted.\r\n"
    moveErr = fmt.Errorf("too exhausted")
} else {
    // ... existing success body (set RoomVNum, adjust light, result = newRoom) ...
}
```

Note the short-circuit: immortals never call `SpendMove`, so they neither pay nor
get blocked. Confirm `p.Level` is readable here without a deadlock — the
surrounding code holds `w.mu`; `p.Level` is a plain field read used elsewhere in
this function, so it's fine (match the existing access pattern; don't add a
lock).

### 4. Do NOT touch the secondary `doSimpleMove` path's own deduction

`act_movement.go` `doSimpleMove` is a secondary/legacy path (the audit notes the
live path is `MovePlayer`). Just fixing its table (step 1) is enough — do not
rewire its flow. Keep this PR surgical.

## Tests (add to pkg/game)

- Table sanity: assert `movementLoss` has 16 entries and equals the one-true
  table above (guards the swap from regressing). Explicitly assert
  `movementLoss[SECT_UNDERWATER] == 2` and `movementLoss[SECT_FLYING] == 6` with
  a comment pointing at this brief.
- `sectorMoveCost`: sample a few sectors incl. DESERT(10)=8 and an out-of-range
  index returning the INSIDE default (not panicking).
- Immortal exemption: build a level-`LVL_IMMORT` player and a mortal, move each
  between two rooms of known sectors; assert the mortal's move points dropped by
  `(src+dst)/2` and the immortal's did not. Use the existing
  `newCombatTestWorld` / room-spawn helpers in pkg/game tests as scaffolding
  (see combat_test.go); add a second room with a non-default sector if needed.

## Definition of done

- One `movementLoss` table, correct 16 values, live `sectorMoveCost` reads it.
- Immortals pay 0 movement; mortals pay `(src+dst)>>1`.
- `go build ./...`, `go test ./pkg/game/...`, `-race` on pkg/game, `gofumpt -l`
  (must be empty), `go vet ./pkg/game/...` all clean.
- Commit trailer: `Co-Authored-By: <your executor id>`. PR body notes DP-1029 / F9.

## Scope guard

Touch only `pkg/game/world.go`, `pkg/game/act_movement.go`, and a new/existing
pkg/game test file. Do NOT touch combat, damage, death, or spell files — a
parallel Claude task (F2/DP-1022) owns those; conflicts there create a git mess.
