# Port Fidelity Audit: Module 39 (`new_cmds.c`)

This audit examines the port fidelity between the legacy C source file `src/new_cmds.c` and its Go counterparts in `pkg/game/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/new_cmds.c` (2,793 lines)
- **Functions & Features**:
  - **Custom Object Transformations**:
    - `do_mold`: Reshapes moldable items (halos, clay, playdough) into custom objects with custom keywords, short, and long descriptions.
    - `do_carve`: Carves edible meat items (such as `8015`, `12`, `13`, `14`) from room corpses using a wielded slashing or piercing weapon.
    - `do_behead`: Decapitates room corpses to spawn a severed head (`16`) and convert the corpse into a headless corpse (`17`), retaining contents.
  - **Special Martial Arts & Combat Skills**:
    - `do_headbutt`: Standard martial hit dealing headbutt damage with a 25% chance of self-stunning on failure.
    - `do_bearhug`: Bare-handed martial grab dealing level-scaled squeeze damage.
    - `do_cutthroat`: Sneaky thief combat skill slitting the throat of a non-fighting target using a dagger, applying `AFF_CUTTHROAT` (silence + stat penalties) and dealing minor `level/2` damage.
    - `do_trip`: Thief trip skill causing a standing, non-flying target to fall to the ground.
    - `do_otouch`: Immortal fun command ("orgasmic touch").

### Go Port Files
- **Combat & Stealth Skills**:
  - [pkg/game/skill_advanced.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/skill_advanced.go): Implements `DoCarve` (carving) and `DoCutthroat` (throat slitting).
  - [pkg/game/skill_special.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/skill_special.go): Implements `DoBehead` (beheading) and `DoBearhug` (bearhug).
  - [pkg/game/skill_combat.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/skill_combat.go): Implements `DoTrip` (tripping) and `DoHeadbutt` (headbutt).
  - [pkg/game/skills.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/skills.go): Implements `DoMold` (molding).

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. Balance-Breaking Instant-Kill Cutthroat Bug (Severe Exploit)
- **Source Context**: `pkg/game/skill_advanced.go#L75` (`DoCutthroat`), `src/new_cmds.c#L650` (`do_cutthroat`).
- **Fidelity Bug**: In legacy C, succeeding on a `cutthroat` attempt only deals minor damage scaled to half the player's level (`GET_LEVEL(ch)/2`) and joins the `AFF_CUTTHROAT` silence affect. 
  In the Go port, **`DoCutthroat` has been turned into a 100% execution instant-kill command**:
  ```go
  // Instant kill: set target to -1 HP
  damage := target.GetHP() + 1
  target.TakeDamage(damage)
  ```
  - **Impact**: Any thief with a single point of `cutthroat` skill can instantly one-shot any creature or boss in the game from behind, completely breaking combat balance and quest progression.

### 2. Hilarious Self-Beheading Decapitation Bug
- **Source Context**: `pkg/game/skill_special.go#L125-L132` (`DoBehead`).
- **Fidelity Bug**: In legacy C, decapitating a corpse creates a head object (`16`) whose short description is dynamically concatenated with the *corpse's* short description (e.g. `a bloody head hacked from a town crier`).
  However, in the Go implementation, the severed head and headless corpse descriptions are dynamically formatted with **the attacker's name (`ch.Name`) instead of the victim's name**:
  ```go
  headObj.Runtime.ShortDesc = fmt.Sprintf("the severed head of %s", ch.Name)
  headlessCorpseObj.Runtime.ShortDesc = fmt.Sprintf("the headless corpse of %s", ch.Name)
  ```
  - **Impact**: Decapitating *any* corpse in the room results in spawning `"the severed head of <PlayerName>"` and `"the headless corpse of <PlayerName>"`, making it look like the player decapitated themselves!

### 3. Corpse Duplication Food Carving Bug
- **Source Context**: `pkg/game/skill_advanced.go#L31-L35` (`DoCarve`), `src/new_cmds.c#L168-L177` (`do_carve`).
- **Fidelity Bug**: In legacy C, carving a corpse checks keywords and spawns a specialized food object (`8015`, `12`, `13`, `14`) corresponding to the meat type.
  In the Go port, `DoCarve` creates the food instance by duplicating the corpse container's VNum itself:
  ```go
  food := &ObjectInstance{
      VNum:    corpse.VNum,
      RoomVNum: ch.GetRoomVNum(),
  }
  food.Runtime.ShortDescOverride = "some carved meat from " + corpse.GetShortDesc()
  ```
  - **Impact**: The carved food item retains all the container/corpse attributes (high weight, wrong object type) instead of becoming a real eatable `ITEM_FOOD` item, corrupting inventory dynamics.

### 4. Self-Contradictory Bearhug Level Check
- **Source Context**: `pkg/game/skill_special.go#L160-L162` (`DoBearhug`), `src/new_cmds.c#L525` (`do_bearhug`).
- **Fidelity Bug**: In legacy C, if an attacker is immortal, `percent` is set to `101` (representing a **guaranteed failure** in C's bearhug formula). 
  The Go port replicates this by setting `percent = 101` for high-level characters, but includes an ironic developer comment:
  ```go
  // Immortals always succeed, sleeping targets always hit
  if ch.GetLevel() > 60 {
      percent = 101
  }
  ```
  Because `percent` is compared as `percent > prob` (where `prob` maxes at 100), setting `percent = 101` causes immortals to **always fail** to bearhug, directly contradicting the comment.

---

## 3. Go Improvements Over C

### 1. Unified Pronoun Interpolation
- **Go Enhancement**: C’s action output macros (`act("$n rips the head off $p with $s bare hands...", ...)`) required complex parser expansions for possessive genders. Go implements clean, type-safe gendered pronoun interpolation structures (`GetPronouns(ch.Name, ch.GetSex())` and `ActMessage`), improving code readability.

### 2. Standardized Inventory Transfers
- **Go Enhancement**: Go manages item movements using unified, thread-safe callbacks (`world.MoveObjectToPlayerInventory`, `world.MoveObjectToNowhere`), automatically ensuring item counts and weights update cleanly without memory leaks.

---

## 4. Concurrency & Thread Safety

- **Safe Target Extractions**:
  - The combat skills leverage a thread-safe `FindTargetInRoom` utility that performs lookups under read-locks, preventing race conditions with concurrent player movement.
- **RWMutex Isolation**:
  - HP deductions and damage allocations (`target.TakeDamage`) utilize atomic updates or protected write locks on target models, ensuring multi-threaded combat ticks do not corrupt vital statistics.

---

## 5. Summary of Recommended Fixes

1. **Restore Parity to `DoCutthroat`**:
   Replace the instant-kill logic in `DoCutthroat` with standard level-scaled damage (`ch.GetLevel() / 2`) and re-introduce the `AFF_CUTTHROAT` silence effect.
2. **Correct Decapitation Descriptions**:
   Fix `DoBehead` so that it retrieves the short description of the *corpse* object and formats the newly spawned head and headless corpse descriptions with the corpse's details instead of the player's name (`ch.Name`).
3. **Correct Carved Food VNums**:
   Update `DoCarve` to map the corpse keywords to the standard food VNums (`8015`, etc.) rather than duplicating the container's VNum.
