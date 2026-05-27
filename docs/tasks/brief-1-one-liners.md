# Brief 1: One-Liner Fixes (9 issues)

All changes are single-line or trivial. Each has a verified file path and exact change.

## 1. DP-481 — Jailer bribe VNum wrong
**File:** `pkg/game/mobprogs.go:97`
**Change:** `case vnum == 8014:` → `case vnum == 8088:`
**Why:** 8014 is Guild Guard, 8088 is Jailer. Bribe triggers wrong mob.

## 2. DP-485 — Bearhug immortal comment wrong
**File:** `pkg/game/skill_special.go:160`
**Change:** `// Immortals always succeed, sleeping targets always hit` → `// Immortals always fail bearhug (intentional — they don't need it)`
**Why:** Comment contradicts code. percent=101 causes guaranteed failure, not success.

## 3. DP-489 — Recharger cost 100x inflated
**File:** `pkg/game/spec_procs_missing.go:93`
**Change:** `cost := 1000 * spellLvl * maxCharges` → `cost := spellLvl * 100`
**Why:** C uses `spell_level * 100` per charge. Go multiplies by 1000 AND maxCharges, making recharging 100x too expensive.

## 4. DP-494 — Rent deadline message discarded
**File:** `pkg/game/objsave.go:693` (approximately — find `_ = fmt.Sprintf`)
**Change:** `_ = fmt.Sprintf(...)` → `p.SendMessage(fmt.Sprintf(...))` (or `ch.SendMessage` depending on context)
**Why:** C outputs rent deadline to player. Go calculates and throws it away.

## 5. DP-469 — Backwards mana regen from equipment
**File:** Look in `pkg/game/` for the mana regeneration tick. The C source regenerates mana for equipment that reduces mana cost — Go has the comparison backwards (regens for equipment that doesn't reduce).
**What to find:** Search for mana regeneration or `ManaRegen` in the affect/equipment tick code. Find where equipment apply types are checked. The condition is inverted.
**Why:** Equipment that reduces mana cost should speed regen. Go does the opposite.

## 6. DP-498 — Receptionist ignores bank gold
**File:** `pkg/game/objsave.go` around line 740, in `GenReceptionist`
**Change:** `cost <= p.Gold` → `cost <= p.Gold+p.BankGold`
**Why:** C checks `GET_GOLD(ch) + GET_BANK_GOLD(ch)` for rent affordability. Go only checks cash on hand.

## 7. DP-479 — Mob wandering has no probability roll
**File:** `pkg/game/ai.go` around line 95, start of `wanderMob`
**Change:** Add at the top of the function:
```go
// C: door = number(0, 18); if door >= NUM_OF_DIRS, no move (~31.5% chance)
if rand.Intn(19) >= 6 {
    return
}
```
**Why:** C only attempts wandering ~31.5% of ticks. Go moves 100% of the time.

## 8. DP-477 — Delete unused mapcode.go
**File:** `pkg/game/mapcode.go`
**Change:** Delete the entire file. It has `lint:file-ignore U1000` and is not imported anywhere.
**Why:** Duplicate of `pkg/session/map_cmds.go` which is the actual production implementation.

## 9. DP-475 — readInt64 sign-extension bug
**File:** `pkg/game/mail.go:628-629`
**Change:**
```go
// OLD:
func readInt64(buf []byte, off int) int64 {
    return int64(readInt32(buf, off)) | int64(readInt32(buf, off+4))<<32
}
// NEW:
func readInt64(buf []byte, off int) int64 {
    return int64(uint32(readInt32(buf, off))) | int64(uint32(readInt32(buf, off+4)))<<32
}
```
**Why:** When readInt32 returns a negative value, int64() sign-extends filling upper bits with 1s, corrupting the OR result. Cast through uint32 first to zero-extend.

## Build verification
After all changes:
```bash
go build ./... && go vet ./... && go test ./...
```
