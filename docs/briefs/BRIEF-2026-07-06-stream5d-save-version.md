# BRIEF — Stream 5: Save format version field (F18)

**Linear:** DP-958 (F18 — player save format has no version field)
**Effort:** S
**Agent:** Reek (DeepSeek)
**Source of truth:** docs/reports/REVIEW-2026-07-05-full-audit.md — F18

## Goal

Add a `SaveVersion int` field to `savePlayerData` (and `saveWorldData`). Log a warning on load when version mismatch is detected. Establish the version at 1 (current format).

## Problem

The player save format (`pkg/game/save.go`) has no version field. `json.Unmarshal` silently assigns zero values to missing fields and silently discards unknown fields. A field rename drops data with no warning. The L-11 affect-loss bug (April 2026) was exactly this class: affects loaded but never restored from save data.

## Current Save Structs

### `savePlayerData` (save.go:22-65)
No version field. ~35 fields including Stats, SpellMap, Skills, Inventory, Equipment, Affects, etc.

### `saveWorldData` (save.go:397)
No version field. Fields: NextMobID, NextObjID, DoorStates, Mobs, RoomItems, Gossip.

## Fix

### 1. Add SaveVersion to both structs

```go
type savePlayerData struct {
    SaveVersion int `json:"save_version"` // bumped on save format changes
    // ... existing fields ...
}

type saveWorldData struct {
    SaveVersion int `json:"save_version"`
    // ... existing fields ...
}
```

### 2. Set version on save

In `SavePlayer()` (and wherever `savePlayerData` is constructed):

```go
data.SaveVersion = CurrentSaveVersion
```

Define the constant:
```go
const CurrentSaveVersion = 1 // initial versioned save format
```

### 3. Check version on load

In `LoadPlayer()` / `DeserializePlayer()`, after `json.Decode`:

```go
if data.SaveVersion != 0 && data.SaveVersion != CurrentSaveVersion {
    slog.Warn("player save version mismatch",
        "player", data.Name,
        "file_version", data.SaveVersion,
        "expected_version", CurrentSaveVersion,
        "action", "loading with possible data loss")
}
```

Key detail: `SaveVersion == 0` means "old save file, no version field" — log at `slog.Info` level (not warning), since this is expected for existing saves upgrading to the versioned format. Only `SaveVersion != 0 && SaveVersion != CurrentSaveVersion` is a warning (future save format was downgraded or corrupted).

Do the same for `saveWorldData` loading.

### 4. Tests

- `TestSavePlayer_IncludesVersion` — save a player, unmarshal, verify SaveVersion == 1
- `TestLoadPlayer_OldSave_NoVersion_Warns` — load a save JSON without save_version field, verify no error (graceful upgrade) and version is 0
- `TestLoadPlayer_FutureVersion_Warns` — load a save JSON with save_version: 99, verify warning logged and data still loaded

## Files

| File | Change |
|---|---|
| `pkg/game/save.go` | Add SaveVersion field to both structs, set on save, check on load |
| `pkg/game/save_test.go` | Add version field tests |

## Build Gate

```bash
go build ./...
go vet ./...
go test -race $(go list ./... | grep -v /tests/unit) -timeout 120s
gofumpt -l .
golangci-lint run ./...
```

## Constraints

1. **Do NOT reject saves with wrong version.** The load must always succeed — this is a warning system, not a gate. Rejecting saves would lock players out.
2. **Do NOT implement a full migration system.** That's a future enhancement. This PR just establishes the version field and the warning.
3. **Do NOT change any existing field names or JSON tags.** That's a separate, dangerous change.
4. **`SaveVersion: 0` means old format.** Existing save files don't have the field — `json.Unmarshal` assigns 0. This must not be treated as an error.
5. Single PR.

## C Fidelity

C's save format (db.c) had no version field either — it used struct size checks. The Go JSON approach is strictly better. This is a Go-era improvement, no C behavior to match.

## Future Enhancement (NOT in scope)

A future PR could implement per-version migration functions (e.g., `migrateSaveV1ToV2(data)`) that run when `SaveVersion < CurrentSaveVersion`. This establishes the foundation for that.
