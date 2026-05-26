# Subagent Brief: Quick Fidelity Fixes (Batch 1)

**Objective:** Fix 4 low-complexity fidelity issues in the Dark Pawns codebase. Each fix is a few lines of code.

**Working directory:** `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo/`

**Before committing:** Run `go build ./... && go vet ./...` to verify.

---

## Fix 1: DP-354 — Gender Pronoun Mapping Inverted

**File:** `pkg/game/death.go:663`
**Function:** `genderPronoun(sex int) string`

The switch cases are wrong:
```go
// CURRENT (broken):
case 1: return "her"   // MALE should be "his"
case 2: return "its"   // FEMALE should be "her"
default: return "his"  // NEUTRAL should be "its"
```

**Fix:** Swap the return values:
```go
case 1: return "his"   // MALE
case 2: return "her"   // FEMALE
default: return "its"  // NEUTRAL
```

Reference: C `src/utils.h:505` `HSHR(ch)` macro: MALE(1)→"his", FEMALE(2)→"her", NEUTRAL(0)→"its".

---

## Fix 2: DP-356 — Qcomm Quest Flag Check Missing

**File:** `pkg/session/act_comm.go:39`
**Function:** `cmdQcomm(s *Session, args []string) error`

The function broadcasts to all sessions without checking if the player is on a quest. In C, this is gated by `PRF_FLAGGED(ch, PRF_QUEST)`.

**Fix:** Add a quest flag check at the top of the function, after the empty args check:

```go
// Check if player is on a quest
if s.player.GetFlags()&(1<<game.PrfQuest) == 0 {
    s.Send("You aren't even part of the quest!")
    return nil
}
```

The `PrfQuest` constant is defined at `pkg/game/other_helpers.go:56` with value 49. Import the `game` package if not already imported.

---

## Fix 3: DP-361 — PLR_NOSHOUT Bypass in Session Commands

**Files:** `pkg/session/comm_cmds.go` (cmdTell, cmdReply) and `pkg/session/act_comm.go` (cmdWhisper, cmdAsk)

The `PlrNoshout` flag exists (`pkg/game/player_flags.go:13`, value 8) and is checked in game-side code, but the session-side commands don't check it. Muted players can still tell, reply, whisper, and ask.

**Fix:** Add this check near the top of each of these 4 functions (after the empty args check):

```go
if s.player.GetFlags()&(1<<game.PlrNoshout) != 0 {
    s.Send("You cannot tell anyone anything!")
    return nil
}
```

For whisper and ask, change the message to:
```go
s.Send("You cannot communicate at all!")
```

Functions to modify:
1. `cmdTell` in `pkg/session/comm_cmds.go:32`
2. `cmdReply` in `pkg/session/comm_cmds.go:86`
3. `cmdWhisper` in `pkg/session/act_comm.go:68`
4. `cmdAsk` in `pkg/session/act_comm.go:124`

---

## Fix 4: DP-357 — AFK Pronoun Bug

**File:** `pkg/game/comm_tell.go:13`

The AFK message uses `hisHer(vict.Sex)` which returns possessive pronouns (his/her/its). But the sentence "X is AFK right now, ___ may not hear you" needs subject pronouns (he/she/it).

**Fix:** The `hisHer` function at `pkg/game/skills.go:355` returns possessive pronouns. You need to either:
- Create a `heSheIt(sex int) string` function that returns subject pronouns, OR
- Change the AFK message text to use possessive grammar: `"%s is AFK right now, %s ears may not hear you.\r\n"` (awkward)

Best approach: add a subject pronoun helper and use it:
```go
func heSheIt(sex int) string {
    switch sex {
    case 1: return "he"
    case 2: return "she"
    default: return "it"
    }
}
```

Then in `comm_tell.go:13`, change `hisHer(vict.Sex)` to `heSheIt(vict.Sex)`.
