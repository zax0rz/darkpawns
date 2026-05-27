# Brief 2: Small Targeted Fixes (15 issues)

Each fix is isolated to one function or file. All have verified C source references.

## 1. DP-484 — Cutthroat is instant kill (CRITICAL combat balance)
**File:** `pkg/game/skill_advanced.go:54-77`
**Change:** Replace lines 75-77:
```go
// OLD:
damage := target.GetHP() + 1
target.TakeDamage(damage)

// NEW:
damage := ch.GetLevel() / 2
target.TakeDamage(damage)
// TODO: Apply AFF_CUTTHROAT silence affect (requires affect system integration)
```
**Why:** C deals `GET_LEVEL(ch)/2` damage and applies silence. Go instant-kills. Any thief one-shots any boss.
**Note:** If applying the cutthroat affect is complex, at minimum fix the damage. The affect can be a follow-up.

## 2. DP-491 — Behead uses attacker name instead of victim
**File:** `pkg/game/skill_special.go:122-132`
**Change:** Lines 125-126 and 130-131 use `ch.Name`. Replace with victim name derived from corpse:
```go
// Extract victim name from corpse keywords (first keyword after "corpse")
victimName := extractVictimName(corpse)
headObj.Runtime.ShortDesc = fmt.Sprintf("the severed head of %s", victimName)
headObj.Runtime.Name = fmt.Sprintf("head %s", victimName)
headlessCorpseObj.Runtime.ShortDesc = fmt.Sprintf("the headless corpse of %s", victimName)
headlessCorpseObj.Runtime.Name = fmt.Sprintf("corpse headless %s", victimName)
```
The `extractVictimName` helper should parse the corpse's keywords to find the name portion (look at how C's `do_behead` gets the corpse's name).

## 3. DP-490 — DoCarve creates food with corpse VNum
**File:** `pkg/game/skill_advanced.go:31-39`
**Change:** Instead of `VNum: corpse.VNum`, map to standard food VNums:
```go
// Map corpse keywords to food VNums (matching C's do_carve)
foodVNum := 8015 // default meat
keywords := strings.ToLower(corpse.GetKeywords())
if strings.Contains(keywords, "deer") || strings.Contains(keywords, "elk") {
    foodVNum = 12
} else if strings.Contains(keywords, "wolf") || strings.Contains(keywords, "bear") {
    foodVNum = 13
} else if strings.Contains(keywords, "dragon") {
    foodVNum = 14
}
food, err := world.SpawnObject(foodVNum, ch.GetRoomVNum())
```
**Why:** C maps to real ITEM_FOOD objects. Go duplicates the corpse container.

## 4. DP-488 — DoDisarm is flavor text only
**File:** `pkg/game/skills2.go` in `DoDisarm`, after the success messages
**Change:** Add equipment unequip after successful disarm:
```go
// After success messages, unequip the target's weapon
if targetObj, ok := target.(interface{ GetEquipment() interface{} }); ok {
    // Find wielded weapon and move to inventory
    // Look at how C does: obj_to_char(unequip_char(vict, WEAR_WIELD), vict)
    // Adapt to Go's equipment system
}
```
**Note:** Check how the Go equipment system handles unequipping. Look for `Unequip` or `RemoveEquipment` methods on the target type.

## 5. DP-487 — Beholder anti-magic bypassed
**File:** `pkg/game/spec_procs_missing.go` in `specBeholder`
**Change:** Before the `if cmd != "" { return false }` line, add:
```go
// Block spellcasting in beholder's presence (matches C's command intercept)
if cmd == "cast" || cmd == "recite" {
    ch.SendMessage("You feel your magic dissipate in the beholder's presence!\r\n")
    return true
}
```
**Why:** C intercepts cast/recite commands. Go returns false for all commands.

## 6. DP-480 — Missing onpulse_all/onpulse_pc/sound triggers
**File:** `pkg/game/mobact.go` in `mobileActivityForMob`, after the scavenger block
**Change:** Add script pulse triggers:
```go
// Sound trigger (C: 1-in-16 chance)
if rand.Intn(16) == 0 {
    w.MpSound(ch)
}

// onpulse triggers (check if room has occupants)
// Find how to check for players in the mob's room
// If room has human players: ch.RunScript("onpulse_pc")
// If room has any occupants: ch.RunScript("onpulse_all")
```
**Note:** Check how Lua script triggers are invoked elsewhere in the codebase. Look for `RunScript` or `ExecuteScript` calls.

## 7. DP-482 — Mob wandering has no sector constraints
**File:** `pkg/game/ai.go` in `wanderMob`, inside the exit loop (around line 118-129)
**Change:** After checking MOB_STAY_ZONE, add terrain checks:
```go
// Check sector constraints (C: mobact.c:130-138)
sector := targetRoom.SectorType // or however sector is accessed
if (sector == "water_swim" || sector == "water_noswim") && !mobCanSwim(mob) {
    continue
}
if sector == "flying" && !mobIsFlying(mob) {
    continue
}
```
**Note:** Check how sector types are represented in Go (string constants? int constants?). Check if mobs have swim/fly flags or if you need to check their race/flags.

## 8. DP-483 — huntVictim never called
**File:** `pkg/game/mobact.go` in `mobileActivityForMob`, after the standing check
**Change:** Add hunter call:
```go
// Hunter mobs chase their targets (C: mobact.c)
if ch.GetPosition() == combat.PosStanding && hasMobFlag(ch, "hunter") && ch.GetFighting() == "" {
    w.huntVictim(ch)
}
```
**Why:** `huntVictim` is fully implemented in graph.go but never called.

## 9. DP-493 — OnHuntItems callback never wired
**File:** Wherever the game loop callbacks are initialized (search for `cb.OnHuntItems` or `Callbacks{` in cmd/server/ or pkg/engine/)
**Change:** Wire the callback:
```go
cb.OnHuntItems = func() { world.HuntItems() }
```
**Note:** You may need to implement `HuntItems()` on the World type if it doesn't exist yet. Check what `hunt_items` does in `src/new_cmds2.c:765` and port the logic.

## 10. DP-495 — Poof messages not persistent
**File:** `pkg/session/wiz_info.go:264-275`
**Change:** Instead of `s.SetTempData("poofin", msg)`, persist to player save:
```go
// Save to player record instead of session temp data
player := s.GetPlayer()
if player != nil {
    player.SetPoofIn(msg)  // or however custom fields are stored
    // Also save default poofs if empty
}
```
**Note:** Check how the player save system handles custom fields. You may need to add PoofIn/PoofOut fields to the player save data structure.

## 11. DP-492 — DoFirstAid only works on players
**File:** `pkg/game/skills2.go:123` in `DoFirstAid`
**Change:** The `target.(*Player)` type assertion fails for mobs. Add mob handling:
```go
if p, ok := target.(*Player); ok {
    // existing player logic
} else if mob, ok := target.(*MobInstance); ok {
    // Same first aid logic for mobs — revive to 1 HP
    if mob.GetHP() > 0 {
        return SkillResult{Success: false, MessageToCh: "They don't really need first aid.\r\n"}
    }
    mob.SetHP(1)
    // Send messages
}
```

## 12. DP-476 — Mail system has no global mutex
**File:** `pkg/game/mail.go` — add a global mail mutex
**Change:** Add a package-level mutex and lock it around `storeMail` and `readDelete`:
```go
var mailGlobalMu sync.Mutex

func storeMail(...) {
    mailGlobalMu.Lock()
    defer mailGlobalMu.Unlock()
    // existing code
}

func readDelete(...) {
    mailGlobalMu.Lock()
    defer mailGlobalMu.Unlock()
    // existing code
}
```
**Why:** `mailIndex`, `freeList`, `fileEndPos` are globals accessed concurrently with no protection.

## 13. DP-462 — Red portals never decay
**File:** Find where portal objects are spawned (search for portal creation or the moongate code)
**Change:** When creating red portal objects, set a decay timer:
```go
// After spawning portal object, set timer
portal.SetTimer(ObjectTimer, 2) // 2 ticks like C
```
**Why:** C gives portal objects a 2-tick lifespan. Go portals persist forever, creating infinite gate objects.

## 14. DP-473 — Commands dropped during cooldown
**File:** Find the command cooldown/cooldown system
**Change:** The cooldown check is eating the command even for non-combat commands. Add an exception:
```go
if onCooldown && !isNonCombatCommand(cmd) {
    ch.SendMessage("You're still recovering!\r\n")
    return
}
```
**Note:** Search for where commands are rejected during cooldown. The issue is that non-combat commands (look, inventory, say) are being blocked by combat cooldowns.

## 15. DP-460 — specMoonGate hardcodes MortalStartRoom
**File:** Find `specMoonGate` in the spec_procs code
**Change:** Replace hardcoded room vnum with gate_phases table lookup:
```go
// OLD: teleports to hardcoded MortalStartRoom
// NEW: look up destination from gate_phases table based on current moon phase
phase := getCurrentMoonPhase()
destRoom := gatePhases[phase]
```
**Note:** Check how moon phases are tracked in Go. The gate_phases table maps phase -> destination room vnum.

## Build verification
After all changes:
```bash
go build ./... && go vet ./... && go test ./...
```
