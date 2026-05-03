# Dark Pawns MUD — Gap Analysis

Generated: 2026-05-03
Scope: stubs, missing features, broken wiring, and incomplete ports from C source.

---

## CRITICAL

### 1. `cmdGet` doesn't support "get X from container" or "get all from corpse"
- **File:** `pkg/session/cmd_inventory.go` line 229
- **Problem:** The registered `cmdGet` handler only searches room items by short description substring match. It does NOT parse `get <item> <container>` or `get all` syntax. The game layer has a fully-ported `doGet()` in `pkg/game/item_transfer.go` that handles containers, corpses, `get all`, dotmode — but it is **never called** from the session layer.
- **Fix:** Replace or augment `cmdGet` to call `w.doGet(ch, nil, "get", arg)`. The game-layer function already handles all cases.

### 2. "give" command not registered
- **File:** `pkg/session/commands.go`
- **Problem:** `doGive()` is fully implemented in `pkg/game/item_transfer.go` (handles give object, give N coins, give all, dotmode) but **no "give" command is registered** in the command registry. Players cannot give items or gold.
- **Fix:** Register `"give"` command calling through to `w.doGive(ch, nil, "give", arg)`.

### 3. "put" command not registered
- **File:** `pkg/session/commands.go`
- **Problem:** `performPut()` exists in `pkg/game/item_container.go` and handles putting items into containers, but **no "put" command is registered**.
- **Fix:** Register `"put"` command calling through to the game layer's put logic.

### 4. `RegisterCommand` in session_command.go is a stub
- **File:** `pkg/session/session_command.go` line 10
- **Problem:** `Manager.RegisterCommand()` is a debug stub that only logs and discards. Any code path that tries to register commands dynamically (plugins, modules) will silently fail.
- **Fix:** Wire to actual registry or remove the method.

### 5. World deserialization not implemented
- **File:** `pkg/game/save.go` line 359
- **Problem:** `world deserialization not implemented yet` — player/world saves likely broken for full state.
- **Fix:** Implement deserialization or document current save/load scope.

---

## HIGH

### 6. MovePlayer doesn't check dark rooms, sector types, or move costs
- **File:** `pkg/game/world.go` `MovePlayer()` ~line 490
- **Problem:** Only checks exit existence and target room validity. Does NOT:
  - Check `IS_DARK(room)` + player's ability to see in dark
  - Check sector type (underwater, flying requirements)
  - Deduct move points based on sector movement cost
  - Check `AFF_BLIND` or visibility
  - The C source's `do_move()` in `act.movement.c` checks all of these.
- **Fix:** Add darkness check, sector validation, and move point deduction before allowing movement.

### 7. Starvation/dehydration doesn't cause damage
- **File:** `pkg/game/limits_condition.go` (GainCondition), C reference `src/limits.c` point_update
- **Problem:** The C `point_update()` applies poison damage, cutthroat damage, and incapacitated damage every tick. The Go port handles condition messages (hungry/thirsty) but **does not apply damage when hunger/thirst reach 0**. In C, the gain functions reduce HP/mana/move regen by 75% when hungry/thirsty, but there's no separate starvation damage — the damage comes from reduced regen. The Go `HitGain`/`ManaGain` implementations need to verify they include the hunger/thirst penalty (they appear to based on `limits_gain.go`), but `point_update()` should verify poison/cutthroat/incap damage ticks are ported.
- **Fix:** Audit `PointUpdate()` in `limits_condition.go` to ensure poison damage, cutthroat damage, and incapacitated/mortally wounded damage ticks match C.

### 8. Autoloot not wired to death/combat
- **File:** `pkg/session/autoloot.go`
- **Problem:** `autoLootState` is a session-layer toggle with a `cmdAutoLoot` handler, but nothing in the game layer's death or combat code checks `IsAutoLootEnabled()`. Corpses are created on death (`pkg/game/death.go`) but auto-loot is never triggered. The game layer has `party.go` with manual looting functions, but those aren't wired either.
- **Fix:** After mob death and corpse creation, check `IsAutoLootEnabled(killer)` and auto-transfer items/gold to killer's inventory.

### 9. cmdLook doesn't handle "look <target>", "look in <container>", or extra descriptions
- **File:** `pkg/session/cmd_look.go`
- **Problem:** Only implements bare `look` (room view). Missing:
  - `look <mob/player/item>` — examine a specific target
  - `look in <container>` — see container contents
  - `look <direction>` — look through an exit/door
  - Extra descriptions (keywords in rooms)
- **Fix:** Parse args for target/container/direction and dispatch accordingly.

### 10. Scripting engine has many unimplemented functions
- **File:** `pkg/scripting/engine.go` lines 2151, 2162, 2191, 2202, 2515
- **Problem:** Multiple mob script functions are stubs: `direction`, `set_hunt`, `unaffect`, `equip_char`, `steal`. These log debug and return no-ops.
- **Fix:** Implement each function with proper World access.

---

## MEDIUM

### 11. Mail system stub
- **File:** `pkg/game/mail.go` lines 456-457
- **Problem:** Mail message entry via string writer is a placeholder. `log the intent and store a placeholder`.
- **Fix:** Implement full mail write/entry functionality.

### 12. Gossip broadcasts to room only, not all players
- **File:** `pkg/scripting/integration_test.go` line 1248
- **Problem:** `gossip implemented — sends to room (TODO: broadcast to all players)`.
- **Fix:** Change gossip to broadcast globally like the C original.

### 13. Skill review is a placeholder
- **File:** `pkg/game/skill_special.go` line 409
- **Problem:** Returns a message that review was requested — no actual review logic.
- **Fix:** Implement skill review mechanics.

### 14. Pick lock is a placeholder in skill_stealth.go
- **File:** `pkg/game/skill_stealth.go` line 187
- **Problem:** Comments say "actual pick lock logic is in door_commands.go" but function returns immediately as placeholder.
- **Fix:** Verify door_commands.go has the real implementation and wire through.

### 15. Affect manager has two placeholder methods
- **File:** `pkg/engine/affect_manager.go` lines 574, 578
- **Problem:** Two methods are "For now, just a placeholder" — no actual affect processing.
- **Fix:** Implement affect application/removal logic.

### 16. Combat helpers placeholder
- **File:** `pkg/game/combat_helpers.go` line 51
- **Problem:** "This is a placeholder implementation."
- **Fix:** Implement real combat helper logic.

### 17. Skill commands registration is a no-op
- **File:** `pkg/command/skill_commands.go` lines 1720-1726
- **Problem:** "Registration placeholder — commands are called directly via Cmd* handlers." This means the skill command system has a split path between registration and direct dispatch.
- **Fix:** Either remove dead registration code or unify the dispatch path.

### 18. Socials work for players but mob social targeting is limited
- **File:** `pkg/session/cmd_social.go`
- **Problem:** Socials are functional for players (6+ message format with $n/$N substitution, target found/not found/self-target). However, `actToRoomMob()` in `world.go` uses hardcoded "him"/"his" pronouns instead of checking mob gender. Also mob social matching uses exact `EqualFold` on full short desc — won't match partial names like `get mob guard`.
- **Fix:** Add partial name matching for mob targets, fix pronoun resolution.

### 19. `cmdSocial` victim matching uses exact match on mob short descriptions
- **File:** `pkg/session/cmd_social.go` ~line 100
- **Problem:** `strings.EqualFold(m.GetShortDesc(), targetName)` requires the full short description (e.g. "A city guard" not "guard"). Player names do exact match too but players typically type short names.
- **Fix:** Use partial/keyword matching consistent with how `findPlayerByName` works.

### 20. "password" and "prompt" commands missing
- **File:** `pkg/session/commands.go`
- **Problem:** No `password` (change password) or `prompt` (set prompt string) commands registered. These are basic MUD essentials.
- **Fix:** Implement `cmdPassword` and `cmdPrompt` handlers.

### 21. Wiznet TODO — future save-state snapshots
- **File:** `pkg/session/wizard_cmds.go` line 81
- **Problem:** `switch` command notes "Save-state snapshots before/after switch (requires DB)".
- **Fix:** Implement state snapshots for switch/return.

### 22. `get all` / `get all.<item>` not wired in session layer
- **File:** `pkg/session/cmd_inventory.go` line 229
- **Problem:** Even if `doGet` were called, the session `cmdGet` would need to handle multi-word args properly. Currently `args` are joined into a single string then searched — `get all sword` becomes a single search for "all sword".
- **Fix:** Wire to game-layer `doGet` which already handles dotmode.

---

## LOW

### 23. Lua player lookup not implemented
- **File:** `pkg/scripting/engine.go` line 796
- **Problem:** `_ = ch // placeholder until Lua player lookup is implemented`
- **Fix:** Implement Lua-side player lookup.

### 24. Extra description table not implemented in scripting
- **File:** `pkg/scripting/integration_test.go` line 1048
- **Problem:** "extra stubbed (extra desc table not implemented)"

### 25. `isname()` matching inconsistent between layers
- **File:** Various
- **Problem:** Session layer `cmdGet` uses `strings.Contains` (substring match on short desc). Game layer `doGet` uses `isname()` (keyword-based matching from C). Different matching behavior means the same command can succeed or fail depending on which code path runs.
- **Fix:** Standardize on `isname()` / keyword matching everywhere.

### 26. Mob `executeMobCommand` drop/get/give not implemented
- **File:** `pkg/game/world.go` ~lines 337-339
- **Problem:** `drop`, `get`, `give` for mob-executed commands just log "not yet implemented".
- **Fix:** Wire to existing game-layer functions.

### 27. Socials gender pronouns are hardcoded for mobs
- **File:** `pkg/game/world.go` `actToRoomMob()`
- **Problem:** `$m` → "him", `$s` → "his" regardless of mob sex.
- **Fix:** Check mob `GetSex()` and use appropriate pronouns.

### 28. Duplicate `bug`/`typo`/`idea`/`todo` registrations
- **File:** `pkg/session/commands.go`
- **Problem:** `bug` is registered at line ~387 with aliases `typo`, `idea`, `todo`, then `typo`, `idea`, `todo` are registered again individually at lines ~389-391. The later registrations overwrite the earlier ones with separate handler functions.
- **Fix:** Remove duplicate registrations, keep one set with proper aliasing.

---

## Prioritized Fix Order

| Priority | Gap # | What | Effort |
|----------|-------|------|--------|
| 1 | #1 | Wire `cmdGet` → `doGet` (enables get from container/corpse) | Low |
| 2 | #2 | Register "give" command → `doGive` | Low |
| 3 | #3 | Register "put" command → `performPut` | Low |
| 4 | #6 | Add dark room + sector checks to MovePlayer | Medium |
| 5 | #7 | Audit PointUpdate for poison/incap damage ticks | Medium |
| 6 | #8 | Wire autoloot to death handler | Medium |
| 7 | #9 | Expand cmdLook for targets/containers/exits | Medium |
| 8 | #5 | Implement world deserialization (save/load) | High |
| 9 | #10 | Implement missing scripting functions | High |
| 10 | #20 | Add password/prompt commands | Low |
| 11 | #4 | Wire or remove RegisterCommand stub | Low |
| 12 | #25 | Standardize name matching (isname vs Contains) | Medium |
| 13 | #12 | Fix gossip to broadcast globally | Low |
| 14 | #18-19 | Fix social mob targeting (partial match) | Low |
| 15 | #11 | Implement mail entry | Medium |
| 16-28 | Remaining | Stubs and polish | Varies |

**Quick wins (items 1-3):** The game layer has fully-ported `doGet`, `doGive`, and `performPut`. Wiring them to the session command registry is trivial and unlocks core MUD item interaction immediately.
