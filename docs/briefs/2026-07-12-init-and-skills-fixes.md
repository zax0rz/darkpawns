# Dark Pawns — init_char() Fidelity + Skill Assignment Refactor (2 Issues)

**Target files:**
- `pkg/game/player.go` — player initialization (`NewCharacter()`)
- `pkg/game/character.go` — `GiveStartingSkills()`

**Repo:** `/Users/zach/.openclaw/workspace/darkpawns_repo`
**Branch:** Create from `main`, name `fix/creation-fidelity-init-and-skills`
**After fixing:** Run `go build ./... && go vet ./... && go test ./...`. All must pass.
**Push:** `git push origin fix/creation-fidelity-init-and-skills`

---

## Fix 1: Add height/weight randomization (DP-1075)

**File:** `pkg/game/player.go`

**C source** (db.c:3041-3047) in `init_char()`:
```c
/* make favors for sex */
if (ch->player.sex == SEX_MALE) {
    ch->player.weight = number(120, 180);
    ch->player.height = number(160, 200);
} else {
    ch->player.weight = number(100, 160);
    ch->player.height = number(150, 180);
}
```

**Go currently:** No `Height` or `Weight` fields exist anywhere on the `Player` struct. These are character physical attributes — separate from `CarriedWeight`, `MaxCarryWeight`, etc.

**What to add:**

1. Add fields to the `Player` struct (near line ~25, in the identity/core stats section):
```go
Height int // Height in cm — randomized by sex at creation (db.c:3041-3047)
Weight int // Weight in kg — randomized by sex at creation (db.c:3041-3047)
```

2. Add JSON tag (lowercase, snake_case or camelCase — match the existing pattern in the struct). Look at how other fields are tagged (e.g. `WimpLevel int \`json:"wimp_level"\``).

3. In `NewCharacter()` (player.go, around line ~310 after the `p.Drunk = 0` line), add height/weight initialization:
```go
// Random height/weight by sex — db.c:3041-3047
rand.Seed(time.Now().UnixNano()) // if not already seeded
if p.Sex == 0 { // SEX_MALE = 0
    p.Weight = 120 + rand.Intn(61)  // 120-180
    p.Height = 160 + rand.Intn(41)  // 160-200
} else {
    p.Weight = 100 + rand.Intn(61)  // 100-160
    p.Height = 150 + rand.Intn(31)  // 150-180
}
```

**Note on Go random:** Check if the codebase uses `math/rand` or `crypto/rand` or some other RNG pattern elsewhere. Use whatever pattern the codebase already uses for randomization. Do NOT use `rand.Seed()` if the module uses the newer `rand.New(rand.NewSource(...))` pattern — Go 1.20+ deprecates `rand.Seed`. Check `go.mod` for the Go version.

**Also from C `init_char()` — check if Go already handles these:**

| C field | C value | Go equivalent | Status |
|---------|---------|---------------|--------|
| `ch->player.time.birth` | `time(0)` | `Birth: now.Unix()` | ✅ Already set in `NewPlayer()` |
| `ch->player.time.played` | `0` | `PlayedDuration: 0` | ✅ Already set in `NewPlayer()` |
| `ch->player.time.logon` | `time(0)` | `ConnectedAt: now` | ✅ Already set in `NewPlayer()` |
| `ch->points.armor` | `100` | `AC: 10` in `NewPlayer()` | ⚠️ Check: C says 100, Go says 10. Read the AC scale. If C AC is inverted (lower= worse, like CircleMUD), 100 base is correct and Go's 10 is wrong. If Go AC is D&D-style (lower=better), 10 is correct. **Do NOT change this without understanding the AC scale.** Leave it for now if unsure. |
| All skills to 0 | `SET_SKILL(ch, i, 0)` | `InitializeDefaultSkills()` | ✅ Handled |
| `GET_ORIG_CON(ch) = GET_CON(ch)` | Save original constitution | Not present | Skip — not used yet |
| Weight/Height | Random by sex | Not present | ❌ **Fix this** |

**IMPORTANT:** Do NOT change `AC` unless you understand the CircleMUD AC scale. The fidelity review flagged it but changing it without understanding the combat formula could break combat. Leave a comment if it looks wrong but don't change it.

---

## Fix 2: Consolidate duplicate skill assignments (DP-1079)

**Files:** `pkg/game/player.go` (lines 330-352) and `pkg/game/character.go` (lines 296-316)

**The problem:** Two functions set starting skills for the same classes:

1. `NewCharacter()` in `player.go` sets thief/assassin skills (sneak, hide, steal, backstab, pick_lock), kender steal, warrior bash/rescue, and kick for all classes.
2. `GiveStartingSkills()` in `character.go` sets thief/assassin skills (sneak, hide, **peek**, steal, backstab, pick_lock), kender steal, and minotaur headbutt.

Both are called during character creation — `NewCharacter()` first, then `GiveStartingSkills()` in `completeCharCreation()`. Since `SetSkill` overwrites, the second call wins. But the overlap is confusing and error-prone.

**C source** (class.c:554-570) only sets skills in `do_start()`:
```c
case CLASS_THIEF:
case CLASS_ASSASSIN:
    SET_SKILL(ch, SKILL_SNEAK, 10);
    SET_SKILL(ch, SKILL_HIDE, 5);
    SET_SKILL(ch, SKILL_PEEK, 15);   // ← peek is here in C
    SET_SKILL(ch, SKILL_STEAL, 15);
    SET_SKILL(ch, SKILL_BACKSTAB, 10);
    SET_SKILL(ch, SKILL_PICK_LOCK, 10);
    break;

if (GET_RACE(ch) == RACE_KENDER)
    SET_SKILL(ch, SKILL_STEAL, 25);
if (GET_RACE(ch) == RACE_MINOTAUR)
    SET_SKILL(ch, SKILL_HEADBUTT, 25);
```

C's `do_start()` does NOT grant kick, bash, or rescue to any class. Those are Go additions.

**Fix:** Remove ALL skill assignments from `NewCharacter()` in `player.go` (lines ~330-352). Keep `GiveStartingSkills()` in `character.go` as the single source of truth for starting skills. This makes `GiveStartingSkills()` match the C source exactly, with the Go-specific additions (kick, bash, rescue) clearly visible as non-C skills if they're intentional.

**Decision needed on Go-specific skills:** The Go code grants `kick` to all classes and `bash`/`rescue` to warriors. C does NOT do this. These are either:
- **Intentional additions** — keep them in `GiveStartingSkills()` with a comment saying "Go addition — not in C source"
- **Accidental additions** — remove them

**For now: move them to `GiveStartingSkills()` with a comment marking them as Go-specific.** Do not remove them. The decision to keep or cut can be made later.

**After the fix, `NewCharacter()` should NOT contain any `SetSkill()` calls.** All skill initialization lives in `GiveStartingSkills()` only.

---

## Summary

| # | Issue | File | Change |
|---|-------|------|--------|
| 1 | DP-1075 | player.go | Add `Height`/`Weight` fields, randomize by sex in `NewCharacter()` |
| 2 | DP-1079 | player.go + character.go | Remove skill assignments from `NewCharacter()`, consolidate into `GiveStartingSkills()` |

**Commit message:** `fix: character creation fidelity — height/weight init, skill consolidation (DP-1075 DP-1079)`

**After commit:** `go build ./... && go vet ./... && go test ./...` — must pass. Then push.
