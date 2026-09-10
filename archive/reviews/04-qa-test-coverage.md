# QA & Test Coverage Audit

**Date:** 2026-05-03

---

## Part 1: QA Findings

### Critical Bugs

#### 1. `movementLoss` Array Index Mismatch — `act_movement.go:87-104`

The array entries for FLYING and UNDERWATER are labeled in reverse relative to the Go constants:

```go
var movementLoss = []int{
    ...
    2, // FLYING — this is index 8 in C, but C has UNDERWATER=8 and FLYING=9
    6, // UNDERWATER (index 8 in C structs.h)
```

But the Go constants are `SECT_FLYING = 9` and `SECT_UNDERWATER = 8`. This means:
- A FLYING room (sector=9) uses `movementLoss[9]` = 6 — the underwater cost
- An UNDERWATER room (sector=8) uses `movementLoss[8]` = 2 — the flying cost

Result: flying rooms are expensive (6 move/sector, same as underwater) and underwater rooms are nearly free (2 move/sector, same as indoor). The values need to be swapped, or the constants need to match the original C ordering.

#### 2. Double Death Message — `death.go:323-327` and `357`

In `handlePlayerDeath`, the player receives two experience loss messages:
- Line 325: `"You lose %d experience points.\r\n"` (grammatically "lose")
- Line 357: `"You lost %d experience.\r\n"` (grammatically "lost", different tense)

The second message at line 357 is always sent, even when `expLoss == 0`. The first message at line 325 is gated on `expLoss > 0`. Both messages are sent to the same player on every death, producing duplicate output.

#### 3. Backstab Mob Logic Inversion — `combat_basic.go:208-215`

When `vict == nil` (no player found by name), the code loops through mobs to find a match. When the mob IS found, it sends "They aren't here." and returns instead of attempting the backstab:

```go
for _, m := range w.GetMobsInRoom(ch.RoomVNum) {
    if strings.Contains(strings.ToLower(m.GetName()), strings.ToLower(victName)) {
        ch.SendMessage("They aren't here.\r\n")  // BUG: mob WAS found
        return true
    }
}
```

The backstab command cannot target mobs at all — it either says "They aren't here" (mob found) or "Backstab who?" (mob not found). The mob path in `doDisembowel` has the same issue (`combat_basic.go:211`).

#### 4. `doInventory` / `doEquipment` Nil Prototype Dereference — `info_commands.go:72,87`

Both functions access `item.Prototype.ShortDesc` without checking if `Prototype` is nil. Synthetic objects (corpses, money, ash) have `Prototype: nil`. If a player has such an item in inventory or equipment, the score/inventory commands will panic.

```go
ch.SendMessage(fmt.Sprintf("[%2d] %s\r\n", i+1, item.Prototype.ShortDesc)) // panics if Prototype==nil
```

#### 5. Moderation Commands Not Wired to Game Chat — `pkg/moderation/manager.go`

`Manager.CheckMessage` and `IsMuted` are implemented and tested, but are not called from any game communication handlers (`comm_say.go`, `comm_channel.go`, `comm_tell.go`). Admin `mute` command stores the penalty but it has no effect on gameplay — muted players can still say, gossip, shout, and tell freely.

---

### Edge Case Issues

#### 6. `doKill` Self-Same Level Check — `combat_basic.go:174`

```go
if vict.GetLevel() == ch.GetLevel() {
```

Only blocks killing someone at exactly the same level. An implementor at level 54 can kill anyone at level 53 or below with one command. This should likely be `>=` for meaningful protection.

#### 7. `doFlee` Locked Door Check — `combat_control.go:137`

The flee code checks `exit.DoorState == 1` (closed) but not `== 2` (locked). A character can flee through a locked door. Locked door state should also be blocked.

#### 8. `doSimpleMove` Tunnel Check Off-By-One — `act_movement.go:273-278`

The tunnel check blocks entry when `len(players) >= 1`. This means a tunnel can hold exactly 0 players — any attempt to enter a tunnel, even an empty one, succeeds (0 < 1), but a single player in the tunnel blocks everyone else. Intended behavior should be "max 2" or similar. The original CircleMUD used `!number(0,1)` style checks. Verify the intended capacity.

#### 9. `AffectUpdate` Race Condition — `affect_update.go:46-72`

The function reads `p.ActiveAffects` under `mu.RLock()` but then writes `p.ActiveAffects` under `w.mu.Lock()` (world mutex). These are two different mutexes. Between the read-lock release and the write-lock acquisition, another goroutine could modify `p.ActiveAffects`, causing the write to clobber concurrent changes.

#### 10. `ManaGain` Equipment Affect Only Applies While Sleeping — `limits_gain.go:35`

```go
gain += p.sumEquipAffect(ApplyManaRegen, pos == PosSleeping)
```

The `ApplyManaRegen` positive bonus applies only when sleeping. The comment says "Positive modifier only applies while sleeping; negative always applies." This matches the original C source at lines 89-95. The behavior is correct but surprising — equipment with `ApplyManaRegen` has no positive benefit unless sleeping.

#### 11. `HitGain` Equipment Regen Position Check — `limits_gain.go:112`

```go
gain += p.sumEquipAffect(ApplyHitRegen, false)
```

Called inside `case PosSleeping` but passes `false` as the sleeping parameter, which is inconsistent with the comment `"limits.c:156-162, only while sleeping"`. This may mean HP regen equipment always applies regardless of position, contradicting the source comment. Verify against C source behavior.

#### 12. `doBackstab` Does Not Check `vict.IsFighting()` for Player — `combat_basic.go:238`

The check `vict.IsFighting()` at line 238 blocks backstabbing fighting players. But the check uses `vict.GetPosition() >= posSleeping` at line 243, which means a sleeping NPC will notice the backstab (causing it to enter combat). However, a sleeping PC can be backstabbed silently. Sleeping PCs and NPCs should have consistent handling.

---

### New Feature Verification

#### Autoloot — WORKS CORRECTLY
`death.go:193-213`: After mob death, if killer has `PRF_AUTOLOOT` flag, all takeable items are moved from corpse to player inventory. Non-takeable items are skipped. Empty corpses get an appropriate message. Logic is solid.

#### Pick Lock — WORKS CORRECTLY
`act_movement.go:547-558`, `okPick:585-603`: When `scmdPick` succeeds, `doorLocked` transitions to `doorClosed` (unlocked). The reverse door is updated. The door IS actually unlocked. Magic lock guard at line 660 fires before `okPick` so the flow is correct.

#### Admin Mute — PARTIALLY BROKEN
`admin_commands.go:199-248`: Penalty is stored in `moderation.Manager.activePenalties` correctly. `IsMuted()` checks it correctly. But no game communication handler calls `IsMuted` or `CheckMessage`, so muted players are not actually silenced.

#### Admin Ban — PARTIALLY BROKEN  
Same issue as mute — ban is stored but not checked at login or command dispatch. Banned players are not blocked from connecting.

#### Weather Events — WORKS IF WIRED
`weather.go:115-183`: All broadcasts route through the `sendToOutdoor func(string)` callback. If the caller (game loop) passes a proper broadcast function, all players in outdoor rooms receive weather messages. If `nil` is passed, all weather messages silently drop. Verify that `main.go` wires this correctly.

#### Help System — FRAGILE
`command/registry.go:27,66`: `HelpText` is stored per command entry. `Lookup("help")` would return the entry, but there's no `do_help` command registered in the registry. The help system relies on callers implementing their own lookup. No test or registration found.

#### Shop Save/Load — NOT TESTED
The world save/load test (`save_world_test.go`) does not include shop state in the round-trip. Shop inventory and state are not serialized in `SerializeWorld`/`DeserializeWorld`. After a server restart, all shop inventory would be lost.

#### Password Change — NOT IMPLEMENTED
No `ChangePassword` function found in any game package. No bcrypt operations found in `pkg/game/`. The feature referenced in the task brief does not appear to exist in the codebase.

---

## Part 2: Test Coverage

### Current Coverage Summary

| File | Package | What It Tests |
|------|---------|--------------|
| `load_test/load_test.go` | load_test | Connection stress test (not functional unit tests) |
| `pkg/auth/ratelimit_test.go` | auth | HTTP rate limiting, X-Forwarded-For proxy header parsing, trusted proxy CIDR |
| `pkg/engine/affect_test.go` | engine | Affect creation, tick/expiry, apply/remove stat modifiers, stacking rules, poison/regen periodic damage |
| `pkg/engine/skill_test.go` | engine | Skill creation, CanLearn, Learn, Practice, Use, GetDisplayLevel, CanTeach, SkillManager CRUD, slots |
| `pkg/events/lua_integration_test.go` | events | Lua event integration |
| `pkg/events/queue_test.go` | events | Event queue publish/subscribe, concurrency |
| `pkg/game/object_movement_test.go` | game | Object location tracking: room→inventory, equip/unequip, container nesting, extract, MoveObject rollback, mob equipment, save/load Location round-trip. Has 6 BUG tests that now pass (bugs were fixed). |
| `pkg/game/save_world_test.go` | game | World serialize/deserialize: empty world, door states, ID counters, gossip history, full round-trip |
| `pkg/game/systems/door_test.go` | systems | Door states, open/close/lock/unlock/pick/bash, DoorManager add/remove/lookup |
| `pkg/game/systems/shop_test.go` | systems | Shop CRUD, price calculations, type checking, buy/sell transactions |
| `pkg/metrics/metrics_test.go` | metrics | Prometheus metrics counters |
| `pkg/moderation/manager_test.go` | moderation | Word filter censor/block, regex, spam detection, CheckMessage pipeline. `hasPenalty` is private so the test documents but doesn't test it. |
| `pkg/parser/wld_test.go` | parser | WLD world file parsing |
| `pkg/privacy/client_test.go` | privacy | Privacy client |
| `pkg/scripting/integration_test.go` | scripting | Engine creation, spell damage formula documentation |
| `pkg/scripting/integration_test_batchd_test.go` | scripting | Batch Lua script processing |
| `tests/unit/combat_test.go` | unit | `CalculateHitChance` hit probability, `CalculateDamage` minimum damage, `RollDice` bounds |

---

### Critical Untested Paths (Top 20)

1. **Player death / corpse creation flow** — `death.go:247-358`  
   Why critical: Central player experience. Tests needed: exp loss (combat vs non-combat), CON loss at level 6+ and level 21+, inventory/equipment transferred to corpse, gold in corpse, respawn at MortalStartRoom, double death message bug.

2. **Mob death / autoloot** — `death.go:144-233`  
   Why critical: Mob kill is the primary gameplay loop. Tests needed: corpse created in correct room, all mob inventory in corpse, autoloot with PRF_AUTOLOOT set, SPELL_DISINTEGRATE scatters items (makeDust), spawner instance count decremented.

3. **Combat damage formulas** — `pkg/combat/formulas.go`, `fight_core.go`  
   Why critical: All combat outcomes depend on these. Tests needed: THAC0 table lookup per class/level, STR damage bonus application, position modifiers (sleeping takes double), critical hit detection.

4. **HP/Mana/Move regeneration** — `limits_gain.go:3-214`  
   Why critical: Determines game balance. Tests needed: ManaGain per class (mage/cleric doubles, psionic +25%), position multipliers (sleeping=2×), poison/flaming/cutthroat penalties (÷4), veteran bonus, ROOM_REGENROOM bonus, hunger/thirst penalty.

5. **AffectUpdate ticker** — `affect_update.go:34-74`  
   Why critical: Spell duration. Tests needed: duration countdown, wear-off message sent, `AffectFromChar` called on expiry, permanent affect (-1 duration) never expires.

6. **doSimpleMove sector/movement cost** — `act_movement.go:226-348`  
   Why critical: Contains the `movementLoss` index bug. Tests needed: move cost calculation per sector pair, water/boat check, tunnel capacity, death room instant-kill, sneak suppresses messages.

7. **Player login / SavePlayer / LoadPlayer** — `save.go`, `objsave.go`  
   Why critical: Persistence. Tests needed: player saved to disk, loaded back, inventory items restored with correct Location, equipment restored to correct slots, gold/exp/stats preserved.

8. **doGenDoor / door state machine** — `act_movement.go:606-674`  
   Why critical: Core exploration mechanic. Tests needed: open/close/lock/unlock transitions, key check (wrong key rejected), pickproof door rejected by `okPick`, pick success unlocks door, bidirectional exit sync.

9. **Combat start / stop** — `combat_advanced.go` `startCombatBetween`, `fight_core.go`  
   Why critical: All combat depends on this. Tests needed: initiating combat sets fighting state on both parties, death stops combat, flee updates fighting state.

10. **doBackstab / doDisembowel mob targeting** — `combat_basic.go:193-324`  
    Why critical: Core rogue/assassin mechanics; has the mob-targeting logic inversion bug. Tests needed: backstab player succeeds from stealth, backstab fighting target fails, backstab mob (currently broken), piercing weapon required.

11. **Admin mute/ban enforcement** — `pkg/command/admin_commands.go`  
    Why critical: Moderation has no effect without this. Tests needed: muted player's `say`/`gossip` commands are blocked, ban blocks login, expired mute allows speech again.

12. **Shop buy/sell transactions via game commands** — `pkg/command/shop_commands.go`  
    Why critical: The `systems/shop_test.go` tests the data model but not the command dispatch. Tests needed: `buy` command transfers gold and item, `sell` command credits gold and removes item, insufficient gold rejected, wrong item type rejected.

13. **Weather broadcast to outdoor rooms** — `weather.go`  
    Why critical: Weather messages only reach players if `sendToOutdoor` is wired. Tests needed: `WeatherAndTime` with a mock broadcast function sends sunrise/sunset/storm messages, `SetWeatherWorld` correctly scopes broadcasts.

14. **Spell affect application to players** — `player_affects.go`  
    Why critical: Spells are the primary mage/cleric mechanic. Tests needed: `HasSpellAffect` detects active spell, affect applied modifies stats, affect expired removes modifier.

15. **Zone reset / mob spawning** — `spawner.go`, `zone_dispatcher.go`  
    Why critical: World population. Tests needed: zone reset respawns killed mobs, mob instance count tracked correctly, items in zone reset to default positions.

16. **World save/load including shop state** — `save.go` + `systems/shop_manager.go`  
    Why critical: Shop data lost on restart. Tests needed: serialized world includes shop inventory, deserialized world restores shop items.

17. **Lua scripting API surface** — `pkg/scripting/`  
    Why critical: Scripts drive all zone-specific behavior. Current tests only check engine creation, not API functions. Tests needed: `isfighting()`, `getchar()`, `damage()`, `teleport()`, `create_item()` return correct values.

18. **Player position / combat state transitions** — `limits_condition.go`, `other_status.go`  
    Why critical: Position affects regen, hit chance, and ability to act. Tests needed: sleep/rest/sit transitions send correct messages, standing from sleep resets position, fighting position blocked from sleeping.

19. **Container cycle detection** — `object_movement_test.go:559-589`  
    Why critical: A→B→A container cycle corrupts `GetTotalWeight` and any recursive walk. The BUG test now passes, but the fix should be verified as a property-based test with deeper nesting.

20. **EXP/XP award on mob kill** — `death.go:97`  
    Why critical: Leveling progression. `AwardMobKillXP` is called but untested. Tests needed: solo kill awards full exp, group kill splits exp, max XP cap (`maxExpGain = 100000`) enforced, XP triggers level-up check.

---

### Test Quality Assessment

**Strengths:**
- `object_movement_test.go` is excellent — thorough, documents surprising behavior, has BUG-labeled tests that were fixed.
- `save_world_test.go` covers the round-trip well.
- `engine/affect_test.go` and `engine/skill_test.go` are comprehensive for their packages.
- `systems/door_test.go` covers all door state transitions including edge cases.

**Weaknesses:**
- **No tests for the `game` package core logic**: combat handlers, death, movement, limits/regen, communication — all untested.
- **62 files tagged `//nolint:unused`**: These are complete C ports not yet wired to the command registry. They compile but are unreachable. Any bugs in them are invisible.
- **Moderation tests don't test enforcement** — `hasPenalty` is private and the integration (mute → block chat) is entirely untested.
- **No integration tests**: No test exercises a complete game session (login → move → fight → die → respawn).
- **Combat test is minimal**: `tests/unit/combat_test.go` only tests that hit chance produces some hits, damage is ≥1, and dice are in bounds. No edge case coverage.

---

## Recommendations

### Priority 1 — Fix Bugs (before any new features)

1. Fix `movementLoss` array: swap indices 8 and 9 or rename constants to match C ordering.
2. Fix `doBackstab`/`doDisembowel` mob targeting: the loop should attempt the skill, not print "They aren't here."
3. Remove duplicate death XP message in `handlePlayerDeath`.
4. Add nil checks in `doInventory` and `doEquipment` before accessing `item.Prototype.ShortDesc`.
5. Wire `IsMuted`/`IsBanned` into `comm_say.go`, `comm_channel.go`, `comm_tell.go`, and session login.

### Priority 2 — Test Writing Plan (ordered by risk)

1. Death flow test (`TestPlayerDeath`, `TestMobDeath`) — highest risk, untested critical path
2. Regen formula tests (`TestManaGain`, `TestHitGain`, `TestMoveGain`) — pure functions, easy to test
3. Combat state tests (`TestStartCombat`, `TestFlee`, `TestBackstabMob`) — validates the fixes above
4. Door integration test (`TestPickLockSuccess`, `TestPickLockPickproof`) — validates the wire between act_movement and door state
5. Moderation enforcement test (`TestMutedPlayerCannotTalk`, `TestBannedPlayerCannotLogin`)
6. Save/load round-trip with items (`TestPlayerSaveLoadInventory`) — validates objsave Location bug

### Priority 3 — Wire Unwired Code

62 files are tagged `nolint:unused`. Before wiring any of them to the command registry, audit each for the `movementLoss`-style index bugs that accumulate during C→Go ports. Prioritize: `combat_basic.go`, `combat_melee.go`, `act_movement.go`, `info_commands.go`.
