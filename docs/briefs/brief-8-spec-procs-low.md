# Brief 8: Spec Procs — Low Priority

**Issues:** DP-514, DP-512
**Priority:** LOW
**Files:** `pkg/game/spec_procs.go`
**C Source:** `src/spec_procs.c`

---

## Problem

Two mob special procedures are simplified stubs. Neither affects gameplay mechanics or causes data loss — they are fidelity gaps in NPC behavior that reduce world immersion.

---

## Issues in This Brief

### DP-512 — specMayor static — missing walking routine (LOW)

**Go:** `pkg/game/spec_procs.go:524-543`
```go
func specMayor(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
    if cmd != "" || randN(4) != 0 {
        return false
    }
    mayorSayings := []string{
        "Hello, mate!", "Nice to see you!", "How d'you do?",
        "Another fine day!", "Welcome to New Thalos!", "Good day!",
        "Lovely to meet you!",
    }
    saying := mayorSayings[randN(len(mayorSayings))]
    w.roomMessage(me.RoomVNum, me.GetName()+" says, '"+saying+"'")
    return true
}
```

**C:** `src/spec_procs.c:823-924`

The C mayor has a complex walking schedule driven by encoded path strings:
- At hour 6 (sunrise): follows `open_path` = `"W3a3003b33000c111d0d111Oe333333Oe22c222112212111a1S."`
- At hour 20 (sunset): follows `close_path` = `"W3a3003b33000c111d0d111CE333333CE22c222112212111a1S."`
- Each character in the path is an action:
  - `0-3`: move in that direction
  - `W`: wake up ("$n awakens and groans loudly.")
  - `S`: go to sleep ("$n lies down and instantly falls asleep.")
  - `a`: "Hello Honey!" + smirk
  - `b`: "What a view! I must get something done about that dump!"
  - `c`: "Vandals! Youngsters nowadays have no respect for anything!"
  - `d`: "Good day, citizens!"
  - `e`: "I hereby declare the bazaar open!"
  - `E`: "I hereby declare Bourbon closed!"
  - `O`: unlock + open gate
  - `C`: close + lock gate
  - `.`: stop moving (path complete)

**Fix:** Port the path-walking system. This requires:
1. Storing the path strings and current index as static state on the mob (or in a map keyed by mob VNUM)
2. On each tick (when `cmd == ""`), advance one step through the path
3. Execute the action for the current character
4. At `.`, stop and wait for the next schedule trigger (hour 6 or 20)

The `do_gen_door` calls for gate unlock/open/close/lock need to be translated to Go's door manipulation API. Check how `doGenDoor` or equivalent is implemented in Go — likely in `pkg/game/` somewhere.

**Note:** This is low priority but adds significant world flavor. The mayor's daily routine (waking up, walking around New Thalos, opening the bazaar at dawn, closing it at dusk, going to sleep) is a visible piece of living world behavior.

### DP-514 — specCuchi Easter egg stubbed (LOW)

**Go:** `pkg/game/spec_procs.go:587-606`
```go
func specCuchi(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
    if cmd != "" || randN(4) != 0 {
        return false
    }
    cuchiSayings := []string{...}
    saying := cuchiSayings[randN(len(cuchiSayings))]
    w.roomMessage(me.RoomVNum, me.GetName()+" says, '"+saying+"'")
    return true
}
```

**C:** `src/spec_procs.c:1034-1071`
```c
SPECIAL(cuchi) {
    if (!CMD_IS("pat"))
        return FALSE;
    if (!strcmp(GET_NAME(ch), "Orodreth")) {
        // Orodreth pats Cuchi → promoted to Implementor level
        GET_LEVEL(ch) = LVL_IMPL;
        stc("Cuchi purrs at you contently.\r\n", ch);
    } else {
        // Anyone else pats Cuchi → gets 10 gold
        GET_GOLD(ch) += 10;
        stc("Cuchi purrs at you and bestows a gift from the gods.\r\n", ch);
    }
    return TRUE;
}
```

**Bug:** Go ignores the "pat" command entirely. C has a special Easter egg: patting Cuchi awards 10 gold, and patting Cuchi as "Orodreth" promotes you to Implementor level.

**Fix:** Add "pat" command detection:
```go
func specCuchi(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
    if cmd != "pat" {
        return false
    }
    // Pat response (matches C src/spec_procs.c:1034-1071)
    w.roomMessage(me.RoomVNum, ch.GetName()+" pats "+me.GetName()+" on the head and rubs around her ears.")
    sendToChar(ch, "You pat "+me.GetName()+" on the head and rub around her ears.\r\n")

    if ch.GetName() == "Orodreth" {
        ch.SetLevel(lvlImpl)  // Check the actual implementor level constant
        sendToChar(ch, "Cuchi purrs at you contently.\r\n")
        w.roomMessage(me.RoomVNum, me.GetName()+" purrs contently at "+ch.GetName()+".")
    } else {
        ch.Gold += 10
        sendToChar(ch, "Cuchi purrs at you and bestows a gift from the gods.\r\n")
        w.roomMessage(me.RoomVNum, me.GetName()+" purrs at "+ch.GetName()+" and bestows a gift from the gods.")
    }
    return true
}
```

Check the actual implementor level constant — it's likely `LVL_IMPL` or a Go equivalent like `consts.LvlImpl` or similar. Also check `sendToChar` vs `ch.SendMessage` — use whatever the codebase uses for private player messages.

**Note:** This is pure flavor. Orodreth is one of the original Dark Pawns implementors. The Easter egg is a tribute to the people who built this world. It should work.

---

## Execution Order

1. **DP-514 (cuchi)** — simpler, pure command detection
2. **DP-512 (mayor)** — complex path system, needs door API investigation

## Verification

After all fixes:
```bash
cd darkpawns_repo
go build ./...
go vet ./...
go test ./...
```
