**Last updated:** 2026-05-08

# Skill System

The world has walls, and some mortals learn to move through them differently. Dark Pawns inherits the CircleMUD skill model: each skill is a proficiency (0–100), gated by class and level, improved through use and practice at a guildmaster. No skill points for basic combat skills. No D&D spell schools. The nine rooms of backstab, bash, kick, sneak, steal — they open or they don't, depending on who you are and how long you've been at it.

Two layers exist here and they only partially talk to each other. Know which one you're touching before you touch it.

---

## The Two Layers

### Layer 1: CircleMUD Skills (`pkg/game/`)

The real skill system. Direct port from `class.c` and `act.offensive.c`. Skills are stored as `map[string]int` on the `Player` struct — name to proficiency (0–100). No struct overhead, no registration required.

**Getting/setting proficiency:**
```go
level := ch.GetSkill(SkillBackstab)   // 0 = doesn't know it
ch.SetSkill(SkillBackstab, 75)         // 75% proficiency
```

**Class/level requirements** are defined in `pkg/game/skills.go` as `SkillClassReq`:
```go
var SkillClassReq = map[string]map[int]int{
    SkillBackstab: {ClassThief: 1, ClassAssassin: 1},
    SkillBash:     {ClassWarrior: 3, ClassPaladin: 3, ClassRanger: 3},
    // ...
}
```

A class not in the map cannot use that skill. `CanUseSkill(ch, skillName)` enforces this plus position requirements.

**Skill improvement** happens on use via `improveSkill()` in `pkg/game/combat_helpers.go`:
```go
func improveSkill(ch *Player, skill string) {
    cur := ch.GetSkill(skill)
    if cur <= 0 || cur >= 100 { return }
    if rand.Intn(100)+1 > cur {                      // harder to improve as you get better
        chance := (ch.GetInt() + ch.GetWis()) / 4    // INT+WIS averaged
        if rand.Intn(100) < chance {
            ch.SetSkill(skill, cur+1)
            ch.SendMessage(fmt.Sprintf("You feel a bit more competent in %s.\r\n", skill))
        }
    }
}
```

Higher proficiency = smaller window for improvement. INT and WIS matter. That's it.

**Guild practice** is handled by the `guild` mob special (`pkg/game/spec_procs.go`). When a player types `practice <skill>` near a guildmaster mob, `specGuild()` fires: checks that the skill is learned, calls `ch.SkillManager.PracticeSkill()`, and sends feedback. The guildmaster is the only way to practice — you can't grind it by standing in a room and typing practice repeatedly.

---

### Layer 2: SkillManager (`pkg/engine/`)

A newer overlay system sitting on top. Stores skills as a `map[string]*Skill` with full progression metadata. Used by the `skills`, `learn`, `forget`, `practice`, and `use` player commands in `pkg/command/skill_commands.go`. Also registers the Dark Pawns core skills so the command layer knows about them.

**The Skill struct** (`pkg/engine/skill.go`):
```go
type Skill struct {
    Name        string    // unique key (e.g. "backstab")
    DisplayName string    // player-facing name
    Type        SkillType // SkillTypeCombat / SkillTypeMagic / SkillTypeUtility
    Level       int       // proficiency 0-100
    Practice    int       // practice points accumulated 0-100
    Difficulty  int       // 1-10 (affects learning requirements)
    MaxLevel    int       // cap, usually 100
    Learned     bool      // whether learned
    LastUsed    time.Time
}
```

**SkillManager API** (`pkg/engine/skill_manager.go`):

| Method | What it does |
|---|---|
| `NewSkillManager()` | Creates manager with 10 slots, 0 points |
| `RegisterSkill(skill)` | Adds skill to registry without learning it |
| `LearnSkill(skill, charLevel, stat)` | Attempts to learn (checks slots, points, requirements) |
| `PracticeSkill(name, charLevel, stat)` | Accumulates 10–30 practice points; levels up on 100+ |
| `UseSkill(name, charLevel, stat, targetLevel)` | Rolls for success; small improvement chance |
| `ForgetSkill(name)` | Marks unlearned; refunds half difficulty in points |
| `TeachSkill(name, target, ...)` | Player-to-player teaching (level 50+ skill required) |
| `GetSkill(name)` | Returns `*Skill` or nil |
| `HasSkill(name)` | Returns true if skill is registered AND Learned |
| `GetSkillLevel(name)` | Returns level or 0 if not learned |
| `GetLearnedSkills()` | Sorted by level desc |
| `GetAllSkills()` | All registered, learned or not |
| `AddSkillPoints(n)` / `GetSkillPoints()` | Manage the skill point pool |
| `GetSlots()` / `GetUsedSlots()` / `GetAvailableSlots()` / `IncreaseSlots(n)` | Manage the slot limit |
| `InitializeDefaultSkills()` | Registers the full skill set |

**Learning requirements** enforced by `LearnSkill`:
- `charLevel >= skill.Difficulty`
- Stat threshold: Combat ≥ 10 STR/DEX, Magic ≥ 12 INT/WIS, Utility ≥ 8 DEX/INT
- Skill points ≥ difficulty
- Used slots < max slots (default 10)

**Practice progression**: Each call to `PracticeSkill()` adds 10–30 random practice points. When practice hits 100+, a success check fires: `50 + (stat*2) - (difficulty*5) + (charLevel - skillLevel)`, capped 10–90%. Success → level up + reset practice. Failure → lose 30 practice points.

**Proficiency titles** from `GetDisplayLevel()`:

| Level | Title |
|---|---|
| 0 | Unlearned |
| 1–24 | Novice |
| 25–49 | Apprentice |
| 50–74 | Journeyman |
| 75–89 | Expert |
| 90–99 | Master |
| 100 | Grandmaster |

---

## Skills That Actually Exist

### Core CircleMUD Skills

Defined in `pkg/game/skills.go` as string constants. These are the ones with class/level gating in `SkillClassReq`. Column headers: Skill | Classes | Min Level.

**Combat (offensive):**

| Skill | Classes | Min Level |
|---|---|---|
| `backstab` | Thief, Assassin | 1 |
| `bash` | Warrior, Paladin, Ranger | 3 |
| `kick` | All classes except Druid/Ranger at 1 | 1 |
| `trip` | Thief, Assassin | 9 |
| `headbutt` | Warrior (5), Paladin (5), Ranger (7) | 5–7 |
| `rescue` | Warrior (4), Paladin (3), Ranger (5) | 3–5 |

**Stealth / Utility:**

| Skill | Classes | Min Level |
|---|---|---|
| `sneak` | Thief, Assassin | 2 |
| `hide` | Thief, Assassin (5), Ranger (10) | 5–10 |
| `steal` | Thief, Assassin | 3 |
| `pick_lock` | Thief, Assassin | 4 |

**Wave 1 / Wave 2 skills** (from `new_cmds.c` and `new_cmds2.c` ports): These are implemented in `skills.go` and `skill_special.go` but most don't have class requirements in `SkillClassReq` yet — they use `CanUseSkill()` which returns false if not in the map:

`carve`, `cutthroat`, `strike`, `compare`, `scan`, `sharpen`, `scrounge`, `peek`, `stealth`, `appraise`, `scout`, `first_aid`, `disarm`, `mindlink`, `detect`, `serpent_kick`, `dig`, `turn`, `mold`, `behead`, `bearhug`, `slug`, `smackheads`, `bite`, `tag`, `point`, `groinrip`, `review`, `whois`, `palm`, `flesh_alter`

**C-10 Advanced Combat** (`pkg/game/skill_c10_combat.go`):

`disembowel`, `dragon_kick`, `tiger_punch`, `shoot`, `subdue`, `sleeper`, `neckbreak`, `ambush`, `parry`, `escape`, `retreat`

### SkillManager Default Registry

`InitializeDefaultSkills()` registers these in the SkillManager for use by the `learn`/`skills`/`practice` command layer:

**Combat:** `swords`, `axes`, `maces`, `daggers`, `polearms`, `archery`, `unarmed`, `shield`, `parry`, `dodge`, plus the core CircleMUD skills: `backstab`, `bash`, `kick`, `trip`, `rescue`

**Magic:** `evocation`, `abjuration`, `conjuration`, `divination`, `enchantment`, `illusion`, `necromancy`, `transmutation`

**Utility:** `stealth`, `lockpick`, `disarm`, `search`, `track`, `heal`, `craft`, `appraise`, `diplomacy`, `intimidate`, `sneak`, `hide`, `steal`, `pick_lock`

The magic skills and several utility entries (evocation, swords, diplomacy, etc.) are registered in the SkillManager but don't yet have `SkillClassReq` entries — they won't gate via `CanUseSkill()`, which is a known gap.

---

## Commands

### Player Commands

| Command | Description |
|---|---|
| `skills` | Show all learned skills with level and progress |
| `practice <skill>` | Practice at a guildmaster (guild mob special) |
| `learn <skill>` | Learn a skill from the SkillManager registry |
| `list` | Show all available skills with difficulty and status |
| `forget <skill>` | Forget a skill (refunds half difficulty in points); requires `confirm forget` |
| `use <skill> [target]` | Generic skill use (calls SkillManager.UseSkill) |
| `skillinfo <skill>` | Show detailed info for one skill |
| `backstab`, `bash`, `kick`, `trip`, `headbutt`, `rescue` | Direct combat skill commands |
| `sneak`, `hide`, `steal`, `pick` | Direct stealth/utility commands |
| `disembowel`, `dragon_kick`, `tiger_punch`, `shoot`, `subdue`, `sleeper`, `neckbreak`, `ambush`, `parry` | C-10 advanced combat commands |

All direct skill commands (`CmdBackstab`, `CmdBash`, etc.) live in `pkg/command/skill_commands.go`. They call `CanUseSkill()` for class/level gate, then the `Do*` function in `pkg/game/`.

### Player Methods

```go
ch.GetSkill(name string) int         // proficiency 0-100; 0 = doesn't know it
ch.SetSkill(name string, level int)  // set proficiency directly
ch.SkillManager                      // access SkillManager overlay
```

---

## How Skills Actually Improve

Two paths, same world:

**1. Use-based improvement** (CircleMUD path): Every time you successfully use a skill (backstab, bash, etc.), `improveSkill()` runs. Chance to improve = `(INT + WIS) / 4` percent, but only if `rand(100) > current_level` (i.e., higher level = smaller improvement window). You stop improving at 100.

**2. Practice at a guildmaster** (SkillManager path): Type `practice <skill>` near a guildmaster mob. `specGuild()` calls `ch.SkillManager.PracticeSkill()`. This accumulates practice points toward a level-up roll. Guildmaster practice is currently the only deliberate grind path that doesn't require combat.

The two improvement paths increment different counters. `improveSkill()` writes to `Player.Skills` (the map). `PracticeSkill()` writes to `Skill.Level` inside the SkillManager. These are bridged when skill commands read via `GetSkill()`, but are not always in sync. This is a known architectural quirk.

---

## Integration with Player

```go
type Player struct {
    Skills       map[string]int        // CircleMUD proficiency map
    SkillManager *engine.SkillManager  // overlay with full Skill structs
    // ...
}
```

New characters need `InitializeDefaultSkills()` called on their SkillManager at creation. Starting skill points depend on class. The CircleMUD skills (`backstab`, `kick`, etc.) are initialized via pfile load or character creation — they don't go through `LearnSkill()`.

---

## Testing

```bash
go test ./pkg/engine -run TestSkill
go test ./pkg/game -run TestSkill
```

---

## Known Gaps

- Magic skills and several utility skills exist in `InitializeDefaultSkills()` but have no `SkillClassReq` entries. `CanUseSkill()` returns false for them (not in map). They're available in the SkillManager command layer only.
- The two improvement paths (`Player.Skills` map vs `SkillManager.Skill.Level`) are parallel and can drift.
- `CmdPractice` in `skill_commands.go` and `specGuild` in `spec_procs.go` both call `PracticeSkill()` but via different callers. The guildmaster path is the intended one; the command path bypasses mob presence.
- Teaching system (`TeachSkill`) exists in the code but has no in-game mob or command wired to it yet.
