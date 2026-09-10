# Port Fidelity Audit: Module 37 (`mobprog.c`)

This audit examines the port fidelity between the legacy C source file `src/mobprog.c` and its Go counterparts in `pkg/game/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/mobprog.c` (647 lines)
- **Functions & Features**:
  - **In-Game Mob Programs**: Implements hardcoded, highly specific behaviors for various NPCs including:
    - `mp_greet` and `mp_ride_greet`: Triggers when players enter a room. Managed actions for dogs (licking/growling based on alignment), shopkeepers muttering about dogs, and tavern tenders welcoming guests.
    - `mp_give` and `mp_bribe`: Triggers when mobs receive items or gold. Managed mercenaries swearing allegiance (getting charmed), dogs devouring food or playing with items, soul eaters (`IS_DEMON`) trading soul stones (`9900`) for shimmering portals (`19611`), and jail guards (`8088`) accepting bribes to free players.
    - `entry_prog`: Custom events when mobs enter rooms (e.g. Captain Aversin `8059` waking up sleeping city guards).
    - `mp_sound`: Hardcoded ambient echoes for beggars, minstrels, carpenters, zealots, and dogs (relieving themselves to spawn a puddle `20` with a decay timer).
    - **Helpers**: Group defense, cityguard protection, and rescue utilities (`kill_bad_guy`, `npc_rescue`).

### Go Port Files
- **Dormant Framework**:
  - [pkg/game/mobprogs.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/mobprogs.go): Contains ports of `MpGreet`, `MpRideGreet`, `MpGive`, `MpBribe`, `EntryProg`, `NpcRescue`, and `MpSound`.
- **Combat & Shop Helpers**:
  - [pkg/game/combat_helpers.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/combat_helpers.go): Implements `IsMobShopkeeper` (checking shop spec assignments).

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. VNum Mix-up Bug in Jailer Cell Bribe (Major Gameplay Bug)
- **Source Context**: `pkg/game/mobprogs.go#L97` (`MpGive`), `src/mobprog.c#L197` (`mp_bribe`).
- **Fidelity Bug**: In legacy C, the cell bribe logic that frees the player and their mount and moves them to real room `8117` is registered to the Jailer's VNum **`8088`**:
  ```c
  } else if (GET_MOB_VNUM(mob)==8088) {
      ...
      act("$N throws you out of the cell!", TRUE, ch, 0, mob, TO_CHAR);
      char_from_room(ch);
      char_to_room(ch, real_room(8117));
  ```
  However, the Go port checks VNum **`8014`** (which is actually the **Guild Guard**) instead of `8088`:
  ```go
  case vnum == 8014:
      w.roomMessage(mob.GetRoom(), "$n says, 'Now get outta here!'")
      ch.SendMessage("$N throws you out of the cell!\r\n")
      ...
      w.MovePlayerToRoom(ch, 8117)
  ```
  - **Impact**: Bribing the Guild Guard (`8014`) will trigger cell-throwing actions and move players to room `8117` (highly incorrect behavior). Conversely, bribing the actual Jailer (`8088`) will do nothing because the case is missing in Go.

### 2. Entire Hardcoded MobProg system is Dormant (Dead Code)
- **Source Context**: `pkg/game/mobprogs.go` (marked as `//nolint:unused` and `//lint:file-ignore U1000`).
- **Fidelity Bug**: All ported functions in `pkg/game/mobprogs.go` (including `MpGreet`, `MpRideGreet`, `MpGive`, `MpBribe`, `EntryProg`, `NpcRescue`, and `MpSound`) are **completely uncalled and un-integrated** into the Go server's event, movement, item transfer, or coin giving execution pathways. None of these legacy hardcoded behaviors (e.g. mercenaries charmed by gold, prostitutes pulling players into shadows, dogs devouring food) are active in-game.

### 3. Hardcoded Greet & Give Triggers Omitted
- **Source Context**: `pkg/game/mobprogs.go#L37` (`MpGreet`), `pkg/game/mobprogs.go#L90` (`MpGive`).
- **Fidelity Bug**: 
  - **Greets**: Go's `MpGreet` only handles the "Inner Circle" gatekeeper logic on `8014`. The legacy C checks for Dog reactions, mutant shopkeepers complaining about dogs, and fireplace tenders welcoming players are entirely omitted.
  - **Gives**: Go's `MpGive` only handles gold bribes. The legacy C item-giving events (giving food to dogs, giving soul stones to demons, giving junk to janitors) are completely unported.

### 4. Missing Shopkeeper VNum Overrides
- **Source Context**: `pkg/game/combat_helpers.go#L73` (`IsMobShopkeeper`), `src/mobprog.c#L495-L509` (`is_shopkeeper`).
- **Fidelity Bug**: In legacy C, the shopkeeper check includes explicit VNum overrides (mobs with VNums `8003-8011`, `8078` are automatically classified as shopkeepers even if they do not have a spec procedure assigned). Go's `IsMobShopkeeper` only checks spec assignments, omitting these hardcoded VNum fallbacks.

### 5. Dog Puddle Creation Dormancy
- **Source Context**: `pkg/game/mobprogs.go#L395` (`MpSound`).
- **Fidelity Bug**: The Go port of `MpSound` includes the 1-in-26 chance for dogs to relieve themselves and spawn a puddle object (`VNum 20` with timer = 2). However, because `MpSound` is never called, dogs will never spawn puddles.

---

## 3. Go Improvements Over C

### 1. Decoupling the Custom Lua Scripting Engine
- **Go Enhancement**: Rather than hardcoding quest mob logic into massive C switch-statements, the Go server utilizes a modern Lua-based scripting engine (`pkg/scripting/`). While the legacy hardcoded behaviors inside `mobprog.c` were ported to `mobprogs.go` as a fallback, Go’s modern architecture cleanly abstracts complex quest behaviors into modular, hot-reloadable Lua script files.

### 2. Structured Gold Deduction and Mount Transfers
- **Go Enhancement**: Where C relied on manual room shifts for mounts and players:
  ```c
  char_from_room(ch);
  char_to_room(ch, real_room(8117));
  if (get_mount(ch)) { ... }
  ```
  Go wraps these in safe structural mutations (`w.MovePlayerToRoom` and `w.GetMount`) which automatically maintain integrity of the player's session and mounted variables.

---

## 4. Concurrency & Thread Safety

- **Local Object Spawning**:
  - The object spawner in `CreateObject` maps safety-clamped values and locks the room slice locally during insertion, preventing race conditions with other game threads attempting to manipulate the room inventory contents.
- **RWMutex Isolation**:
  - Player values (such as alignment or gold) are fetched via safe thread-safe getters (`who.GetLevel()`, `who.ClanID`), ensuring concurrent MUD combat threads do not corrupt values during tick notifications.

---

## 5. Summary of Recommended Fixes

1. **Fix the Jailer Cell VNum**:
   Correct the switch case in `pkg/game/mobprogs.go#L97` to check for VNum `8088` (Jailer) instead of `8014` (Guild Guard).
2. **Wire MobProgs into World Events**:
   Wire `MpGreet` into the player room-entry dispatcher, `MpGive`/`MpBribe` into the player coin/item giving endpoints, and `MpSound` into the mob activity pulse tick.
3. **Restore Item Giving Events**:
   Port the missing `mp_give` item evaluations (dogs devouring food, demons taking soul stones to spawn portals) to Go.
