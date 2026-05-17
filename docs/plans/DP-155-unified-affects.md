# DP-155: Unified Affect System — Implementation Plan

## The Problem

Two independent affect systems in `pkg/engine/`:

1. **AffectManager** — Go-native, `AffectType` iota enum, central map, managed `Tick()`, stacking, locking
2. **MasterAffect** — Ported from C handler.c, integer spell types + `APPLY_*` constants, stored on Player, standalone functions

They don't share state. Removing one system's affects doesn't account for the other's. `AffectTotal` (stat recalculation) ignores AffectManager. `AffectManager.Tick()` ignores MasterAffects. Equipment affects and spell affects operate in parallel universes.

## The Goal

**One system. One data model. One tick. One stat recalculation.**

MasterAffect is absorbed into AffectManager. The `AffectType` enum is replaced with integer fields that match CircleMUD's model (`SpellID` + `Location`). All callers — spells, equipment, items, spells.go helpers — flow through a single AffectManager.

## Data Model Change

### Before: Two structs

```go
// AffectManager system
type Affect struct {
    ID        string
    Type      AffectType    // iota enum
    SpellID   int           // unused by AffectManager
    Duration  int
    Magnitude int
    Flags     uint64
    Source    string
    // ... stacking fields
}

// MasterAffect system
type MasterAffect struct {
    Type      int    // SPELL_* integer
    Duration  int
    Location  int    // APPLY_* constant
    Modifier  int
    Bitvector uint64
    ByType    int    // BY_SPELL, BY_ITEM
    ObjNum    int    // Object VNUM
}
```

### After: One struct

```go
type Affect struct {
    ID        string    // unique ID
    SpellID   int       // SPELL_* or SKILL_* number (replaces both AffectType and MasterAffect.Type)
    Duration  int       // duration in ticks
    Location  int       // APPLY_* constant (replaces AffectType's implicit mapping)
    Magnitude int       // stat modifier magnitude
    Flags     uint64    // AFF_* bitvector
    Source    string    // human-readable source name
    Origin    int       // BY_SPELL, BY_ITEM (new — MasterAffect had this, Affect didn't)
    ObjNum    int       // Object VNUM if from equipment (new)

    // Metadata
    AppliedAt time.Time
    ExpiresAt time.Time

    // Stacking
    StackID   string
    MaxStacks int
}
```

**Key change:** `AffectType` enum is gone. `SpellID` + `Location` replace it. The `Location` (APPLY_* constant) tells the system *which stat to modify*. The `SpellID` tells it *which spell created this*. Status flags are set via `Flags` bitvector, same as before.

### Storage Change

```go
// Before: key by AffectType (one affect per type per entity)
affects map[string]map[AffectType]*Affect

// After: flat slice per entity (multiple affects per spell, same entity)
affects map[string][]*Affect
```

## Stat Modification: Table-Driven

Replace `applyAffectImmediate`'s 40-case switch with a table lookup:

```go
var applyLocationTable = map[int]string{
    ApplyStr:          "STR",
    ApplyDex:          "DEX",
    ApplyInt:          "INT",
    ApplyWis:          "WIS",
    ApplyCon:          "CON",
    ApplyCha:          "CHA",
    ApplyMana:         "Mana",
    ApplyHit:          "HP",
    ApplyMove:         "Move",
    ApplyAC:           "AC",
    ApplyHitroll:      "Hitroll",
    ApplyDamroll:      "Damroll",
    ApplySavingPara:   "SavingPara",
    ApplySavingRod:    "SavingRod",
    ApplySavingPetri:  "SavingPetri",
    ApplySavingBreath: "SavingBreath",
    ApplySavingSpell:  "SavingSpell",
}
```

`applyAffectImmediate` becomes:
```go
func (am *AffectManager) applyAffectImmediate(entity Affectable, affect *Affect) {
    if statName, ok := applyLocationTable[affect.Location]; ok {
        addStat(entity, statName, affect.Magnitude)
    }
    // Status flags via bitvector
    if affect.Flags != 0 {
        entity.SetStatusFlag(affect.Flags)
    }
}
```

`removeAffectImmediate` is the same but subtracts and clears flags.

**Note on status flags:** The old system set/cleared individual bits based on `AffectType`. The new system stores the flag bitvector directly on the Affect. When applying, OR the flags onto the entity. When removing, we need reference counting on flag bits (DP-152) — or enforce MaxStacks=1 for flag affects so only one instance exists.

## Affect Lookup: Composite Key

For `HasAffect` and `RemoveAffectsByType`, we need to look up by spell type. Since `SpellID` is the spell identifier:

```go
func (am *AffectManager) HasAffectBySpell(entity Affectable, spellID int) bool {
    // iterate entity's affects, check SpellID
}

func (am *AffectManager) RemoveAffectsBySpell(entity Affectable, spellID int) int {
    // remove all affects with matching SpellID
}
```

This replaces `AffectedBySpell` and `AffectFromChar`.

## Duration Tracking: Per-Spell

Multiple Affects from the same spell (e.g. Bless → STR + hitroll) share a duration. When the spell expires, ALL its affects expire together.

**Approach:** Duration is tracked per-Affect, but when creating multiple affects from one spell cast, they all get the same initial duration. `Tick()` decrements each independently — when one expires, we remove ALL affects with the same `SpellID`.

```go
func (am *AffectManager) Tick() {
    // ... decrement durations ...
    // When an affect expires, also remove all affects with same SpellID on same entity
    expiredSpellID := aff.SpellID
    am.removeAffectsBySpell(entityID, expiredSpellID)
}
```

## RecalculateStats: The New AffectTotal

```go
func (am *AffectManager) RecalculateStats(entity Affectable) {
    // Phase 1: Strip all stat modifications (both our affects AND equipment)
    // Phase 2: Re-apply all in order

    // Strip pass: iterate all affects, subtract Magnitude from each Location
    for _, aff := range am.affects[entityID] {
        am.removeAffectImmediate(entity, aff)
    }

    // Also strip equipment affects (if entity provides them)
    if ep, ok := entity.(EquipAffectProvider); ok {
        for _, ea := range ep.GetEquipAffects() {
            am.removeAffectImmediate(entity, &Affect{Location: ea.Location, Magnitude: -ea.Modifier, Flags: ea.Bitvector})
        }
    }

    // Re-apply pass
    for _, aff := range am.affects[entityID] {
        am.applyAffectImmediate(entity, aff)
    }
    if ep, ok := entity.(EquipAffectProvider); ok {
        for _, ea := range ep.GetEquipAffects() {
            am.applyAffectImmediate(entity, &Affect{Location: ea.Location, Magnitude: ea.Modifier, Flags: ea.Bitvector})
        }
    }

    // Clamp pass (STR 3-18, etc.)
    am.clampStats(entity)
}
```

**Important:** Equipment affects are NOT stored in AffectManager. They're recalculated on demand via `EquipAffectProvider.GetEquipAffects()`. This matches C's `affect_total` which strips and reapplys both eq and spell affects each time.

## Implementation Phases

### Phase 1: Extend Affect struct + Kill AffectType enum

**Files:** `pkg/engine/affect.go`, `pkg/engine/affect_manager.go`

1. Add `Location`, `Origin`, `ObjNum` fields to `Affect`
2. Add `applyLocationTable` mapping APPLY_* → stat names
3. Rewrite `applyAffectImmediate` / `removeAffectImmediate` to use table
4. Change storage from `map[string]map[AffectType]*Affect` to `map[string][]*Affect`
5. Add `HasAffectBySpell(spellID)` and `RemoveAffectsBySpell(spellID)` methods
6. Add `RecalculateStats(entity)` method
7. Keep `AffectType` enum temporarily as aliases (deprecated) so callers compile during migration

**Test:** Existing `affect_test.go` must still pass after adapting to new struct fields.

### Phase 2: Update AffectManager callers

**Files:** `pkg/engine/affect_helpers.go`, `pkg/engine/affect_tick.go`

1. Rewrite `AffectToChar` → calls `am.ApplyAffect` with Origin=BySpell
2. Rewrite `AffectRemove` → calls `am.RemoveAffect`
3. Rewrite `AffectFromChar` → calls `am.RemoveAffectsBySpell`
4. Rewrite `AffectedBySpell` → calls `am.HasAffectBySpell`
5. Rewrite `AffectJoin` → finds existing by SpellID+Location, merges or replaces
6. Rewrite `AffectTotal` → calls `am.RecalculateStats`
7. Delete `MasterAffect` struct, `StatModifiable` interface (no longer needed — entity just implements `Affectable`)
8. `affect_update.go` `AffectUpdate` → delegates to `am.Tick()` (or becomes redundant)

**Test:** `affect_test.go` + new tests for RecalculateStats.

### Phase 3: Update spell system

**Files:** `pkg/spells/affect_spells.go`, `pkg/spells/affect_effects.go`

1. `spellAffectMap` changes from `map[int]engine.AffectType` to a richer structure:
   ```go
   type spellAffectDef struct {
       Location int    // APPLY_* constant
       MagnitudeFn func(casterLevel int) int  // duration/magnitude calculators
       DurationFn  func(casterLevel int) int
   }
   ```
2. `ApplySpellAffects` creates Affects with `SpellID` + `Location` instead of `AffectType`
3. `MagAffects` switch cases create Affects with `SpellID` + `Location`
4. Remove all `engine.AffectType(...)` casts

**Test:** Spell casting tests.

### Phase 4: Update game package callers

**Files:** `pkg/game/player.go`, `pkg/game/player_stats.go`, `pkg/game/affect_update.go`, `pkg/game/save.go`, `pkg/session/` files

1. Remove `MasterAffects` field from Player struct
2. `Player.ActiveAffects` becomes the canonical list (or AffectManager holds it)
3. `GetMasterAffects`/`SetMasterAffects`/`AddMasterAffect`/`RemoveMasterAffect` → removed
4. `GetEquipAffects` stays (equipment provides affects on demand)
5. `save.go` serialization: save `[]*Affect` directly (not `saveAffect` wrapper)
6. All `AffectToChar`/`AffectRemove`/`AffectTotal` callers updated
7. `equipment_ac.go`, `eat_cmds.go`, `use_cmds.go`, `tattoo.go` — update to use new API
8. `player_affects.go` — update `IsAffected` to check AffectManager

**Test:** Full game test suite, save/load roundtrip.

### Phase 5: Remove AffectType enum + cleanup

**Files:** `pkg/engine/affect.go`

1. Delete `AffectType` enum constants (AffectStrength, AffectBlind, etc.)
2. Delete `applyLocFromAffectType` (no longer needed)
3. Clean up any remaining references
4. Update tests

## Testing Strategy

### Unit Tests (per phase)
- `affect_test.go`: Affect creation, stacking, duration, tick expiration
- `RecalculateStats` test: cast buff → equip item → remove buff → verify stats
- `RecalculateStats` test: equip item → cast debuff → unequip → verify debuff still active
- Spell affect tests: each spell creates correct Location + Magnitude + Duration

### Integration Tests
- Save/load roundtrip: save player with affects, load, verify affects restored
- Equipment + spell interaction: full equip/cast/remove/unequip cycle
- Duration tick: verify all affects from one spell expire together

### Manual Verification
- Cast Bless → verify STR and hitroll both increase
- Equip +1 STR item → cast Bless → remove Bless → verify STR still has item bonus
- Cast Blindness → verify hitroll penalty + blind flag both apply
- Wait for duration expiry → verify all effects removed cleanly

## Risk Assessment

**Medium risk.** The changes are mechanical but touch the entire affect pipeline:
- Every spell in `affect_spells.go` (44 usages of AffectType)
- Every equipment affect path
- Save/load serialization
- All callers of `AffectToChar`/`AffectRemove`/`AffectTotal`

**Mitigation:**
- Phase-by-phase with tests passing at each phase
- Keep deprecated AffectType aliases during migration (compile-time safety)
- Full test suite + race detector after each phase
- Manual verification of key spell/equipment interactions

## Estimated Effort

- Phase 1 (data model): 1 session
- Phase 2 (AffectManager callers): 1 session
- Phase 3 (spell system): 1 session
- Phase 4 (game package): 1-2 sessions
- Phase 5 (cleanup): 0.5 session
- **Total: 4-5 focused sessions**
