# Brief: Batch 2 — Fidelity Fixes (DP-669, DP-924, DP-923)

**Workspace:** `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.

---

## Fix 1: DP-669 — Duration-0 affects treated as permanent (C/Go contract mismatch)

**File:** `pkg/game/affect_update.go:53-68`
**C source:** `src/magic.c:441-450` (affect_update)

### C behavior (ground truth)

```c
if (af->duration >= 1)
    af->duration--;
else if (af->duration == -1)    /* No action */
    af->duration = -1;    /* GODs only! unlimited */
else {
    // duration == 0 OR any unexpected value → EXPIRES
    if ((af->type > 0) && (af->type <= MAX_SPELLS))
        if (!af->next || (af->next->type != af->type) ||
        (af->next->duration > 0))
            if (*spell_wear_off_msg[af->type])
                send_to_char(spell_wear_off_msg[af->type], i);
    affect_remove(i, af);
}
```

- `duration >= 1` → decrement (active spell, ticking down)
- `duration == -1` → permanent (immortal only, never expires)
- `duration == 0` → **expires this tick** (wear-off message + remove)

### Go behavior (current — WRONG)

```go
if af.Duration <= 0 {
    // Permanent affect: 0 is the engine.NewAffect contract,
    // -1 is a legacy implementor-level sentinel (CircleMUD items).
    remaining = append(remaining, af)
    continue
}
```

Go treats `duration == 0` as permanent. C treats it as "expires on next tick." This means any spell cast with duration 0 (which C treats as a 1-tick spell) lasts forever in Go.

### Fix

Replace the `af.Duration <= 0` early-return with C-faithful logic:

```go
for _, af := range affects {
    if af.Duration == -1 {
        // Permanent affect (immortal only). Matches C: duration == -1 → no action.
        remaining = append(remaining, af)
        continue
    }
    if af.Duration >= 1 {
        // Active spell — decrement. Matches C: duration >= 1 → duration--.
        af.Duration--
        if af.Duration > 0 {
            remaining = append(remaining, af)
        } else {
            // Duration just hit 0 — expires this tick.
            if msg := SpellWearOffMsg(af.SpellID); msg != "" {
                p.SendMessage(msg + "\r\n")
            }
            engine.AffectFromChar(p, af.SpellID)
        }
        continue
    }
    // Duration == 0 (or unexpected negative) — expires immediately.
    // Matches C: else branch → affect_remove.
    if msg := SpellWearOffMsg(af.SpellID); msg != "" {
        p.SendMessage(msg + "\r\n")
    }
    engine.AffectFromChar(p, af.SpellID)
}
```

### Test

```go
func TestAffectUpdateDurationZero(t *testing.T) {
    // Create player with a duration-0 affect
    // Run AffectUpdate
    // Assert affect was removed and wear-off message sent
}

func TestAffectUpdateDurationOne(t *testing.T) {
    // Create player with duration-1 affect
    // Run AffectUpdate
    // Assert duration decremented to 0 (still present)
    // Run AffectUpdate again
    // Assert affect removed
}

func TestAffectUpdateDurationNegativeOne(t *testing.T) {
    // Create player with duration-1 (permanent)
    // Run AffectUpdate
    // Assert affect still present with duration -1
}
```

---

## Fix 2: DP-924 — Movement panics on malformed room sector values

**File:** `pkg/game/act_movement.go:252`
**C source:** `src/act.movement.c:135-136`

### C behavior

```c
need_movement = (movement_loss[SECT(ch->in_room)] +
         movement_loss[SECT(EXIT(ch, dir)->to_room)]) >> 1;
```

C arrays with out-of-bounds access produce undefined behavior (silently reads garbage). The game "works" with bad sector values — it just uses wrong movement costs. No crash.

### Go behavior (current — PANICS)

```go
needMovement := (movementLoss[room.Sector] + movementLoss[toRoom.Sector]) >> 1
```

`movementLoss` has 16 entries (indices 0-15). If `room.Sector` or `toRoom.Sector` is < 0 or >= 16, Go panics with index-out-of-range. A malformed world file crashes the server on the first movement command.

### Fix

Add bounds check before indexing:

```go
// Validate sector types before indexing movementLoss (DP-924).
// C uses SECT() macro with no bounds check (UB on bad values);
// Go panics on out-of-range, so we guard explicitly.
if room.Sector < 0 || room.Sector >= len(movementLoss) ||
    toRoom.Sector < 0 || toRoom.Sector >= len(movementLoss) {
    sendToChar(ch, "You can't go that way.\r\n")
    return false
}

needMovement := (movementLoss[room.Sector] + movementLoss[toRoom.Sector]) >> 1
```

### Test

```go
func TestMovementBadSector(t *testing.T) {
    // Create world with a room having Sector = 99
    // Attempt to move into it
    // Assert no panic, movement blocked with error message
}
```

---

## Fix 3: DP-923 — Combat engine retains stale shopkeeper combat pairs

**File:** `pkg/combat/engine.go:315-318`
**C source:** `src/fight.c:1359-1366`

### C behavior

```c
if (!ok_damage_shopkeeper(ch, victim) || is_shopkeeper(victim))
{
    stc("Ha ha... Don't think so.\r\n", ch);
    if (FIGHTING(ch))
        stop_fighting(ch);      // stop attacker
    if (FIGHTING(victim))
        stop_fighting(victim);  // stop shopkeeper too
    return FALSE;
}
```

Both sides of the fight are stopped. The shopkeeper returns to idle.

### Go behavior (current — INCOMPLETE)

```go
if IsShopkeeper != nil && IsShopkeeper(defender.GetName()) {
    ce.StopCombat(attacker.GetName())
    return
}
```

Only the attacker's combat is stopped. The shopkeeper's `FIGHTING` pointer and `CombatPair` remain — `IsFighting` continues reporting combat, and the engine keeps processing the stale pair.

### Fix

Stop both sides, matching C:

```go
if IsShopkeeper != nil && IsShopkeeper(defender.GetName()) {
    ce.StopCombat(attacker.GetName())
    ce.StopCombat(defender.GetName())
    return
}
```

### Test

```go
func TestShopkeeperCombatClearsBothSides(t *testing.T) {
    // Set up attacker fighting a shopkeeper
    // Process one combat round
    // Assert shopkeeper's combat pair is cleared
    // Assert attacker's combat pair is cleared
    // Assert neither IsFighting
}
```

---

## Linear Updates (after merge)

For each fix, add a comment with the commit hash and move to Done:
- DP-669: "Fixed — duration-0 affects now expire per C magic.c:441-450, commit <hash>"
- DP-924: "Fixed — sector bounds check before movementLoss indexing, commit <hash>"
- DP-923: "Fixed — StopCombat called for both attacker and shopkeeper, commit <hash>"
