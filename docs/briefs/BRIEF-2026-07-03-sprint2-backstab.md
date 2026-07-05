# Brief: Sprint 2 HIGH — Backstab C Gates — 2026-07-03

**Workspace:** `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.
**Milestone:** Fable Review (2026-07-03)
**Depends on:** DP-900 (combat reciprocity) and DP-901 (skill damage pipeline) should land first.

---

## Fix: DP-906 — Backstab missing C gates (HIGH)

**File:** `pkg/game/skill_combat.go` — `DoBackstab()` (line ~11)

**Problem:**
The Go `DoBackstab` is missing gates that C's `do_backstab` (`src/act.offensive.c:165`) enforces. Live path vs C:

| Gate | C (`act.offensive.c:165`) | Go (`skill_combat.go:11`) | Status |
|------|--------------------------|--------------------------|--------|
| Skill check | `GET_SKILL(ch, SKILL_BACKSTAB)` | ✅ `ch.GetSkill(SkillBackstab) == 0` | OK |
| Target in room | `get_char_room_vis(ch, buf)` | Caller resolves target | OK (caller's job) |
| Self-check | `vict == ch` | Missing | **ADD** |
| Must wield weapon | `GET_EQ(ch, WEAR_WIELD)` | ✅ checks `GetWeaponDamage()` | OK (but see note) |
| **Piercing weapon** | `GET_OBJ_VAL(GET_EQ(ch, WEAR_WIELD), 3) != TYPE_PIERCE - TYPE_HIT` | **MISSING** — only checks dice > 0 | **ADD** |
| Mounted check | `IS_MOUNTED(ch)` | Missing | **ADD** |
| Target fighting | `FIGHTING(vict)` | ✅ `target.GetFighting() != ""` | OK |
| **MOB_AWARE counter** | `MOB_FLAGGED(vict, MOB_AWARE) && AWAKE(vict)` → `hit(vict, ch)` | **MISSING** | **ADD** (blocked on DP-898 until flag table fixed, but add the gate now) |
| **Miss starts combat** | `damage(ch, vict, 0, SKILL_BACKSTAB)` on miss | **MISSING** — returns SkillResult with no damage | **ADD** |
| **str-to-damage bonus** | C: `dam = str_app[...].todam + GET_DAMROLL(ch) + weapon_dice` | Only `weaponDam + ch.GetDamroll()` | **ADD** |

**Note on weapon check:** The current code checks `GetWeaponDamage()` returns dice > 0. A fresh thief backstabbed bare-handed in the playtest. Either creation grants a starting weapon or `GetWeaponDamage()` returns nonzero defaults. The piercing-weapon check is the real gate — it should use `weapon.Prototype.Values[3] != TYPE_PIERCE - TYPE_HIT` like `DoCircle` already does (skill_combat.go:~400).

**Fix:**

```go
func DoBackstab(ch *Player, target combat.Combatant, world *World) SkillResult {
    // 1. Skill check
    if ch.GetSkill(SkillBackstab) == 0 {
        return SkillResult{Success: false, MessageToCh: "You have no idea how."}
    }

    // 2. Self-check (C: act.offensive.c:185)
    if target.GetName() == ch.Name {
        return SkillResult{Success: false, MessageToCh: "How can you sneak up on yourself?"}
    }

    // 3. Must wield a weapon (C: act.offensive.c:189)
    weapon, weaponOK := ch.Equipment.GetItemInSlot(combat.SlotWield)
    if !weaponOK || weapon == nil {
        return SkillResult{Success: false, MessageToCh: "You need to wield a weapon to make it a success."}
    }

    // 4. Piercing weapon required (C: act.offensive.c:194)
    // TYPE_PIERCE = 11, TYPE_HIT = 0, so C checks GET_OBJ_VAL(..., 3) != TYPE_PIERCE - TYPE_HIT
    if weapon.Prototype.Values[3] != 11 { // TYPE_PIERCE - TYPE_HIT
        return SkillResult{
            Success:     false,
            MessageToCh: "Only piercing weapons can be used for backstabbing.\r\n",
        }
    }

    // 5. Mounted check (C: act.offensive.c:206)
    if ch.IsMounted() {
        return SkillResult{Success: false, MessageToCh: "Dismount first!\r\n"}
    }

    // 6. Target must not be fighting (C: act.offensive.c:209)
    if target.GetFighting() != "" {
        return SkillResult{Success: false, MessageToCh: "You can't backstab a fighting person -- they're too alert!"}
    }

    // 7. MOB_AWARE counter-attack (C: act.offensive.c:212)
    //     Even though DP-898 (flag table) isn't landed yet, add the gate.
    //     When AWARE flag is properly wired, this will trigger.
    if mob, ok := target.(*MobInstance); ok && mob.HasMobFlag(MobFlagAware) && target.GetPosition() > combat.PosSleeping {
        victPronouns := GetPronouns(target.GetName(), target.GetSex())
        chPronouns := GetPronouns(ch.Name, ch.GetSex())
        // Mob notices and retaliates — start combat with mob as attacker
        if target.GetFighting() == "" {
            target.SetFighting(ch.Name)
        }
        return SkillResult{
            Success:       false,
            MessageToCh:   ActMessage("$e notices you lunging at $m!", victPronouns, &chPronouns, ""),
            MessageToVict: ActMessage("You notice $N lunging at you!", victPronouns, &chPronouns, ""),
            MessageToRoom: ActMessage("$n notices $N lunging at $m!", victPronouns, &chPronouns, ""),
        }
    }

    // 8. Roll for success (C: act.offensive.c:220)
    //    C: percent = number(1, 101); prob = subcmd ? number(50,100) : GET_SKILL(ch, SKILL_BACKSTAB)
    //    Go subcmd concept doesn't exist here — always use skill level.
    // #nosec G404 — game RNG
    percent := rand.IntN(101) + 1 // 1-101
    skillLevel := ch.GetSkill(SkillBackstab)
    prob := skillLevel

    chPronouns := GetPronouns(ch.Name, ch.GetSex())
    victPronouns := GetPronouns(target.GetName(), target.GetSex())

    if target.GetPosition() > combat.PosSleeping && percent > prob {
        // MISS — but C calls damage(ch, vict, 0, SKILL_BACKSTAB) which starts combat
        // Return a miss result that signals combat should start
        return SkillResult{
            Success:       false,
            MessageToCh:   ActMessage("You try to backstab $N, but $E notices you!", chPronouns, &victPronouns, ""),
            MessageToVict: ActMessage("$n tries to backstab you, but you notice $m in time!", chPronouns, &victPronouns, ""),
            MessageToRoom: ActMessage("$n tries to backstab $N, but fails.", chPronouns, &victPronouns, ""),
            StartCombat:   true, // NEW FIELD — signals caller to initiate combat
            WaitCh:        1,
        }
    }

    // 9. HIT — calculate damage
    //    C: dam = str_app[STRENGTH_APPLY_INDEX(ch)].todam + GET_DAMROLL(ch) + weapon_dice
    //    Then: dam *= backstab_mult(GET_LEVEL(ch))
    weaponNum, weaponSides := ch.Equipment.GetWeaponDamage()
    weaponDam := combat.RollDice(weaponNum, weaponSides)
    strToDam := ch.GetStrToDam() // NEW METHOD — needs to exist on Player
    dam := weaponDam + ch.GetDamroll() + strToDam
    mult := combat.BackstabMult(ch.GetLevel())
    dam = int(float64(dam) * mult)

    improveSkill(ch, SkillBackstab)

    return SkillResult{
        Success:       true,
        Damage:        dam,
        MessageToCh:   "Your deadly backstab strikes deep!",
        MessageToVict: ActMessage("$n sneaks up from behind and plunges a dagger into you!", chPronouns, &victPronouns, ""),
        MessageToRoom: ActMessage("$n sneaks up from behind and backstabs $N!", chPronouns, &victPronouns, ""),
        WaitCh:        1,
    }
}
```

### Supporting changes needed:

**1. Add `StartCombat` field to `SkillResult`** (in `pkg/game/skill_types.go` or wherever SkillResult is defined):
```go
type SkillResult struct {
    // ... existing fields ...
    StartCombat bool // Signal to caller: initiate combat even on miss (C: damage(ch, vict, 0, skill))
}
```

**2. Add `GetStrToDam()` method to Player** (in `pkg/game/player.go`):
```go
// GetStrToDam returns the strength to-damage bonus.
// C: str_app[STRENGTH_APPLY_INDEX(ch)].todam
func (p *Player) GetStrToDam() int {
    // Look up from str_app table using player's strength
    // See pkg/game/tables.go for str_app or similar
    str := p.GetStat(StatStr)
    if str < len(strAppTable) {
        return strAppTable[str].ToDam
    }
    return 0
}
```

If `strAppTable` doesn't exist yet, check `pkg/game/tables.go` or `pkg/game/limits.go` for the strength application table. The C source is `src/structs.h` — `struct str_app_type` has `todam` field.

**3. Wire `sendSkillResult` to respect `StartCombat`** (in `pkg/command/skill_commands.go`):
After the existing damage block, add:
```go
if result.StartCombat && target != nil {
    // C: damage(ch, vict, 0, skill) starts combat even on miss
    // DP-900 fix should handle this via DoSpellDamage or direct combat engine call
    if s.player.GetFighting() == "" {
        // Start combat — both sides should be fighting
        s.world.combatEngine.StartCombat(s.player, target)
    }
}
```

**Cite:** C source — `act.offensive.c:165` (`do_backstab`). Key differences: piercing check (line 194), MOB_AWARE (line 212), miss-init (line 227 — `damage(ch, vict, 0, SKILL_BACKSTAB)`), str-to-dam (referenced from `fight.c` damage calc).

**Regression Test:**
```go
func TestBackstabRequiresPiercingWeapon(t *testing.T) {
    // Equip a slashing weapon
    // Attempt backstab
    // Assert: "Only piercing weapons can be used for backstabbing."
}

func TestBackstabMissStartsCombat(t *testing.T) {
    // Backstab a high-HP mob (will miss due to RNG or low skill)
    // Assert: StartCombat field is true in result
    // Assert: both attacker and defender are fighting after caller processes result
}

func TestBackstabStrToDam(t *testing.T) {
    // Create player with known strength
    // Backstab with known weapon
    // Assert damage includes str_app todam bonus
}

func TestBackstabBareHandedFails(t *testing.T) {
    // Unequip weapon
    // Attempt backstab
    // Assert: "You need to wield a weapon"
}
```

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Execution Order

1. Add `StartCombat` field to `SkillResult`
2. Add `GetStrToDam()` to Player (or find existing str_app table)
3. Rewrite `DoBackstab` with all C gates
4. Wire `sendSkillResult` to handle `StartCombat: true`
5. Add regression tests

## After All Fixes

```bash
cd /Users/zach/.openclaw/workspace-daeron/darkpawns_repo
git add pkg/game/skill_combat.go pkg/game/skill_types.go pkg/game/player.go pkg/command/skill_commands.go
git commit -m "fix: backstab C gates — piercing check, MOB_AWARE, miss-init, str-to-dam (DP-906)"
git push -u origin fix/dp-906-backstab-gates
gh pr create --title "fix: backstab C gates (DP-906)" --body "Fixes DP-906. See docs/briefs/BRIEF-2026-07-03-sprint2-backstab.md"
```

## Linear Updates (after merge)

- DP-906: Add comment "Fixed — all C gates ported: piercing check, MOB_AWARE, miss-init combat, str-to-dam bonus", commit hash, move to Done
