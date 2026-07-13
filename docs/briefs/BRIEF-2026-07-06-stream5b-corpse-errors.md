# BRIEF — Stream 5: Corpse/donation error handling (F9)

**Linear:** DP-956 (F9 — corpse/donation item transfers swallow errors)
**Effort:** M
**Agent:** Reek (DeepSeek)
**Source of truth:** docs/reports/REVIEW-2026-07-05-full-audit.md — F9

## Goal

Replace all 8 swallowed `_ =` errors on `MoveObjectToContainer`/`MoveObjectToRoom` in `death.go` and `item_donate.go` with proper error handling: log at `slog.Error` with context, and attempt rollback where possible.

## Problem

Eight `_ =` swallowed errors on item transfers in the most emotionally charged moment the game has: death and donation. If a move fails, items/gold **vanish permanently** with no log and no fallback. The player has already had the item removed from inventory but it never reaches the destination.

## Affected Sites

### death.go — makeCorpse (3 sites)

| Line | Call | Context | Silent failure consequence |
|---|---|---|---|
| 805 | `w.MoveObjectToContainer(item, corpse)` | Transfer inventory item to corpse | Item vanishes, not in corpse |
| 810 | `w.MoveObjectToContainer(item, corpse)` | Transfer equipped item to corpse | Gear vanishes |
| 818 | `w.MoveObjectToContainer(moneyObj, corpse)` | Transfer gold to corpse | Gold lost |

### death.go — makeDust (3 sites)

| Line | Call | Context | Silent failure consequence |
|---|---|---|---|
| 830 | `w.MoveObjectToRoom(item, roomVNum)` | Scatter inventory to room floor | Items vanish |
| 837 | `w.MoveObjectToRoom(item, roomVNum)` | Scatter equipped to room floor | Gear vanishes |
| 844 | `w.MoveObjectToRoom(moneyObj, roomVNum)` | Scatter gold to room floor | Gold lost |

### item_donate.go (2 sites)

| Line | Call | Context | Silent failure consequence |
|---|---|---|---|
| 34 | `w.MoveObjectToRoom(obj, donationRoom)` | Donate item to donation room | Item already removed from player, never arrives |
| 65 | `w.MoveObjectToRoom(moneyObj, donationRoom)` | Donate gold to donation room | Gold already deducted, never arrives |

**Notable contrast:** Line 862 in `makeDust` already does proper error handling for the ash object:
```go
if err := w.MoveObjectToRoom(ash, roomVNum); err != nil {
    slog.Warn("MoveObjectToRoom failed in makeDust", "room", roomVNum, "error", err)
}
```

Follow this pattern.

## Fix

### 1. Replace `_ =` with error check + slog.Error

For each site, change from:
```go
_ = w.MoveObjectToContainer(item, corpse)
```
to:
```go
if err := w.MoveObjectToContainer(item, corpse); err != nil {
    slog.Error("failed to move item to corpse",
        "player", ch.GetName(), "item_vnum", item.VNum, "error", err)
    // Attempt to give the item back to the player as fallback
    w.MoveObjectToRoom(item, roomVNum)
}
```

### 2. Rollback strategy

When a move fails, the item is in limbo — detached from the player but not at the destination. Attempt a fallback move:

- **Corpse context:** If moving item to corpse fails, scatter to room floor (`MoveObjectToRoom(item, roomVNum)`). The item is at least recoverable.
- **Dust context:** If scattering fails, this is already the fallback. Log and move on — the item is truly lost, but we have the log.
- **Donation context:** If moving to donation room fails, give item back to player inventory. Player already saw the donation message, so also send an error message.

### 3. Donation rollback messages

For donation failures, send a player-visible message:
```go
if err := w.MoveObjectToRoom(obj, donationRoom); err != nil {
    slog.Error("donation move failed", "player", ch.GetName(), "item_vnum", obj.VNum, "error", err)
    ch.SendMessage("Something went wrong. Your item was not donated.\r\n")
    // Return item to player inventory as fallback
    ch.Inventory.AddItem(obj)
}
```

### 4. Tests

- `TestMakeCorpse_MoveError_LogsAndScattersToRoom` — mock MoveObjectToContainer to fail, verify slog.Error called and MoveObjectToRoom fallback attempted
- `TestPerformDispose_MoveError_ReturnsToPlayer` — mock MoveObjectToRoom to fail on donation, verify item returned to player inventory
- Since these are hard to unit test without mocking the World methods, at minimum add a test that exercises the code path. If mocking World is too invasive, skip tests for now and document why.

## Files

| File | Change |
|---|---|
| `pkg/game/death.go` | Replace 6 `_ =` with error check + fallback (lines 805, 810, 818, 830, 837, 844) |
| `pkg/game/item_donate.go` | Replace 2 `_ =` with error check + rollback (lines 34, 65) |

## Build Gate

```bash
go build ./...
go vet ./...
go test -race $(go list ./... | grep -v /tests/unit) -timeout 120s
gofumpt -l .
golangci-lint run ./...
```

## Constraints

1. Do NOT change the save file format.
2. Do NOT change the order of operations in death.go — the item detach → move sequence is intentional.
3. Do NOT add new dependencies — use existing `slog.Error` and `slog.Warn`.
4. Follow the existing pattern at death.go:862 (ash object error handling).
5. Single PR.

## C Fidelity

C's `obj_to_obj()` and `obj_to_room()` return void — they can't fail. Go's `MoveObjectToContainer`/`MoveObjectToRoom` return errors because they need to update location tracking state. The error handling is a Go-specific addition that makes the system more robust than C.
