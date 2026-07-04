# DP-904 — U1000 dead-code inventory (burn-down backlog)

_Generated 2026-07-03 via `staticcheck -checks U1000 ./...` with all `//lint:file-ignore U1000` suppressions stripped._

## Status

- **DP-905 done:** `MakeHit()` shadow damage path deleted; `pkg/combat/fight_core.go` no longer needs its suppression.
- **Suppressions retired this pass (6):** `pkg/combat/fight_core.go`, `pkg/game/act_informative.go`, `pkg/game/other_economy.go`, `pkg/game/show.go`, `pkg/game/spec_procs3.go`, `pkg/session/manager.go` — all verified U1000-clean.
- **Also deleted:** dead `(*Session).ctx()` accessor (zero callers) in `pkg/session/session_helpers.go`.
- **Remaining:** 154 genuinely-dead symbols across 43 files still behind 40 file-level suppressions. Each is a C-port function not yet wired to the command registry — **wire or delete, per system.**

## Why not remove all 46 + add a CI gate now

A blanket `staticcheck U1000` CI gate today would be **red** on 154 symbols. These are intentional C-port stubs awaiting wiring, not accidental cruft — mass-deleting them discards fidelity work; mass-wiring them is a large effort. The safe path: **burn down per system, then flip the gate on** (or add a non-increasing ratchet first). Suppressions are retired file-by-file only once a file reaches zero dead symbols, as done for the 6 above.

## Remaining dead symbols by file (154)

### `pkg/game/info_commands.go` (21)

- L8: `func (*World).doScore`
- L41: `func (*World).doWho`
- L57: `func (*World).getWhoTitle`
- L65: `func (*World).doInventory`
- L80: `func (*World).doEquipment`
- L96: `func (*World).doWhere`
- L110: `func (*World).doLevels`
- L122: `func (*World).doColor`
- L149: `func (*World).doToggle`
- L176: `func onOff`
- L183: `func flagOnOff`
- L194: `func (*World).doAbils`
- L207: `func (*World).doSkills`
- L219: `func (*World).doUsers`
- L232: `func (*World).doExamine`
- L254: `func (*World).doCoins`
- L263: `func (*World).doDescription`
- L273: `func (*World).doCommands`
- L290: `func (*World).doDiagnose`
- L366: `func (*World).doConsider`
- L379: `func (*World).doHelp`

### `pkg/game/item_helpers.go` (14)

- L152: `const scmdDrink`
- L153: `const scmdSip`
- L154: `const scmdEat`
- L155: `const scmdTaste`
- L160: `const scmdPour`
- L161: `const scmdFill`
- L305: `var drinkAff`
- L346: `func contIsCloseable`
- L359: `func contIsLocked`
- L363: `func contSetClosed`
- L371: `func contSetLocked`
- L380: `func drinkLiquidIndex`
- L484: `func removeFromSlice`
- L642: `func moneyDesc`

### `pkg/game/act_movement.go` (10)

- L149: `func getExitByDirStr`
- L406: `func doMove`
- L687: `func doEnter`
- L718: `func doLeave`
- L747: `func doStand`
- L770: `func doSit`
- L792: `func doRest`
- L814: `func doSleep`
- L836: `func doWake`
- L872: `func doFollow`

### `pkg/game/act_comm.go` (9)

- L51: `const condDrunk`
- L398: `var drunkSyllables`
- L424: `func speakDrunk`
- L471: `func (*World).getCharVis`
- L506: `func deleteAnsiControls`
- L553: `type lastTellersData`
- L557: `func (*World).initLastTellers`
- L563: `func (*World).setLastTeller`
- L568: `func (*World).getLastTeller`

### `pkg/game/combat_basic.go` (7)

- L32: `func (*World).doAssist`
- L92: `func (*World).doHit`
- L150: `func (*World).doKill`
- L191: `func (*World).doBackstab`
- L262: `func (*World).doBackstabMob`
- L300: `func (*World).doDisembowel`
- L367: `func (*World).doDisembowelMob`

### `pkg/game/combat_advanced.go` (6)

- L32: `func (*World).doRetreat`
- L109: `func (*World).doSubdue`
- L218: `func (*World).doSleeper`
- L325: `func (*World).doNeckbreak`
- L401: `func (*World).doAmbush`
- L493: `func (*World).startCombatBetween`

### `pkg/game/modify.go` (6)

- L17: `func (*World).doSet`
- L258: `func (*World).doStat`
- L300: `func (*World).doGecho`
- L316: `func (*World).doSocial`
- L351: `func (*World).doSkillset`
- L448: `func (*World).doString`

### `pkg/game/remort_helpers.go` (6)

- L12: `func findRemortClass`
- L93: `func doFirstRemortAdjust`
- L105: `func doSecondRemortAdjust`
- L119: `func advanceLevel`
- L171: `func setExp`
- L204: `func (*World).kenderSteal`

### `pkg/game/item_door.go` (5)

- L9: `func doorIsOpenable`
- L14: `func (*World).doOpen`
- L65: `func (*World).doClose`
- L111: `func (*World).doUnlock`
- L184: `func (*World).doLock`

### `pkg/game/combat_helpers.go` (4)

- L28: `const lvlImpl`
- L39: `func isOutlaw`
- L61: `func isShopkeeper`
- L76: `func isPiercingWeapon`

### `pkg/game/combat_melee.go` (4)

- L32: `func (*World).doBash`
- L221: `func (*World).doKick`
- L273: `func (*World).doDragonKick`
- L326: `func (*World).doTigerPunch`

### `pkg/game/mobprogs.go` (4)

- L22: `func isJanitor`
- L24: `func isMercenary`
- L220: `func (*World).getBadGuy`
- L247: `func (*World).killBadGuy`

### `pkg/game/other_character.go` (4)

- L14: `func (*World).doPractice`
- L91: `func (*World).doTitle`
- L124: `func (*World).doGroup`
- L242: `func (*World).doUngroup`

### `pkg/game/other_stealth.go` (4)

- L15: `func (*World).doNotHere`
- L24: `func (*World).doSneak`
- L59: `func (*World).doHide`
- L120: `func (*World).doSteal`

### `pkg/session/info_cmds.go` (4)

- L12: `func className`
- L32: `func positionName`
- L58: `func conditionLabel`
- L82: `func cmdInfo`

### `pkg/game/comm_tell.go` (3)

- L5: `func (*World).performTell`
- L30: `func (*World).doTell`
- L74: `func (*World).doReply`

### `pkg/game/damage_stubs.go` (3)

- L113: `func (*World).hitSkill`
- L148: `func (*World).doMurder`
- L168: `func (*World).updatePosFromHP`

### `pkg/game/mobact.go` (3)

- L363: `func (*World).scanForMob`
- L374: `func (*World).scanForPlayer`
- L384: `func mobAlive`

### `pkg/game/other_helpers.go` (3)

- L90: `func getPlayerByName`
- L100: `func strCompare`
- L141: `func getMount`

### `pkg/session/movement_cmds.go` (3)

- L179: `func cmdFleeMovement`
- L299: `func cmdFollowMovement`
- L361: `func cmdSneak`

### `pkg/session/tattoo.go` (3)

- L52: `func useTattoo`
- L110: `func tattooAf`
- L190: `func applyModifier`

### `pkg/game/clans.go` (2)

- L350: `func (*ClanManager).findClanOrError`
- L365: `func (*World).resolveClanContext`

### `pkg/game/combat_control.go` (2)

- L32: `func (*World).doOrder`
- L92: `func (*World).doFlee`

### `pkg/game/comm_channel.go` (2)

- L139: `func (*World).doWrite`
- L250: `func (*World).doPage`

### `pkg/game/comm_say.go` (2)

- L80: `func (*World).doSay`
- L120: `func (*World).doGSay`

### `pkg/game/graph.go` (2)

- L115: `func (*World).canGo`
- L132: `func (*World).doTrack`

### `pkg/game/item_consumable.go` (2)

- L10: `func (*World).doDrink`
- L102: `func (*World).doEat`

### `pkg/game/combat_ranged.go` (1)

- L30: `func (*World).doShoot`

### `pkg/game/constants.go` (1)

- L9: `const sendBufSize`

### `pkg/game/gates.go` (1)

- L45: `const numGates`

### `pkg/game/item_equipment.go` (1)

- L313: `func (*World).doRemove`

### `pkg/game/look.go` (1)

- L464: `func (*World).doExits`

### `pkg/game/mail.go` (1)

- L76: `var noMail`

### `pkg/game/other_session.go` (1)

- L14: `func (*World).doQuit`

### `pkg/game/skills.go` (1)

- L390: `func heSheIt`

### `pkg/game/spec_procs2.go` (1)

- L112: `func guardAssist`

### `pkg/game/world.go` (1)

- L116: `field lastTellers`

### `pkg/scripting/engine.go` (1)

- L116: `func matchKeyword`

### `pkg/session/examine.go` (1)

- L14: `func itemCheck`

### `pkg/session/fight.go` (1)

- L7: `func cmdParry`

### `pkg/session/shop.go` (1)

- L7: `func cmdNotBuy`

### `pkg/session/time_weather.go` (1)

- L20: `var timePeriods`

### `pkg/session/wiz_system.go` (1)

- L353: `func cmdBroadcast`

## Pre-existing unsuppressed dead code (not part of the 46)

These already trip standalone staticcheck (no suppression) and are separate from the DP-904 file-ignore set:

- `pkg/game/mobprogs.go`: `isJanitor`, `isMercenary`
