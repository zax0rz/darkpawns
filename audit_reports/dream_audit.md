# Port Fidelity Audit: Module 23 (`dream.c`)

This audit examines the port fidelity between the legacy C source file `src/dream.c` (dreaming engine and dream travel) and its Go implementations in `pkg/game/dreams.go` and `pkg/session/tattoo.go`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/dream.c` (224 lines)
- **Core Functions**:
  - `dream(struct char_data *ch)` (manages sleeping character dream ticks, death nightmares, and level-based flavor dreams)
  - `dream_travel(struct char_data *ch, int subcmd)` (processes dream teleportation destination selection, prints messages, and relocates the player)
  - `dtravel[]` (table of dream travel destinations)
- **File**: `src/tattoo.c` (187 lines)
  - `use_tattoo(struct char_data *ch)` (triggers special active tattoo abilities)
  - `tattoo_af(struct char_data *ch, bool add)` (applies/removes stat-modifying passive tattoo affects)

### Go Port Files
- `pkg/game/dreams.go` (main dreaming engine, containing `Dream()`, `DreamTravelFn()`, and the `DreamContext` interface)
- `pkg/session/tattoo.go` (tattoo activation, passive affect modification switch-statements, and command structures)
- `pkg/game/deferred_fight_fns.go` (defines tattoo constants and `GetTattooBonuses` helper functions)
- `pkg/game/limits_condition.go` (decrements the tattoo cooldown timer during the hourly `PointUpdate` tick)

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. Tattoo Serialization/Persistence Omission (Permanent Loss on Logout)
- **Source Context**: `pkg/db/player.go`, `pkg/db/convert.go`, `pkg/game/player.go#L196-L198`
- **Fidelity Bug**: In the legacy C codebase, `GET_TATTOO(ch)` and `TAT_TIMER(ch)` are macros mapping directly to `ch->player_specials->saved.tattoo` and `ch->player_specials->saved.tattimer`. Because they reside in the `saved` sub-structure, both values are automatically serialized to the character's disk file and fully persist across logins and reboots.
  In Go, `Tattoo` and `TatTimer` are defined on the memory `Player` struct but are **completely omitted from the SQLite schema, `PlayerRecord` struct, and mapping converters** in `pkg/db/`.
- **Impact**: Any character who obtains a tattoo or starts a tattoo cooldown will permanently lose their tattoo identity and timer resets as soon as they log out or the server restarts.

### 2. Broken Teleport Affect Removal (Stranded Sleeping Body)
- **Source Context**: `pkg/game/dreams.go#L217-L242` (`DreamTravelFn`)
- **Fidelity Bug**: In `DreamTravelFn`, when a dream destination is selected, the code removes the dream affect flag immediately and issues the teleportation command:
  ```go
  ch.SendToChar(fmt.Sprintf("You have a dream %s \r\n", dt.Descrip))
  ch.SendToRoom("The sleeping body of $n fades from existence.")
  ch.MoveToRoom(dt.RoomNum)
  ch.SendToRoom("The sleeping body of $n fades into existence.")
  ch.RemoveAffect(AFF_DREAM_BIT)
  ```
  If `ch.MoveToRoom(dt.RoomNum)` fails (e.g. because the target room VNum does not exist in the active world data, causing `PlayerTransfer` to return an error), the Go port safely prevents a crash and keeps the player in their current room. However, it still outputs the visual side-effects and removes `AFF_DREAM_BIT` anyway!
- **Impact**: A player who fails to teleport due to a missing or unloaded room will have their `AFF_DREAM` bit permanently stripped and stay stranded in their original room, receiving confusing visual cues ("$n fades from existence... $n fades into existence" in the exact same room) and losing the ability to try the teleport again.

### 3. Hardcoded Switch Drift (Stat Application Maintenance Pain)
- **Source Context**: `pkg/session/tattoo.go#L110-L185` (`tattooAf`)
- **Fidelity Bug**: In legacy C, `tattoo_af` was powered by a generic table structure using an array of `affected_type` records loaded with parameters, matching the structural pattern of normal spells. 
  In the Go port, `tattooAf` constructs a local table via nested switch-statements. In addition, `applyModifier` hardcodes stat index modifications (e.g. `case 0: p.Stats.Str += modifier`).
- **Impact**: Modifying or introducing new tattoos requires altering complex switch blocks and direct pointer-state offsets rather than simply updating a central structural array.

### 4. Exceptional Level Dream Scaling Drift
- **Source Context**: `pkg/game/dreams.go#L147-L207` (`Dream`)
- **Fidelity Bug**: The switch checking `GET_LEVEL(ch)` in Go uses CircleMUD groupings up to 30. However, Dark Pawns contains custom exceptional immortal level ranges scaling above level 31. The default case in `Dream()` assumes anyone above `LVLImmort` (31) is an immortal and prints the Orodreth fear text and wakes them up.
- **Impact**: High-level characters or custom immortal ranges may experience incorrect dream ticks or premature wake-ups.

---

## 3. Go Improvements Over C

### 1. Interface Decoupling & Testability
- **Fidelity Improvement**: In legacy C, the `dream()` function relied heavily on direct global variable access and pointer manipulation on `char_data` structs. Go encapsulates all of this behind the `DreamContext` interface:
  ```go
  type DreamContext interface {
      GetLevel() int
      GetLastDeath() int64
      SetLastDeath(t int64)
      HasAffect(bitNum int) bool
      RemoveAffect(bitNum int)
      SendToChar(msg string)
      SendToRoom(msg string)
      WakeUp()
      MoveToRoom(roomVNum int)
      CurrentTime() int64
  }
  ```
  This allows testing sleeping states, nightmares, and level-based flavor text with mock characters, bypassing full SQLite or World setup.

### 2. Time Overflow Protection
- **Fidelity Improvement**: Legacy C stored `lastdeath` in a 32-bit `long` representation. Go uses `int64` for all Unix timestamps, safeguarding the game from the Year 2038 problem.

---

## 4. Summary of Recommended Fixes

1. **Add Tattoo Fields to SQLite & db/convert.go**:
   Add `tattoo` and `tattoo_timer` columns to the `players` database table:
   ```sql
   ALTER TABLE players ADD COLUMN IF NOT EXISTS tattoo INTEGER DEFAULT 0;
   ALTER TABLE players ADD COLUMN IF NOT EXISTS tattoo_timer INTEGER DEFAULT 0;
   ```
   Map these fields in the `PlayerRecord` struct in `pkg/db/player.go`, and add them to `PlayerToRecord` and `RecordToPlayer` conversions in `pkg/db/convert.go`.

2. **Fix Teleport Rollback in DreamTravelFn**:
   Modify `DreamTravelFn` (or the underlying `MoveToRoom` interface implementation) to return an error, and wrap the message outputs and affect removal in a success check:
   ```go
   if err := ch.MoveToRoom(dt.RoomNum); err == nil {
       ch.SendToChar(fmt.Sprintf("You have a dream %s \r\n", dt.Descrip))
       // Only remove affect and output messages on successful transfer!
       ch.RemoveAffect(AFF_DREAM_BIT)
   }
   ```
