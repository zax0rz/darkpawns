# Dark Pawns — Character Creation Trivial Fixes (8 Issues)

**Target files:**
- `pkg/game/player.go` — player initialization
- `pkg/session/char_creation.go` — character creation flow, menus

**Repo:** `/Users/zach/.openclaw/workspace/darkpawns_repo`
**Branch:** Create from `main`, name `fix/creation-fidelity-trivial`
**After fixing:** Run `go test ./...` from repo root. All tests must pass.
**Push:** `git push origin fix/creation-fidelity-trivial`

---

## Fix 1: Starting Move Points (DP-1069)

**File:** `pkg/game/player.go`
**Line:** 302
**Current:** `p.MaxMove = 100`
**Change to:** `p.MaxMove = 82`
**Reason:** C source (db.c:3053) sets `ch->points.max_move = 82` for new characters.

Also check: does `p.Move` get set elsewhere? It should equal `p.MaxMove` after init. If `Move` is set to 100 on another line, change that too.

---

## Fix 2: Starting Hunger/Thirst (DP-1070)

**File:** `pkg/game/player.go`
**Lines:** 306-307
**Current:**
```go
p.Hunger = 24
p.Thirst = 24
```
**Change to:**
```go
p.Hunger = 36
p.Thirst = 36
```
**Reason:** C source (class.c:578-579) sets `GET_COND(ch, THIRST) = 36` and `GET_COND(ch, FULL) = 36`. Scale is 0-48. 36 = well-fed. 24 = half-full (wrong).

---

## Fix 3: Wimp Level (DP-1071)

**File:** `pkg/game/player.go`
**Location:** In `NewCharacter()` or `NewPlayer()`, wherever `WimpLevel` should be initialized.
**Current:** `WimpLevel` defaults to 0 (Go zero-value for int).
**Add:** `p.WimpLevel = 5`
**Reason:** C source (class.c:588) sets `GET_WIMP_LEV(ch) = 5`. This is the HP threshold at which a player auto-flees from combat. 0 means never flee — dangerous default.
**Note:** Find the right place to add this. It should be in the player initialization function, near the other stat/level/condition lines (~lines 295-310).

---

## Fix 4: Starting Practices (DP-1072)

**File:** `pkg/game/player.go`
**Location:** In `NewCharacter()` or `NewPlayer()`, near where Practices is declared.
**Current:** `Practices` defaults to 0 (Go zero-value for int).
**Add:** `p.Practices = 2`
**Reason:** C source (class.c:590) does `GET_PRACTICES(ch) += 2`. Practices are the currency players spend at guildmasters to learn/improve skills. Without these, new players cannot practice any skills at all. This is a gameplay-blocking bug.

---

## Fix 5: Race Typo — "Minotauran" → "Minotaur" (DP-1078)

**File:** `pkg/session/char_creation.go`
**Line:** 632
**Current:** `{"M", "Minotauran"},`
**Change to:** `{"M", "Minotaur"},`
**Reason:** C source (constants.c:200) has `[M]inotaur`. The Go code added an extra "-an" suffix. The race is called "Minotaur" everywhere else in the codebase (help text, race constants, etc.).

---

## Fix 6: Class Typo — "Magic-user" → "Mage" (DP-1081)

**File:** `pkg/session/char_creation.go`
**Line:** 646
**Current:** `{"M", "Magic-user"},`
**Change to:** `{"M", "Mage"},`
**Reason:** Three different names exist for this class:
- C class menu (class.c:69): `"Magic User"` (two words, no hyphen)
- C help file (constants.c:1298): `"Mage"`
- Go display name (character.go:54): `"Mage"` (via `ClassMageUser` → `"Mage"`)
- Go creation menu (this line): `"Magic-user"` (hyphenated)

The creation menu should match the Go display name "Mage" since that's what the player sees everywhere else (score, skills, remort into Magus). The `M` key selector stays the same.

---

## Fix 7: Name Rejection — Don't Disconnect (DP-1074)

**File:** `pkg/session/char_creation.go`
**Line:** 167
**Current:**
```go
case "N":
    s.sendText("Okay, what IS it, then? ")
    s.Close()
```
**Change to:**
```go
case "N":
    s.sendText("Okay, what IS it, then? ")
    s.charStage = "get_name"
    s.charName = ""
```
**Reason:** C source (interpreter.c:1850-1853) sends "Okay, what IS it, then?" and sets `STATE(d) = CON_GET_NAME` so the player can try a different name. The Go code sends the same message but then calls `s.Close()`, disconnecting the player. This is a usability bug — new players who typo their name get kicked out instead of re-prompted.

**Important:** After setting `s.charStage = "get_name"` and clearing `s.charName`, the session state machine should loop back to the name prompt automatically. Make sure no other state (charPassword, charClass, etc.) from a previous attempt leaks through.

---

## Fix 8: Display Flags — SKIP THIS ONE (DP-1073)

Go doesn't have a `PRF_DISPHP`/`PRF_DISPMANA`/`PRF_DISPMOVE` toggle mechanism. HP/mana/move bars appear to be always-on in the Go client. The `AutoExit` field is already set to `true` by default (player.go:262). The other three C flags (`PRF_DISPHP`, `PRF_DISPMANA`, `PRF_DISPMOVE`) have no Go equivalent because Go always shows them.

**Do not change anything for this issue.** It will be handled separately as a "wontfix — client always displays" resolution.

---

## Summary

| # | Issue | File | Lines | Change |
|---|-------|------|-------|--------|
| 1 | DP-1069 | player.go | 302 | `MaxMove = 82` |
| 2 | DP-1070 | player.go | 306-307 | `Hunger = 36, Thirst = 36` |
| 3 | DP-1071 | player.go | (find) | `WimpLevel = 5` |
| 4 | DP-1072 | player.go | (find) | `Practices = 2` |
| 5 | DP-1078 | char_creation.go | 632 | `"Minotaur"` |
| 6 | DP-1081 | char_creation.go | 646 | `"Mage"` |
| 7 | DP-1074 | char_creation.go | 167 | Don't disconnect, re-prompt |
| 8 | DP-1073 | — | SKIP | No Go equivalent for PRF_DISPHP etc. |

**Commit message:** `fix: character creation fidelity — numeric values, typos, name re-prompt (DP-1069 DP-1070 DP-1071 DP-1072 DP-1078 DP-1081 DP-1074)`

**After commit:** `go test ./...` — must pass. Then push.
