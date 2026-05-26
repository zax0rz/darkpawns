# Claude Code Prompt — Dark Pawns Fidelity Fixes

You are working on the Dark Pawns MUD codebase at `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo/`. This is a Go port of a 1994 CircleMUD C codebase. A fidelity audit has identified issues where the Go port deviates from the original C behavior.

**Your job:** Fix the issues below, in priority order. Each issue includes the exact file and line, the C source reference, and what needs to change.

**Build requirement:** After EVERY fix, run `cd /Users/zach/.openclaw/workspace-daeron/darkpawns_repo && go build ./... && go vet ./...` to verify it compiles. Do not proceed to the next fix if the build is broken.

**Commit convention:** `fix: description` (e.g., `fix: DP-348 XP level-difference penalty`)

---

## CRITICAL (fix these first)

### DP-348: XP Level-Difference Penalty Missing

**File:** `pkg/game/party.go` — `AwardMobKillXP()` function
**C source:** `src/fight.c:1949`

In C, XP gained from killing a mob is reduced by a percentage based on the level difference between the killer and the mob:
- If killer is higher level than the mob, XP is reduced proportionally
- Formula: `diff = GET_LEVEL(ch) - GET_LEVEL(mob)`, then `gain = gain * (1 - diff/20)` capped at minimum 1
- This prevents power-leveling by killing trivial mobs

The Go function `AwardMobKillXP` currently awards flat XP with no level-difference scaling.

**Fix:** After calculating `xp` in AwardMobKillXP, add:
```go
diff := ch.Level - mob.Level
if diff > 0 {
    xp = xp * (20 - diff) / 20
    if xp < 1 {
        xp = 1
    }
}
```

Reference C code to match (fight.c:1949-1970):
```c
if ((IS_NPC(ch) || (GET_LEVEL(ch) < LVL_IMMORT)) &&
    (GET_LEVEL(victim) > 0)) {
    if (GET_LEVEL(ch) > GET_LEVEL(victim))
        gain = gain * GET_LEVEL(victim) / GET_LEVEL(ch);
    else if (GET_LEVEL(ch) < GET_LEVEL(victim))
        gain = gain * (2 * GET_LEVEL(victim) - GET_LEVEL(ch)) / GET_LEVEL(victim);
    if (gain < 1) gain = 1;
}
```

---

### DP-358: Race Say Syllable Translation Dead Code

**Files:**
- `pkg/session/act_comm.go:16` — `cmdRaceSay` (active, no translation)
- `pkg/game/comm_say.go:6` — `doRaceSay` (has translation but dead code)

**C source:** `src/act.comm.c:635` — `do_race_say()`

The proper implementation exists in `comm_say.go` but is marked `//lint:file-ignore U1000 ... not yet wired to command registry`. The active command `cmdRaceSay` in `act_comm.go` broadcasts plaintext with no syllable substitution.

**Fix:** Wire the game-layer `doRaceSay` into the session command. Change `cmdRaceSay` in `act_comm.go` to call `doRaceSay` (or the syllable substitution logic) instead of broadcasting plaintext. Look at how `cmdSay` calls into the game layer for the pattern. Remove the lint ignore comment from `comm_say.go`.

---

## HIGH (fix after critical)

### DP-352: Sleep Spell Missing Checks

**File:** `pkg/spells/affect_spells.go:74` — `case SpellSleep:`
**C source:** `src/magic.c` — `spell_sleep()`

C has multiple gates before applying sleep:
1. Reagent check (requires sand)
2. Outlaw protection check
3. Level difference limit (caster level ±3)
4. MOB_NOSLEEP immunity flag
5. Position mutation to POS_SLEEPING
6. NPC retaliation on failed save

Go only does a saving throw and applies the affect.

**Fix:** Add these checks before the saving throw:
- Check `MOB_NOSLEEP` flag on victim (if NPC) — skip if immune
- Check level difference: if abs(casterLevel - victimLevel) > 3, reduce success chance or block
- After applying affect, set victim position to `POS_SLEEPING` (find the position constant in the codebase)
- After failed save, trigger NPC retaliation via `npcRetaliate(victim, ch)`

The reagent system may not be implemented in Go — check before adding. If there's no reagent infrastructure, document it but don't add one.

---

### DP-353: Curse Spell Damroll Discarded

**File:** `pkg/spells/affect_spells.go:60-69`
**C source:** `src/magic.c:967`

The Curse case constructs two affects but only applies the first one:
```go
case SpellCurse:
    ...
    aff = engine.NewAffectDirect(SpellCurse, engine.ApplyNone, curseDur, -3, engine.AFFCurse, "curse")
    applyAffect(victim, aff)
    aff = engine.NewAffect(SpellCurse, engine.ApplyDamroll, curseDur, -3, "curse")
    // NO applyAffect call — falls through to SpellInvisible, overwriting aff
```

**Fix:** Add `applyAffect(victim, aff)` after the damroll affect is constructed (after the `aff = ...ApplyDamroll` line), BEFORE the fall-through to SpellInvisible. Also check if C also applies a HITROLL affect — if so, add that too.

---

### DP-351: Poison Missing STR and Hitroll Penalties

**File:** `pkg/spells/affect_spells.go:104`
**C source:** `src/magic.c` — `spell_poison()`

C applies two affects:
1. `APPLY_STR` with modifier -2
2. `APPLY_HITROLL` with modifier -2

Go applies a single affect with `APPLY_NONE` and modifier -2.

**Fix:** Replace the single affect with two separate affects:
```go
case SpellPoison:
    ...
    poisonDur := 1 + getLevel(ch)/4
    aff = engine.NewAffect(SpellPoison, engine.ApplyStr, poisonDur, -2, "poison")
    applyAffect(victim, aff)
    aff = engine.NewAffect(SpellPoison, engine.ApplyHitroll, poisonDur, -2, "poison")
    applyAffect(victim, aff)
    aff = engine.NewAffectDirect(SpellPoison, engine.ApplyNone, poisonDur, 0, engine.AFFPoison, "poison")
```

Check the ApplyStr and ApplyHitroll constants exist — search for `ApplyStr` and `ApplyHitroll` in `pkg/spells/` or `pkg/game/`.

---

### DP-346: Parry/Dodge Round-Wide Penalty Missing

**File:** `pkg/combat/engine.go:275`
**C source:** `src/fight.c:1251`

C applies a round-wide penalty: a successful parry or dodge gives the defender a combat advantage for the entire round (reduces attacker's hit chance on subsequent attacks that round). Go treats each attack independently.

**Fix:** This is a structural change. In C's `circle_attack`, if parry/dodge succeeds, the attacker's subsequent attacks in the same round get a penalty. The Go combat loop in `processCombatPair` fires multiple attacks per round (based on attacks-per-round).

Add a `roundPenalty` flag to the combat pair. If parry or dodge succeeds on one attack, set `roundPenalty = true`. On subsequent attacks in the same round loop, add this penalty to the hit chance calculation via the `HitModifiers` struct.

---

### DP-347: Data Race on XP/Gold Mutations

**File:** `pkg/game/party.go` — `AwardMobKillXP()`
**C source:** N/A (concurrency issue — C was single-threaded)

`AwardMobKillXP` modifies `ch.Xp`, `ch.Gold` without holding `ch.mu`. Multiple goroutines can process combat simultaneously.

**Fix:** Add mutex locking around the XP and gold mutations in `AwardMobKillXP`:
```go
ch.mu.Lock()
ch.Xp += xp
ch.Gold += gold
ch.mu.Unlock()
```

Also check `Death()`, `performLoot()`, and `DeathCut` for similar unguarded mutations — if they exist and are called concurrently, fix those too.

---

### DP-350: Tattoos Apply Zero Bonuses

**File:** `pkg/game/deferred_fight_fns.go:354` — `TattooAf()`
**C source:** `src/tattoo.c:104`

`TattooAf()` is a stub that does nothing:
```go
func TattooAf(ch *Player, i int) {
    _ = bonuses
    _ = add
}
```

In C, tattoos apply stat bonuses (STR, INT, WIS, DEX, CON, CHA) based on tattoo type and position.

**Fix:** Port the tattoo bonus logic from `src/tattoo.c:104`. The function should look up the tattoo type and apply the appropriate stat modifier. Check what data the Player struct has for tattoos (search for `Tattoo` or `tattoo` in the player struct).

---

### DP-355: Hellfire Area Spell Dead Stub

**File:** `pkg/spells/affect_spells.go:1583` — `case SpellHellfire:`
**C source:** `src/spells.c:701`

`mag_areas()` case for Hellfire is `return;` — completely dead.

In C, Hellfire:
1. Does fire damage to all enemies in the room
2. Knocks victims to POS_SITTING
3. Instant-kills targets under level 5

**Fix:** Port the Hellfire logic from `src/spells.c:701`. This is a complex spell — implement the damage + position change + level check. If any sub-function isn't available in Go, stub it with a comment and implement what you can.

---

## MEDIUM (fix last)

### DP-360: Soundproof Room Flag Ignored

**Files:** `pkg/session/comm_cmds.go` (cmdTell, cmdReply, cmdShout), `pkg/session/act_comm.go` (cmdWhisper)
**C source:** `src/act.comm.c` — ROOM_SOUNDPROOF checks

**Fix:** Add `ROOM_SOUNDPROOF` checks to session commands. Check if the player's room has the soundproof flag. If so, send "The walls seem to absorb your words.\r\n" and return.

Search for `ROOM_SOUNDPROOF` or `Soundproof` in the codebase to find the flag constant and how to check room flags.

---

### DP-362: InfoBar Data Race

**File:** `pkg/session/display_cmds.go:453` — `cmdInfoBarUpdate()`
**Type:** CONCURRENCY

**Fix:** Acquire `p.mu` before reading player stats for the infobar template. Lock early, read all fields, unlock before sending output.

---

### DP-363: InfoBar Wrong XP Formula

**File:** `pkg/session/display_cmds.go:476` — `cmdInfoBarUpdate()`

Dynamic updates use `is.expNeededForLevel = 1000 * is.level` instead of the class-specific `findExp()`.

**Fix:** Replace `1000 * is.level` with `findExp(s.player.Class, s.player.Level+1)` — same call used in the initial render.

---

### DP-349: Kill Milestone Blessings Disabled

**File:** `pkg/game/` — search for `counter_procs` or `kill_milestone` or `blessing`
**C source:** `src/fight.c:660`

**Fix:** This is the counter_procs system from Session 38. Check if `counter_procs` exists in the Go codebase. If it does, verify it's wired. If it doesn't, port it from the C source. This is lower priority — the feature was already identified as missing.

---

### DP-354: Gender Pronoun Inverted

**File:** `pkg/game/death.go:663` — `genderPronoun()`

If this hasn't been fixed by the subagent already, swap the return values:
- `case 1: return "his"` (MALE)
- `case 2: return "her"` (FEMALE)
- `default: return "its"` (NEUTRAL)

---

### DP-356: Qcomm Quest Flag Check

**File:** `pkg/session/act_comm.go:39` — `cmdQcomm()`

If this hasn't been fixed by the subagent, add:
```go
if s.player.GetFlags()&(1<<game.PrfQuest) == 0 {
    s.Send("You aren't even part of the quest!")
    return nil
}
```

---

## AFTER ALL FIXES

Run the full test suite:
```bash
cd /Users/zach/.openclaw/workspace-daeron/darkpawns_repo
go build ./...
go vet ./...
go test ./...
```

If tests fail, fix them before committing.
