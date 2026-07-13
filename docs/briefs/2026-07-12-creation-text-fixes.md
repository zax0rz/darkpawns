# Dark Pawns — Character Creation Text/String Fixes (5 Issues)

**Target files:**
- `pkg/session/char_creation.go` — character creation flow, text prompts
- `pkg/session/session_send.go` — welcome message (possibly)

**Repo:** `/Users/zach/.openclaw/workspace/darkpawns_repo`
**Branch:** Create from `main`, name `fix/creation-fidelity-text`
**After fixing:** Run `go build ./... && go vet ./... && go test ./...`. All must pass.
**Push:** `git push origin fix/creation-fidelity-text`

---

## Fix 1: Missing "fantasy name" reminder (DP-1065)

**File:** `pkg/session/char_creation.go`
**Location:** In the `handleCharInput` function, in the `case "confirm_name"` block, when the user says "Y" to confirm their name.

**C source** (interpreter.c:1812 and 1781) sends this BEFORE the "Did I get that right?" prompt:
```
Please remember to choose an appropriate fantasy-oriented name.\r\n
```

**Go source** (line ~161-163) currently jumps straight from name acceptance to either password or color prompt without this reminder.

**Fix:** After name confirmation (the "Y" case), before sending the password/color prompt, add:
```go
s.sendText("Please remember to choose an appropriate fantasy-oriented name.\r\n")
```

This appears in two places in C (returning player and new player paths). In Go, add it once in the `case "Y"` block of `confirm_name`, right before the `if s.charPasswordSupplied` check.

---

## Fix 2: Missing "New character." announcement (DP-1066)

**Status: ALREADY FIXED.** This string already exists in the Go code at char_creation.go:163:
```go
s.sendCharCreatePrompt("create_password", fmt.Sprintf("New character.\r\nGive me a password for %s: ", s.charName), nil)
```

**Do nothing for this issue.** It will be resolved as "already present" on merge.

---

## Fix 3: Missing welcome message text (DP-1068)

**C source** (config.c:256) has a `WELC_MESSG` string:
```
\r\n
Welcome to Dark Pawns! May your visit here be... Interesting.
\r\n\r\n
```

This is sent via `send_to_char(WELC_MESSG, d->character)` at interpreter.c:2187 when a returning player enters the game (after pressing "1" from CON_MENU).

**Go source:** The `sendWelcome()` function in `session_send.go` sends JSON state data (player stats, room info, token). It does NOT send any text welcome message. The MOTD is sent separately as a `MsgEvent` of type "motd" — that's correct.

**Fix:** In `pkg/session/session_send.go`, inside `sendWelcome()`, after sending the MOTD message (around line ~85) and before sending the state message (around line ~93), add:

```go
// Welcome text — matches C WELC_MESSG (config.c:256)
welcomeMsg, err := json.Marshal(ServerMessage{
	Type: MsgEvent,
	Data: EventData{
		Type: "text",
		Text: "\r\nWelcome to Dark Pawns! May your visit here be... Interesting.\r\n",
	},
})
if err == nil {
	s.send <- welcomeMsg
}
```

**Note:** This welcome should appear for BOTH new characters (from `completeCharCreation`) and returning players (from `handleLogin`). Both call `sendWelcome()`, so adding it there covers both paths.

---

## Fix 4: Missing text strings audit — verify these are correct

The Go code already has `"New character.\r\n"` and `"Did I get that right, %s (Y/N)? "` in the right places. Verify these match C:

| String | C location | Go location | Status |
|--------|-----------|-------------|--------|
| `"Okay, what IS it, then? "` | interpreter.c:1850 | char_creation.go:167 | ✅ Already present |
| `"Did I get that right, %s (Y/N)? "` | interpreter.c:1813 | char_creation.go:169 | ✅ Already present |
| `"New character.\r\n"` | interpreter.c:1843 | char_creation.go:163 | ✅ Already present |
| `"Give me a password for %s: "` | interpreter.c:1847 | char_creation.go:163 | ✅ Already present |
| `"Please type Yes or No: "` | interpreter.c:1856 | char_creation.go:173 | ✅ Already present |
| `"Please remember to choose an appropriate fantasy-oriented name.\r\n"` | interpreter.c:1781 | **MISSING** | ❌ Fix #1 above |
| `"Welcome to Dark Pawns! May your visit here be... Interesting.\r\n"` | config.c:256 | **MISSING** | ❌ Fix #3 above |

---

## Fix 5: DP-1077 — room 8004 vs 8099

**SKIP.** Already marked as intentional change. Go uses `MortalStartRoom` (8004) instead of C's 8099 because 8099 has no exits in current world data. Not a bug.

---

## Summary

| # | Issue | File | Action |
|---|-------|------|--------|
| 1 | DP-1065 | char_creation.go | Add fantasy name reminder text |
| 2 | DP-1066 | — | SKIP — already present |
| 3 | DP-1068 | session_send.go | Add welcome text in sendWelcome() |
| 4 | Audit | — | Verify all strings match (table above) |
| 5 | DP-1077 | — | SKIP — intentional change |

**Commit message:** `fix: character creation fidelity — missing text strings (DP-1065 DP-1068)`

**After commit:** `go build ./... && go vet ./... && go test ./...` — must pass. Then push.
