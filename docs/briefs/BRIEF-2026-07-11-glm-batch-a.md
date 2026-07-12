# Brief: GLM Batch A — 2026-07-11

**Workspace:** `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.

---

## Fix 1: DP-1043 — Weapon damage message tiers re-bucketed vs C (LOW)

**File:** `pkg/combat/fight_core.go` — `damMessageTiers` slice (line ~615)

**Problem:**
The Go damage message tiers have different boundaries and different texts than the C source. Go uses 14 tiers with boundaries 0/1/3/5/7/11/18/26/36/48/60/80/101/10000. The C source uses 12 tiers with boundaries 0/2/4/6/10/14/19/23/33/43/53+. The Go messages are reworded variants (labeled "CRIT-010") that don't match the C originals. For example, C has "massacres... to small fragments" at 20-23 damage, but Go puts that text at tier 7 starting at damage 26. C has "OBLITERATES" at 24-33, Go at 36-47. C has "ROCKS THE HELL OUT OF" at >53, Go doesn't have this text at all (its highest tier uses a fabricated 10000 boundary).

**Fix:**
Replace the `damMessageTiers` slice with the exact C tier boundaries and primary message texts. The C source is `src/fight.c:895-992`. The 12 tiers with their exact boundary values and canonical messages are:

| Tier | C Boundary | C Room Message | C Char Message | C Victim Message |
|------|-----------|----------------|----------------|------------------|
| 0 | miss (0) | "$n tries to #w $N, but misses." | "You try to #w $N, but miss." | "$n tries to #w you, but misses." |
| 1 | 0-1 | "$n misses $N." | "You miss $N." | "$n misses you." |
| 2 | 2-3 | "$n scratches $N." | "You scratch $N." | "$n scratches you." |
| 3 | 4-5 | "$n barely #W $N." | "You barely #w $N." | "$n barely #W you." |
| 4 | 6-9 | "$n #W $N." | "You #w $N." | "$n #W you." |
| 5 | 10-13 | "$n #W $N hard." | "You #w $N hard." | "$n #W you hard." |
| 6 | 14-18 | "$n #W $N very hard." | "You #w $N very hard." | "$n #W you very hard." |
| 7 | 19-22 | "$n #W $N extremely hard." | "You #w $N extremely hard." | "$n #W you extremely hard." |
| 8 | 23-32 | "$n massacres $N to small fragments with $s #w!" | "You massacre $N to small fragments with your #w!" | "$n massacres you to small fragments with $s #w!" |
| 9 | 33-42 | "$n OBLITERATES $N with $s deadly #w!!" | "You OBLITERATE $N with your deadly #w!!" | "$n OBLITERATES you with $s deadly #w!!" |
| 10 | 43-52 | "$n EVISCERATES $N with $s incredible #w!!" | "You EVISCERATE $N with your incredible #w!!" | "$n EVISCERATES you with $s incredible #w!!" |
| 11 | 53+ | "$n ROCKS THE HELL OUT OF $N with $s #w!!" | "You ROCK THE HELL OUT OF $N with your #w!!" | "$n ROCKS THE HELL OUT OF you with $s #w!!" |

**Verified C source (fight.c:895-992):**
| Tier | Boundary | C Room | C Char | C Victim |
|------|----------|--------|--------|----------|
| 0 | dam==0 | misses | miss | misses |
| 1 | 1-2 | scratches...as $e #W $M | scratch...as you #w $M | scratches...as $e #W you |
| 2 | 3-4 | barely #W | barely #w | barely #W |
| 3 | 5-6 | #W $N | #w $N | #W you |
| 4 | 7-10 | #W $N hard | #w $N hard | #W you hard |
| 5 | 11-14 | #W $N very hard | #w $N very hard | #W you very hard |
| 6 | 15-19 | #W $N extremely hard | #w $N extremely hard | #W you extremely hard |
| 7 | 20-23 | massacres...small fragments | massacre...small fragments | massacres...small fragments |
| 8 | 24-33 | OBLITERATES...deadly #w!! | OBLITERATE...deadly #w!! | OBLITERATES...deadly #w!! |
| 9 | 34-43 | EVISCERATES...incredible #w!! | EVISCERATE...incredible #w!! | EVISCERATES...incredible #w!! |
| 10 | 44-53 | DESTROYS...ungodly #w!! | DESTROY...ungodly #w!! | DESTROYS...ungodly #w!! |
| 11 | 54+ | ROCKS THE HELL OUT OF...ultimate #w!! | ROCK THE HELL OUT OF...ultimate #w!! | ROCKS THE HELL OUT OF...ultimate #w!! |

**Important notes:**
- The `#w` and `#W` placeholders are weapon name placeholders — preserve them as-is
- The C if/else chain determines tier: `if (dam<=2) msgnum=1; else if (dam<=4) msgnum=2; ...` — this is a standard chained if/else, no overlap
- The Go `DamMessage` function (line ~851) iterates tiers from highest to lowest — keep this logic, just fix the tier table
- Keep the existing Go multiple-variant approach (random flavor text per hit), but the FIRST variant in each tier MUST match the C original text. Additional variants are fine as extras
- C text uses `destroys`/`destroy` (lowercase) — Go currently has it capitalized. Match C exactly

**Cite:** `src/fight.c:895-992` — dam_weapons[] table and dam assignment chain

**Regression Test:**
No unit test needed — this is a data table fix. The existing DamMessage tests (if any) should still pass with the corrected boundaries. Verify manually: 0 damage = miss, 2 = scratch, 6 = light hit, 23 = massacres, 53 = ROCKS.

---

## Fix 2: DP-1040 — counter_procs milestone rewards don't reproduce C fallthrough (LOW)

**File:** `pkg/game/death.go` — `counter_procs()` method (line ~966)

**Problem:**
The C `counter_procs()` at `src/fight.c:1280-1290` uses `switch(number(1,3))` with NO breaks between cases. This means:
- Roll 1: +2 max_hit, +1 mana, +1 move (case 1 falls through to case 2)
- Roll 2: +1 hit, +1 mana, +1 move (case 2 falls through to case 3)
- Roll 3: +1 move, +1 hit (case 3 — no mana)

Wait — re-reading more carefully. The C source at fight.c:1280-1290:
```c
switch (number(1, 3)) {
  case 1:
    GET_MAX_HIT(ch) += 2;
    /* fall through */
  case 2:
    GET_MAX_MANA(ch) += 1;
    /* fall through */
  case 3:
    GET_MAX_MOVE(ch) += 1;
    break;
}
```

So the actual fall-through behavior is:
- Roll 1: +2 max_hit, +1 mana, +1 move (falls through 1→2→3)
- Roll 2: +1 mana, +1 move (falls through 2→3)
- Roll 3: +1 move only

**Go current (line ~971-975):** Always gives +1 max_hit, +1 mana, +1 move — matches roll 2 exactly, but misses the variance.

**Fix:** The C switch at fight.c:1280-1290 has NO breaks — cases fall through:
```c
switch (number(1, 3)) {
  case 1:
    GET_MAX_HIT(ch) += 2;   // falls through
  case 2:
    GET_MAX_MANA(ch) += 1;  // falls through
  case 3:
    GET_MAX_MOVE(ch) += 1;  // break
}
```

Actual outcomes per roll:
- Roll 1: +2 max_hit, +1 mana, +1 move (falls through all three)
- Roll 2: +1 mana, +1 move (skips case 1, falls through 2→3)
- Roll 3: +1 move only

Replace the Go code at death.go:971-975 with:
```go
roll := GetRoller().IntN(3)
ch.Lock()
defer ch.Unlock()
ch.MaxMove++
if roll <= 1 { // cases 1 and 2: +1 mana
    ch.MaxMana++
}
if roll == 0 { // case 1 only: +2 hit
    ch.MaxHealth += 2
}
```

**Cite:** `src/fight.c:1280-1290` — counter_procs() switch with fall-through

**Regression Test:**
Add a test in `pkg/game/death_test.go` that calls `counter_procs` at a milestone kill count (e.g., 1000) and verifies that max_hit, max_move, and max_man a all increased by at least 1. The test can't assert exact values due to the random roll, but can verify the stats increased and the blessing message was sent.

---

## Fix 3: DP-1038 — Carry weight (CAN_CARRY_W) not enforced (MED)

**File:** `pkg/game/inventory.go` — `addItem()` (line ~26) and `SetCapacity()` (line ~145)

**Problem:**
`SetCapacity` only sets `inv.Capacity` (item count limit via CAN_CARRY_N). The `addItem` check is `len(inv.Items) >= inv.Capacity` — count only. The `GetWeight()` method exists (line ~130) but nothing checks weight on pickup. The C source at `utils.h:448` defines `CAN_CARRY_W(ch) = str_app[STRENGTH_APPLY_INDEX(ch)].carry_w` and `utils.h:543-545` requires both weight and count headroom for CAN_GET_OBJ.

**Fix:**
1. Add a `MaxWeight` field to `Inventory` struct
2. In `SetCapacity`, also set `MaxWeight` using the `strApp` table from `pkg/combat/formulas.go`:
   - The `strApp` type already has a `CarryW` field (line ~150: `Fields: {tohit, todam, carry_w, carry_n}`)
   - Need a public accessor: `func GetCarryWeight(strIndex int) int` in formulas.go (or use the existing pattern)
   - `inv.MaxWeight = strApp[strIndex].CarryW`
3. In `addItem`, add a weight check before the count check:
   ```go
   func (inv *Inventory) addItem(item *ObjectInstance) error {
       if inv.MaxWeight > 0 && inv.GetWeight()+item.GetTotalWeight() > inv.MaxWeight {
           return ErrInventoryTooHeavy // new error
       }
       if len(inv.Items) >= inv.Capacity {
           return ErrInventoryFull
       }
       inv.Items = append(inv.Items, item)
       return nil
   }
   ```
4. Add `ErrInventoryTooHeavy` to the error variables
5. Callers that handle `addItem` errors should send a player-facing message like "You can't carry that much weight." — check existing error handling for `ErrInventoryFull` and follow the same pattern

**Important:**
- `SetCapacity` is called from `pkg/game/player_identity.go` at character creation and login. It currently takes `(dex, level)`. It needs a third parameter `str int` to look up carry_w from strApp.
- The `strApp` table is in `pkg/combat/formulas.go`. You'll need to either export `strApp` or add a getter function. Follow the pattern of `GetStrDamage` / `GetStrToHit` — add `func GetCarryWeight(strIndex int) int` that returns `strApp[idx].CarryW`.
- Need a `strToIndex` or similar to map raw STR to strApp index (same as `strIndex` in formulas.go).

**Cite:** `utils.h:448` — CAN_CARRY_W definition, `utils.h:543-545` — CAN_GET_OBJ weight check

**Regression Test:**
Add a test in `pkg/game/inventory_test.go`:
- Create an Inventory with a known MaxWeight
- Add items until weight exceeds MaxWeight
- Assert `addItem` returns `ErrInventoryTooHeavy`
- Verify count-based full still works independently

---

## Execution Order

1. **Fix 1 (DP-1043)** — damage tiers. Zero risk, data table only.
2. **Fix 2 (DP-1040)** — counter_procs. Small, isolated to death.go.
3. **Fix 3 (DP-1038)** — carry weight. Crosses inventory.go + formulas.go, but well-scoped.

All three are independent — no file overlap. Can be implemented in any order.

## After All Fixes

```bash
cd /Users/zach/.openclaw/workspace-daeron/darkpawns_repo
git add -A
git commit -m "fix: restore C damage tier boundaries, counter_procs fallthrough, carry weight enforcement (DP-1043, DP-1040, DP-1038)"
git push -u origin fix/glm-batch-a
```

Then wait for Daeron to review and merge. Do NOT merge the PR yourself.

## Linear Updates (after merge)

- DP-1043: Add comment "Fixed — damage tiers restored to C boundaries (fight_core.go)", commit hash, move to Done
- DP-1040: Add comment "Fixed — counter_procs now rolls 1-in-3 with C fallthrough (death.go)", commit hash, move to Done
- DP-1038: Add comment "Fixed — carry weight enforced via str_app table (inventory.go, formulas.go)", commit hash, move to Done
