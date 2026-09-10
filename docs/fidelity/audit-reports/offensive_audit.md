# Port Fidelity Audit: Module 10 (`act.offensive.c`)

This audit examines the port fidelity between the legacy C source file `src/act.offensive.c` and its Go counterparts in `pkg/game/`, `pkg/session/`, `pkg/combat/`, and `pkg/command/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/act.offensive.c` (1,510 lines)
- **Functions**: `do_assist`, `do_hit`, `do_kill`, `do_backstab`, `do_disembowel`, `do_order`, `do_flee`, `do_bash`, `do_rescue`, `do_kick`, `do_dragon_kick`, `do_tiger_punch`, `do_shoot`, `do_retreat`, `do_subdue`, `do_sleeper`, `do_neckbreak`, `do_ambush`.

### Go Port Files
- **Active Commands**:
  - `pkg/session/commands.go` (Command registrations)
  - `pkg/session/combat_cmds.go` (`cmdHit`, `cmdFlee`)
  - `pkg/session/cmd_combat_basic.go` (`cmdAssist`)
  - `pkg/session/cmd_combat_special.go` (`cmdOrder`)
  - `pkg/command/skill_commands.go` (`CmdBackstab`, `CmdBash`, `CmdKick`, `CmdTrip`, `CmdHeadbutt`, `CmdRescue`, `CmdDisembowel`, `CmdDragonKick`, `CmdTigerPunch`, `CmdShoot`, `CmdSubdue`, `CmdSleeper`, `CmdNeckbreak`, `CmdAmbush`, `CmdParry`)
  - `pkg/game/skill_combat.go` (`DoBackstab`, `DoBash`, `DoKick`, `DoTrip`, `DoHeadbutt`, `DoRescue`)
  - `pkg/game/skill_c10_combat.go` (`DoDisembowel`, `DoDragonKick`, `DoTigerPunch`, `DoShoot`, `DoSubdue`, `DoSleeper`, `DoNeckbreak`, `DoAmbush`, `DoParry`, `CheckParry`, `CheckNPCDodge`)
- **Dead / Unwired Files** (marked with `//lint:file-ignore U1000 Game logic port — not yet wired`):
  - `pkg/game/combat_advanced.go` (duplicate methods on `World` for subdue, sleeper, neckbreak, ambush, retreat)
  - `pkg/game/combat_melee.go` (duplicate methods on `World` for bash, rescue, kick, dragon_kick, tiger_punch)
  - `pkg/game/combat_control.go` (duplicate methods on `World` for order, flee)

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. The Wimpy / Auto-Flee System is Silently Broken
- **Source Context**: `pkg/combat/fight_core.go#L546-L560` (Wimpy check)
- **Logic Gap**: In the core combat tick, if a player drops below their configured wimpy HP threshold, the engine attempts to trigger an automatic flee or retreat by invoking package-level callbacks:
  ```go
  if hasRetreat || hasEscape {
      if DoRetreat != nil {
          DoRetreat(victimName)
      }
  } else if DoFlee != nil {
      DoFlee(victimName)
  }
  ```
- **Fidelity Bug**: The global hooks `DoFlee` and `DoRetreat` are defined as package-level hook variables in `fight_core.go` but **are never initialized or wired in production** (only mocked in `round_test.go`). Consequently, they are `nil` at runtime, causing the entire automatic wimpy-flee system to fail silently.

### 2. Player Flee Command (`cmdFlee`) Bypasses XP Penalties for Mortals
- **Source Context**: `pkg/session/combat_cmds.go#L148-L169` (`cmdFlee`)
- **Logic Gap**: The C implementation of `do_flee` penalizes character experience point totals based on the damage inflicted on their opponent (`loss = MaxHP(opp) - HP(opp); loss *= Level(opp)`), regardless of player level, and adds a level-scaled bonus penalty if level > 10.
- **Fidelity Bug**: In Go, the entire experience subtraction call is enclosed inside an `if level > 10` block:
  ```go
  level := s.player.GetLevel()
  if level > 10 {
      xpLoss += int(500 * (float64(level) / 2.6))
      s.player.LoseExp(xpLoss)
      ...
  }
  ```
  As a result, mortal characters of level 10 or less suffer **zero experience point loss** when fleeing from combat, even if their opponent was heavily damaged.

### 3. Flee Mechanism Discrepancies
- **Source Context**: `pkg/session/combat_cmds.go#L138-L144` (`cmdFlee`)
- **Logic Gap**: The C implementation of `do_flee` loops up to 6 times trying random directions. If a direction contains a valid, open, and non-death room exit, it executes a simple move. If it fails to move due to closed doors or blocked rooms, it prints a specific message.
- **Fidelity Bug**: In Go, fleeing is mapped to a flat 50% chance failure check:
  ```go
  if rand.Intn(100) > 50 {
      s.Send("You attempt to flee but fail!")
      return nil
  }
  ```
  If it succeeds, it picks a random direction exit and calls `MovePlayer` directly, which could bypass closed exit door checks or crash on blocked rooms.
- *Note*: The dead file `pkg/game/combat_control.go` has a much more faithful implementation of `doFlee` using the 6-loop directional search, but it is completely unused.

### 4. Ignored Stun Flag (`StunTarget`) Causes Silent Stunning Failure
- **Source Context**: `pkg/command/skill_commands.go#L1371-L1417` (`sendSkillResult`)
- **Logic Gap**: Skills like `DoSubdue`, `DoSleeper`, `DoHeadbutt`, and `DoBash` return `StunTarget: true` in their `SkillResult` payload to indicate that the victim was successfully stunned or knocked out.
- **Fidelity Bug**: The command mapper `sendSkillResult` completely ignores the `StunTarget` flag. It applies `SelfStumble` (sits the attacker) and `TargetFalls` (sits the target), but never checks `result.StunTarget` or changes the target's position to `PosStunned`. Consequently, stun-based skills fail to stun their victims.

### 5. Silent Failure in Sleeper Hold (`DoSleeper`)
- **Source Context**: `pkg/game/skill_c10_combat.go#L180-L220` (`DoSleeper`)
- **Logic Gap**: The C version of `do_sleeper` sets `GET_POS(vict) = POS_SLEEPING` and applies the `AFF_SLEEP` affect to put the target out of action.
- **Fidelity Bug**: In Go, `DoSleeper` merely returns `StunTarget: true` in its result:
  ```go
  return SkillResult{
      Success: true, Damage: 0, StunTarget: true, WaitCh: 2,
      MessageToCh: ActMessage("You put $N in a sleeper hold.", chPronouns, &victPronouns, ""),
      ...
  }
  ```
  Because `sendSkillResult` ignores `StunTarget` and `DoSleeper` does not apply any sleep affect directly to the character, the sleeper hold skill is **mechanically useless** — it emits messages claiming the target went to sleep, but the victim remains active, awake, and standing.
- *Note*: The dead method `World.doSleeper` in `combat_advanced.go` correctly set `vict.SetPosition(combat.PosSleeping)` and `vict.SetAffect(affSleep, true)` but is un-wired.

### 6. Mobs Exempt from Knockdown/Sitting Positions
- **Source Context**: `pkg/command/skill_commands.go#L1376-L1381` (`sendSkillResult`)
- **Logic Gap**: When a skill like Bash or Trip succeeds, the target should be knocked down to a sitting position (`combat.PosSitting`).
- **Fidelity Bug**: The Go implementation explicitly ignores mobs:
  ```go
  if result.TargetFalls && target != nil {
      if p, ok := target.(*game.Player); ok {
          p.SetPosition(combat.PosSitting)
      }
      // Mobs don't have SetPosition in current interface — would need Combatant extension
  }
  ```
  Mobs never sit down when bashed or tripped, leaving combat-positioning benefits completely dead against NPCs.

### 7. Ranged Shooting (`do_shoot`) Highly Truncated
- **Source Context**: `pkg/game/skill_c10_combat.go#L110-L136` (`DoShoot`)
- **Logic Gap**: The legacy C `do_shoot` command allowed players to wield a bow/sling, aim in a direction, and fire projectiles into adjacent rooms. The hit calculated damage and dragged the targeted mob into the shooter's room to attack them.
- **Fidelity Bug**: The Go implementation of `DoShoot` is restricted entirely to the **same room**, ignoring exit directions, closed doors, and adjacent rooms. It operates as a standard instant melee hit with ranged flavor text.

### 8. Mortal Players Unable to Assist or Order
- **Source Context**: `pkg/session/commands.go#L211, L221` (Command Registry)
- **Fidelity Bug**: The `assist` and `order` commands are registered with a minimum level of `LVL_IMMORT` (level 31):
  ```go
  cmdRegistry.Register("assist", wrapArgs(cmdAssist), "Assist a target in combat.", LVL_IMMORT, combat.PosFighting)
  cmdRegistry.Register("order", wrapArgs(cmdOrder), "Order a pet or follower.", LVL_IMMORT, 0)
  ```
  In legacy C, both commands are available to mortals (e.g. to assist party members or order charmed mobs). Restricting them to immortals breaks core gameplay mechanics.

### 9. Charmed Orders (`cmdOrder`) are a Mock
- **Source Context**: `pkg/session/cmd_combat_special.go#L9-L39` (`cmdOrder`)
- **Fidelity Bug**: The C version of `do_order` allowed players to command charmed followers to execute any command using `command_interpreter`. The Go implementation in `cmdOrder` is a mock stub: it searches for a mob matching the name in the room and prints a line saying it obeys your order, but **does not actually parse or execute the command on the mob**.

---

## 3. Secondary Discrepancies & Stubs

### 1. Implementor Kill Command Missing
- **Fidelity Gap**: In C, implementors have access to `do_kill` as a brutal instakill slay. For mortals, it delegates to `do_hit`. In Go, `kill` is a simple alias to `hit` for all levels, and the immortal instakill function is omitted.

### 2. Automatic Dismount on Attack Missing
- **Fidelity Gap**: In C, if a player is mounted and hits a target, the game automatically dismounts them via `do_dismount` to initiate fighting. In Go, `cmdHit` completely lacks the mount-checking code, allowing players to engage in normal combat rounds while remaining mounted.

### 3. Retreat / Escape Commands are Unimplemented
- **Fidelity Gap**: Although the dead file `combat_advanced.go` contains `doRetreat`, there is no `CmdRetreat` or registration for `"retreat"` or `"escape"` in the active command registry `pkg/session/commands.go`. These commands are dead and unavailable to players.

---

## 4. Concurrency & Thread Safety

- **Session-Engine State Sharing**:
  - Combat rounds are performed autonomously on the background `CombatEngine` ticker (`pkg/combat/engine.go:84-95`).
  - Read/Write concurrency exists when session command threads modify character attributes (such as moves or hitpoints) while the combat ticker reads them (e.g. to deduct move points or apply round changes). Ensure `Player.mu` or `MobInstance.mu` locks are held during all mutations.
- **Lock Ordering**:
  - The `handleDeath` method in the combat engine documents a lock contract:
    1. `CombatEngine.mu` (guards pair map)
    2. `World.mu` (guards players/mobs maps)
    3. `Player.mu` (guards player fields)
  - This contract must be strictly followed when wiring future combat hooks to avoid ABBA deadlocks.

---

## 5. Summary of Recommended Fixes

1. **Wire Wimpy Hooks**: Wire up `DoFlee` and `DoRetreat` callbacks in `pkg/session/manager.go` to invoke the session-based `cmdFlee` (or a corrected flee function) to restore functional wimpy triggers.
2. **Fix `sendSkillResult` Stuns**: Update `sendSkillResult` in `pkg/command/skill_commands.go` to check `result.StunTarget` and correctly alter player and mob positions to `combat.PosStunned`.
3. **Extend Combatant interface for Mob Positions**: Add `SetPosition(int)` to the `combat.Combatant` interface and implement it on `MobInstance` so that bashes, trips, and sleeper holds correctly sat/slid mobs to sitting or sleeping states.
4. **Remove Level Limits on Mortal Commands**: Lower the minimum level requirement for `"assist"` and `"order"` in `pkg/session/commands.go` from `LVL_IMMORT` to `0`, restoring mortal gameplay.
5. **Fix Mortal Flee XP Loss Bug**: Move the `s.player.LoseExp(xpLoss)` call in `cmdFlee` outside the `level > 10` check block, so all levels pay the base damage-based XP fee for fleeing, leaving only the bonus penalty level-gated.
6. **Implement Real Sleeper Hold Effects**: Modify `DoSleeper` in Go to apply the `affSleep` affect and set position to `PosSleeping`.
7. **Clean up Dead Files**: Remove the duplicate, unused files `combat_advanced.go`, `combat_melee.go`, and `combat_control.go` to prevent codebase clutter and developer confusion.
