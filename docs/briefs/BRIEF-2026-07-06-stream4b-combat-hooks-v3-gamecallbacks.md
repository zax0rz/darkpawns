# Brief: Stream 4b v3 — GameCallbacks Struct Migration (F8)

## Context

DP-952 from the Fable Audit 2026-07-05 (originally C-02 from April review).
54 live package-level function hooks in `pkg/combat/fight_core.go:14-77`.

These hooks are the **bridge** between the combat package (which only sees the
`Combatant` interface with 25 methods) and the game layer (Player/Mob structs
with direct state access). The combat package uses name-based lookups to query
game state (affects, flags, gold, groups, skills, etc.) that the Combatant
interface doesn't expose.

**Problem:** Package-level vars written at boot, read at runtime with no
synchronization. Blocks testing (can't wire two engines with different
callbacks). Nil-panic risk on partial init. Every hook call is a name-based
dispatch that couples combat to the naming layer.

**Audit's suggested fix (from REVIEW-2026-07-05-full-audit.md):**
> Incremental — introduce a `GameCallbacks` struct on `CombatEngine`, migrate
> hooks in batches, validate non-nil at construction.

**Linear:** DP-952 (F8)
**Branch:** `refactor/stream4b-gamecallbacks`
**Agent:** Kimi

## Architecture

### Why a struct, not interface methods?

Adding hooks to the `Combatant` interface won't work because most hooks query
*other* characters, not self. Example: `HasMobFlag(victimName)` — the attacker
needs to check the *victim's* mob flags. Putting that on the Combatant interface
would require every combatant to know about every other combatant's data.

Adding function parameters to `TakeDamage` won't work either — it already has
complex control flow and touches 10 hook groups. Adding 40+ parameters is
unmanageable.

The `GameCallbacks` struct encapsulates all game-layer queries in one place,
passed once at engine construction, and validated for completeness.

### CombatEngine already has some callbacks

The engine path uses struct callbacks (BroadcastFunc, MessageFunc, DeathFunc,
OnCombatAction, etc.). The fight_core hooks serve the legacy fight_core path.
After migration, both paths use the same `GameCallbacks` struct — the engine
passes it through to fight_core functions.

## Migration Plan — 3 PRs

### PR 1 — Foundation + Messaging Hooks

**Goal:** Create `GameCallbacks` struct, add it to `CombatEngine`, migrate the
7 messaging/output hooks. Also delete the 3 truly dead hooks from v2.

**New file:** `pkg/combat/callbacks.go`

```go
type GameCallbacks struct {
    // Messaging (Group 1)
    Broadcast    func(roomVNum int, msg string, exclude string)
    SendToChar   func(name string, msg string)
    SkillMessage func(dam int, ch, vict string, attackType int, roomVNum int) bool
    BroadChat    func(chName string, msg string)
    Log          func(msg string, level string, minLevel int, toLog bool)
}
```

**Changes:**
1. Add `Callbacks *GameCallbacks` field to `CombatEngine` struct
2. Move `BroadcastMessage`, `SendToCharFunc`, `SkillMessageFunc`, `BroadChatFunc`,
   `LogMessage` from package vars to `GameCallbacks` struct
3. Update call sites: `BroadcastMessage(room, msg, "")` → `cb.Broadcast(room, msg, "")`
4. Pass `Callbacks` to fight_core functions that need it (or access via
   package-level var set from engine during construction)
5. Wire in `cmd/server/main.go` and `pkg/session/manager.go`
6. Delete 3 dead hooks: `BuildTHAC0`, `RunFightScript`, `IsInRoom`
7. Delete 6 dead wrapper functions: `Die`, `MakeCorpse`, `MakeDust`, `SkillMessage`,
   `Appear`, `fleshAlteredType`
8. Add `GetExp` nil guard at lines 565 and 1088
9. Add construction-time validation for Broadcast and SendToChar

**Transition pattern during migration:**

During the multi-PR migration, fight_core functions need access to `GameCallbacks`
but don't receive it as a parameter. Use a package-level var that's set during
engine construction:

```go
var callbacks *GameCallbacks

// Set during CombatEngine initialization
func SetCallbacks(cb *GameCallbacks) { callbacks = cb }
```

This is temporary — removed in PR 3 after all hooks are migrated.

**Files:**
- `pkg/combat/callbacks.go` — NEW
- `pkg/combat/fight_core.go` — modify (remove dead hooks/functions, update call sites)
- `pkg/combat/engine.go` — modify (add Callbacks field, wire SetCallbacks)
- `pkg/combat/skill_messages.go` — modify (update BroadcastMessage/SendToCharFunc)
- `pkg/session/manager.go` — modify (wire callbacks struct instead of package vars)
- `cmd/server/main.go` — modify (wire callbacks, add validation)

---

### PR 2 — Character State Hooks

**Goal:** Migrate all "read/write character state" hooks to GameCallbacks.

**Add to GameCallbacks:**

```go
// Character identity
GetRace      func(name string) int
GetRaceHate  func(name string, index int) int
GetAlignment func(name string) int
SetAlignment func(name string, val int)
GetSex       func(name string) int
GetSkill     func(name string, skillNum int) int

// Affects
HasAffect        func(name string, aff int) bool
HasAffectStr     func(name string, aff string) bool
RemoveAffect     func(name string, skillNum int)
RemoveAllAffects func(name string)

// Player/Mob/Room flags
HasPlrFlag    func(name string, flag string) bool
SetPlrFlag    func(name string) bool
HasPrfFlag    func(name string, flag string) bool
HasMobFlag    func(name string, flag string) bool
HasMobVNum    func(name string, vnum int) bool
HasRoomFlag   func(roomVNum int, flag string) bool
HasScriptFlag func(name string, flag string) bool
IsShopkeeper  func(name string) bool

// Equipment & Mounts
IsMounted     func(name string) bool
Dismount      func(name string)
Unmount       func(name string)
GetWeaponInfo func(chName string) (wType, damDice, damSize int, isBlessed bool)

// Room navigation
GetAdjacentRoom func(roomVNum, door int) int

// Kill/Death/Stats
GainExp         func(name string, amount int)
GetExp          func(name string) int
GetKills        func(name string) int64
SetKills        func(name string, kills int64)
GetDeaths       func(name string) int64
SetDeaths       func(name string, deaths int64)
SetLastDeath    func(name string, t int64)
GetPks          func(name string) int64
SetPks          func(name string, pks int64)
GetConstitution func(name string) int
SetConstitution func(name string, val int)

// Corpse & Extraction
MakeCorpse    func(victim string, attackType int)
MakeDust      func(victim string, attackType int)
ExtractChar   func(name string)
RunDeathScript func(killer, victim string, roomVNum int)
```

**Wire from game layer:** Create a `WireCombatCallbacks()` function in
`pkg/game/` or `cmd/server/main.go` that sets up all callbacks using game-layer
functions. Most are simple lookups:

```go
cb.GetRace = func(name string) int {
    if p := world.GetPlayerByName(name); p != nil { return p.GetRace() }
    if m := world.GetMobByName(name); m != nil { return m.GetRace() }
    return 0
}
```

**Replace `NowUnix`:** This is a clock, not a callback. Replace
`combat.NowUnix()` calls with `time.Now().Unix()` directly.

**Files:**
- `pkg/combat/callbacks.go` — extend struct
- `pkg/combat/fight_core.go` — update all call sites
- `pkg/combat/formulas.go` — update GetWeaponInfo call sites
- `pkg/combat/skill_messages.go` — update GetCharacterSex call sites
- `cmd/server/main.go` or new `pkg/game/combat_wire.go` — wire all callbacks

---

### PR 3 — Group, Gold, Items, Commands, World Hooks

**Goal:** Migrate remaining hooks, remove all package-level vars, remove
temporary `callbacks` package-level var.

**Add to GameCallbacks:**

```go
// Group/Party
GetFollowersInRoom       func(name string, roomVNum int) int
GetMasterInRoom          func(name string, roomVNum int) bool
GetFellowFollowersInRoom func(name string, roomVNum int) bool
CountGroupMembers        func(leaderName string, roomVNum int) int
ApplyToGroupMembers      func(leaderName string, roomVNum int, fn func(name string))

// Gold
GetGold func(name string) int
SetGold func(name string, gold int)

// Items
JunkInventoryItems func(chName string)

// Commands
PerformCommand func(chName, cmd string)

// Flee/Retreat
DoFlee    func(name string)
DoRetreat func(name string)

// World
IncreaseMaxStat func(name string, stat string)
HealAllPlayers  func()
```

**After this PR:**
- Zero package-level function hooks remain in fight_core.go
- All hooks are on `GameCallbacks` struct, validated at construction
- `combat.NowUnix` replaced with `time.Now().Unix()`
- The temporary `var callbacks *GameCallbacks` is removed
- CombatEngine validates all required callbacks are non-nil at construction
- `fight_core` functions receive `*GameCallbacks` as parameter or access via
  engine's Callbacks field

**Files:**
- `pkg/combat/callbacks.go` — final struct
- `pkg/combat/fight_core.go` — final cleanup, remove all package-level vars
- `pkg/combat/engine.go` — validate all callbacks at construction
- `cmd/server/main.go` — wire remaining callbacks

---

## Key Rules

### Do NOT modify function behavior

The migration is **structural only**. Every function must produce exactly the
same results before and after. The only change is HOW hooks are accessed
(package var → struct field), not WHAT they do.

### Do NOT gut functions

If a hook is not yet migrated in the current PR, it stays as a package-level var.
Do NOT remove the call site. Do NOT make the function a no-op. The 3-PR split
means some hooks migrate in PR 1, some in PR 2, some in PR 3. All call sites
must compile and work at every PR boundary.

### Nil-guard during transition

While hooks are migrating, use the pattern:
```go
if cb.GetRace != nil {
    race = cb.GetRace(name)
}
```

After PR 3, all required callbacks are validated at construction and guaranteed
non-nil, so nil guards can be removed in a follow-up cleanup.

### Keep all existing tests green

The existing test suite sets package-level vars directly. During PR 1-2, tests
continue to work because we keep the package-level vars alongside the struct.
In PR 3, tests are updated to set `GameCallbacks` instead.

## Build Gate

```bash
go build ./... && go vet ./... && go test ./... && gofumpt -l . | grep -v vendor
```

All four must pass before committing. CI adds `go test -race`.

## Commits

```
PR 1: refactor: introduce GameCallbacks struct, migrate messaging hooks, delete dead code (DP-952)
PR 2: refactor: migrate character state hooks to GameCallbacks (DP-952)
PR 3: refactor: complete GameCallbacks migration, remove all package-level hooks (DP-952)
```
