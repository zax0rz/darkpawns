# Dark Pawns — Spawn New Characters in Room 8099 (Burning Hut) (DP-1077)

**Target file:** `pkg/session/char_creation.go`
**Small fix — ~10 lines.**

**Repo:** `/Users/zach/.openclaw/workspace/darkpawns_repo`
**Branch:** Create from `main`, name `fix/newbie-start-room-8099`
**After fixing:** `go build ./... && go vet ./... && go test ./...`
**Push:** `git push origin fix/newbie-start-room-8099`

---

## What the C Source Does

C places brand-new characters in room 8099 (A Burning Hut) as an intro moment. The player sees the room via `look_at_room()`, then immediately gets teleported to their actual hometown load room.

Reference: `src/interpreter.c:2241`
```c
char_from_room(d->character);
char_to_room(d->character, real_room(8099));
look_at_room(d->character, 0);
```

## Current Go Behavior

In `completeCharCreation()` (`pkg/session/char_creation.go`, around line 538):
```go
// Room 8099 (A Burning Hut) is the C source intro room (interpreter.c:2241)
// but it has no exits and no mob spawns in the current world data.
// For now, use LoginStartRoom which accounts for immortal/frozen status.
s.player.RoomVNum = game.LoginStartRoom(s.player)
```

Go skips the Burning Hut entirely and puts players directly in their hometown.

## The Fix

**`game.NewbieStartRoom`** already exists as a constant in `pkg/game/death.go:77`:
```go
const NewbieStartRoom = 8099
```

1. In `completeCharCreation()`, BEFORE `AddPlayer()`, set the initial room to 8099:
```go
// C source intro: spawn in the Burning Hut (interpreter.c:2241)
s.player.RoomVNum = game.NewbieStartRoom
```

2. AFTER `AddPlayer()` and `GiveStartingItems()`, send the room state so the player sees the Burning Hut:
```go
// Send initial room view (Burning Hut)
s.sendPlayerState()
```
(Use whatever method the session already uses to send room state to the client — check `session_send.go` for `sendPlayerState()` or equivalent.)

3. THEN immediately teleport to the real hometown load room:
```go
// Teleport to actual hometown (C: char_from_room + char_to_room to load room)
s.player.RoomVNum = game.LoginStartRoom(s.player)
```

4. **Do NOT send another room render after the teleport.** The player will see their real room when they get the full state message after entering the game (which happens later in the flow). OR — if you want them to see the hometown room immediately, send the state again after teleport. Both approaches are valid. C only shows the Burning Hut, not the hometown — the hometown is seen after `look_at_room` when entering the game proper. Match C's behavior.

## Summary

Replace the current comment + `LoginStartRoom` with:
1. Set room to `NewbieStartRoom` (8099) before `AddPlayer()`
2. Send room state so client renders the Burning Hut
3. Teleport to hometown load room

**Commit message:** `fix: spawn new characters in Burning Hut (8099) like C source (DP-1077)`
