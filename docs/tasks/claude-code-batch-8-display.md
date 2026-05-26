# Claude Code Batch — Run 8: Information & Display Commands

## Overview
4 fidelity issues — commands that display information to the player are either stubs, missing, or using fabricated data. All in session-layer files. Zero overlap between tasks (different functions in different files).

## Issues
- DP-379: Score command stub (HIGH)
- DP-380: coins/abils/levels/toggle missing (HIGH)
- DP-378: Consider command fabricated damage (HIGH)
- DP-406: Peek command mock (MEDIUM)

---

## Task 1: Score command — full RPG layout (DP-379)

**File:** `pkg/session/cmd_info.go` — `cmdScore()` (lines 9-22)

**Current Go (debug stub):**
```go
func cmdScore(s *Session) error {
    p := s.player
    s.Send(fmt.Sprintf("Name: %s  Level: %d  XP: %d/%d", p.Name, p.Level, p.Exp, 1000))
    s.Send(fmt.Sprintf("HP: %d/%d  Mana: %d/%d  Move: %d/%d", p.Health, p.MaxHealth, p.Mana, p.MaxMana, p.Move, p.MaxMove))
    s.Send(fmt.Sprintf("Str: %-14s  Int: %-14s  ..."))
    s.Send(fmt.Sprintf("AC:%d  Hitroll:%d  Damroll:%d  Align:%d  Gold:%d", p.AC, p.Hitroll, p.Damroll, p.Alignment, p.Gold))
    return nil
}
```

**C source:** `src/act.informative.c:1168` — `do_score()`. Prints a full RPG character sheet with these sections (in order):

1. **Name + Age** — `GET_NAME(ch)` + `GET_AGE(ch)` + birthday check
2. **HP/Mana/Move** — colored current(max) format
3. **Alignment text** — 12 tiers from "Epitome of Righteousness" (1000) to "Epitome of Evil" (-1000). See C lines 1213-1238 for exact thresholds and strings.
4. **AC text** — 14 tiers from "naked, have you no shame?" (100) to "armored like a god!" (-175). See C lines 1240-1265.
5. **Experience** — raw number
6. **Gold carried + Gold in bank** — `GET_GOLD(ch)` + `GET_BANK_GOLD(ch)`
7. **Kills/PKs/Deaths** — `GET_KILLS(ch)`, `GET_PKS(ch)`, `GET_DEATHS(ch)`
8. **XP to next level** — only for levels below LVL_IMMORT-1
9. **Play time** — days + hours from `playing_time(ch)`
10. **Veteran status** — `is_veteran(ch)` check
11. **Citizenship** — hometown name
12. **Clan info** — rank name + clan name (if in a clan with rank > 0)
13. **Title line** — "This ranks you as Name Title (level N)."
14. **Race + Class** — "You are a race class."
15. **Pack weight** — 6 tiers from "empty" to "almost too heavy to lift" based on `IS_CARRYING_W/CAN_CARRY_W` ratio
16. **Position** — 9 positions from POS_DEAD to POS_STANDING
17. **Status conditions** — intoxicated (DRUNK > 10), hungry (FULL == 0), thirsty (THIRST == 0)
18. **Active affects** — AFF_BLIND, PRF_SUMMONABLE, AFF_WEREWOLF, AFF_VAMPIRE, AFF_MOUNT, AFF_FLESH_ALTER
19. **Spell affects list** — `ch->affected` linked list, print each spell name (excluding AFF_SNEAK, AFF_DODGE, AFF_KUJI_KIRI without modifier, AFF_ROBBED, AFF_BERSERK)

**Go fields available on Player:**
- `p.Name`, `p.Level`, `p.Exp`, `p.Health`, `p.MaxHealth`, `p.Mana`, `p.MaxMana`, `p.Move`, `p.MaxMove`
- `p.Alignment`, `p.AC`, `p.Hitroll`, `p.Damroll`, `p.Gold`, `p.BankGold`
- `p.Stats.Str/Int/Wis/Dex/Con/Cha`
- `p.Class`, `p.Race`, `p.RoomVNum`, `p.Position`, `p.Title`, `p.Description`
- `p.Hunger`, `p.Thirst`, `p.Drunk`
- `p.ClanID`, `p.ClanRank`
- `p.GetFlags()` — PRF flags
- `p.ActiveAffects` — `[]*engine.Affect` (each has `Type`, `Flags`, `Duration`, `Modifier`, `Location`)

**Implement:**
- Replace the stub with the full layout matching the C source order
- Use `game.AlignmentText(alignment)` helper — create it if it doesn't exist
- Use `game.ACText(ac)` helper — create it if it doesn't exist
- Use `game.ClassNames[p.Class]` and `game.RaceNames[p.Race]` for text
- For kills/PKs/deaths: check `p.Kills`, `p.Pks`, `p.Deaths` fields (may need to add to Player if missing — check first)
- For active affects: iterate `p.ActiveAffects`, filter out AFF_SNEAK/AFF_DODGE/AFF_BERSERK/AFF_ROBBED, print spell name
- Use `fmt.Sprintf` with `\r\n` line endings — this is a telnet MUD

**Helper functions to add (in `cmd_info.go` or a new file):**
```go
func alignmentText(alignment int) string {
    switch {
    case alignment == 1000: return "You are the Epitome of Righteousness!"
    case alignment >= 900: return "You're so good, you make the angels jealous."
    case alignment >= 750: return "You are feeling pretty righteous."
    case alignment >= 500: return "You are aligned with the path of right."
    case alignment >= 350: return "You are feeling pretty good today."
    case alignment >= 100: return "You are a little more good than neutral, but yet still bland."
    case alignment > -100: return "You are neutral, how boring."
    case alignment > -350: return "You are little more evil than neutral, but not very exciting."
    case alignment > -500: return "I actually think you would kill your own mother."
    case alignment > -750: return "You are so evil it hurts."
    case alignment > -900: return "Charles Manson is in your fan club."
    default: return "You are the Epitome of Evil!"
    }
}

func acText(ac int) string {
    switch {
    case ac == 100: return "You are naked, have you no shame?"
    case ac > 70: return "You are lightly clothed."
    case ac > 40: return "You are pretty well clothed."
    case ac > 10: return "You are lightly armored."
    case ac > -10: return "You are well armored."
    case ac > -40: return "You are getting pretty sweaty with all that armor on."
    case ac > -20: return "You are very well armored."
    case ac > -50: return "You are extremely well armored."
    case ac > -75: return "You are decked out in full battle armor."
    case ac > -125: return "You are armored like a wyvern!"
    case ac > -150: return "You are armored like a dragon!"
    case ac > -175: return "You could walk through the gates of Hell in all that armor!"
    default: return "You are armored like a god!"
    }
}
```

---

## Task 2: Register coins, abils, levels + fix toggle (DP-380)

**4 sub-tasks, all in different files:**

### 2a. Register `coins` command

**File:** `pkg/session/commands.go` — add registration:
```go
cmdRegistry.Register("coins", wrapArgs(cmdCoins), "Display your gold and bank balance.", 0, 0)
```

**File:** `pkg/session/cmd_info.go` — add function:
```go
func cmdCoins(s *Session) error {
    p := s.player
    s.Send(fmt.Sprintf("You are currently carrying %d coins,\r\n", p.Gold))
    s.Send(fmt.Sprintf("and in your bank account, you have %d coins.\r\n", p.BankGold))
    s.Send(fmt.Sprintf("Your current net-worth is %d coins.\r\n", p.Gold+p.BankGold))
    return nil
}
```

**C source:** `src/act.informative.c:2743` — `do_coins()`. Simple 3-line display.

### 2b. Register `abils` command

**File:** `pkg/session/commands.go` — add registration:
```go
cmdRegistry.Register("abils", wrapArgs(cmdAbils), "Show your ability scores.", 0, 0)
```

**File:** `pkg/session/cmd_info.go` — add function:
```go
func cmdAbils(s *Session) error {
    p := s.player
    s.Send("Your current ability scores:\r\n")
    s.Send(fmt.Sprintf("Strength:      (%s)\r\n", getAbilName(p.Stats.Str)))
    s.Send(fmt.Sprintf("Dexterity:     (%s)\r\n", getAbilName(p.Stats.Dex)))
    s.Send(fmt.Sprintf("Intelligence:  (%s)\r\n", getAbilName(p.Stats.Int)))
    s.Send(fmt.Sprintf("Wisdom:        (%s)\r\n", getAbilName(p.Stats.Wis)))
    s.Send(fmt.Sprintf("Constitution:  (%s)\r\n", getAbilName(p.Stats.Con)))
    s.Send(fmt.Sprintf("Charisma:      (%s)\r\n", getAbilName(p.Stats.Cha)))
    return nil
}
```

Note: `getAbilName` already exists in `cmd_info.go` (used by cmdScore). If it doesn't exist, create it — it maps stat values (3-25) to text names like "Weak", "Average", "Strong", etc.

**C source:** `src/act.informative.c:1077` — `do_abils()`. Uses `abil_names[]` array.

### 2c. Register `levels` command

**File:** `pkg/session/commands.go` — add registration:
```go
cmdRegistry.Register("levels", wrapArgs(cmdLevels), "Show XP table for your class.", 0, 0)
```

**File:** `pkg/session/cmd_info.go` — add function:
```go
func cmdLevels(s *Session) error {
    p := s.player
    var buf strings.Builder
    for i := 1; i < 31; i++ { // LVL_IMMORT = 31
        xpNeeded := game.ExpNeededForLevel(p.Class, i)
        xpPrev := game.ExpNeededForLevel(p.Class, i-1)
        fmt.Fprintf(&buf, "[%2d] %8d-%-8d    (%6d)\r\n", i, xpPrev, xpNeeded, xpNeeded-xpPrev)
    }
    s.Send(buf.String())
    return nil
}
```

**C source:** `src/act.informative.c:2311` — `do_levels()`. Iterates 1 to LVL_IMMORT-1, prints XP range per level.

**Note:** `game.ExpNeededForLevel(class, level)` may not exist. Check `pkg/game/level.go` for the XP table or `find_exp()` equivalent. If missing, create it using the C source `src/class.c` XP tables.

### 2d. Expand `toggle` command

**File:** `pkg/session/act_informative.go` — `cmdToggle()` (line 158)

**Current Go:** Only supports `autoexit`. Rejects everything else with "Unknown toggle".

**C source:** `src/act.informative.c:2500` — `do_toggle()` displays 24 toggles in a formatted grid.

**Fix:** Expand the toggle command to support all toggles. The C version displays but doesn't toggle inline — it's a status display. The Go version should match: show all toggles when called with no args, toggle a specific one when called with a toggle name.

**Toggles to support (from C source):**

| Toggle | PRF Flag | ON/OFF |
|--------|----------|--------|
| hitpoint | PRF_DISPHP | ON=show HP in prompt |
| brief | PRF_BRIEF | ON=short room descriptions |
| summonable | PRF_SUMMONABLE | OFF=can be summoned |
| move | PRF_DISPMOVE | ON=show move in prompt |
| compact | PRF_COMPACT | ON=less spacing |
| quest | PRF_QUEST | YES=on quest |
| mana | PRF_DISPMANA | ON=show mana in prompt |
| notell | PRF_NOTELL | ON=can't receive tells |
| norepeat | PRF_NOREPEAT | OFF=repeat chat |
| autoexit | PRF_AUTOEXIT | ON=show exits in room |
| autoloot | PRF_AUTOLOOT | ON=auto loot corpses |
| autogold | PRF_AUTOGOLD | ON=auto get gold |
| autosplit | PRF_AUTOSPLIT | ON=auto split gold |
| deaf | PRF_DEAF | ON=deaf to channels |
| wimpy | WIMP_LEV | numeric (0=off) |
| nogossip | PRF_NOGOSS | OFF=gossip on |
| noauction | PRF_NOAUCT | OFF=auction on |
| nogratz | PRF_NOGRATZ | OFF=gratz on |
| disptank | PRF_DISPTANK | ON=show tank stat |
| disptarget | PRF_DISPTARGET | ON=show fightg stat |
| color | COLOR_LEV | 0-3 (off/sparse/normal/complete) |
| newbie | PRF_NONEWBIE | OFF=newbie on |
| noctell | PRF_NOCTELL | OFF=clan tells on |
| nobroad | PRF_NOBROAD | OFF=broadcasts on |

**Implementation:** For each toggle, check if the PRF flag constant exists in `pkg/game/player_flags.go`. The flag bit positions need to match the C source. Use the same ONOFF/YESNO display format as the C source (3-column grid).

**Important:** Check what PRF flags are already defined in `pkg/game/player_flags.go`. Some may be missing. If a flag is missing, add it with the correct bit value matching the C source.

---

## Task 3: Consider command — real damage calculations (DP-378)

**File:** `pkg/session/consider.go` — `cmdConsider()` (lines 61-140)

**C source:** `src/act.informative.c:2330` — `do_consider()`. Three-part sentence:

**Part 1 — Damage comparison:**
- Get player's wielded weapon: `GET_EQ(ch, WEAR_WIELD)`
- Get victim's wielded weapon: `GET_EQ(victim, WEAR_WIELD)`
- Base damage = `str_app[STRENGTH_APPLY_INDEX(ch)].todam + GET_DAMROLL(ch)`
- If wielded: add `dice(obj.Val[1], obj.Val[2])` (numdice, sizedice)
- If bare-handed player: add `number(0, GET_LEVEL(ch)/3)`
- If NPC: add `dice(mob.damnodice, mob.damsizedice)`
- Same calculation for victim
- `damdiff = victdam - chardam`
- 7 text tiers from "eat you for lunch" (>20) to "not even worth the effort" (<-10)

**Part 2 — HP comparison:**
- `hitdiff = GET_HIT(victim) - GET_HIT(ch)`
- 7 text tiers based on multiples of player's max HP:
  - > 4x: "much better physical shape"
  - > 2x: "a lot better physical shape"
  - > 1x: "better physical shape"
  - >= 0: "about the same physical shape"
  - > -25: "a little worse physical shape"
  - > -50: "worse physical shape"
  - else: "a lot worse physical shape"

**Part 3 — Level confidence:**
- `leveldiff = GET_LEVEL(victim) - GET_LEVEL(ch)`
- 7 text tiers:
  - > ch level: "moves with an ease telling of many won battles"
  - > ch level/2: "seems to know his opponent"
  - >= 0: "seems about as confident in battle as you do"
  - > -3: "seems less confident in battle than you do"
  - > -5: "seems much less confident in battle than you do"
  - > -7: "seems ready to run from a fight"
  - else: "seems like he's never been in battle before"

**Go implementation — fix the damage calculation:**
1. Get player's wielded weapon from equipment: `s.player.Equipment.GetItemAtSlot(game.WearWield)` (or similar — check how equipment is accessed)
2. Get victim's wielded weapon similarly
3. Use the same `str_app` todam table (check if it exists in Go — might be `game.StrApp` or similar)
4. Add weapon dice damage if wielded, bare-hand damage if not
5. Same for victim (check if victim is NPC → use mob damnodice/damsizedice)
6. Fix Part 2 to use HP ratios relative to player's max HP (not hardcoded thresholds)
7. Add Part 3 (level confidence) — completely missing in Go

**Check for existing helpers:**
- `game.StrApp` or `str_app` table for todam
- How to get wielded weapon from player equipment
- How to get mob damage dice (damnodice, damsizedice on MobInstance)

---

## Task 4: Peek command — list actual items (DP-406)

**File:** `pkg/game/other_utility.go` — `doPeek()` (lines 15-60)

**C source:** `src/act.other.c:1665` — `do_peek()`. On success, calls `look_at_char(victim, ch)` which displays the victim's equipment and inventory to the peeker.

**Current Go:** Runs skill check correctly, then prints static "[Equipment and inventory]" header and never lists items.

**Fix:** After the skill check succeeds, list the victim's equipment and inventory:

```go
// After successful skill check:
ch.SendMessage(fmt.Sprintf("You peek at %s's belongings:\r\n", victimPl.Name))

// List equipment
ch.SendMessage("[Equipment]\r\n")
for i := 0; i < game.NumWearSlots; i++ {
    item := victimPl.Equipment.GetItemAtSlot(i)
    if item != nil {
        ch.SendMessage(fmt.Sprintf("  %s\r\n", item.Prototype.Name))
    }
}

// List inventory
ch.SendMessage("[Inventory]\r\n")
for _, item := range victimPl.Inventory.Items {
    if item != nil {
        ch.SendMessage(fmt.Sprintf("  %s\r\n", item.Prototype.Name))
    }
}
```

**Check:** How does the player's equipment/inventory work in Go? Look at:
- `pkg/game/equipment.go` — how to iterate wear slots
- `pkg/game/inventory.go` — how to iterate inventory items
- `item.Prototype.Name` or `item.Name` — the item name field

The C version uses `look_at_char()` which does a full character description. For peek, a simple list is fine — we don't need the full look_at_char behavior, just the item names.

---

## Execution Order
All 4 tasks are independent — different functions in different files. They can execute in any order.

**Recommended order:**
1. Task 2 (coins/abils/levels/toggle) — simplest, 4 small functions
2. Task 4 (peek) — small, self-contained
3. Task 3 (consider) — medium, needs damage calculation helpers
4. Task 1 (score) — largest, needs alignment/AC text helpers

## Verification
1. `go build ./...` — must pass after each task
2. `go vet ./...` — must pass
3. `go test ./...` — must pass
