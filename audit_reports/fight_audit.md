# Port Fidelity Audit: Module 25 (`fight.c`)

This audit examines the port fidelity between the legacy C source file `src/fight.c` (combat loop, damage formulas, death, and party experience/loot systems) and its Go implementations in `pkg/combat/` and `pkg/game/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/fight.c` (2,034 lines)
- **Core Functions**:
  - `perform_violence` (combat round tick loop executing rounds for all active characters)
  - `hit`, `damage` (melee hit checks, THAC0 calculations, and damage application)
  - `die`, `die_with_killer`, `raw_kill` (character death routines, experience penalties, and stat drops)
  - `make_corpse`, `make_dust` (creates item container and transfers equipment/inventory/gold)
  - `group_gain`, `solo_gain` (experience calculations and party/group division math)
  - `attitude_loot`, `brag_message` (aggro mob autonomous looting routines and global brag chat notifications)

### Go Port Files
- `pkg/combat/engine.go` (coordinates active combat pairs, triggers rounds every 2 seconds, and handles position recovery)
- `pkg/combat/fight_core.go` (implements damage messages, constitution loss, auto-loot/auto-split command dispatch, and milestone counter-procs)
- `pkg/combat/formulas.go` (calculates hit chances using class THAC0, parry/dodge checks, and AC damage reduction)
- `pkg/game/death.go` (implements the game-layer `HandleDeath` hook, managing gold splits, experience awards, and concrete corpse spawning)
- `pkg/game/deferred_fight_fns.go` (implements AC-based damage reduction math, mounting, and mob memories)
- `pkg/session/manager.go` (instantiates the thread-safe combat engine and hooks up websocket broadcasts)

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. Integer Division Fidelity Preserved (Brilliant AC Multiplier)
- **Source Context**: `pkg/combat/formulas.go#L490-L501` (`CalculateDamage`)
- **Fidelity Note**: In legacy CircleMUD, the position multiplier for damage applied to a sitting, sleeping, or incapacitated target was:
  ```c
  dam *= 1 + (POS_FIGHTING - GET_POS(victim)) / 3;
  ```
  While comments in C claimed this resulted in multipliers like `1.33` (sitting) or `1.66` (resting), C's native integer division truncated these, meaning sitting/resting did **not** scale damage at all, while sleeping/stunned was a flat `2.00x` multiplier, and mortally wounded was `3.00x`.
  Many modern ports mistakenly use floating point math (breaking game balance). The Go port **faithfully preserved the integer division behavior**, guaranteeing identical damage ratios to the original game.

### 2. AutoGold Duplication Guard
- **Source Context**: `pkg/game/death.go#L243-L250` (`handleMobDeath`)
- **Fidelity Bug**: When a player with `AutoGold` enabled kills a mob, the MUD awards the mob's gold directly to their wallet.
  In Go, `handleMobDeath` contains a safeguard checking if the player has `AutoGold` active, and if so, it sets `corpseGold = 0` to prevent the gold from appearing in the corpse and being duplicated.
  However, in `pkg/combat/fight_core.go#L587`, the combat engine unconditionally dispatches a command execution:
  ```go
  if HasPrfFlag != nil && HasPrfFlag(chName, "PRF_AUTOGOLD") {
      PerformCommand(chName, "get all gold corpse")
  }
  ```
  If `AutoGold` is enabled, the command tries to loot a corpse that has `0` gold. If `AutoGold` is disabled, the gold is in the corpse, but the player doesn't run the `get all gold` command.
- **Impact**: While safe from gold-duplication exploits, this results in useless, redundant string command executions ("get all gold corpse") dispatched from the combat thread for players who already have the money.

### 3. Faithful Fall-Through Bug (Milestone Counter-Procs)
- **Source Context**: `pkg/combat/fight_core.go#L1360-L1375` (`CounterProcs`)
- **Fidelity Bug**: In legacy C, the switch checking player kill milestones (e.g. 10,000 kills) missed `break` keywords between stat cases, resulting in all cases executing.
  The Go port **intentionally reproduces the fall-through bug** to preserve this beloved server quirk:
  ```go
  case 1000, 2000, 10000, 20000, 30000, 40000, 50000:
      // Major milestones: random +1 max stat, full heal...
      // Since case 3 falls through to default and all lack breaks,
      // ALL THREE stats get +1 (case 1+3 hit, case 2 mana, case 3 move).
      if IncreaseMaxStat != nil {
          IncreaseMaxStat(ch.GetName(), "hp")
          IncreaseMaxStat(ch.GetName(), "mana")
          IncreaseMaxStat(ch.GetName(), "move")
      }
  ```
- **Impact**: Players receive boosts to all three stats instead of just one, maintaining 100% gameplay mechanics alignment.

---

## 3. Go Improvements Over C

### 1. Robust Lock Ordering Protocol (Deadlock Prevention)
- **Fidelity Improvement**: Multi-threaded combat in Go poses severe ABBA deadlock risks when locking the world state vs individual characters. Go establishes a strict lock-ordering contract:
  `1. CombatEngine.mu` (guards combat pair maps) $\rightarrow$ `2. World.mu` (guards active entity slices) $\rightarrow$ `3. Player.mu` (guards character stats)
  This allows parallel telnet readers, AI ticks, and combat loops to run safely without thread starvation or locking conflicts.

### 2. Typed Combatant Interface
- **Fidelity Improvement**: In legacy C, combat routines accepted `char_data *ch` pointers and utilized unsafe void castings or macro checks. Go abstracts this behind a clean `Combatant` interface:
  ```go
  type Combatant interface {
      GetName() string
      IsNPC() bool
      GetRoom() int
      GetHP() int
      GetMaxHP() int
      GetPosition() int
      GetLevel() int
      GetClass() int
      GetStr() int
      GetStrAdd() int
      GetDex() int
      GetInt() int
      GetWis() int
      GetHitroll() int
      GetDamroll() int
      GetAC() int
      TakeDamage(amount int)
      Heal(amount int)
      SendMessage(msg string)
      SetFighting(name string)
      StopFighting()
      GetFighting() string
  }
  ```
  This decouples the physical MUD entities from the mathematical combat rules, making the combat formulas extremely clean and modular.

### 3. Asynchronous Position Recovery
- **Fidelity Improvement**: Go implements a separate `StartMobPositionRecovery` routine that periodically scans idle mobs and stands them up if they are sitting or sleeping outside of combat. This decouples position recovery from the intensive 2-second combat round loops.

---

## 4. Summary of Recommended Fixes / Enhancements

1. **Avoid Redundant Get-Gold Commands**:
   Refactor `pkg/combat/fight_core.go` line 587 to skip executing `"get all gold corpse"` if the player's gold was already directly deposited into their wallet during `handleMobDeath`.
2. **Standardize Combatant Property Mutators**:
   Currently, the combat engine uses a massive list of global `var` hooks (e.g. `GetConstitution`, `SetConstitution`, `GetAlignment`) in `fight_core.go`. These should eventually be refactored to clean methods on the `Combatant` interface itself to avoid runtime `nil` pointer checks on hooks.
