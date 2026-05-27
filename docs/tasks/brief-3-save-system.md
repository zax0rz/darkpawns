# Brief 3: Save System Cluster (Sonnet — 5 issues)

These all touch the player save/load path. Test together. One regression = data loss.

## CRITICAL: DP-499 — Idlesave deletes entire player character

**File:** `pkg/game/objsave.go:725-726`
**Bug:** `DeleteCrashFile` calls `DeletePlayer(name)` which erases the entire character JSON.
**C behavior:** `Crash_delete_file` only deletes `<name>.objs` (item save), NOT `<name>.plr` (character sheet).

**Fix:**
```go
func DeleteCrashFile(name string) bool {
    // Load the player save record
    record, err := LoadPlayerRecord(name)
    if err != nil {
        slog.Error("DeleteCrashFile: failed to load", "name", name, "error", err)
        return false
    }
    // Clear only the item fields, NOT the character
    record.Inventory = nil
    record.Equipment = nil
    record.RentCode = 0
    record.RentTime = 0
    record.NetCostPerDiem = 0
    // Re-save the character with cleared items
    if err := SavePlayerRecord(name, record); err != nil {
        slog.Error("DeleteCrashFile: failed to save cleared record", "name", name, "error", err)
        return false
    }
    slog.Debug("DeleteCrashFile", "player", name)
    return true
}
```

**Verify:** Check how `SavePlayerRecord` and `LoadPlayerRecord` work (or their equivalents). The key insight: we want to keep the character but clear the items.

---

## DP-496 — DB save path strips all item state

**File:** `pkg/db/convert.go:117-139`
**Bug:** `inventoryVnums` returns `[]int` (VNums only). `equipmentVnums` returns `map[string]int`. All custom object state lost.

**Fix:** Change these functions to return `saveItemData` structs instead of raw VNums:
```go
func inventorySaveData(inv *game.Inventory) []game.SaveItemData {
    if inv == nil {
        return []game.SaveItemData{}
    }
    items := inv.FindItems("")
    result := make([]game.SaveItemData, 0, len(items))
    for _, item := range items {
        result = append(result, game.SaveItemData{
            VNum:   item.GetVNum(),
            Count:  1,
            Locate: 0,
            State:  item.GetSaveState(),
        })
    }
    return result
}
```

**Note:** You'll need to export `saveItemData` (rename to `SaveItemData`) from `pkg/game/save.go` so `pkg/db` can use it. Or create a shared type.

---

## DP-497 — Container nesting hierarchy lost

**File:** `pkg/game/save.go:70-75` (saveItemData struct)
**Bug:** No `ParentContainer` field. Nested items are flattened to root inventory.

**Fix:** Add nesting support to the save format:
```go
type saveItemData struct {
    VNum             int                    `json:"vnum"`
    Count            int                    `json:"count"`
    Locate           int                    `json:"locate"`
    State            map[string]interface{} `json:"state,omitempty"`
    ContainerVNum    int                    `json:"container_vnum,omitempty"` // NEW: parent container VNum (0 = root inventory)
    ContainerIndex   int                    `json:"container_index,omitempty"` // NEW: index of container in the save list
}
```

On save: when serializing items, track which container each item is in by walking the container hierarchy.
On load: reconstruct nesting by matching ContainerIndex references.

**Note:** This is the most complex change in this brief. If it's too risky, defer it — the other fixes are more urgent.

---

## DP-498 — Receptionist ignores bank gold

**File:** `pkg/game/objsave.go` in `GenReceptionist`, around line 740
**Change:** `cost <= p.Gold` → `cost <= p.Gold+p.BankGold`

---

## DP-494 — Rent deadline message discarded

**File:** `pkg/game/objsave.go` around line 693
**Change:** `_ = fmt.Sprintf(...)` → send the message to the player

---

## Build verification
```bash
go build ./... && go vet ./... && go test ./...
```

## Testing notes
- DP-499 is the highest priority. Test by: idling out a test character with empty inventory, verifying character still exists on next login.
- DP-496/497: test by equipping a renamed item, saving, reloading, verifying the name persists.
