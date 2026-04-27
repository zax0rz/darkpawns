# Skill System Documentation

## Overview

The skill system in Dark Pawns allows characters to learn, practice, and improve various abilities. Skills are categorized into three types: Combat, Magic, and Utility. Each skill has a proficiency level (0-100) that increases through practice and use.

## Skill Types

### 1. Combat Skills
- **Purpose**: Physical combat abilities
- **Examples**: Swordsmanship, Archery, Shield Use, Dodging
- **Primary Stats**: Strength, Dexterity
- **Requirements**: Character level ≥ skill difficulty, STR/DEX ≥ 10

### 2. Magic Skills
- **Purpose**: Spellcasting and magical abilities
- **Examples**: Evocation, Abjuration, Necromancy, Illusion
- **Primary Stats**: Intelligence, Wisdom
- **Requirements**: Character level ≥ skill difficulty, INT/WIS ≥ 12

### 3. Utility Skills
- **Purpose**: Non-combat practical abilities
- **Examples**: Stealth, Lock Picking, Healing, Crafting
- **Primary Stats**: Dexterity, Intelligence
- **Requirements**: Character level ≥ skill difficulty, DEX/INT ≥ 8

## Skill Structure

Each skill has the following attributes:

| Attribute | Description | Range |
|-----------|-------------|-------|
| Name | Unique identifier | lowercase, no spaces |
| DisplayName | Player-facing name | Any string |
| Type | Skill category | Combat, Magic, Utility |
| Level | Proficiency level | 0-100 |
| Practice | Practice points accumulated | 0-100 |
| Difficulty | Learning difficulty | 1-10 |
| MaxLevel | Maximum achievable level | Usually 100 |
| Learned | Whether skill is known | true/false |
| LastUsed | Timestamp of last use | time.Time |

## Skill Progression

### Learning New Skills
1. **Requirements**: Meet level and stat requirements
2. **Skill Points**: Spend skill points equal to difficulty
3. **Available Slots**: Have empty skill slots
4. **Command**: `learn <skillname>`

### Practicing Skills
1. **Requirement**: Skill must be learned
2. **Process**: Accumulate practice points (10-30 per practice)
3. **Level Up**: When practice reaches 100, chance to increase level
4. **Success Chance**: Based on stats, difficulty, and level difference
5. **Command**: `practice <skillname>`

### Using Skills
1. **Skill Checks**: Automatic during combat or specific actions
2. **Manual Use**: `use <skillname> [target]`
3. **Improvement Chance**: Small chance to gain practice on successful use
4. **Success Formula**: `level + stat_bonus + difficulty_modifier`

## Skill Levels and Titles

| Level Range | Title | Description |
|-------------|-------|-------------|
| 0 | Unlearned | Skill not known |
| 1-24 | Novice | Basic understanding |
| 25-49 | Apprentice | Developing competence |
| 50-74 | Journeyman | Solid proficiency |
| 75-89 | Expert | Highly skilled |
| 90-99 | Master | Exceptional ability |
| 100 | Grandmaster | Peak mastery |

## Commands

### Player Commands

| Command | Aliases | Description |
|---------|---------|-------------|
| `skills` | `sk` | Display all learned skills |
| `practice <skill>` | | Practice a learned skill |
| `learn <skill>` | | Learn a new skill |
| `list` | `listskills` | Show all available skills |
| `forget <skill>` | | Forget a learned skill |
| `use <skill> [target]` | | Use a skill manually |
| `skillinfo <skill>` | `sinfo` | Show detailed skill information |

### Administrative Commands

| Command | Description |
|---------|-------------|
| `skillpoints <player> <amount>` | Grant skill points to player |
| `teach <player> <skill>` | Teach skill to another player |
| `setskill <player> <skill> <level>` | Set skill level directly |

## Skill Manager API

### Creating a Skill Manager
```go
sm := engine.NewSkillManager()
```

### Registering Skills
```go
// Individual skill
skill := engine.NewSkill("swords", "Swordsmanship", engine.SkillTypeCombat, 3)
sm.RegisterSkill(skill)

// Default skills
sm.InitializeDefaultSkills()
```

### Learning Skills
```go
success := sm.LearnSkill(skill, playerLevel, playerStat)
```

### Practicing Skills
```go
leveledUp := sm.PracticeSkill(skillName, playerLevel, playerStat)
```

### Using Skills
```go
success, improved := sm.UseSkill(skillName, playerLevel, playerStat, targetLevel)
```

### Skill Information
```go
// Get skill by name
skill := sm.GetSkill(skillName)

// Check if skill is learned
hasSkill := sm.HasSkill(skillName)

// Get skill level
level := sm.GetSkillLevel(skillName)

// Get all learned skills
learnedSkills := sm.GetLearnedSkills()

// Get all available skills
allSkills := sm.GetAllSkills()
```

## Integration with Player System

### Player Structure Update
The `Player` struct now includes a `SkillManager` field:
```go
type Player struct {
    // ... other fields
    SkillManager *engine.SkillManager
}
```

### Player Methods
- `SetSkill(name string, level int)` - Set skill level
- `GetSkill(name string) int` - Get skill level

### Character Creation
New characters automatically get:
- Initialized SkillManager
- Default skills registered
- Starting skill points based on class/race

## Default Skills

### Combat Skills
- `swords` - Swordsmanship (Difficulty: 3)
- `axes` - Axe Fighting (4)
- `maces` - Mace Fighting (3)
- `daggers` - Dagger Fighting (2)
- `polearms` - Polearm Fighting (5)
- `archery` - Archery (4)
- `unarmed` - Unarmed Combat (1)
- `shield` - Shield Use (2)
- `parry` - Parrying (3)
- `dodge` - Dodging (3)

### Magic Skills
- `evocation` - Evocation Magic (6)
- `abjuration` - Abjuration Magic (5)
- `conjuration` - Conjuration Magic (7)
- `divination` - Divination Magic (4)
- `enchantment` - Enchantment Magic (6)
- `illusion` - Illusion Magic (5)
- `necromancy` - Necromancy (8)
- `transmutation` - Transmutation Magic (7)

### Utility Skills
- `stealth` - Stealth (3)
- `lockpick` - Lock Picking (4)
- `disarm` - Trap Disarming (5)
- `search` - Searching (2)
- `track` - Tracking (3)
- `heal` - Healing (4)
- `craft` - Crafting (3)
- `appraise` - Appraisal (2)
- `diplomacy` - Diplomacy (4)
- `intimidate` - Intimidation (3)

## Skill Points and Slots

### Skill Points
- **Acquisition**: Level up, quest rewards, training
- **Usage**: Learning new skills (cost = difficulty)
- **Refund**: Half points refunded when forgetting skill

### Skill Slots
- **Default**: 10 slots per character
- **Increase**: Through abilities, items, or leveling
- **Management**: `GetUsedSlots()`, `GetAvailableSlots()`, `IncreaseSlots()`

## Teaching System

### Requirements for Teaching
1. Teacher skill level ≥ 50
2. Teacher level ≥ student skill level + 20
3. Student meets skill requirements
4. Student has available skill points and slots

### Teaching Process
```go
success := teacherSM.TeachSkill(skillName, studentSM, teacherLevel, studentLevel, studentStat)
```

## Testing

Run skill system tests:
```bash
go test ./pkg/engine -run TestSkill
```

## Future Enhancements

### Planned Features
1. **Skill Synergies**: Bonus for related skills
2. **Skill Specializations**: Focus within a skill category
3. **Skill Degradation**: Skills decay without use
4. **Skill Books**: Learn from items
5. **Skill Quests**: Special quests for rare skills

### Integration Points
1. **Combat System**: Skill checks in combat formulas
2. **Crafting System**: Utility skills for item creation
3. **Social System**: Diplomacy/intimidation in NPC interactions
4. **Exploration System**: Search/track for hidden content

## Examples

### Learning a Skill
```
> learn swords
You successfully learn Swordsmanship!
```

### Practicing a Skill
```
> practice swords
You practice Swordsmanship. Progress: 45% (Level 1)
```

### Using a Skill
```
> use stealth
You attempt to use Stealth... Success!
You feel like you've improved your understanding of this skill.
```

### Viewing Skills
```
> skills
╔══════════════════════════════════════════════════════╗
║                     Your Skills                      ║
╠══════════════════════════╦══════╦════════╦═══════════╣
║ Skill                    ║ Level║ Progress║ Type     ║
╠══════════════════════════╬══════╬════════╬═══════════╣
║ Swordsmanship            ║    5 ║  75%   ║ Combat    ║
║ Stealth                  ║    3 ║  30%   ║ Utility   ║
╚══════════════════════════╩══════╩════════╩═══════════╝

Skill points: 2 | Slots: 2/10 (8 available)
```