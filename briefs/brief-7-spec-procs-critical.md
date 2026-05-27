# Brief 7: Spec Procs — Critical + High

**Issues:** DP-513, DP-511, DP-510
**Priority:** CRITICAL + HIGH
**Files:** `pkg/game/spec_procs.go`
**C Source:** `src/spec_procs.c`

---

## Problem

Three mob special procedures in `spec_procs.go` have fidelity bugs that range from character-wiping (CRITICAL) to rendering entire city defense systems non-functional (MEDIUM). Each is an independent fix in the same file.

---

## Issues in This Brief

### DP-513 — CRITICAL: specFido deletes player gear (URGENT)

**Go:** `pkg/game/spec_procs.go:483-494`
```go
func specFido(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
    // ...
    for _, obj := range items {
        if strings.Contains(obj.GetKeywords(), "corpse") {
            w.roomMessage(me.RoomVNum, me.GetName()+" savagely devours "+obj.GetShortDesc()+".")
            if err := w.MoveObjectToNowhere(obj); err != nil { ... }
            return true
        }
    }
```

**C:** `src/spec_procs.c:724-746`
```c
SPECIAL(fido) {
    for (i = world[ch->in_room].contents; i; i = i->next_content) {
        if (GET_OBJ_TYPE(i) == ITEM_CONTAINER && GET_OBJ_VAL(i, 3)) {
            act("$n savagely devours a corpse.", FALSE, ch, 0, 0, TO_ROOM);
            for (temp = i->contains; temp; temp = next_obj) {
                next_obj = temp->next_content;
                obj_from_obj(temp);       // extract from corpse
                obj_to_room(temp, ch->in_room);  // spill to room floor
            }
            extract_obj(i);  // now extract empty corpse
            return (TRUE);
        }
    }
```

**Bug:** Go calls `MoveObjectToNowhere(obj)` on the corpse without first extracting its nested contents. In C, fido iterates `i->contains`, moves each item to the room floor with `obj_to_room`, then extracts the empty corpse.

**Impact:** Any player who dies in a room with a fido dog has their entire inventory permanently deleted — weapons, armor, quest items, cash. The dog eats the corpse and everything inside it vanishes from existence.

**Fix:** Before `MoveObjectToNowhere`, iterate the corpse's inventory and move each item to the room:
```go
if strings.Contains(obj.GetKeywords(), "corpse") {
    w.roomMessage(me.RoomVNum, me.GetName()+" savagely devours "+obj.GetShortDesc()+".")
    // Spill corpse contents to room floor (matches C src/spec_procs.c:735-741)
    for _, content := range obj.GetContents() {
        if err := w.MoveObjectToRoom(content, me.GetRoom()); err != nil {
            slog.Warn("Failed to spill corpse content", "obj_vnum", content.GetVNum(), "error", err)
        }
    }
    if err := w.MoveObjectToNowhere(obj); err != nil {
        slog.Warn("MoveObjectToNowhere failed in fido spec", "obj_vnum", obj.GetVNum(), "error", err)
    }
    return true
}
```

Check `MoveObjectToRoom` exists on World — if not, use the equivalent method that places an object in a room. The key point: contents must be extracted BEFORE the corpse is removed.

**Note on C difference:** C checks `ITEM_CONTAINER && GET_OBJ_VAL(i, 3)` (container flag = closable). Go checks `strings.Contains(keywords, "corpse")`. The Go approach is more specific (only corpses, not any container) — this is acceptable, keep the Go check.

### DP-511 — specJanitor deletes items instead of adding to inventory (HIGH)

**Go:** `pkg/game/spec_procs.go:501`
```go
w.RemoveItemFromRoom(obj, me.GetRoom())
```

**C:** `src/spec_procs.c:750-768`
```c
SPECIAL(janitor) {
    for (i = world[ch->in_room].contents; i; i = i->next_content) {
        if (!CAN_WEAR(i, ITEM_WEAR_TAKE) || (isname((i)->name, "corpse")))
            continue;
        act("$n picks up some trash.", FALSE, ch, 0, 0, TO_ROOM);
        obj_from_room(i);
        obj_to_char(i, ch);    // <-- goes INTO janitor's inventory
        return TRUE;
    }
}
```

**Bug:** Go calls `RemoveItemFromRoom` which deletes the item from existence. C calls `obj_to_char(i, ch)` — the janitor picks items up into its own inventory. Players can kill the janitor to retrieve dropped items.

**Impact:** Items dropped on the ground in janitor rooms are permanently deleted instead of being recoverable.

**Fix:** Replace `RemoveItemFromRoom` with adding to the janitor's inventory:
```go
if !strings.Contains(obj.GetKeywords(), "corpse") && randN(2) == 0 {
    w.roomMessage(me.GetRoom(), me.GetName()+" picks up "+obj.GetShortDesc()+".")
    // Move to janitor's inventory, not nowhere (matches C obj_to_char)
    if err := me.GetInventory().AddItem(obj); err != nil {
        slog.Warn("Janitor failed to pick up item", "obj_vnum", obj.GetVNum(), "error", err)
        return false
    }
    w.RemoveItemFromRoom(obj, me.GetRoom())
    return true
}
```

Check the actual inventory API — it might be `me.Inventory.AddItem(obj)` or `me.GetCharData().carrying = obj`. The pattern is: add to inventory first, then remove from room.

### DP-510 — specCityguard only blocks movement (MEDIUM)

**Go:** `pkg/game/spec_procs.go:508-521`
```go
func specCityguard(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
    if isMoveCmd(cmd) && ch.GetRoomVNum() == me.RoomVNum {
        flags := ch.GetFlags()
        if flags&1 != 0 { // PLR_OUTLAW
            sendToChar(ch, me.GetName()+" says, 'HALT!  You are under arrest!'")
            w.roomMessage(me.RoomVNum, me.GetName()+" bars "+ch.GetName()+"'s way!")
            return true
        }
    }
    if me.IsFighting() {
        return true
    }
    return false
}
```

**C:** `src/spec_procs.c:771-821`
The C cityguard does FOUR things:
1. If currently fighting → delegate to `fighter()` skill routine (bash, parry, headbutt)
2. Scan room for outlaws → attack them with `hit()`
3. Call `breed_killer()` → attack aggressive breed mobs
4. Scan room for combatants → attack the most evil one if they're fighting a good-aligned target

**Bug:** Go only blocks outlaw movement. Never attacks. Never protects. Towns are completely undefended.

**Fix:** Port the full cityguard logic. This is the most complex fix in this brief:
```go
func specCityguard(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
    if cmd != "" || !me.IsAwake() {
        return false
    }
    // If already fighting, use fighter skills (bash/parry/headbutt)
    if me.IsFighting() {
        return fighterSpec(w, ch, me, cmd, arg)
    }
    // Scan for outlaws and attack
    for _, tch := range w.GetPlayersInRoom(me.RoomVNum) {
        if !tch.IsNPC() && tch.CanBeSeenBy(me) && tch.IsOutlaw() {
            w.roomMessage(me.RoomVNum, me.GetName()+" says, 'We don't like OUTLAWS like you in this city!'")
            me.Attack(tch)
            return fighterSpec(w, ch, me, cmd, arg)
        }
    }
    // Find most evil combatant and protect good targets
    var evil *Player
    maxEvil := 1000
    for _, tch := range w.GetCharactersInRoom(me.RoomVNum) {
        if tch.CanBeSeenBy(me) && tch.IsFighting() {
            if tch.GetAlignment() < maxEvil && (tch.IsNPC() || tch.GetTarget().IsNPC()) {
                maxEvil = tch.GetAlignment()
                evil = tch
            }
        }
    }
    if evil != nil && evil.GetTarget() != nil && evil.GetTarget().GetAlignment() >= 0 {
        w.roomMessage(me.RoomVNum, me.GetName()+" says, 'You just pissed me off, "+evil.GetTarget().GetName()+"!'")
        me.Attack(evil)
        return fighterSpec(w, ch, me, cmd, arg)
    }
    return false
}
```

**Note:** This requires understanding the combat initiation API. Check how other spec_procs start combat (e.g., `specSnake`, `specSummoner`). The `fighterSpec` call delegates to the fighter skill routine — check if `specFighter` exists in Go and can be called directly, or if you need to implement the skill selection inline.

---

## Execution Order

1. **DP-513 (fido)** — CRITICAL, highest priority, relatively simple fix
2. **DP-511 (janitor)** — HIGH, simple fix (one function call change)
3. **DP-510 (cityguard)** — MEDIUM, most complex, requires combat API understanding

## Verification

After all fixes:
```bash
cd darkpawns_repo
go build ./...
go vet ./...
go test ./...
```

Manually verify:
- `grep -n "MoveObjectToNowhere" pkg/game/spec_procs.go` — fido should extract contents first
- `grep -n "RemoveItemFromRoom" pkg/game/spec_procs.go` — janitor should NOT use this (use inventory add)
- `grep -n "specCityguard" pkg/game/spec_procs.go` — should have combat logic, not just movement blocking
