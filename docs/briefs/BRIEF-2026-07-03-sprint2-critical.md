# Brief: Sprint 2 CRITICAL — Combat Reciprocity + Skill Damage Pipeline — 2026-07-03

**Workspace:** `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.
**Milestone:** Fable Review (2026-07-03)

---

## Fix 1: DP-900 — Combat is one-sided — defenders never swing (CRITICAL)

**File:** `pkg/combat/engine.go` — `StartCombat()` (line ~144), `PerformRound()` (line ~223)

**Problem:**
`StartCombat` creates exactly one `CombatPair{Attacker, Defender}`. The defender never gets a reciprocal pair. `PerformRound` iterates pairs and calls `processCombatPair` which only processes `pair.Attacker`'s attacks. Defenders never swing.

C's `perform_violence()` (`src/fight.c:1398`) iterates the combat *list* — every character with `FIGHTING` set attacks. In Go, only the attacker in each pair attacks.

Live repro: Player attacks a mob → 22+ rounds of "You miss!" → mob never retaliates → player HP never moves.

**Fix:**
Restructure `PerformRound` to iterate all combatants with a fighting target, not just the attacker side of pairs. This is Option B from the Linear issue — closer to C semantics and handles retaliatory retarget for free.

In `pkg/combat/engine.go`, replace the `PerformRound` method:

```go
// PerformRound executes one round of combat for all active fighters.
// Source: fight.c perform_violence() — iterates ALL combatants with FIGHTING set.
func (ce *CombatEngine) PerformRound() {
    ce.mu.RLock()

    // Collect unique combatants from all pairs.
    // Both attacker and defender may need to swing — C's perform_violence
    // iterates the full combat list, not pairs.
    type combatEntry struct {
        attacker Combatant
        defender Combatant
    }
    seen := map[string]bool{}
    var entries []combatEntry

    for _, pair := range ce.combatPairs {
        // Attacker → Defender
        aName := pair.Attacker.GetName()
        if !seen[aName] {
            target := pair.Attacker.GetFighting()
            if target != "" {
                // Find the defender Combatant by name
                if d, ok := ce.getCombatantByName(target); ok {
                    entries = append(entries, combatEntry{attacker: pair.Attacker, defender: d})
                    seen[aName] = true
                }
            }
        }
        // Defender → Attacker (reciprocal)
        dName := pair.Defender.GetName()
        if !seen[dName] {
            target := pair.Defender.GetFighting()
            if target != "" {
                if a, ok := ce.getCombatantByName(target); ok {
                    entries = append(entries, combatEntry{attacker: pair.Defender, defender: a})
                    seen[dName] = true
                }
            }
        }
    }

    ce.mu.RUnlock()

    // Process each combatant's attack
    for _, entry := range entries {
        pair := &CombatPair{
            Attacker: entry.attacker,
            Defender: entry.defender,
        }
        ce.processCombatPair(pair)
    }

    // C-10: decrement wait states each round
    if ce.OnRoundEnd != nil {
        ce.OnRoundEnd()
    }
}
```

You'll also need to add a helper method to find a combatant by name:

```go
// getCombatantByName finds a Player or MobInstance by name from the world.
// Must be called with ce.mu held (at least RLock).
func (ce *CombatEngine) getCombatantByName(name string) (Combatant, bool) {
    // Check all pairs for this name
    for _, pair := range ce.combatPairs {
        if pair.Attacker.GetName() == name {
            return pair.Attacker, true
        }
        if pair.Defender.GetName() == name {
            return pair.Defender, true
        }
    }
    return nil, false
}
```

**Important:** The CombatEngine has callbacks like `GetCombatTarget`, `IsFighting`, etc. These already handle bidirectional lookup (checking both attacker and defender sides of pairs). They don't need changes.

**Cite:** C source — `fight.c:1398` (`perform_violence`). C iterates a flat combat list; Go has pairs. The fix bridges the gap by iterating both sides of each pair.

**Regression Test:**
```go
func TestCombatReciprocity(t *testing.T) {
    ce := NewCombatEngine()
    // Create two mock combatants
    // StartCombat(a, b)
    // Verify both a and b have fighting targets set
    // PerformRound
    // Verify both sides dealt damage (HP decreased on both)
}

func TestCombatBothSidesDealDamage(t *testing.T) {
    // Start combat, run 10 rounds, verify both lost HP
}
```

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Fix 2: DP-901 — Skill damage path bypasses death handling (CRITICAL)

**File:** `pkg/command/skill_commands.go` — `sendSkillResult()` (line ~1544)

**Problem:**
`sendSkillResult` on `result.Damage > 0` calls `target.TakeDamage(dam)`; if HP ≤ 0 it only prints `"%s is dead!"` — no corpse, no XP, no removal from world, no `handleMobDeath`/`rawKill`. And no `StartCombat` anywhere.

The correct implementation already exists: `DoSpellDamage` in `pkg/game/damage_stubs.go` (line ~28) handles both `*Player` and `*MobInstance`, calls `rawKill`/`handleMobDeath`, and sets fighting state. But `sendSkillResult` doesn't use it.

Also: `doDamage` (line ~56) silently no-ops for mobs (`vict.(*Player)` type assertion fails). `hitSkill` (line ~67) uses flat `randRange(1,8)+2` ignoring skill and weapon. `diceRoll` (line ~89) calls `rand.IntN(d)` twice and discards the first roll.

**Fix:**

### 2a. Fix `sendSkillResult` to route through `DoSpellDamage`

In `pkg/command/skill_commands.go`, replace the damage block (~line 1544):

```go
// BEFORE (broken):
if result.Damage > 0 && target != nil {
    target.TakeDamage(result.Damage)
    if target.GetHP() <= 0 {
        _ = s.SendMessage(fmt.Sprintf("%s is dead!\r\n", target.GetName()))
    }
}

// AFTER (fixed):
if result.Damage > 0 && target != nil {
    // Route through DoSpellDamage which handles both Player and Mob death,
    // corpse creation, XP, and combat initiation.
    s.world.DoSpellDamage(s.player, target, result.Damage, "")
}
```

### 2b. Fix `doDamage` to handle mobs

In `pkg/game/damage_stubs.go`, change `doDamage` to accept `interface{}` like `DoSpellDamage`:

```go
func (w *World) doDamage(ch, vict interface{}, dam int, skill string) bool {
    if dam <= 0 {
        if p, ok := vict.(*Player); ok {
            p.SendMessage(fmt.Sprintf("%s hits you, but it doesn't hurt!\r\n", getAttackerName(ch)))
        }
        return false
    }

    attackerName := getAttackerName(ch)
    switch v := vict.(type) {
    case *Player:
        v.TakeDamage(dam)
        v.SetFighting(attackerName)
        if v.GetHP() <= 0 {
            w.rawKill(v, 303)
        }
        return true
    case *MobInstance:
        v.TakeDamage(dam)
        v.SetFighting(attackerName)
        if v.GetHP() <= 0 {
            w.handleMobDeath(v, nil, 303)
        }
        return true
    default:
        return false
    }
}
```

### 2c. Fix `hitSkill` to use weapon damage

```go
func (w *World) hitSkill(ch, vict interface{}, skill string) bool {
    dam := randRange(1, 8) + 2
    w.doDamage(ch, vict, dam, skill)
    return true
}
```

(This is a stub — leave as-is for now since real skill damage goes through DoBackstab/DoKick/etc. Just ensure it doesn't no-op on mobs.)

### 2d. Fix `diceRoll` — discard first roll

```go
// BEFORE (bug):
func diceRoll(n, d int) int {
    total := 0
    for i := 0; i < n; i++ {
        rand.IntN(d)  // discarded!
        total += rand.IntN(d) + 1
    }
    return total
}

// AFTER (fixed):
func diceRoll(n, d int) int {
    total := 0
    for i := 0; i < n; i++ {
        total += rand.IntN(d) + 1
    }
    return total
}
```

**Cite:** C source — `fight.c` `damage()` function handles death for all character types. The Go port split this into two paths (combat engine + skill stubs) and the skill path lost the death handling.

**Regression Test:**
```go
func TestSkillDamageVsMob(t *testing.T) {
    // Create a mob with low HP
    // Apply skill damage via DoSpellDamage
    // Assert mob is dead and removed from world
    // Assert corpse exists in room
}

func TestDoDamageHandlesMobs(t *testing.T) {
    // Create a mob
    // Call doDamage with dam > 0
    // Assert mob HP decreased (was silently no-oping before)
}
```

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Execution Order

1. **Fix 1 (DP-900)** first — combat reciprocity is the foundation
2. **Fix 2 (DP-901)** after — skill damage pipeline depends on combat working

## After All Fixes

```bash
cd /Users/zach/.openclaw/workspace-daeron/darkpawns_repo
git add pkg/combat/engine.go pkg/command/skill_commands.go pkg/game/damage_stubs.go
git commit -m "fix: combat reciprocity + skill damage pipeline (DP-900, DP-901)"
git push -u origin fix/dp-900-901-critical-combat
gh pr create --title "fix: CRITICAL combat + skill damage (DP-900, DP-901)" --body "Fixes DP-900 (defenders never swing) and DP-901 (skill damage bypasses death). See docs/briefs/BRIEF-2026-07-03-sprint2-critical.md"
```

Then wait for Daeron to review and merge.

## Linear Updates (after merge)

- DP-900: Add comment "Fixed — PerformRound now iterates all combatants (both sides of pairs), defenders swing", commit hash, move to Done
- DP-901: Add comment "Fixed — sendSkillResult routes through DoSpellDamage, doDamage handles mobs, diceRoll fixed", commit hash, move to Done
