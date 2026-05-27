# Fidelity Testing: Concrete Examples

This document shows what C fidelity testing looks like in practice using `AssertBehaviorMatchesC`. Three levels of complexity, all targeting core game functions.

---

## Level 1: Simple Value Mapping (Attack Hit Text)

The simplest case — C has a lookup table, Go should match it exactly.

**C source** (fight.c:84):
```c
struct attack_hit_type attack_hit_text[] = {
   {"hit", "hits"},       /* 0  */
   {"sting", "stings"},   /* 1  */
   {"whip", "whips"},     /* 2  */
   {"slash", "slashes"},  /* 3  */
   {"bite", "bites"},     /* 4  */
   {"bludgeon", "bludgeons"}, /* 5 */
   {"crush", "crushes"},  /* 6  */
   {"pound", "pounds"},   /* 7  */
   {"claw", "claws"},     /* 8  */
   {"maul", "mauls"},     /* 9  */
   {"thrash", "thrashes"}, /* 10 */
   {"pierce", "pierces"}, /* 11 */
   {"blast", "blasts"},   /* 12 */
   {"punch", "punches"},  /* 13 */
   {"stab", "stabs"}      /* 14 */
};
```

**Go test:**
```go
func TestAttackHitText_Fidelity(t *testing.T) {
    cTable := []struct{ singular, plural string }{
        {"hit", "hits"}, {"sting", "stings"}, {"whip", "whips"},
        {"slash", "slashes"}, {"bite", "bites"}, {"bludgeon", "bludgeons"},
        {"crush", "crushes"}, {"pound", "pounds"}, {"claw", "claws"},
        {"maul", "mauls"}, {"thrash", "thrashes"}, {"pierce", "pierces"},
        {"blast", "blasts"}, {"punch", "punches"}, {"stab", "stabs"},
    }

    for i, expected := range cTable {
        goSingular := combat.AttackHitText[i].Singular
        goPlural := combat.AttackHitText[i].Plural

        testutil.AssertBehaviorMatchesC(t,
            fmt.Sprintf("attack_hit_text[%d].singular", i),
            func() string { return goSingular },
            expected.singular,
        )
        testutil.AssertBehaviorMatchesC(t,
            fmt.Sprintf("attack_hit_text[%d].plural", i),
            func() string { return goPlural },
            expected.plural,
        )
    }
}
```

**What this catches:** Any off-by-one in the array, any typo in singular/plural forms, any reordering of the table.

---

## Level 2: Formula Comparison (Damage Calculation)

More complex — C has a formula, Go should produce the same result for known inputs.

**C source** (fight.c:107, `damage()`):
```c
// Base damage = dice(num, size) + str_bonus - armor_reduction
// Minimum damage = 1
int dam = dice(GET_OBJ_VAL(weapon, 0), GET_OBJ_VAL(weapon, 1));
dam += str_app[GET_STR(ch)].todam;
dam -= GET_AC(victim) / 10;
if (dam < 1) dam = 1;
```

**Go test:**
```go
func TestDamageCalculation_Fidelity(t *testing.T) {
    tests := []struct {
        name        string
        diceNum     int
        diceSize    int
        strBonus    int
        victimAC    int
        expectedMin int
        expectedMax int
    }{
        {"dagger vs unarmored", 1, 4, 0, 0, 1, 4},
        {"sword vs chainmail", 2, 4, 2, 50, 1, 8},  // AC 50 = 5 reduction
        {"mace vs plate", 2, 6, 3, 80, 1, 12},       // AC 80 = 8 reduction
        {"fist vs god", 1, 2, 0, 100, 1, 1},          // Max reduction, floor at 1
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Get the C expected range
            cMin := tt.diceNum + tt.strBonus - tt.victimAC/10
            if cMin < 1 { cMin = 1 }
            cMax := tt.diceNum * tt.diceSize + tt.strBonus - tt.victimAC/10
            if cMax < 1 { cMax = 1 }

            // Test Go's damage calculation
            goMin := combat.CalculateDamageMin(tt.diceNum, tt.diceSize, tt.strBonus, tt.victimAC)
            goMax := combat.CalculateDamageMax(tt.diceNum, tt.diceSize, tt.strBonus, tt.victimAC)

            testutil.AssertBehaviorMatchesC(t,
                tt.name+" min damage",
                func() string { return fmt.Sprintf("%d", goMin) },
                fmt.Sprintf("%d", cMin),
            )
            testutil.AssertBehaviorMatchesC(t,
                tt.name+" max damage",
                func() string { return fmt.Sprintf("%d", goMax) },
                fmt.Sprintf("%d", cMax),
            )
        })
    }
}
```

**What this catches:** Armor reduction formula changes, strength bonus miscalculation, minimum damage floor bugs, dice math errors.

---

## Level 3: State Machine Comparison (Position Updates)

The most complex — C has a state machine with branching logic, Go should replicate every branch.

**C source** (fight.c:186, `update_pos()`):
```c
void update_pos(struct char_data * victim) {
  if ((GET_HIT(victim) >= 1) && (GET_POS(victim) > POS_STUNNED))
    return;
  else if ((GET_HIT(victim) >= 1) && (GET_POS(victim) == POS_STUNNED))
    GET_POS(victim) = POS_STANDING;
  else if (GET_HIT(victim) == -1)
    GET_POS(victim) = POS_MORTALLYW;
  else if (GET_HIT(victim) == -2)
    GET_POS(victim) = POS_STUNNED;
  else if (GET_HIT(victim) <= -3)
    GET_POS(victim) = POS_DEAD;
}
```

**Go test:**
```go
func TestUpdatePosition_Fidelity(t *testing.T) {
    // Position constants from C source (structs.h)
    const (
        POS_DEAD     = 0
        POS_MORTALLYW = 1
        POS_STUNNED  = 2
        POS_SLEEPING = 3
        POS_RESTING  = 4
        POS_SITTING  = 5
        POS_FIGHTING = 6
        POS_STANDING = 7
    )

    tests := []struct {
        name      string
        hitPoints int
        startPos  int
        expected  int
    }{
        // From C source logic:
        {"healthy standing stays standing", 10, POS_STANDING, POS_STANDING},
        {"healthy fighting stays fighting", 10, POS_FIGHTING, POS_FIGHTING},
        {"stunned with HP becomes standing", 1, POS_STUNNED, POS_STANDING},
        {"HP -1 becomes mortally wounded", -1, POS_STANDING, POS_MORTALLYW},
        {"HP -2 becomes stunned", -2, POS_STANDING, POS_STUNNED},
        {"HP -3 becomes dead", -3, POS_STANDING, POS_DEAD},
        {"HP -10 becomes dead", -10, POS_STANDING, POS_DEAD},
        {"HP 0 stunned stays stunned", 0, POS_STUNNED, POS_STUNNED},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            player := testutil.NewTestPlayer("tester", 3, 0)
            player.Health = tt.hitPoints
            player.Position = tt.startPos

            combat.UpdatePosition(player)

            testutil.AssertBehaviorMatchesC(t,
                tt.name+" position",
                func() string { return fmt.Sprintf("%d", player.Position) },
                fmt.Sprintf("%d", tt.expected),
            )
        })
    }
}
```

**What this catches:** Off-by-one in position thresholds, wrong position constants, missing edge cases, reordered conditionals.

---

## Level 4: Menu Text Comparison (Race Menu)

The simplest but most visible — exact string comparison against C source text.

**C source** (constants.c:196):
```c
const char *race_menu =
"\r\n"
"Choose a race:\r\n"
"  [H]uman        [E]lven       [D]warven      [K]enderkin\r\n"
"  [M]inotaur     [R]akshasan   [S]sauran\r\n"
"  [?]Help on races in general\r\n"
"  [?<race abbreviation>] Help on a specific race (i.e ?D for help on dwarves)"
"\r\n";
```

**Go test:**
```go
func TestRaceMenuText_Fidelity(t *testing.T) {
    cExpected := "\r\n" +
        "Choose a race:\r\n" +
        "  [H]uman        [E]lven       [D]warven      [K]enderkin\r\n" +
        "  [M]inotaur     [R]akshasan   [S]sauran\r\n" +
        "  [?]Help on races in general\r\n" +
        "  [?<race abbreviation>] Help on a specific race (i.e ?D for help on dwarves)" +
        "\r\n"

    testutil.AssertBehaviorMatchesC(t,
        "race_menu text",
        func() string { return session.RaceMenuText },
        cExpected,
    )
}
```

**What this catches:** Typos, missing `\r\n`, changed wording, reordered races, missing `?` help option.

---

## Implementation Priority

| Priority | Test Target | Why |
|----------|-------------|-----|
| 1 | Race/class/hometown menu text | Already ported, verify exact match |
| 2 | Attack hit text table | 15 entries, simple comparison |
| 3 | Position update state machine | 8 branches, critical for combat |
| 4 | Damage formula (min/max) | Core combat loop |
| 5 | Saving throw formula | Affects every spell |
| 6 | Movement sector costs | 12 sector types, simple mapping |
| 7 | Spell damage dice | 40+ spells to verify |
| 8 | Skill check formulas | Backstab, sneak, hide |

---

## Pattern Summary

Every fidelity test follows the same pattern:

1. **Find the C source** — exact file and line number
2. **Determine the expected output** — for a specific input, what does C produce?
3. **Call the Go function** — with the same input
4. **Assert equality** — using `AssertBehaviorMatchesC`

The helper handles the comparison and reporting. The test writer handles the C source research.

For simple value tables (arrays, maps), loop through and compare each entry.
For formulas, test with known inputs and verify outputs.
For state machines, enumerate every branch and verify the transition.
For text, exact string comparison.
