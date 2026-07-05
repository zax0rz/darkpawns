# Brief: Sprint 2 MED/LOW — Fable Review Batch — 2026-07-03

**Workspace:** `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.
**Milestone:** Fable Review (2026-07-03)

These are independent fixes. Each can be done in parallel by a different agent. No dependencies between them.

---

## Fix 1: DP-907 — Target resolution inconsistent across commands (MED)

**File:** `pkg/command/` — multiple commands use different target resolvers

**Problem:**
Different commands use different target resolvers: `FindTargetInRoom`, `getCharRoomVis`, and per-command ad-hoc loops. `consider postman` → "They aren't here" while `kick postman` finds the postman standing right there.

**Fix:**
Audit all commands in `pkg/command/` that take a target argument. Ensure they all use the same resolver. The canonical resolver should be `FindTargetInRoom` (or `getCharRoomVis` — pick one and standardize).

Check these commands specifically: `consider`, `kick`, `backstab`, `bash`, `trip`, `rescue`, `steal`, `give`, `wear`, `remove`, `eat`, `drink`, `quaff`, `recite`, `brandish`.

For each: verify the target lookup uses the same function. Replace ad-hoc loops with the canonical resolver.

**Cite:** C source — `act.other.c` / `act.item.c` — all use `get_char_room_vis()`.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Fix 2: DP-908 — World degrades over uptime (MED)

**File:** `pkg/game/` — wander/mob activity code, zone reset

**Problem:**
Room population drifts far from reset state within minutes of boot. After the flag fixes (DP-898/899), SENTINEL/STAY_ZONE should be respected, but wander cadence and zone-reset equilibrium need verification.

**Fix:**
1. Verify `MobileActivity` respects SENTINEL flag (mobs with SENTINEL should never wander)
2. Verify STAY_ZONE flag is checked before wandering (mob shouldn't leave its zone)
3. Check wander probability gate (DP-479 from earlier — should already be fixed)
4. Verify zone reset timers are running and zones reset correctly
5. Add a debug log: when a mob wanders, log "mob X wandered from room Y to room Z" at TRACE level

This is primarily a verification task — most of the fixes should already be in place from earlier sprints. The key is confirming the flag fixes from DP-898/899 actually work end-to-end.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Fix 3: DP-909 — Character creation menus nondeterministic (MED)

**File:** `pkg/game/char_creation.go` — `getRaceOptions()`, `getClassOptions()`, `getHometownOptions()` (line ~300)

**Problem:**
Race/class/hometown options are passed as Go `map[string]string`, so the option list order is **shuffled on every render**. This matters for AI agents keying off option labels. Also: doubled menus and double-password flow in no-DB path.

**Fix:**
1. Replace `map[string]string` with `[]struct{Key, Label string}` or a sorted slice to ensure deterministic ordering
2. Verify no-DB path doesn't double-prompt for password
3. Verify menus aren't rendered twice (check if char creation state machine has a re-entry bug)

The C source uses arrays — deterministic by definition.

**Cite:** C source — `act.other.c` `do_gen_components()` — options are in static arrays.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Fix 4: DP-910 — JWT generation silently fails on short secret (MED)

**File:** `pkg/session/` — JWT token generation path

**Problem:**
Token generation fails at runtime when `JWT_SECRET` < 32 chars — error is logged as ERROR and then ignored. Server keeps running with token issuance broken. The e2e smoke suite's own secret is `e2e-smoke-test-secret` (21 chars), so CI silently exercises the failing path.

**Fix:**
1. At server startup, validate `JWT_SECRET` length >= 32 chars. If too short, either:
   - Panic with a clear message: "JWT_SECRET must be at least 32 characters"
   - Or pad/derive a 32-char key from the provided secret (less ideal)
2. Fix the e2e test secret to be >= 32 chars
3. If token generation fails at runtime, return the error instead of ignoring it

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Fix 5: DP-911 — Save errors swallowed (MED)

**File:** `pkg/game/char_mgmt.go:159`, `pkg/game/clan_admin.go:157`

**Problem:**
Save errors are discarded with `_ = SavePlayer(p)`. After the silent-postgres-ownership incident, no save error should be discardable.

**Fix:**
Replace `_ = SavePlayer(p)` with error handling:
```go
if err := SavePlayer(p); err != nil {
    // Log the error with context — who was being saved and why
    slog.ErrorContext(ctx, "failed to save player after operation",
        "player", p.Name,
        "operation", "char_mgmt/clan_admin",
        "error", err,
    )
    // Send message to the player/admin if possible
    // Don't crash — but don't silently swallow either
}
```

Apply at both locations: `char_mgmt.go:159` and `clan_admin.go:157`.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Fix 6: DP-912 — Login-screen lifecycle (MED)

**File:** `pkg/session/` — login/connection handling

**Problem:**
`quit` returns to the login banner and the connection idles there indefinitely. No idle timeout on login/banner screen. Guest name assignment is quirky (two sequential sessions both received `Guest_7000`).

**Fix:**
1. Add an idle timeout (120 seconds) on pre-auth connections. If no input received within timeout, close the connection with a message.
2. Fix guest name assignment to use an atomic counter or UUID-based naming to avoid collisions.
3. The timeout should be configurable via flag/env var.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Fix 7: DP-913 — Cosmetic string bugs (LOW)

**File:** `pkg/game/` — score display, say command

**Problem:**
1. `score` renders "You You are well armored." — double "You"
2. `say` self-echo: "You says 'test'" — should be "You say 'test'"

**Fix:**
1. Find the armor line in score display. The string likely starts with "You" and the caller prepends "You " again. Remove the duplicate.
2. Find the say self-echo. Second-person conjugation should be "say" not "says". Check `pkg/command/` for the say command handler.

**Cite:** C source — `act.info.c` for score, `act.comm.c` for say.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Execution Order

These are independent — execute in parallel if possible. No ordering constraints.

## After Each Fix

```bash
git add <specific-files>
git commit -m "fix: <description> (DP-XXX)"
git push -u origin fix/dp-XXX-<slug>
gh pr create --title "fix: <description> (DP-XXX)" --body "Fixes DP-XXX. See docs/briefs/BRIEF-2026-07-03-sprint2-med-low.md"
```

## Linear Updates (after each merge)

- DP-XXX: Add comment "Fixed — <what changed>", commit hash, move to Done
