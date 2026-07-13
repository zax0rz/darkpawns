# BRIEF 2026-07-12 — Kimi: DP-1054 jail-guard intercept gates on phantom vnums (QA Q2)

**Executor:** Kimi k2.7-code (GLM acceptable — this is multi-file callback plumbing, mechanical but touchy). **Branch:** `fix/dp1054-jail-vnums` (fresh off current `main`, your own clone/worktree — never share a HEAD with another executor).
**One PR.** Claude verified the spec-assignment table, the callback wiring, and both call sites against `origin/main` @ `2daf4ce`.
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git`
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.

The jail-guard subdue logic is faithful and complete — it's just **unreachable**, because it gates on two mob vnums (8102, 8103) that don't carry the jail spec in either codebase. This fix re-gates it on the actual spec assignment via a callback, so it can finally fire. The subdue behavior itself (HP=1, unmount, forget/clear-hunting, move to 8118, `max(2, level/2)` timer, messages) is correct — **do not touch it.**

**Scope lock — you own exactly these files:**
- `pkg/combat/callbacks.go` (new callback field + accessor)
- `pkg/game/combat_wire.go` (wire the callback to the spec table)
- `pkg/combat/engine.go` (the LIVE call site, line 509)
- `pkg/combat/fight_core.go` (the DEAD-spine call site, line 352 — consistency only)
- `pkg/combat/engine_test.go` (rewire the existing jail test + add a negative test)

Do NOT touch `pkg/game/death.go`, `pkg/game/party.go`, `pkg/game/mobact.go`, `pkg/game/world.go`, or `pkg/game/spec_assign.go` (the table is already correct — read it, don't edit it). Do NOT modify the subdue callback (`JailGuardSubdue`) or any `src/*.c` file.

---

## The bug

`pkg/combat/engine.go:508-518` (`applyMobCombatRedirects`, the live mob-damage path):
```go
if !defender.IsNPC() &&
    (cbHasMobVNum(attackerName, 8102) || cbHasMobVNum(attackerName, 8103)) &&
    attacker.GetHP() > attacker.GetMaxHP()/2 &&
    !cbHasAffectStr(defenderName, AFF_STR_VAMPIRE) &&
    !cbHasAffectStr(defenderName, AFF_STR_WEREWOLF) {
    if cbJailGuardSubdue(attackerName, defenderName) { ... }
}
```
Neither **8102** nor **8103** carries a jail spec. The `take_to_jail` spec is assigned to mobs **8001, 8002, 8020, 8027, 8059**, and `wall_guard_ns` to **8060** — verified in `pkg/game/spec_assign.go:108-115` (identical to C `src/spec_assign.c:285-291`). The 8102/8103 numbers were invented by the dead `fight_core.go:352` port and were wrong from the start. Result: the entire jail-guard intercept is dead code — city guards beat PKs to death instead of jailing them.

---

## Authoritative C — `src/spec_assign.c:285-291` (spec assignment)

The gate that matters is "does this mob carry the jail-guard spec," not "is this mob a specific vnum." In C the spec is what routes behavior; the Go port should gate the same way. The correct spec-bearing vnums (already in Go's `MobSpecAssign`):

| VNum | Spec |
|---|---|
| 8001, 8002, 8020, 8027, 8059 | `take_to_jail` |
| 8060 | `wall_guard_ns` |

**Cite:** `src/spec_assign.c:285-291`; mirrored in `pkg/game/spec_assign.go:108-115` (`MobSpecAssign`).

---

## The fix

### Step 1 — add a callback field (`pkg/combat/callbacks.go`)

In the `GameCallbacks` struct (around line 36, next to `HasMobVNum`), add:
```go
// MobHasJailGuardSpec reports whether the named mob carries the take_to_jail or
// wall_guard_ns spec (src/spec_assign.c:285-291). Used to gate the jail-guard
// subdue redirect — the correct replacement for the phantom 8102/8103 vnum check.
MobHasJailGuardSpec func(name string) bool
```

### Step 2 — add the package-level accessor (`pkg/combat/callbacks.go`)

Next to `cbHasMobVNum` (around line 250), mirror the same nil-guarded pattern:
```go
func cbMobHasJailGuardSpec(name string) bool {
    if cb := callbacks; cb != nil && cb.MobHasJailGuardSpec != nil {
        return cb.MobHasJailGuardSpec(name)
    }
    return false
}
```
(Nil-guard returns false, so existing tests that don't set the field simply won't trigger the subdue — matching today's effective behavior.)

### Step 3 — wire it in the game layer (`pkg/game/combat_wire.go`)

Next to `cb.HasMobVNum` (around line 156), add:
```go
cb.MobHasJailGuardSpec = func(name string) bool {
    m := w.GetMobByName(name)
    if m == nil || m.Prototype == nil {
        return false
    }
    switch MobSpecAssign[m.Prototype.VNum] {
    case "take_to_jail", "wall_guard_ns":
        return true
    default:
        return false
    }
}
```

### Step 4 — fix the LIVE call site (`pkg/combat/engine.go:509`)

Replace **only** the vnum sub-expression:
```go
// before:
(cbHasMobVNum(attackerName, 8102) || cbHasMobVNum(attackerName, 8103)) &&
// after:
cbMobHasJailGuardSpec(attackerName) &&
```
Leave every other condition in that `if` (the `!defender.IsNPC()`, the HP>half guard, the vampire/werewolf guards) and the whole subdue body exactly as-is.

### Step 5 — fix the DEAD-spine call site (`pkg/combat/fight_core.go:352`) — consistency only

`fight_core.go:352` has the identical phantom check inside the dead `damage()` spine:
```go
(cbHasMobVNum(chName, 8102) || cbHasMobVNum(chName, 8103)) {
```
Replace it with `cbMobHasJailGuardSpec(chName) {` too. This spine is not on a live path (no behavior change), but leaving the wrong vnums here is a landmine — a future port could copy them again (that's exactly how they got into engine.go). Do NOT delete or restructure the surrounding dead code; just swap the condition.

### Out of scope (note, don't do)

Fable flagged that C's jail block also checks `CAN_SEE(ch, victim)`. There is **no** `CanSee` callback in `pkg/combat` today, and adding one is more surface than this fix warrants. Leave a `// TODO(DP-1054): C also gates on CAN_SEE(ch, victim); no CanSee callback exists yet` comment on the engine.go block and note it in the PR body as a deliberate deferral. Do not build a CanSee callback in this PR.

---

## Regression Tests — `pkg/combat/engine_test.go`

**`TestMobRedirect_JailGuardSubduesInsteadOfDamaging` (line 175) WILL BREAK** — it currently makes the subdue fire via `HasMobVNum: ... vnum == 8102`, which the engine no longer checks. Rewire it:
- Remove the `HasMobVNum` stub's relevance and instead set:
  ```go
  MobHasJailGuardSpec: func(name string) bool { return name == "jail guard" },
  ```
- Keep the rest of the test identical; it should still assert the subdue callback fired, no live-path melee damage landed, and combat cleared.

**Add `TestMobRedirect_NonJailMobDoesNotSubdue`:** same setup, but `MobHasJailGuardSpec` returns `false` for the attacker (a normal aggressive mob). Assert `JailGuardSubdue` is **never** called and the combat proceeds normally (the redirect returns false). This is the test that would have caught the original bug — a mob without the jail spec must not subdue.

Do not weaken or delete the other redirect tests (charmed-pet, switcheroo) — they must still pass untouched.

---

## Execution order

1. Steps 1-2 (callback field + accessor in callbacks.go).
2. Step 3 (wire in combat_wire.go).
3. Steps 4-5 (both call sites).
4. Tests.
5. Build gate.

## After all fixes

```bash
git checkout -b fix/dp1054-jail-vnums main
# ... implement ...
go build ./... && go vet ./... && go test ./...
git add -A
git commit -m "fix: jail-guard intercept gates on spec assignment, not phantom vnums (DP-1054)"
git push -u origin fix/dp1054-jail-vnums
gh pr create --title "fix: jail-guard intercept gates on spec assignment (DP-1054)" \
  --body "The jail-guard subdue redirect gated on vnums 8102/8103, which carry no jail spec, so it never fired (QA Q2). Re-gates on MobSpecAssign (take_to_jail / wall_guard_ns) via a new callback; fixes the same phantom check in the dead damage() spine for consistency. CAN_SEE gate deferred (no callback exists yet). See docs/briefs/BRIEF-2026-07-12-kimi-dp1054-jail-vnums.md. Fixes DP-1054."
```

Then STOP. Do not merge. Claude reviews against `origin/main`, runs the build gate, verifies the callback resolves the correct vnums, and merges.
