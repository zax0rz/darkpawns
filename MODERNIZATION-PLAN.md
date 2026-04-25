# Dark Pawns Go Modernization Plan

Dark Pawns builds and vets clean, so this should be treated as a behavior-preserving modernization pass, not a rewrite. The goal is to remove the sharpest C-port artifacts while leaving game logic, spec procs, socials, and TODO feature work alone.

## Executive priority

1. Replace object ownership `interface{}` with typed owner/location abstractions.
2. Stop swallowing operational errors, especially inventory/equipment transfer failures.
3. Replace ad hoc `fmt.Printf` with structured `slog`.
4. Split command files by command family once tests/ownership are safer.
5. Replace `CustomData map[string]interface{}` with typed runtime state where it matters.
6. Later: chip away at magic numbers/status strings/bit macros.

The only change that probably wants a focused branch is object ownership. Everything else can be incremental.

---

## Current patterns observed

### 1. Object ownership uses `interface{}` and is already inconsistent

References:

- `pkg/game/object.go:19` — `Carrier interface{} // *MobInstance or *Player`
- `pkg/game/object.go:23` — `EquippedOn interface{} // *MobInstance or *Player`
- `pkg/game/object.go:241-246` — `GetCarrier() interface{}` / `SetCarrier(carrier interface{})`
- `pkg/game/inventory.go:29` — `item.Carrier = inv`, despite the comment saying carrier is `*MobInstance or *Player`
- `pkg/game/equipment.go:173` and `pkg/game/equipment.go:195` — `item.EquippedOn = eq`, despite the comment saying equipped-on is `*MobInstance or *Player`
- `pkg/game/world.go:407-413` — type-switch only handles `*Player` and `*MobInstance`, so objects carried by `*Inventory` or equipped on `*Equipment` can be missed
- `pkg/game/mob.go:201`, `pkg/game/mob.go:226` — mob code sets `Carrier` / `EquippedOn` to the mob itself
- `pkg/game/systems/shop.go:197`, `pkg/game/systems/shop_manager.go:181` — shop code also sets carrier-like ownership through common object interfaces

This is the worst modernization target because it is both non-idiomatic and potentially wrong today. The field comments and actual assignments disagree.

### 2. `CustomData map[string]interface{}` is an untyped state leak

References:

- `pkg/game/object.go:30`, `pkg/game/object.go:203-216` — object custom state bag
- `pkg/game/mob.go:50-51` — mob custom state bag
- `pkg/game/save.go:63`, `pkg/game/save.go:203`, `pkg/game/save.go:219` — serialized save state is also `map[string]interface{}`
- `pkg/game/mail.go:489` — `obj.CustomData["mail_text"]`
- `pkg/game/skills.go:832`, `pkg/game/skills.go:1050`, `pkg/game/skills.go:1144-1150` — carved meat, molded objects, severed-head/corpse descriptions
- `pkg/game/spec_procs2.go:323-326`, `pkg/game/spec_procs2.go:390-393` — horse carry/move data
- `pkg/game/spec_procs3.go:454-459` — brain eater damroll bonus type assertion

This is not all bad: some scripted/game-specific state really is dynamic. The problem is that save data, object overrides, mob combat state, and one-off script scratch space all share the same untyped bucket.

### 3. Errors are ignored where state changes can fail

References:

- `pkg/game/world.go:633`, `pkg/game/world.go:644` — starting item inventory errors ignored
- `pkg/game/skills.go:779`, `pkg/game/skills.go:1472` — inventory adds ignored
- `pkg/game/spec_procs2.go:550`, `579`, `606`, `744`, `1091` — spec proc inventory transfer errors ignored
- `pkg/game/act_item.go:745`, `762`, `1002`, `1703` — inventory transfer errors ignored in player-facing item commands
- `pkg/game/world.go:1002-1004` — script error explicitly discarded after combat fight trigger
- `pkg/game/player.go:523-528` — full send channel silently drops messages; acceptable for backpressure, but should at least be a deliberate helper/policy

The highest-risk cases are player-visible transfers: if inventory is full, current code can claim success or lose symmetry between room/container/player state.

### 4. Logging is split between `fmt.Printf` and `slog`

References:

- `pkg/game/serialize.go:55`, `92`, `102`
- `pkg/game/spawner.go:70`, `77`, `85`, `91`, `98`, `104`, `114`, `120`, `136`, `142`
- `pkg/game/world.go:612`, `990`, `1047`, `1053` already use `slog`, so the project has a clear destination

Spawner/deserialize warnings are exactly where structured logs help: vnum, room, zone command, max count, error.

### 5. File organization is still C-shaped

References:

- `pkg/game/act_item.go` — 2021 lines, constants, helpers, command handlers, object transfer, equip, drink/eat, containers
- `pkg/game/act_other.go` — 1718 lines, mixed player commands
- `pkg/game/act_offensive.go` — 1545 lines, offensive command surface
- `pkg/game/spec_procs2.go`, `pkg/game/spec_procs3.go` — large by nature; do not split individual proc logic unless extracting shared helpers
- `pkg/game/limits.go` — ported constants and regen/condition logic in one file

Splitting files is useful, but only after ownership/errors are safer. Otherwise it is just moving the haunted furniture.

### 6. Other C-isms worth cleaning later

- Magic item type constants in `pkg/game/act_item.go:8-45` use ALL_CAPS and plain `int`. Prefer named Go types such as `type ItemType int` plus `const ItemWeapon ItemType = 5`.
- Position/sex/class/race/status values are raw ints/strings in `pkg/game/player.go`, `pkg/game/mob.go`, `pkg/game/limits.go`. Several are faithful ports; convert only at package boundaries where it reduces bugs.
- `MobInstance.Status string` in `pkg/game/mob.go:23` duplicates combat position constants and requires conversion in `pkg/game/mob.go:285-302`.
- Many methods are direct macro ports (`GetStr`, `GetDex`, `IsAffected`, `SetPlrFlag`). That is acceptable for compatibility; modernize names/types gradually, not all at once.

---

## Phase 1 — Typed object ownership and location

**Goal:** Make it impossible for an object to have an unknown carrier/equipment owner at compile time.

**Files/functions affected:**

- `object.go`: `ObjectInstance`, `GetCarrier`, `SetCarrier`
- `inventory.go`: `Inventory.AddItem`, `RemoveItem`, `RemoveItemByVNum`, `Clear`
- `equipment.go`: `Equipment.Equip`, `Unequip`, `UnequipItem`
- `mob.go`: `MobInstance.AddToInventory`, `RemoveFromInventory`, `EquipItem`, `UnequipItem`
- `world.go`: `ExtractObject`, item movement helpers, scriptable item transfers
- `systems/shop.go`, `systems/shop_manager.go`: shop ownership adapter points

**Recommended pattern:** use a small tagged owner/location type, not a huge interface. Go has no sum types, but this domain wants explicit state more than polymorphism.

Before:

```go
type ObjectInstance struct {
    RoomVNum   int
    Carrier    interface{}
    Container  *ObjectInstance
    EquippedOn interface{}
}

if p, ok := obj.Carrier.(*Player); ok {
    p.Inventory.RemoveItem(obj)
} else if m, ok := obj.Carrier.(*MobInstance); ok {
    m.RemoveFromInventory(obj)
}
```

After:

```go
type ObjectLocationKind uint8

const (
    ObjNowhere ObjectLocationKind = iota
    ObjInRoom
    ObjInInventory
    ObjEquipped
    ObjInContainer
    ObjInShop
)

type ObjectLocation struct {
    Kind       ObjectLocationKind
    RoomVNum   int
    PlayerName string // stable key for player inventory/equipment
    MobID      int    // stable key for mob inventory/equipment
    ShopID     int
    ContainerID int
    Slot       EquipmentSlot
}

type ObjectInstance struct {
    ID        int
    VNum      int
    Prototype *parser.Obj
    Location  ObjectLocation
    Contains  []*ObjectInstance
    Runtime   ObjectRuntimeState
}
```

Then centralize movement:

```go
func (w *World) MoveObject(obj *ObjectInstance, dst ObjectLocation) error {
    if err := w.detachObject(obj); err != nil {
        return err
    }
    if err := w.attachObject(obj, dst); err != nil {
        return err
    }
    obj.Location = dst
    return nil
}
```

**Why not `interface{ AddItem/RemoveItem }`?** It is tempting, but it keeps the current ambiguity. Inventory, equipment, mob, player, shop, room, and container are not all the same kind of owner. The game often needs to know *where* the object is, not just who can receive it. A tagged location gives clear invariants and serializes cleanly.

**Risk:** High. This touches core item movement. Behavior should not change, but broken invariants may surface.

**Incremental or big-bang:** Focused big-bang inside object movement. Use compatibility methods during transition:

```go
func (o *ObjectInstance) InInventoryOfPlayer(name string) bool {
    return o.Location.Kind == ObjInInventory && o.Location.PlayerName == name
}
```

**Estimated effort:** 2-4 days including tests.

**Tests to add first:**

- ground -> player inventory -> ground
- inventory -> equipment -> inventory
- mob inventory/equipment extraction
- container nesting and extraction
- shop buy/sell round-trip
- save/load with location state if persisted

---

## Phase 2 — Error handling policy for game commands

**Goal:** State-changing helpers return errors; player commands translate them into MUD messages; background systems log them.

**Files/functions affected:**

- `act_item.go`: inventory adds at `745`, `762`, `1002`, `1703`; transfer/equip/drop paths
- `world.go`: `GiveStartingItems`, `giveItem`, `FireMobFightScript`
- `skills.go`: butcher/carve/mold/steal-like item operations
- `spec_procs2.go`: inventory grants and transfers
- `player.go`: send-drop policy should be wrapped

**Recommended pattern:** typed sentinel errors plus wrapping.

Before:

```go
ch.Inventory.AddItem(obj)
ch.SendMessage("You get it.\n")
```

After:

```go
if err := ch.Inventory.AddItem(obj); err != nil {
    ch.SendMessage("You can't carry that.\n")
    return fmt.Errorf("give %d to %s: %w", obj.VNum, ch.Name, err)
}
ch.SendMessage("You get it.\n")
```

For expected gameplay failures, do not spam error logs from command handlers. Return a `CommandResult` or send the player message and return nil. Log only inconsistent state and subsystem failures.

Suggested error vocabulary:

```go
var (
    ErrInventoryFull = errors.New("inventory full")
    ErrObjectNotFound = errors.New("object not found")
    ErrInvalidObjectLocation = errors.New("invalid object location")
    ErrEquipSlotOccupied = errors.New("equipment slot occupied")
)
```

**Risk:** Medium. This can reveal existing bugs and change player-visible messages. Keep messages compatible where possible.

**Incremental or big-bang:** Incremental. Start with `Inventory.AddItem` call sites and `Equipment.Equip/Unequip`.

**Estimated effort:** 1-2 days for worst offenders; ongoing cleanup as files are touched.

**Skip for now:** nil returns that mean “not found” in lookup helpers. Those are ordinary Go if documented and paired with `ok` where ambiguity matters.

---

## Phase 3 — Structured logging cleanup

**Goal:** All game runtime diagnostics go through `log/slog` with fields.

**Files/functions affected:**

- `serialize.go:55`, `92`, `102`
- `spawner.go:70`, `77`, `85`, `91`, `98`, `104`, `114`, `120`, `136`, `142`
- optionally `mob.go:142`, `world.go:308` direct channel sends should go through `SendMessage`, not logging

Before:

```go
fmt.Printf("Error spawning mob %d: %v\n", cmd.Arg1, err)
```

After:

```go
slog.Error("spawn mob failed", "mob_vnum", cmd.Arg1, "room_vnum", cmd.Arg3, "error", err)
```

**Risk:** Low. No behavior change unless tests assert stdout.

**Incremental or big-bang:** Incremental or one small PR.

**Estimated effort:** 1-2 hours.

---

## Phase 4 — Typed runtime state instead of one `CustomData` bag

**Goal:** Keep genuinely dynamic script state possible, but give common object/mob state real types.

**Files/functions affected:**

- `object.go`: replace or narrow `CustomData`
- `mob.go`: replace or narrow `CustomData`
- `save.go`: save/load typed state
- `mail.go`, `skills.go`, `spec_procs2.go`, `spec_procs3.go`: migrate known keys

**Recommended pattern:** a typed runtime struct plus a quarantined extension map.

```go
type ObjectRuntimeState struct {
    MailText string `json:"mail_text,omitempty"`

    ShortDescOverride string `json:"short_desc_override,omitempty"`
    ShortDesc         string `json:"short_desc,omitempty"`
    Name              string `json:"name,omitempty"`
    MoldName          string `json:"mold_name,omitempty"`
    MoldDesc          string `json:"mold_desc,omitempty"`

    Horse *HorseState `json:"horse,omitempty"`

    // Last resort for scripts/import compatibility. New Go code should not use it.
    Script map[string]any `json:"script,omitempty"`
}

type HorseState struct {
    CarryWeight int `json:"carry_weight"`
    CarryNumber int `json:"carry_number"`
    Move        int `json:"move"`
    MaxMove     int `json:"max_move"`
}

type MobRuntimeState struct {
    DamrollBonus int `json:"damroll_bonus,omitempty"`
    Script map[string]any `json:"script,omitempty"`
}
```

Before:

```go
if v, ok := me.CustomData["damroll_bonus"]; ok {
    cur, _ = v.(int)
}
me.CustomData["damroll_bonus"] = cur + 2
```

After:

```go
me.Runtime.DamrollBonus += 2
```

**Risk:** Medium, mainly save compatibility. Provide migration from old `state` map during load.

**Incremental or big-bang:** Incremental by key. Keep `Script map[string]any` as a pressure valve.

**Estimated effort:** 1-3 days depending on save compatibility tests.

**Do not over-engineer:** Do not create one struct per object prototype. That becomes a second parser/database. Use typed structs for repeated semantic state and leave rare script scratch data in `Runtime.Script`.

---

## Phase 5 — Split large files by behavior, not by original C filenames

**Goal:** Make command code navigable without changing behavior.

**Files affected:**

- `act_item.go`
- `act_other.go`
- `act_offensive.go`
- `limits.go` optionally
- leave `spec_procs2.go` / `spec_procs3.go` mostly intact except helper extraction

**Recommended split for `act_item.go`:**

- `item_types.go` — item constants/types, container flags, wear/equip constants
- `item_find.go` — search/dot-mode helpers (`find all`, `all.foo`, inventory/room/equip lookup)
- `item_transfer.go` — get/drop/give/donate/junk and movement helpers
- `item_equipment.go` — wear/remove/wield/grab/equipment display helpers
- `item_container.go` — put/get/open/close/lock/unlock/pick container flows
- `item_consumable.go` — drink/eat/taste/fill/pour
- `act_render.go` or `messages.go` — `actToChar`, `actToRoom`, substitution helpers if shared beyond item commands

**Recommended split for offensive commands:**

- `combat_commands.go` — command entry points
- `combat_skills.go` — bash/kick/backstab/etc wrappers if not already in `skills.go`
- `combat_targeting.go` — target resolution and validation

**Recommended split for other commands:** by player-facing feature group: `act_player_state.go`, `act_social_admin.go` only if obvious. Do not split just to hit a line count.

**Risk:** Low-medium. Mostly mechanical, but merge conflicts are likely.

**Incremental or big-bang:** Incremental file moves after tests exist. Use `go test ./...` after each slice.

**Estimated effort:** 1 day for `act_item.go`; 1-2 more for the others.

---

## Phase 6 — Type aliases/enums for constants and status

**Goal:** Remove the “C macro in Go clothes” feel where it causes bugs.

**Files affected:**

- `act_item.go`, `limits.go`, `player.go`, `mob.go`, class/race/sex constants wherever defined

Recommended replacements:

```go
type ItemType int

const (
    ItemLight ItemType = 1
    ItemWeapon ItemType = 5
    ItemContainer ItemType = 15
)

func (o *ObjectInstance) Type() ItemType {
    return ItemType(o.GetTypeFlag())
}
```

```go
type Position int

const (
    PositionDead Position = combat.PosDead
    PositionStanding Position = combat.PosStanding
)
```

For mob status, prefer position plus explicit combat target over arbitrary strings:

```go
type MobInstance struct {
    Position Position
    FightingTarget string
}
```

**Risk:** Medium if done broadly; low if done around touched areas.

**Incremental or big-bang:** Incremental. Do not churn every constant in one PR.

**Estimated effort:** ongoing; 0.5-1 day per constant family.

---

## TODO Dependency Mapping

Phase 1 (`ObjectLocation`) is the highest-leverage phase because it unblocks object persistence (`houses.go`), makes extract/death safer (`world.go`, `death.go`), and gives every item operation a single code path to audit.

### `limits.go`

- **Line 90 — `playing_time` and kill count fields**
  - **Phase:** Phase 6 simplifies this by typing/organizing player status/stat fields; no hard dependency.
  - **Independent:** Yes.
  - **Needed:** Add explicit player runtime/persistent fields for playing time and kill counters, wire save/load if they should persist, and preserve current C-port semantics.
- **Line 420 — `PRF_INACTIVE` flag check**
  - **Phase:** Phase 6 simplifies this by replacing raw flag macros/ints with typed player preference flags.
  - **Independent:** Yes.
  - **Needed:** Define or map the inactive preference flag and use the existing player flag helpers consistently.
- **Line 433 — call dream when implemented**
  - **Phase:** No modernization phase; this is feature/system work.
  - **Independent:** Yes, once the dream system exists.
  - **Needed:** Add the dream subsystem/API, then call it from the idle/sleep path with compatible messaging and timing.
- **Line 697 — wizlist update**
  - **Phase:** Phase 5 may simplify placement if admin/player-state commands are split; no hard dependency.
  - **Independent:** Yes.
  - **Needed:** Decide the authoritative wizlist source, update it when immortality/admin level changes, and persist or regenerate it deterministically.
- **Lines 801, 836 — `AFF_FLESH_ALTER` handling**
  - **Phase:** Phase 6 simplifies this by typing affect flags/status constants.
  - **Independent:** Yes.
  - **Needed:** Port the missing affect flag semantics, including stat/body modifications and reversal behavior in regen/update paths.
- **Line 871 — full idle handling (`char_data.timer`, `was_in`, `desc` fields)**
  - **Phase:** Phase 6 simplifies typed player/session state; Phase 2 helps error/log policy for forced state transitions.
  - **Independent:** Partially. Basic timer fields can be added independently; full reconnect/room restore benefits from clearer session state.
  - **Needed:** Add idle timer/session fields, track previous room, handle link-dead transitions, and preserve descriptor/reconnection behavior.

### `houses.go`

- **Lines 64, 69 — player database lookup by ID/name**
  - **Phase:** No direct modernization phase; Phase 6 can make player identifiers less ad hoc.
  - **Independent:** Yes.
  - **Needed:** Add a player repository/index that resolves stable player IDs and canonical names without loading the whole live player object incorrectly.
- **Line 307 — `ObjFromStore` / full object loading**
  - **Phase:** Phase 1 enables this cleanly; Phase 4 helps if stored objects include typed runtime state.
  - **Independent:** No, not safely. Object persistence should wait for or be built alongside `ObjectLocation`.
  - **Needed:** Implement serialized object restoration with location/container metadata, prototype lookup, runtime state migration, and house/container attach logic.

### `boards.go`

- **Lines 335, 489 — `BoardSystem` has no `World` reference**
  - **Phase:** No direct modernization phase; Phase 2 helps define subsystem error policy.
  - **Independent:** Yes.
  - **Needed:** Inject the narrow dependency the board system actually needs, preferably a player/message lookup interface rather than a whole `World` pointer unless the whole world is genuinely required.

### `world.go`

- **Line 708 — death handling**
  - **Phase:** Phase 1 makes extract/object movement during death safer; Phase 2 helps command/system error handling around corpse creation and inventory drops.
  - **Independent:** Partially. Minimal death behavior can be ported now, but corpse/equipment/item movement should wait for `ObjectLocation` or be revisited after it.
  - **Needed:** Centralize death flow: corpse creation, inventory/equipment transfer, room messaging, extraction, XP/penalties, and player-vs-mob differences.
- **Line 720 — find actual killer/caster**
  - **Phase:** No direct modernization phase; Phase 6 may help typed combat/skill identifiers.
  - **Independent:** Yes.
  - **Needed:** Track damage attribution through combat/spell effects so delayed or indirect damage can resolve the real attacker.
- **Line 1042 — resolve target player by ID**
  - **Phase:** No direct modernization phase; Phase 6 can clean up typed player identifiers.
  - **Independent:** Yes.
  - **Needed:** Add a stable player ID lookup path against live players and/or persistent player records, with clear not-found behavior.

### `death.go`

- **Line 260 — player reconnection/resurrection**
  - **Phase:** Phase 1 helps recover inventory/equipment/corpse state safely; Phase 6 helps session/player state typing.
  - **Independent:** Partially. Reconnection scaffolding can be started now, but full resurrection should account for object location and death state.
  - **Needed:** Restore player state after death/link loss, reattach descriptors, return to correct room, and reconcile corpse/inventory handling.
- **Line 317 — `TYPE_` and `SKILL_` constants**
  - **Phase:** Phase 6 directly enables this.
  - **Independent:** Yes.
  - **Needed:** Replace raw C constants with typed damage/skill identifiers, then map existing combat messages and death causes through those types.

### `graph.go`

- **Line 179 — weather system penalty**
  - **Phase:** No direct modernization phase.
  - **Independent:** Yes.
  - **Needed:** Expose current weather/terrain modifiers to graph/path cost calculation and verify movement penalty parity with the original game.

### `zone_dispatcher.go`

- **Line 126 — per-zone mob AI ticks**
  - **Phase:** No direct modernization phase; Phase 3 helps diagnostics if AI ticks fail, and Phase 5 may make behavior code easier to navigate later.
  - **Independent:** Yes.
  - **Needed:** Schedule or dispatch mob AI updates per zone, respect zone activity/loading boundaries, and avoid double-ticking mobs already handled elsewhere.

### Already cleared / remove from planning

- **`clans.go` line 1152 — `do_clan_private`**: already ported; do not carry this as remaining work.
- **`spawner.go`**: TODOs were already cleared this session.
- **`spec_procs3.go`**: TODOs were already cleared this session.

---

## What to skip for now

- Nil-check cleanup. The code has many nil checks, but this is normal for live world state and script adapters.
- Spec proc individual logic. It is intentionally weird game content. Extract shared helpers only.
- Social system. Leave it alone if it works.
- Remaining TODOs. Those are feature work, not modernization.
- Full generic/entity-component rewrite. Tempting, wrong, and guaranteed to wake the ancient C ghosts.

---

## Suggested branch/PR sequence

1. **PR 1: Logging cleanup** — replace `fmt.Printf` in `serialize.go` and `spawner.go` with `slog`. Low risk, fast confidence win.
2. **PR 2: Inventory/equipment error audit** — fix ignored `AddItem`/`Equip` results in `act_item.go`, `world.go`, `skills.go`, `spec_procs2.go`; add command-level player messages.
3. **PR 3: Object movement tests** — add tests documenting current item movement/extraction behavior before changing ownership.
4. **PR 4: ObjectLocation refactor** — replace `Carrier`/`EquippedOn` with `ObjectLocation` and centralized `World.MoveObject` helpers.
5. **PR 5: CustomData typed migration** — introduce `ObjectRuntimeState` / `MobRuntimeState`; migrate known keys; preserve `Script map[string]any` for compatibility.
6. **PR 6+: File splits** — split `act_item.go` first, then `act_offensive.go`/`act_other.go` if still painful.

## Validation gates

For every PR:

```bash
go test ./...
go vet ./...
```

For ownership/runtime-state PRs, also add targeted tests for object transfer and save/load. The build being green is not enough here; this is the part where silently wrong object state eats your inventory and then smiles about it.
