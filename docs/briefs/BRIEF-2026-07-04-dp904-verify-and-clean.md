# Brief: DP-904 — U1000 Dead Code Verification & Cleanup

Owner: Kimi (has Linear access)
Scope: Verify each symbol is truly dead, then delete confirmed dead code. Wire anything that's actually needed. Update Linear per issue.

## Context

DP-904 has 7 sub-issues (DP-915 through DP-921) covering ~140 dead symbols across 23 files. These are C-port functions that were ported from CircleMUD C source but never wired into the Go command registry. The `//lint:file-ignore U1000` suppression hides them from the linter.

**Some of these functions ARE called from other files in the same package.** The `//nolint:unused` comments on `act_comm.go` and `mobact.go` prove this — those functions are dead from staticcheck's perspective (because `staticcheck U1000` operates per-file), but they ARE used by other files in `pkg/game/`.

**Your job: verify, then act.** Don't blindly delete. Check callers first.

## Critical Rule: Intra-Package Calls

`staticcheck -checks U1000` is a **per-file** check. A function can be unused within its own file but called by another file in the same package. These are NOT dead — they need to stay (or be wired).

The `//nolint:unused` comments mark functions that ARE called cross-file. If you find a function that's called by another file, **leave it alone** and note it as "intra-package call, not dead."

## Workflow (per file)

1. Read the file
2. For each function/type/const marked dead by `staticcheck U1000`:
   a. Search the ENTIRE `pkg/game/` directory for calls to that symbol: `grep -rn "symbolName" pkg/game/`
   b. If called from another file → it's intra-package, leave it, note it
   c. If NOT called from anywhere → it's genuinely dead, delete it
3. Remove `//lint:file-ignore U1000` from the file IF all dead symbols were deleted
4. Remove `//nolint:unused` comments that are no longer needed
5. Run: `go build ./... && go vet ./...`
6. Do NOT run `go test ./...` — tests take too long for this task

## Per-File Approach

### Group 1: DP-915 — Item helpers (4 files, ~22 symbols)

**`pkg/game/item_helpers.go`** (14 dead):
- `scmdDrink`, `scmdSip`, `scmdEat`, `scmdTaste`, `scmdPour`, `scmdFill` — drink/eat sub-commands. Check if `doDrink`/`doEat`/`doPour` in `item_consumable.go` reference these. They probably DO.
- `drinkAff` — lookup table for drink effects. Check if `doDrink` uses it.
- `contIsCloseable`, `contIsLocked`, `contSetClosed`, `contSetLocked` — container helpers. Check if `item_door.go` or `item_equipment.go` use these.
- `drinkLiquidIndex` — liquid lookup. Check `doDrink`/`doPour`.
- `removeFromSlice` — utility. Search all of `pkg/game/`.
- `moneyDesc` — money description. Search all of `pkg/game/`.

**`pkg/game/item_door.go`** (5 dead):
- `doorIsOpenable`, `doOpen`, `doClose`, `doUnlock`, `doLock` — door commands. Check command registry in `pkg/session/commands.go` for "open"/"close"/"unlock"/"lock". If registered, they're LIVE (called via command dispatch), not dead.

**`pkg/game/item_consumable.go`** (2 dead):
- `doDrink`, `doEat`, `doPour` — same check as above. Likely registered.

**`pkg/game/item_equipment.go`** (1 dead):
- `findEqPos` — equipment position lookup. Check `doWear`/`doRemove` in same file.

### Group 2: DP-916 — Combat shadow (7 files, ~27 symbols)

**`pkg/game/combat_basic.go`** (7 dead):
- `doAssist`, `doHit`, `doKill`, `doBackstab`, `doBackstabMob`, `doDisembowel`, `doDisembowelMob`
- These are ALL command handlers. Check `pkg/session/commands.go` for "hit"/"kill"/"assist"/"backstab"/"disembowel". If registered → LIVE via command dispatch. If NOT registered → dead.

**`pkg/game/combat_advanced.go`** (6 dead):
- `doRetreat`, `doSubdue`, `doSleeper`, `doNeckbreak`, `doAmbush`, `startCombatBetween`
- Same check for command registry. `startCombatBetween` might be called from combat_basic.go.

**`pkg/game/combat_melee.go`** (4 dead):
- `doBash`, `doRescue`, `doKick`, `doDragonKick`, `doTigerPunch`
- Check command registry for "bash"/"rescue"/"kick"/"dragonkick"/"tigerpunch"

**`pkg/game/combat_helpers.go`** (4 dead):
- `isMounted`, `isOutlaw`, `isShopkeeper`, `IsMobShopkeeper`, `isPiercingWeapon`, `improveSkill`, `rawKill`
- `isShopkeeper`/`IsMobShopkeeper`/`rawKill` — search ALL of `pkg/game/` for callers. These are utility functions used by combat flow.
- `isPiercingWeapon` — search `pkg/game/` for callers (backstab needs this).
- `improveSkill` — search for callers (skill improvement on use).

**`pkg/game/damage_stubs.go`** (3 dead):
- `DoSpellDamage`, `doDamage`, `hitSkill`, `getAttackerName`, `executeCommand`, `doForced`, `doMurder`, `diceRoll`, `updatePosFromHP`
- These were wired in session 83 (PR #38 — DoSpellDamage, diceRoll are LIVE). Verify each.

**`pkg/game/combat_control.go`** (2 dead):
- `doOrder`, `doFlee` — check command registry.

**`pkg/game/combat_ranged.go`** (1 dead):
- `doShoot` — check command registry for "shoot"/"range"/"fire".

### Group 3: DP-917 — Communication (3 files, ~14 symbols)

**`pkg/game/act_comm.go`** (9 dead — note: 10 have `//nolint:unused`):
- `condDrunk`, `drunkSyllables`, `speakDrunk` — drunk speech. Check if `comm_say.go` calls these.
- `getCharVis` — has `//nolint:unused // used in comm_channel.go, comm_tell.go, graph.go`. Verify callers exist.
- `deleteAnsiControls` — has `//nolint:unused // used in comm_say.go, comm_tell.go`. Verify.
- `lastTellersData`, `initLastTellers`, `setLastTeller`, `getLastTeller` — have `//nolint:unused // used in comm_tell.go`. Verify.

**`pkg/game/comm_tell.go`** (3 dead):
- `performTell`, `doTell`, `doReply` — check command registry for "tell"/"reply".

**`pkg/game/comm_channel.go`** (2 dead):
- `doSpecComm`, `doShout`, `doWhisper`, `doAsk`, `doWrite`, `doPage`, `doGenComm`, `doQcomm`, `doThink`, `doCTell`, `updateGossipHistory`, `ReviewGossip`
- Check command registry for "shout"/"whisper"/"ask"/"write"/"page"/"gossip"/"qcomm"/"think"/"ctell".

### Group 4: DP-918 — Movement/world (3 files, ~21 symbols)

**`pkg/game/act_movement.go`** (10 dead):
- `getExitByDirStr`, `doMove`, `doEnter`, `doLeave`, `doStand`, `doSit`, `doRest`, `doSleep`, `doWake`, `doFollow`
- Check command registry for "north"/"south"/"enter"/"leave"/"stand"/"sit"/"rest"/"sleep"/"wake"/"follow".
- `doMove` is likely called by `performMove` in the same file — intra-package call, not dead.

**`pkg/game/mobact.go`** (3 dead — note: 4 have `//nolint:unused`):
- `roomHasFlag`, `scanForMob`, `scanForPlayer` — have `//nolint:unused // mob helper`. Verify callers.
- `mobileActivityForMob` — check if called by `MobileActivityForMob`.

### Group 5: DP-919 — Character/remort/misc (3 files, ~29 symbols)

**`pkg/game/remort_helpers.go`** (6 dead):
- `findRemortClass`, `doFirstRemortAdjust`, `doSecondRemortAdjust`, `advanceLevel`, `setExp`, `number`
- Search all of `pkg/game/` for callers. These are core progression functions — likely called by level-up logic.

**`pkg/game/modify.go`** (6 dead):
- `doSet`, `doStat`, `doGecho`, `doSocial`, `doSkillset`, `doString` — wizard commands. Check command registry for "set"/"stat"/"gecho"/"social"/"skillset"/"string".

**`pkg/game/other_helpers.go`** (6 dead):
- `isPlayerNPC`, `actToRoom` (different signature from the one on World), `getPlayerByName`, `strCompare`, `hasRoomFlag`, `isDark`, `isOutdoors`, `getMount`
- These are utility functions. Search ALL of `pkg/game/` for callers.

### Group 6: DP-920 — Session/scripting (2 files, ~9 symbols)

**`pkg/game/world.go`** (3 dead — note: 1 has `//nolint:unused`):
- Check which symbols in world.go are flagged. Many of world.go's 60 functions are LIVE (used everywhere). Only the flagged ones matter.

**`pkg/game/mail.go`** (4 dead — note: 4 have `//nolint:unused`):
- `popFreeList`, `storeMail` — have `//nolint:unused // mail helper`. Verify callers in same file.

### Group 7: DP-921 — Info/score (1 file, 21 symbols)

**`pkg/game/info_commands.go`** (21 dead):
- `doScore`, `doWho`, `getWhoTitle`, `doInventory`, `doEquipment`, `doWhere`, `doLevels`, `doColor`, `doToggle`, `onOff`, `flagOnOff`, `doAbils`, `doSkills`, `doUsers`, `doExamine`, `doCoins`, `doDescription`, `doCommands`, `doDiagnose`, `doConsider`, `doHelp`
- ALL of these are command handlers. Check `pkg/session/commands.go` for "score"/"who"/"inventory"/"equipment"/"where"/"levels"/"color"/"toggle"/"abilities"/"skills"/"users"/"examine"/"coins"/"description"/"commands"/"diagnose"/"consider"/"help".
- If registered → LIVE (called via command dispatch, not direct call).
- This is probably the biggest catch — most of these are likely registered and NOT dead.

## Linear Updates

For each issue (DP-915 through DP-921):
1. Move to "In Progress" when you start
2. Add a comment listing:
   - Symbols confirmed dead and deleted
   - Symbols confirmed intra-package (NOT dead, explain why)
   - Symbols confirmed LIVE via command registry (NOT dead)
   - Files where `//lint:file-ignore U1000` was removed
3. Move to "Done" when complete

## What to report back

For each file, I need to know:
- How many symbols were actually dead vs intra-package vs LIVE
- What was deleted
- What was left (and why)
- Whether the file-level suppression was removed
- Build status after changes

## Non-goals

- Do NOT wire dead code to the command registry — that's a separate effort
- Do NOT modify the command registry
- Do NOT run tests (too slow for this task)
- Do NOT touch files outside `pkg/game/`
- Do NOT change any behavioral logic — only delete dead code or remove suppressions

## Important Notes

- The command registry is at `pkg/session/commands.go` — look for `cmdRegistry.Register("name", ...)` calls
- Functions registered as commands ARE live even if no direct Go caller exists — they're invoked by name at runtime via the command dispatch system
- `//nolint:unused` comments are hints but verify independently — the comment might be stale
- `go build ./...` MUST pass after changes. `go vet ./...` MUST pass.
- One branch per issue group, or one branch for all — your call. Keep commits atomic per file.
