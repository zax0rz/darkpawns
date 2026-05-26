# Port Fidelity Audit: Module 11 (`act.other.c`)

This audit examines the port fidelity between the legacy C source file `src/act.other.c` and its Go counterparts in `pkg/game/`, `pkg/session/`, and the bridging functions in `pkg/game/act_other_bridge.go`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/act.other.c` (1,947 lines)
- **Functions**: `do_quit`, `do_save`, `do_not_here`, `do_sneak`, `do_hide`, `do_steal`, `do_practice`, `do_visible`, `do_title`, `do_group`, `do_ungroup`, `do_report`, `do_split`, `do_use`, `do_wimpy`, `do_display`, `do_gen_write`, `do_gen_tog`, `do_afk`, `do_auto`, `do_transform`, `do_ride`, `do_dismount`, `do_yank`, `do_peek`, `improve_skill`, `do_recall`, `do_stealth`, `do_appraise`, `do_inactive`, `do_scout`, `do_roll`.

### Go Port Files
- **Active Bridge**:
  - `pkg/game/act_other_bridge.go` (Exports `ExecXxx` methods on `World` to bridge session calls to unexported game logic)
- **Session Commands**:
  - `pkg/session/cmd_misc.go` (Active session commands that delegate to `ExecXxx` wrappers)
  - `pkg/session/cmd_inventory.go` (Active `cmdQuit` implementation)
- **Game Logic**:
  - `pkg/game/other_character.go` (`doPractice`, `doVisible`, `doTitle`, `doGroup`, `doUngroup`, `doReport`)
  - `pkg/game/other_stealth.go` (`doNotHere`, `doSneak`, `doHide`, `doSteal`)
  - `pkg/game/other_settings.go` (`doWimpy`, `doDisplay`, `doGenWrite`, `doGenTog`)
  - `pkg/game/other_status.go` (`doAFK`, `doAuto`, `doTransform`)
  - `pkg/game/other_mount.go` (`doRide`, `doDismount`, `doYank`)
  - `pkg/game/other_utility.go` (`doPeek`, `doRecall`, `doStealth`, `doAppraise`, `doInactive`, `doScout`, `doRoll`)

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. Game Logout Security / Position Check Bypassed (`cmdQuit`)
- **Source Context**: `pkg/session/cmd_inventory.go#L11-L35` (`cmdQuit`)
- **Logic Gap**: In legacy C (and the dead `doQuit` in `other_session.go`), players cannot quit unless they are in safe temple rooms or owned rooms. If they try to quit elsewhere, they must use `REALLYQUIT` which deletes their equipment. Furthermore, quitting in combat is strictly blocked.
- **Fidelity Bug**: The active `cmdQuit` command in the session layer completely lacks any room or safety checks. Players can quit anywhere in the MUD (even in high-risk zones) with zero penalties, retaining all their equipment. Crucially, there are **no combat or position checks**, allowing players to quit during active fights to escape death.

### 2. Infinite Max HP Transform Exploit
- **Source Context**: `pkg/game/other_status.go#L131-L142` (`doTransform`)
- **Logic Gap**: Transforming into werewolf/vampire form boosts a character's current HP/mana.
- **Fidelity Bug**: When transforming into a werewolf in Go, the player's Max HP is permanently modified if their new HP exceeds Max HP:
  ```go
  ch.SetHP(ch.GetHP() + bonus)
  if ch.GetHP() > ch.GetMaxHP() {
      ch.SetMaxHP(ch.GetHP())
  }
  ```
  However, when transforming back, **the Max HP is never reverted to its original baseline**. This allows players to repeatedly transform back and forth to gain infinite, permanent Max HP.

### 3. Transformation Time & Moon Phase Restrictions Bypassed
- **Source Context**: `pkg/game/other_status.go#L116-L160` (`doTransform`)
- **Logic Gap**: In legacy C, werewolves and vampires can only transform at night (`weather_info.sunlight == SUN_SET || SUN_DARK`), and full/new moon phases determine the stat bonus magnitude. Hiding/transforming is blocked during the day, and daytime transforms revert automatically.
- **Fidelity Bug**: In Go, time, sunlight, and moon phase checks are entirely bypassed. Players can transform at high noon, and the stat bonus is a flat random roll (`randRange(2, 6) * 10`).

### 4. `peek` Command is a Mock Stub
- **Source Context**: `pkg/game/other_utility.go#L44-L45` (`doPeek`)
- **Logic Gap**: The `peek` command allows thieves and assassins to peek at another player's inventory and equipped items.
- **Fidelity Bug**: The Go implementation of `doPeek` is a mockup that prints a static header:
  ```go
  ch.SendMessage(fmt.Sprintf("You peek at %s's belongings:\r\n", victimPl.Name))
  ch.SendMessage("[Equipment and inventory]\r\n")
  ```
  It never actually queries or lists the victim's carrying or equipped items.

### 5. Stealing Item Syntax Inversion and Crash Bug
- **Source Context**: `pkg/game/other_stealth.go#L104-L115, L156-L160` (`doSteal`)
- **Logic Gap**: In C, the syntax is `steal <item> <victim>` (e.g. `steal sword guard`).
- **Fidelity Bug**:
  - The Go implementation parses arguments by splitting on space: `parts := strings.SplitN(arg, " ", 2)` and treats the *first* argument as the victim: `victimName := parts[0]` and `objName := parts[1]`. This inverts the command syntax to `steal <victim> <item>`.
  - If a player inputs `steal coins guard` (standard C syntax), Go treats `"coins"` as the target character's name, fails to find them, and aborts with `"They aren't here."`
  - Furthermore, if `objName` is anything other than `"coins"` or `"gold"`, the skill aborts with `"You can only steal coins for now."` making actual item thievery completely unimplemented.

### 6. Hiding Logic Inversion (Indoors Block)
- **Source Context**: `pkg/game/other_stealth.go#L67-L70` (`doHide`)
- **Logic Gap**: In C, characters can hide indoors but are blocked from hiding outdoors during the day in exposed terrain (fields, deserts, water, flying).
- **Fidelity Bug**: The Go code checks:
  ```go
  if room != nil && !isOutdoors(room) {
      ch.SendMessage("You can't hide indoors!\r\n")
      return true
  }
  ```
  This completely inverts the logic: players are blocked from hiding indoors (dungeons, taverns, castles) and are forced to only hide outdoors.

### 7. Recall Session Desynchronization
- **Source Context**: `pkg/game/other_utility.go#L89-L98` (`doRecall`)
- **Logic Gap**: Teleporting via recall should update both world and session-pump layers.
- **Fidelity Bug**: `doRecall` directly changes the player's room via `ch.SetRoom(8004)`. However, it never notifies the session manager. As a result, the session manager does not update the player's WebSocket client or room manager state, causing severe room state and broadcast desynchronization until the player manually moves or types `look`.

### 8. Mount Riding Multi-Rider Exploit
- **Source Context**: `pkg/game/other_mount.go#L53-L69` (`doRide`)
- **Logic Gap**: In C, riding sets mount flags on both the player and the mount mob (`AFF_MOUNT`) to mark the mob as ridden.
- **Fidelity Bug**: The Go implementation only sets the flag on the player (`ch.SetAffect(affMounted, true)`). The variable `mountAlreadyRidden` is explicitly ignored and unused, and the mount mob is never flagged as ridden. This allows multiple players to mount and ride the exact same mob at the same time.

---

## 3. Secondary Discrepancies & Stubs

### 1. Dead/Duplicate Quit File
- **Fidelity Gap**: `pkg/game/other_session.go` contains `doQuit`, which implements safe quits, Temple room checks, hometown checks, and dismounting. However, this file is completely unused because `pkg/session/cmd_inventory.go` has its own active (and highly bugged) `cmdQuit` handler.

### 2. Practice/Improvement Formula Divergence
- **Fidelity Gap**: In C, `improve_skill` uses `number(1, 200) > GET_WIS(ch) + GET_INT(ch)` to determine improvement chance, and adds a random amount of `number(1, 3)` percent, printing a message on a roll of `3`. In Go, the active skill improvement checks (e.g. `doSteal`, `doPeek`, `doAppraise`) inline their own divergent logic and hardcode Wis+Int checks or random skill increments.

---

## 4. Concurrency & Thread Safety

- **World-Session State Mutations**:
  - The `doQuit` command and the session `Unregister` routine must be synchronized to prevent duplicate closes on the session send channel (a known source of double-close panics in active sessions).
  - Stat mutations in `doTransform` and gold transfer in `doSplit` modify Player and target struct fields. Since these fields are accessed by concurrent read pumps and game loop cycles, locks must be held during all calculations.

---

## 5. Summary of Recommended Fixes

1. **Fix `cmdQuit` Gates**: Update `cmdQuit` in `pkg/session/cmd_inventory.go` to restore the temple/owned safe-room quit checks, prevent quitting in combat, and implement equipment loss for unsafe quits.
2. **Fix Werewolf HP Exploit**: Store the player's original baseline Max HP before werewolf transformation and restore it upon transforming back.
3. **Fix Steal Command Syntax**: Swap the argument parsing in `doSteal` to match the C standard `steal <item> <victim>` and implement actual item transfer.
4. **Fix Hide Indoors Flag**: Correct the inverted check in `doHide` to allow hiding indoors, and block hiding outdoors during exposed daytime conditions.
5. **Synchronize Recall Room Teleports**: Ensure `doRecall` triggers a session-layer room change update (similar to normal player movement) so that WebSocket clients and room managers are synchronized.
6. **Flag Mounts as Ridden**: Update `doRide` and `doDismount` to apply the `mounted` flag or rider references to the target `MobInstance`, and block multiple players from mounting the same mob.
7. **Complete `doPeek` List Logic**: Replace the mock `doPeek` header with a loop that formats and lists the target's equipped and inventory items.
