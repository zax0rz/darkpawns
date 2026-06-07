# Brief: CODE-001 — Incomplete Class Name Lookup in Test Utility

**Issues:** DP-539 (MEDIUM)
**Priority:** MEDIUM — test tooling (housekeeping, no urgency)
**File:** `cmd/test-race/main.go`

## Problem

`cmd/test-race/main.go:24-30` — The `classes` struct list is hardcoded with only Mage and Warrior. The game has 12 classes (Mage, Cleric, Thief, Warrior, Magus, Avatar, Assassin, Paladin, Ninja, Psionic, Ranger, Mystic) but the test tool only exercises 2.

`allClassNames` (line 33) is assigned from `game.ClassNames` and IS used at line 104 for name lookup in restriction tests — it's just not used for building the class loop list. The test utility can't print race bonus averages for 10 of 12 classes.

This is a test utility issue, not production code. `ValidUserClassChoice` and `RollRealAbils` themselves work correctly.

## Required Fix

### Step 1: Replace hardcoded class list (lines 24-30)

Replace the static struct slice with one built from `game.ClassNames`:

```go
// Before (lines 24-30):
classes := []struct {
    name string
    id   int
}{
    {"Mage", game.ClassMageUser},
    {"Warrior", game.ClassWarrior},
}

// After: build from game.ClassNames, sorted by ID for deterministic output
classes := make([]struct {
    name string
    id   int
}, 0, len(game.ClassNames))
for id, name := range game.ClassNames {
    classes = append(classes, struct {
        name string
        id   int
    }{name, id})
}
sort.Slice(classes, func(i, j int) bool {
    return classes[i].id < classes[j].id
})
```

**Note:** `game.ClassNames` is a `map[int]string` (character.go:53). Map iteration in Go is non-deterministic — must sort after building the slice to get consistent output order. Add `"sort"` to imports.

### Step 2: Add import

Add `"sort"` to the import block (line 4).

### Step 3: Expand restriction test cases (lines 88-97)

Current test cases only cover Ninja, Mage, Warrior, Cleric, Magus, Avatar. Add coverage for remaining classes:

```go
testCases := []struct {
    race  int
    class int
    valid bool
}{
    // Existing:
    {game.RaceHuman, game.ClassNinja, true},      // Ninja: Human only
    {game.RaceElf, game.ClassNinja, false},        // Ninja: non-Human rejected
    {game.RaceHuman, game.ClassMageUser, true},    // Mage: all races
    {game.RaceRakshasa, game.ClassWarrior, true},  // Warrior: all races
    {game.RaceSsaur, game.ClassCleric, true},      // Cleric: all races
    {game.RaceHuman, game.ClassMagus, false},      // Magus: remort-only
    {game.RaceHuman, game.ClassAvatar, false},     // Avatar: remort-only
    // New:
    {game.RaceHuman, game.ClassThief, true},       // Thief: all races
    {game.RaceHuman, game.ClassPsionic, true},     // Psionic: all races
    {game.RaceHuman, game.ClassAssassin, false},   // Assassin: remort-only
    {game.RaceHuman, game.ClassPaladin, false},    // Paladin: remort-only
    {game.RaceHuman, game.ClassRanger, false},     // Ranger: remort-only
    {game.RaceHuman, game.ClassMystic, false},     // Mystic: remort-only
    {game.RaceDwarf, game.ClassNinja, false},      // Ninja: Dwarf rejected
    {game.RaceMinotaur, game.ClassMageUser, true}, // Mage: Minotaur allowed
}
```

ValidUserClassChoice (character.go:234) allows: Mage, Cleric, Thief, Warrior, Psionic for all races. Ninja for Human only. All others (Magus, Avatar, Assassin, Paladin, Ranger, Mystic) are remort-only and return false.

## Verification

1. `go build ./cmd/test-race/`
2. `go vet ./cmd/test-race/`
3. Run the tool — should print all 12 classes with race bonus averages in deterministic order
4. Restriction tests should all pass with ✓ marks
5. `go test ./pkg/game/...` — existing chargen tests still pass

## Context

Pure housekeeping. No urgency — the test tool works, it just doesn't cover all classes. Low-risk, narrow scope, good candidate for a quick Flash pass.
