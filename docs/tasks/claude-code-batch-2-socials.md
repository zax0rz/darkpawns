# Claude Code Batch — Run 2: Social System Fixes

## Issues
- DP-407: Socials bypass correct DoAction engine (URGENT)
- DP-408: Social pronouns use victim gender for all tokens (HIGH)
- DP-412: Social target matching uses substring instead of prefix (MEDIUM)
- DP-413: Socials bypass invisibility/blindness checks (MEDIUM)
- DP-411: Socials ignore victim position (MEDIUM)

## Task: Route socials through DoAction (DP-407)

**File:** `pkg/session/commands.go:473-475` + `pkg/session/cmd_social.go` vs `pkg/game/act_social.go:46`

Two parallel implementations exist:
1. `DoAction` in `pkg/game/act_social.go` — correct engine with pronoun substitution, visibility checks
2. `cmdSocial` in `pkg/session/cmd_social.go` — buggy shortcut used by the command dispatcher

The command dispatcher at `commands.go:473` routes socials to `cmdSocial` instead of `game.DoAction`.

**Fix:** Change the command dispatcher to call `game.DoAction` instead of the local `cmdSocial`. Check how `DoAction` expects to be called (it needs the session layer to provide `sendToChar`, `sendToRoom`, etc. — the game layer can't send directly). If `DoAction` isn't wired for session-layer callbacks, you may need to adapt it.

**C source:** `src/act.social.c — do_action()`

## Task: Fix pronoun substitution (DP-408)

**File:** `pkg/session/cmd_social.go:26-50` (`actSubst`)

If you're routing through DoAction (DP-407), this task may be moot — DoAction handles pronouns correctly.

If cmdSocial is still used as a fallback: `actSubst` uses `targetSex` for ALL tokens. C source uses actor gender for lowercase (`$m/$s/$e`) and victim gender for uppercase (`$M/$S/$E`).

**Fix:** Add `actorSex int` parameter to `actSubst`. Use actorSex for lowercase tokens, targetSex for uppercase.

**C source:** `src/act.social.c — act()` pronoun logic

## Task: Fix target matching (DP-412)

**File:** `pkg/session/cmd_social.go:95-101`

Go uses `strings.Contains` (substring match). C uses prefix matching.

**Fix:** Replace `strings.Contains(strings.ToLower(p.Name), strings.ToLower(targetName))` with `strings.HasPrefix(strings.ToLower(p.Name), strings.ToLower(targetName))`.

**C source:** `src/act.social.c — do_action()` target resolution

## Task: Add invisibility/blindness checks (DP-413)

**File:** `pkg/session/cmd_social.go:61-66` (`sendToRoom`)

C source's `Act()` engine checks `CAN_SEE` before sending messages to each recipient. Invisible players' emotes are hidden from non-detect-invis players.

If routing through DoAction (DP-407), this may already be handled. Otherwise, add visibility checks to `sendToRoom` — skip recipients who can't see the actor.

**C source:** `src/act.social.c — act()` visibility checks

## Task: Add victim position check (DP-411)

**File:** `pkg/session/cmd_social.go` + `pkg/game/socials.go`

C source: each social has `min_victim_position`. If target is below this position, fail with "$N is not in a proper position for that."

**Fix:**
1. Add `MinVictimPosition` field to `Social` struct in `socials.go`
2. In the social handler, check `victim.GetPosition() < social.MinVictimPosition` and send position failure message
3. Parse `min_position` from the socials file (check format — it's likely a number in the social definition)

**C source:** `src/act.social.c — do_action()` position check

## Verification
1. `go build ./...` — must pass
2. `go vet ./...` — must pass
3. `go test ./...` — must pass
