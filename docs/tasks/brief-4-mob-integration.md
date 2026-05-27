# Brief 4: Mob AI Integration (Sonnet — 6 issues)

All touch the mob AI/tick system. Changes interact with each other.

## 1. DP-478 — Entire hardcoded MobProg system never wired (LARGE)

**File:** `pkg/game/mobprogs.go` — all functions are marked `//nolint:unused`
**Bug:** MpGreet, MpGive, MpBribe, MpSound, EntryProg, NpcRescue are implemented but never called.

**Integration points:**
- `MpGreet` → call when a player enters a room with mobs. Find the movement/entry handler.
- `MpGive`/`MpBribe` → call when a player gives items or coins to a mob. Find the `give` command handler.
- `MpSound` → already called from mobileActivityForMob (see DP-480).
- `EntryProg` → call when a mob enters a room (after wandering).
- `NpcRescue` → call from combat helper logic.

**Approach:** Search for each event's dispatch point in the Go code. Look at how Lua triggers are dispatched for reference.

---

## 2. DP-474 — Animate Dead spawns zombie from nothing (CRITICAL)

**File:** `pkg/game/affect_spells.go` in `MagSummons`, case `SpellAnimateDead` (line 486)
**Bug:** Spawns zombie without checking for or consuming a corpse.

**Fix:**
```go
case SpellAnimateDead:
    // Require a corpse in the room
    room := ch.GetRoomVNum()
    items := world.GetItemsInRoom(room)
    var corpse *ObjectInstance
    for _, item := range items {
        if item != nil && strings.Contains(strings.ToLower(item.GetKeywords()), "corpse") {
            corpse = item
            break
        }
    }
    if corpse == nil {
        ch.SendMessage("You don't see a corpse here to animate.\r\n")
        return
    }
    // Consume the corpse
    world.RemoveItemFromRoom(corpse, room)
    // Spawn zombie (existing code)
    // ...
```

**Note:** Check if the manual spell path (ExecuteManualSpell) also needs this fix. The audit says AnimateDead is NOT in that switch, so the routine path may be the only one.

---

## 3. DP-483 — huntVictim never called

**File:** `pkg/game/mobact.go` in `mobileActivityForMob`
**Change:** Add after standing checks:
```go
if ch.GetPosition() == combat.PosStanding && hasMobFlag(ch, "hunter") && ch.GetFighting() == "" {
    w.huntVictim(ch)
}
```

---

## 4. DP-482 — Mob wandering has no sector constraints

**File:** `pkg/game/ai.go` in `wanderMob`, inside exit loop
**Change:** Add terrain checks before adding exit to validDirections:
```go
sector := targetRoom.SectorType
if (sector == SECT_WATER_SWIM || sector == SECT_WATER_NOSWIM) && !canSwim(mob) {
    continue
}
if sector == SECT_FLYING && !isFlying(mob) {
    continue
}
```

**Note:** Check how sector types are represented (constants? strings?). Check how swimming/flying is determined for mobs (race? flags?).

---

## 5. DP-479 — Mob wandering 100% per tick

**File:** `pkg/game/ai.go` at start of `wanderMob`
**Change:** Add probability gate:
```go
if rand.Intn(19) >= 6 {
    return
}
```

---

## 6. DP-480 — Missing onpulse triggers

**File:** `pkg/game/mobact.go` in `mobileActivityForMob`
**Change:** Add script triggers after scavenger block:
```go
// Sound trigger (1-in-16)
if rand.Intn(16) == 0 {
    w.MpSound(ch)
}
// onpulse triggers (check room occupants and fire Lua)
```

**Note:** Check how Lua script triggers are fired elsewhere.

---

## Build verification
```bash
go build ./... && go vet ./... && go test ./...
```
