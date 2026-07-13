# BRIEF — COV-4: Spec proc smoke test + golden fidelity for top-10 specs (DP-965)

**Linear:** DP-965 (COV-4: Spec proc smoke test + golden fidelity for top-10 specs)
**Effort:** M
**Agent:** Kimi
**Source of truth:** docs/reports/REVIEW-2026-07-05-full-audit.md — §3C item 4

## Goal

Create a two-layer test suite for the spec proc system:

1. **Smoke test (all 122 specs):** For every registered spec proc, invoke with a benign command and assert no panic. Catches nil-pointer crashes (the F5/F6 defect class).
2. **Golden fidelity tests (~10 specs):** Behavioral tests for the most-encountered spec procs, asserting correct command dispatch, return values, and player-visible output against C source behavior.

## Background

### The spec proc system

Spec procs are the DikuMUD "special procedure" system — event-driven callbacks attached to mobs, objects, and rooms. When a player types a command, the game checks whether any spec-bearing entity in the room (or the player's equipment/inventory) wants to intercept it.

**Signature** (`pkg/game/spec_assign.go:10`):
```go
type SpecFunc func(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool
```
- `cmd == ""` means the mob triggers on its own pulse/tick (autonomous behavior)
- `cmd != ""` means the player typed a command; if the spec returns `true`, the command is consumed
- `me` is the mob/room that owns the spec; `nil` for object specs and room specs dispatched without a mob
- `ch` is the player who triggered the spec; `nil` during autonomous mob ticks

**Registration** (`pkg/game/spec_procs.go:981-1006`, `spec_procs2.go`, `spec_procs3.go`, `spec_procs4.go`, `spec_procs_missing.go`, `boards.go`, `postmaster.go`):
- 122 specs registered via `RegisterSpec(name, fn)` in `init()` functions
- Lookup: `GetMobSpec(vnum)`, `GetObjSpec(vnum)`, `GetRoomSpec(vnum)` — each resolves VNum → name → `SpecFunc` via `SpecRegistry` map

**Dispatch** (`pkg/session/commands.go:526-594`):
1. Fast-path: `HasSpecInRoom(roomVNum)` gates mob/room/room-item scans
2. Equipment and inventory scans are unconditional
3. First spec that returns `true` consumes the command

**Autonomous dispatch** (`pkg/game/mobact.go:55-64`):
- `callMobSpecSafely` invokes mob specs with `cmd=""` during the AI tick
- Wraps in `defer recover()` to prevent panics from crashing the server

### What the audit found

- **F5 [HIGH]**: `specDump` swallowed the `drop` command — never performed the actual drop, so the XP/gold award was dead code. *This was fixed in QA residuals (PR #110).*
- **F6 [MEDIUM]**: `randN(N)` is exclusive [0,N) but C's `number(0,N)` is inclusive [0,N]. Every `== 0` gate fires slightly too often across ~42 call sites. *Not yet fixed.*
- **F11 [LOW]**: `specSnake` loses TO_VICT message form — victim sees room message instead of "$n bites you!".

These findings demonstrate the defect class: spec procs are fragile because many access `me.*` fields or `ch.*` fields without nil checks, and behavioral regressions (wrong return value, missing messages, wrong RNG) are invisible without tests.

### Existing tests

`pkg/game/spec_procs_test.go` (234 lines) has:
- `TestSpecDumpPerformDrop` — specDump drops item, awards gold
- `TestSpecDumpNonDropPassThrough` — specDump returns false for non-drop
- `TestSpecHorn_UseHornDoesNotPanic` — object-spec dispatch, nil `me` (QA #3 fix)
- `TestSpecDump_ExpGoldLocked` — concurrent specDump gold/exp writes
- `TestActMessageAudienceRouting` — actMessage helper routing

Helper: `newSpecProcTestWorld(t)` creates a minimal world with one room (VNum 1001), one obj (VNum 3001, sword), and a level-5 player.

## Fix

### Part 1: Smoke test — no-panic invocation for all 122 specs

Create a table-driven test that iterates over `AllSpecNames()` (from `spec_assign.go:445`) and calls each spec with a benign command. Assert no panic.

```go
func TestSpecProc_SmokeAll(t *testing.T) {
    w, player, _ := newSpecProcTestWorld(t)
    mob := NewMobInstance(/*需要一个有效的MobPrototype */)

    names := AllSpecNames()
    for name := range names {
        t.Run(name, func(t *testing.T) {
            fn := SpecRegistry[name]
            if fn == nil {
                t.Skipf("spec %q registered but function is nil", name)
            }

            // 测试1: benign command, player present, no mob
            // 大多数specs在cmd不匹配时return false;不应该panic
            assertNotPanic(t, func() {
                fn(w, player, nil, "look", "")
            })

            // 测试2: benign command with mob present
            assertNotPanic(t, func() {
                fn(w, player, mob, "look", "")
            })

            // 测试3: autonomous tick (cmd=""), no player
            // 这模拟mobact的callMobSpecSafely路径
            assertNotPanic(t, func() {
                fn(w, nil, mob, "", "")
            })
        })
    }
}
```

**Key requirements:**
- Use `AllSpecNames()` to enumerate — do NOT hardcode spec names
- Some specs access `me.GetPosition()`, `me.RoomVNum`, `me.GetLevel()` etc. — create a realistic `MobInstance` with position set to standing, level > 0, room set to a valid room
- Some specs access `ch.GetRoomVNum()`, `ch.GetName()`, `ch.GetLevel()` — the player from `newSpecProcTestWorld` satisfies this
- Wrap each call in `defer func() { recover() }()` to catch panics and `t.Fatalf`
- Some specs call `w.doDrop`, `w.actMessage`, `spells.Cast` — these should not panic even if the action has no effect (e.g., no item to drop)

### Part 2: Golden fidelity tests for the top-10 specs

These are the specs with the most VNUM assignments (most encounters in-game). For each, test the specific behaviors the C source defines.

#### 2a. `specGuild` (12 mob VNums — guild masters)

C source: `src/spec_procs.c` — `spec_guild()`
Go: `pkg/game/spec_procs.go:133-180`

**Behaviors to test:**
1. `cmd != "practice"` → returns false (pass-through)
2. `cmd == "practice"`, `arg == ""` → returns true, sends "Practise what?" + skill list
3. `cmd == "practice"`, `arg == "nonexistent"` → returns true, sends "You do not know of that skill."
4. `cmd == "practice"`, skill is learned and below max → returns true, calls `PracticeSkill`, sends "You practice for a while..."
5. NPC player → returns false immediately

**Test approach:** Create a player with a `SkillManager` that has known skills. Call specGuild with various cmd/arg combos. Assert return values and message output.

#### 2b. `specDump` (room spec — dump rooms)

C source: `src/spec_procs.c` — `spec_dump()`
Go: `pkg/game/spec_procs.go:183-209`

**Behaviors to test:**
1. Non-drop command → returns false
2. Drop command with item in inventory → item removed from inventory, appears briefly in room, gold/XP awarded per C formula (`MAX(1,MIN(10,COST/10))`)
3. Drop command with no matching item → returns true (command consumed), no gold awarded
4. Level < 3 player gets EXP; level >= 3 gets gold

Existing tests cover most of this. Add a test for the C-fidelity value formula: cost 100 sword at level 5 → `clamp(100/10, 1, 10) = 10` gold.

#### 2c. `specSnake` (3 mob VNums — snakes)

C source: `src/spec_procs.c` — `spec_snake()`
Go: `pkg/game/spec_procs.go:212-231`

**Behaviors to test:**
1. `cmd != ""` → returns false (only triggers on autonomous tick)
2. `me.GetPosition() != PosFighting` → returns false
3. Fighting but `number(0, 32-level) != 0` → returns false (RNG gate)
4. When bite triggers → victim gets poison spell, correct messages sent via actMessage

**Test approach:** Set mob position to fighting, put a player victim in room, mock the RNG or test the non-trigger path. The RNG gate (`number(0, 32-level) == 0`) is hard to test deterministically — test the guard conditions that return false, and verify no panic on the trigger path.

#### 2d. `specThief` (5 mob VNums — thief mobs)

C source: `src/spec_procs.c` — `spec_thief()`
Go: `pkg/game/spec_procs.go:271-299`

**Behaviors to test:**
1. `cmd != ""` → returns false (autonomous only)
2. `me.GetPosition() != PosStanding` → returns false
3. All players in room are level >= 50 → no steal target, returns false
4. Victim position > sleeping + RNG → "tries to steal" message, no gold lost
5. Victim position <= sleeping + RNG → gold stolen (percentage of victim's gold)

**C reference** (`spec_procs.c`): thief steals `(gold * number(1,10)) / 100`.

#### 2e. `specMagicUser` (55 mob VNums — casters)

C source: `src/spec_procs.c` — `spec_magic_user()`
Go: `pkg/game/spec_procs.go:302-366`

**Behaviors to test:**
1. `cmd != ""` → returns false
2. Not fighting → returns false
3. Fighting with target → casts a spell based on `spellRoll` (random level/2 + level/2)
4. Spell selection: roll ≤ 5 → magic missile, 6-7 → chill touch, 8-10 → blindness, 11-12 → sleep, 13+ → fireball

**Test approach:** Set mob to fighting state with a target in room. The RNG makes exact spell selection non-deterministic — test the guard conditions and verify no panic on the cast path. Assert the function returns true when conditions are met.

#### 2f. `specFighter` (13 mob VNums)

C source: `src/spec_procs.c` — `spec_fighter()`
Go: `pkg/game/spec_procs.go:368-390`

**Behaviors to test:**
1. `cmd != ""` → returns false
2. Not fighting → returns false
3. Fighting → returns true, performs melee (calls `one_hit`)

#### 2g. `specCleric` (17 mob VNums)

C source: `src/spec_procs2.c` — `spec_cleric()`
Go: `pkg/game/spec_procs3.go` (registered as "cleric")

**Behaviors to test:**
1. `cmd != ""` → returns false
2. Not fighting → returns false
3. Fighting, low HP → may heal self (`dispel_evil` or `heal` cast on self)
4. Fighting, healthy → may cast offensive spell (cure serious, cause serious, etc.)

#### 2h. `specCityguard` (8 mob VNums)

C source: `src/spec_procs.c` — `spec_cityguard()`
Go: `pkg/game/spec_procs.go:558-630`

**Behaviors to test:**
1. `cmd != ""` → returns false
2. Scans for outlaws (players with WANTED flag or alignment < -350) → attacks
3. No outlaws → scans for evil players attacking good players → intervenes
4. No targets → returns false

#### 2i. `specDragonBreath` (6 mob VNums)

C source: `src/spec_procs.c` — `spec_dragon_breath()`
Go: `pkg/game/spec_procs.go:754-775`

**Behaviors to test:**
1. `cmd != ""` → returns false
2. Not fighting → returns false
3. Fighting → returns true, casts breath weapon (fire/acid/gas/frost based on mob type)

#### 2j. `specJanitor` (2 mob VNums)

C source: `src/spec_procs.c` — `spec_janitor()`
Go: `pkg/game/spec_procs.go:539-556`

**Behaviors to test:**
1. `cmd != ""` → returns false (autonomous only)
2. Room has items → picks up first item (obj_from_room + obj_to_char), returns true
3. Room empty → returns false
4. Verify item is actually added to mob's inventory

## Files

| File | Change |
|---|---|
| `pkg/game/spec_procs_test.go` | Add `TestSpecProc_SmokeAll` + ~10 golden fidelity test functions |

## Spec proc reference summary

**Total registered:** 122 specs across 7 Go source files + 2 board/postmaster files

| Source file | Registered specs | Notes |
|---|---|---|
| `spec_procs.go` | 23 | Core Merc specs (guild, dump, snake, thief, magic_user, etc.) |
| `spec_procs2.go` | 43 | Zone-specific specs (castle guards, jail, medusa, etc.) |
| `spec_procs3.go` | 31 | Complex specs (cleric, troll, elements puzzle, shop_keeper, etc.) |
| `spec_procs4.go` | 18 | Newer specs (bank, horn, elevator, conductor, dracula, etc.) |
| `spec_procs_missing.go` | 5 | Previously missing specs (no_get, recharger, beholder, zen_master, moon_gate) |
| `boards.go` | 1 | gen_board (bulletin board system) |
| `postmaster.go` | 1 | postmaster (mail system) |

**Top-10 by VNUM assignment count:**
1. `magic_user` — 55 mobs
2. `cleric` — 17 mobs
3. `fighter` — 13 mobs
4. `guild` — 12 mobs
5. `gen_board` — 12 objs (bulletin boards)
6. `moon_gate` — 10 rooms
7. `cityguard` — 8 mobs
8. `guild_guard` — 7 mobs
9. `dragon_breath` — 6 mobs
10. `thief` — 5 mobs

**Spec type distribution:**
- **Mob specs** (trigger on player command or autonomous tick): ~90% — called with `(w, ch, mob, cmd, arg)`. `me` is non-nil.
- **Obj specs** (trigger on player command with object in room/equip/inventory): ~8% — called with `(w, ch, nil, cmd, arg)`. `me` is nil.
- **Room specs** (trigger on player command in room): ~2% — called with `(w, ch, nil, cmd, arg)`. `me` is nil.

## Implementation approach

### Smoke test

1. Call `AllSpecNames()` to get the full set
2. For each name, look up `SpecRegistry[name]`
3. Create a shared test world with `newSpecProcTestWorld(t)`, add a mob
4. For each spec, invoke three times: benign command (with and without mob), and autonomous tick
5. Catch panics with `defer/recover` pattern, report the spec name on failure

### Golden tests

1. Follow the existing pattern from `TestSpecDumpPerformDrop` / `TestSpecHorn_UseHornDoesNotPanic`
2. Use `newSpecProcTestWorld` for basic setup, extend as needed (add mobs, items, set positions)
3. Each golden test should be its own `TestSpec*` function for clear failure reporting
4. For RNG-dependent specs (snake, thief, magic_user): test the guard conditions exhaustively (return false for every non-trigger condition), and verify no panic on the trigger path
5. For command-dispatch specs (guild, dump): test all command/arg combinations

### Mob setup for golden tests

Most combat-oriented specs need a mob that's:
- `me.SetPosition(combat.PosFighting)` or `combat.PosStanding`
- `me.SetRoom(roomVNum)` to a valid room
- `me.SetLevel(N)` for RNG-dependent specs
- `me.SetFighting(playerName)` for combat specs

Create a helper like:
```go
func newTestMob(w *World, vnum int, roomVNum int, level int) *MobInstance {
    proto := w.mobs[vnum] // or a minimal proto if vnum not in world
    mob := NewMob(proto)
    mob.SetRoom(roomVNum)
    mob.SetLevel(level)
    mob.SetPosition(combat.PosStanding)
    return mob
}
```

For the smoke test, you'll need at least one mob proto in the test world. Either use a VNUM from the parser data or create a minimal `MobPrototype` directly.

## Build gate

```bash
go build ./...
go vet ./...
go test -race $(go list ./... | grep -v /tests/unit) -timeout 120s
gofumpt -l .
golangci-lint run ./...
```

## Constraints

1. **Do NOT modify any production spec proc code.** These are tests only.
2. **Use `AllSpecNames()` for the smoke test.** Do NOT hardcode spec names — the registry is the source of truth.
3. **Follow existing test patterns.** Use `newSpecProcTestWorld`, match message capture patterns from existing tests.
4. **Object/room specs are called with `me = nil`.** The smoke test must cover this — many specs crashed on nil `me` before the QA #3 fix (specHorn).
5. **RNG-dependent specs: test guards, not exact outcomes.** `number(0, 32-level) == 0` is non-deterministic. Test that the function returns false when guards fail, and doesn't panic when they pass. Do NOT try to control the RNG.
6. **Keep golden tests focused.** Test the most important behavior per spec. Don't try to test every code path — the smoke test catches panics, the golden tests catch behavioral regressions.
7. Single PR. Commit message: `test: spec proc smoke test + golden fidelity for top-10 specs (COV-4)`

## C fidelity notes

The golden tests should match C behavior where specified:
- `specDump` value formula: `MAX(1, MIN(10, COST/10))` per item (C: `spec_procs.c`)
- `specThief` gold theft: `(victim.gold * number(1,10)) / 100` (C: `spec_procs.c`)
- `specSnake` poison probability: `number(0, 32-level) == 0` (C: `spec_procs.c`)
- `specGuild` practice: requires skill learned, below max level, has practice points

Note F6: `randN(N)` is [0,N) but C's `number(0,N)` is [0,N]. The spec procs already use the `number(from,to)` helper (defined in `spec_procs.go` area or `remort_helpers.go:4`) for most C-fidelity ports. If you see bare `randN()` where C has `number()`, that's the F6 off-by-one — note it but don't fix it in this PR.

## Documentation value

The smoke test serves as a living contract: every registered spec proc must not panic on benign input. If a new spec is added but AllSpecNames() doesn't pick it up (because the VNum→name map wasn't updated), the test still catches it via SpecRegistry enumeration. The golden tests lock in the behavioral contract for the most impactful specs.
