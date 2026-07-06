# Brief: Stream 3 — Fidelity Polish (F6, F7, F19)

## Context

Three fidelity issues from the Fable Audit 2026-07-05. Combat feel (F7) is the
highest-impact; spec proc probability drift (F6) is a systematic fix; F19 is
scoped down after triage (black_horn is dead code in C too).

**Linear issues:** DP-949 (F6), DP-950 (F7), DP-951 (F19)
**Branch:** `fix/stream3-fidelity-polish`
**Agent:** Kimi (implementation), Claude (review)

## Execution Order

1. **F6** (DP-949) — randN fix, foundational for all spec procs
2. **F7** (DP-950) — combat messages, highest player-facing impact
3. **F19** (DP-951) — rent spec procs, verify parity only

---

## F6 (DP-949): randN off-by-one — spec proc probabilities drift from C

**Priority:** Medium | **Labels:** fidelity, drift | **Effort:** S

### Problem

`randN(N)` at `pkg/game/spec_procs.go:39` ports C's inclusive `number(0,N)` but
uses `rand.IntN(n)` which is exclusive `[0,N)`. Every probability gate fires
slightly too often.

| Expression | C range | Go range | P(==0) |
|---|---|---|---|
| `number(0,32)==0` | `[0,32]` (33 values) | `[0,32)` (32 values) | C: 1/33, Go: 1/32 |

44 total call sites across `spec_procs.go`, `spec_procs2.go`,
`spec_procs_missing.go`. Broken into categories:

- **15 sites**: `randN(N) == 0` gates — fire too often
- **14 sites**: `randN(N) != 0` gates — fire too rarely (inverse)
- **2 sites**: `randN(N) >= X` comparisons — distribution shifted
- **10 sites**: `randN(len(array))` for array indexing — **correct** (need exclusive range)
- **3 sites**: `switch randN(N)` for C `number(0,N-1)` — **correct** (accidentally match)

### C Reference

`src/utils.c:53` — `number(from, to)` returns `[from, to]` inclusive:
```c
return (int)(uniform() * (to - from + 1)) + from;
```

### Fix Strategy

**Do NOT change the `randN` definition** — the array-index sites correctly need
exclusive range. Instead:

1. Introduce `number(from, to int) int` in `spec_procs.go` that returns
   `rand.IntN(to - from + 1) + from` — matching C's `number()` semantics.
   (`remort_helpers.go:4` already has a `number()` function — check if it's
   the same signature and reuse it, or move it to a shared location.)

2. Replace all `randN(N) == 0` gates with `number(0, N) == 0`.
   Replace all `randN(N) != 0` guards with `number(0, N) != 0`.
   Replace all `randN(N) >= X` comparisons with `number(0, N) >= X`.

3. Leave `randN(len(array))` array-index sites untouched.
   Leave `switch randN(N)` selection sites untouched (they happen to produce
   the correct range for their purpose).

4. Add a comment to `randN` clarifying it's exclusive (for array indexing),
   and a comment to `number()` clarifying it's inclusive (for C-ported code).

### Affected Sites (gates only — 31 total)

**spec_procs.go** (lines approximate, verify before editing):
- `:181` — `randN(32-me.GetLevel()) != 0` → `number(0, 32-me.GetLevel()) != 0`
- `:220` — `randN(4) == 0` → `number(0, 4) == 0`
- `:236` — `randN(5) == 0` → `number(0, 5) == 0`
- `:248` — `randN(me.GetLevel()) == 0` → `number(0, me.GetLevel()) == 0`
- `:269` — `randN(5) == 0` → `number(0, 5) == 0`
- `:300` — `randN(11) == 0` → `number(0, 11) == 0`
- `:467` — `randN(91) == 0` → `number(0, 91) == 0`
- `:477` — `randN(3) != 0` → `number(0, 3) != 0`
- `:500` — `randN(5) != 0` → `number(0, 5) != 0`
- `:505` — `randN(2) == 0` → `number(0, 2) == 0`
- `:719` — `randN(4) != 0` → `number(0, 4) != 0`
- `:738` — `randN(8) != 0` → `number(0, 8) != 0`
- `:778` — `randN(4) != 0` → `number(0, 4) != 0`
- `:797` — `randN(3) != 0` → `number(0, 3) != 0`
- `:801` — `randN(2) == 0` → `number(0, 2) == 0`
- `:843` — `randN(3) == 0` → `number(0, 3) == 0`
- `:877` — `randN(3) == 0` → `number(0, 3) == 0`

**spec_procs2.go**:
- `:498` — `randN(6) != 0` → `number(0, 6) != 0`
- `:793` — `randN(5) != 0` → `number(0, 5) != 0`
- `:796` — `randN(31) == 0` → `number(0, 31) == 0`
- `:830` — `randN(6) != 0` → `number(0, 6) != 0`
- `:897` — `randN(100) >= pl.GetLevel()*2` → `number(0, 100) >= ...`
- `:907` — `randN(6) != 0` → `number(0, 6) != 0`
- `:927` — `randN(101) > 20` → `number(0, 101) > 20`
- `:956` — `randN(2) != 0` → `number(0, 2) != 0`
- `:1020` — `randN(5) != 0` → `number(0, 5) != 0`
- `:1076` — `randN(4) == 0` → `number(0, 4) == 0`
- `:1185` — `randN(5) == 0` → `number(0, 5) == 0`
- `:1203` — `randN(3) == 0` → `number(0, 3) == 0`
- `:1224` — `randN(4) == 0` → `number(0, 4) == 0`

**spec_procs_missing.go**:
- `:158` — `randN(6) != 0` → `number(0, 6) != 0`

### Regression Test

```go
func TestNumberIsInclusive(t *testing.T) {
    // Verify number(0, N) can return N, unlike randN(N)
    hits := make(map[int]bool)
    for i := 0; i < 10000; i++ {
        hits[number(0, 5)] = true
    }
    for v := 0; v <= 5; v++ {
        if !hits[v] {
            t.Errorf("number(0,5) never produced %d in 10000 iterations", v)
        }
    }
    // Verify randN stays exclusive (for array indexing)
    hits2 := make(map[int]bool)
    for i := 0; i < 10000; i++ {
        hits2[randN(5)] = true
    }
    if hits2[5] {
        t.Error("randN(5) produced 5 — should be exclusive [0,5)")
    }
}
```

---

## F7 (DP-950): Live combat messages bypass golden-tested dam_message tables

**Priority:** Medium | **Labels:** fidelity, drift | **Effort:** M

### Problem

`engine.go`'s `sendHitMessage` (`:414-431`) and `sendMissMessage` (`:434-447`)
output flat generic strings:
```
"You hit Bob for 15 damage!"
"Bob hits you for 15 damage!"
```

C's `dam_message` (`src/fight.c:889-1015`) selects severity-based verbs from a
12-tier table ("barely scratch", "massacre", "OBLITERATE") with weapon-type
substitution (`#w`/`#W`). Go already has the full table in
`fight_core.go:680-935` (`damMessageTiers`, 14 tiers with variants) and
`DamMessage()` (`:937-968`) — golden-tested for tier mapping — but
`engine.go` never calls it.

Two completely disconnected message systems:
| Path | Function | Used by |
|---|---|---|
| `fight_core.go` | `DamMessage()` → `damMessageTiers` | Skill/spell combat only |
| `engine.go` | `sendHitMessage()`/`sendMissMessage()` | Live combat engine tick |

### C Reference

`src/fight.c:889-1015` — `dam_message(dam, ch, victim, dt)`:
1. Select `msgnum` by damage threshold (12 tiers, 0-11)
2. Look up `dam_weapons[msgnum]` for to_room/to_char/to_victim strings
3. Replace `#w` with weapon singular, `#W` with weapon plural
4. Send via `act()` with TO_CHAR, TO_VICT, TO_NOTVICT

### Fix Strategy

1. **Add a `MessageFunc` callback to `CombatEngine`** that routes through the
   existing message infrastructure. The engine shouldn't import fight_core
   directly (circular risk).

2. **Wire it at boot** in `cmd/server/main.go` alongside the other combat
   callbacks. The callback should:
   - For misses: call `DamMessage(0, attacker, defender, attackType)` which
     handles tier 0 (miss messages)
   - For hits: call `DamMessage(damage, attacker, defender, attackType)`
   - Return `bool` — if handled, skip the fallback generic message

3. **Modify `processCombatPair`** to pass `attackType` to `sendHitMessage` and
   `sendMissMessage`. `pair.LastAttackType` is already tracked (`:378`).
   Add `attackType int` parameter to both message functions.

4. **Fallback**: If `MessageFunc` is nil (testing), fall back to current generic
   messages. This keeps engine unit tests working without wiring.

### Key Code Locations

- `pkg/combat/engine.go:414-447` — current generic message functions (MODIFY)
- `pkg/combat/engine.go:338-388` — call sites in `processCombatPair` (MODIFY to pass attackType)
- `pkg/combat/engine.go:24-65` — CombatEngine struct (ADD `MessageFunc` field)
- `pkg/combat/fight_core.go:937-968` — `DamMessage()` (EXISTS, golden-tested)
- `pkg/combat/fight_core.go:680-935` — `damMessageTiers` table (EXISTS)
- `cmd/server/main.go` — combat callback wiring (ADD MessageFunc)

### Important Notes

- `DamMessage()` expects full `Combatant` interface objects (attacker, defender).
  The engine already has these — pass them through.
- The `fight_core.go` message tables use `$n`/`$N`/`#w`/`#W` token replacement
  via `replaceMessageTokens()`. Verify this works for weapon attacks.
- Go has 14 tiers vs C's 12 (extra high-damage tiers 10-12). This is intentional
  and already golden-tested. Keep Go's expanded table.
- `DamMessage()` uses the package-level `BroadcastMessage` and `SendToCharFunc`
  hooks. These must be wired at boot for the messages to deliver.

### Regression Test

```go
func TestSendHitMessageUsesDamMessageFunc(t *testing.T) {
    ce := NewCombatEngine()
    var gotDam, gotAttackType int
    ce.MessageFunc = func(dam int, atkType int) bool {
        gotDam = dam
        gotAttackType = atkType
        return true
    }
    // Stub combatants
    atk := &mockCombatant{name: "Alice", room: 1}
    def := &mockCombatant{name: "Bob", room: 1}
    ce.sendHitMessage(atk, def, 25, 5)
    if gotDam != 25 {
        t.Errorf("MessageFunc got dam=%d, want 25", gotDam)
    }
    if gotAttackType != 5 {
        t.Errorf("MessageFunc got attackType=%d, want 5", gotAttackType)
    }
}

func TestSendHitMessageFallsBackWhenNil(t *testing.T) {
    ce := NewCombatEngine()
    // No MessageFunc — should use generic fallback
    atk := &mockCombatant{name: "Alice", room: 1}
    def := &mockCombatant{name: "Bob", room: 1}
    ce.sendHitMessage(atk, def, 25, 5)
    // Verify attacker got generic "You hit Bob for 25 damage!" message
    if !strings.Contains(atk.lastMsg, "You hit Bob for 25 damage") {
        t.Errorf("fallback message wrong: %q", atk.lastMsg)
    }
}
```

---

## F19 (DP-951): black_horn dead code in C; rent system is dead infrastructure in Go

**Priority:** Low | **Labels:** dead-code | **Effort:** S

### Problem (fully rescoped after triage)

**black_horn is dead code in both codebases.** C declares `SPECIAL(black_horn)` at
`src/new_cmds2.c:624-654` but never assigns it via `ASSIGNOBJ(14500, black_horn)`.
The item isn't placed in any room, carried by any mob, or sold in any shop. **Do
not port this.**

**Rent system is dead infrastructure in Go.** Dark Pawns replaced the C rent
paradigm — players save to SQLite on disconnect, items are always preserved,
no cost, no NPC interaction needed. The ~400 lines of rent code in
`pkg/game/objsave.go` (`OfferRent`, `RentSave`, `CryoSave`, `CrashSave`,
`GenReceptionist`, `ExtractNorents`, `SaveAllPlayers`, `DeleteCrashFile`,
and all rent constants) are never called from outside the file. No rent command
is registered. The `OnAutoSave` gameloop callback is never wired. Rent metadata
fields (`RentCode`, `RentTime`, `NetCostPerDiem`) exist in the save schema but
are always zero.

Even in C, rent was trivialized (`free_rent=YES` means zero cost, items always
preserved) and the receptionist/cryogenicist NPCs were declared but **never
assigned to any mob vnum** — same class as black_horn.

This is the same pattern as the ZoneDispatcher deleted in Stream 2 (DP-948):
implemented but unwired. Per the U1000 ratchet, limbo is worse than either
choice.

### Fix Strategy — Delete the dead rent code

1. **Delete from `pkg/game/objsave.go`:**
   - `OfferRent()`
   - `RentSave()`
   - `CryoSave()`
   - `CrashSave()`
   - `SaveAllPlayers()`
   - `GenReceptionist()`
   - `DeleteCrashFile()`
   - `ExtractNorents()`, `ExtractNorentsFromEquipped()`, `ExtractNorentsList()`
   - `SavePlayerWithRent()`
   - All rent constants: `RentCrash`, `RentRented`, `RentCryo`,
     `RentTimedOut`, `RentForced`, `RentFactor`, `CryoFactor`,
     `MaxObjSave`, `MinRentCost`

2. **Check `IsUnrentable()`** — called from `house_save.go:89`. If houses
   still use this, keep it but simplify (it just checks `FlagNoRent`, negative
   vnum, and `ItemKey`). If houses don't use it either, delete it.

3. **Remove rent fields from save schema** if safe:
   - `RentCode`, `RentTime`, `NetCostPerDiem` in the player save struct.
   - Only if no save files in the wild contain these fields (check with
     `playerToSaveData` / `saveDataToPlayer`).

4. **Remove the `OnAutoSave` hook** from `pkg/engine/gameloop.go` if it's
   exclusively rent-related (check if anything else uses it).

5. **Do NOT port** `receptionist`/`cryogenicist` spec procs — they serve a system
   that doesn't exist in Go.

### Key Files

- `pkg/game/objsave.go` — bulk of deletions (MODIFY)
- `pkg/game/house_save.go:89` — `IsUnrentable` caller (CHECK)
- `pkg/game/save.go` — rent fields in save struct (CHECK)
- `pkg/engine/gameloop.go` — `OnAutoSave` hook (CHECK)
- `src/config.c:100-131` — C rent config (REFERENCE — confirms `free_rent=YES`)

### Regression Test

- `go build ./... && go vet ./... && go test ./...` — verify nothing breaks
- Ensure `house_save.go` still compiles after `IsUnrentable` decision

---

## Build Gate

```bash
go build ./... && go vet ./... && go test ./... && gofumpt -l . | grep -v vendor
```

All four must pass before committing. CI adds `go test -race`.

## Commit

```
fix: fidelity polish — randN inclusive, combat dam_message, delete dead rent code (DP-949, DP-950, DP-951)
```
