# Claude Code Batch — Run 5: Player Commands (Mock Implementations)

## Issues
- DP-402: Steal syntax inverted (HIGH)
- DP-404: Recall desyncs session (HIGH)
- DP-405: Mount not flagged as ridden (HIGH)
- DP-399: Quit bypasses safety checks (URGENT)
- DP-385: Pour command dead (HIGH)
- DP-400: Werewolf transform permanently increases MaxHP (URGENT)
- DP-401: Transform ignores time/moon restrictions (MEDIUM)

## Task: Fix steal syntax (DP-402)

**File:** `pkg/game/other_stealth.go:104-115`

C syntax: `steal <item> <victim>` (e.g. `steal sword guard`)
Go splits `parts[0]` = victim, `parts[1]` = item — inverted.

**Fix:** Swap the argument parsing. Also implement item theft (currently only coins/gold work). The C source has full item theft via `get_obj_in_list`.

**C source:** `src/act.other.c — do_steal()`

## Task: Fix recall session desync (DP-404)

**File:** `pkg/game/other_utility.go:89-98`

`doRecall` sets `ch.SetRoom(recallRoom)` but never notifies the session/WebSocket client.

**Fix:** After changing the room, send a room state update to the client. Look at how movement commands (`cmdMove`) notify the session layer — there should be a `SendRoomState` or similar function. Call the same notification path.

**C source:** `src/act.other.c — do_recall()`

## Task: Flag mount as ridden (DP-405)

**File:** `pkg/game/other_mount.go:53-69`

`doRide` sets `affMounted` on the player but never flags the mount mob. `mountAlreadyRidden` is computed but discarded with `_ =`.

**Fix:**
1. Set `AFF_MOUNT` on the mount mob (not the player — the player gets `affMounted`)
2. Remove the `_ = mountAlreadyRidden` line
3. Add the `mountAlreadyRidden` check before allowing mounting

**C source:** `src/act.other.c — do_mount()`

## Task: Add quit safety checks (DP-399)

**File:** `pkg/session/cmd_inventory.go:11-35`

C source blocks quitting in combat, checks room flags (safe rooms), and applies equipment penalty for unsafe quit.

**Fix:** Add to `cmdQuit`:
1. Check if player is in combat → "No way! You're fighting!"
2. Check if room has DEATH/HOUSE/NOQUIT flags → apply `REALLYQUIT` penalty (drop equipment)
3. Only allow clean quit in safe rooms (temple, etc.)

**C source:** `src/act.other.c — do_quit()`

## Task: Wire pour command (DP-385)

**File:** `pkg/session/commands.go` (command registry)

`doPour` is fully implemented in `pkg/game/item_consumable.go:155` but never registered.

**Fix:** Add to the command registry in `commands.go`:
```go
cmdRegistry.Register("pour", wrapArgs(cmdPour), "Pour liquid.", 0, 0)
```

You'll need a `cmdPour` wrapper in the session layer that calls `game.DoPour`. Check how other item commands (eat, drink) are wrapped.

**C source:** `src/act.item.c — do_pour()`

## Task: Fix werewolf MaxHP exploit (DP-400)

**File:** `pkg/game/other_status.go:131-142`

Transform adds bonus HP, then sets `MaxHP = HP` if HP exceeds MaxHP. Reverting caps HP to MaxHP but MaxHP was never restored.

**Fix:** Save the original MaxHP before transformation. On revert, restore MaxHP to the saved value. Look at the C source for the exact save/restore pattern.

**C source:** `src/act.other.c — do_transform()` werewolf section

## Task: Add transform time/moon checks (DP-401)

**File:** `pkg/game/other_status.go:116-160`

C source: werewolves/vampires can only transform at night. Moon phase determines bonus magnitude.

**Fix:**
1. Check `GetSunlight()` — only allow transform during `SunSet` or `SunDark`
2. Add moon phase logic — check `weather.go` for moon phase data
3. Daytime transforms should revert automatically (check C source for the revert mechanism)

**C source:** `src/act.other.c — do_transform()`

## Verification
1. `go build ./...` — must pass
2. `go vet ./...` — must pass
3. `go test ./...` — must pass
