# BRIEF 2026-07-12 — QA Batch 1: five one-line fidelity fixes (DP-1055, DP-1057, DP-1058, DP-1059, DP-1060)

**Executor:** Kimi or GLM (your call — pure mechanical work, low risk).
**Branch:** `fix/qa-batch1-oneliners` (fresh off current `main`, your own clone/worktree — never share a HEAD with another executor).
**One PR.** Claude verified every C source citation and Go anchor below against the tree at `origin/main` @ `4e5fa33`.
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git`
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.

These are five independent one-to-two-token fixes across four non-overlapping files. All are follow-ups from the 2026-07-12 Fable QA pass (`docs/research/drafts/2026-07-12-fable-fidelity-qa.md`). Do the edits exactly as written — do not refactor surrounding code, do not touch any file not named below.

**Scope lock — touch ONLY these four files:**
- `pkg/game/limits_exp.go`
- `pkg/engine/gameloop.go`
- `pkg/game/mobact.go`
- `pkg/session/commands.go`

(plus test files you add). Do NOT touch `pkg/game/death.go`, `pkg/game/world.go`, `pkg/combat/engine.go`, `pkg/db/convert.go`, or `pkg/game/save.go` — other briefs own those.

---

## Fix 1: DP-1055 — level-up gate lets level-30 mortals advance to 31 (immortal)

**File:** `pkg/game/limits_exp.go` — line 182 (inside `GainExp`).

**Problem:**
```go
if p.Level < LVL_IMPL-1 && p.Exp >= ExpNeededForLevel(p) {
```
`LVL_IMPL` is 40 (`pkg/game/limits.go:23`), so the gate is `< 39` — a level-30 mortal who crosses the XP threshold auto-advances to level 31, which equals `LVL_IMMORT` (`pkg/game/limits.go:21` = 31). That hands a mortal immortal gates (instakill routing, DT immunity, aggro immunity). XP must never take a mortal past level 30.

**Fix:** change `LVL_IMPL-1` to `LVL_IMMORT-1`:
```go
if p.Level < LVL_IMMORT-1 && p.Exp >= ExpNeededForLevel(p) {
```
That is the ONLY change on this line. Leave the `>=` as-is (C uses strict `>` but that off-by-one is a documented nano; not in scope).

**Cite:** C — `src/limits.c:303`: `if (GET_LEVEL(ch) < LVL_IMMORT-1 && GET_EXP(ch) > exp_needed...)`. C caps mortal auto-advance at `LVL_IMMORT-1` (=30).

**Regression Test:** `pkg/game/limits_exp_test.go`
- Add `TestGainExp_MortalCannotReachImmortal`: create a level-30 player, set `p.Exp` to one below `ExpNeededForLevel`, award enough XP to cross the threshold, assert `p.Level == 30` (NOT 31) after `GainExp`. Also assert a level-29 player crossing its threshold *does* advance to 30 (the gate still lets sub-cap advances through).

---

## Fix 2: DP-1057 — gameloop mud hour is 75s, Dark Pawns is 63s

**File:** `pkg/engine/gameloop.go` — line 33.

**Problem:**
```go
SECS_PER_MUD_HOUR = 75 // 75 real seconds per Mud hour (C default)
```
75 is the stock CircleMUD default. It drives `OnAffectUpdate` + `OnWeatherAndTime` (this same file, the mud-hour handlers). Dark Pawns overrides this to 63. Net today: spell durations tick ~19% slow, and this constant disagrees with the already-fixed 63s `PointUpdate` ticker.

**Fix:**
```go
SECS_PER_MUD_HOUR = 63 // 63 real seconds per Mud hour (Dark Pawns override, src/utils.h:135)
```

**Cite:** C — `src/utils.h:135`: `#define SECS_PER_MUD_HOUR 63`. Dark Pawns diverges from stock Circle's 75.

**Regression Test:** none — one-line constant change. Note in the PR body that `go test ./...` must still pass (some tests may assert tick timing; if any break, that's a real signal — surface it, don't paper over it).

---

## Fix 3: DP-1058 — aggressive-mob protect-evil/good roll inverted (skips 5/6, should skip 1/6)

**File:** `pkg/game/mobact.go` — lines 294 and 299 (the aggressive block).

**Problem:**
```go
if vict.IsAffected(12) && mobIsEvil(ch) && rand.IntN(6) != 0 {   // line 294
    continue
}
...
if vict.IsAffected(13) && mobIsGood(ch) && rand.IntN(6) != 0 {   // line 299
    continue
}
```
C's guard is `!number(0,5)` — true only when the roll is 0, i.e. the mob skips the protected victim **1/6** of the time. Go's `rand.IntN(6) != 0` is true **5/6** of the time — the odds are inverted, so protection almost always works instead of almost never.

**Fix:** change both `!= 0` to `== 0` on lines 294 and 299 (nothing else):
```go
if vict.IsAffected(12) && mobIsEvil(ch) && rand.IntN(6) == 0 {
...
if vict.IsAffected(13) && mobIsGood(ch) && rand.IntN(6) == 0 {
```
`rand.IntN(6) == 0` is 1/6, matching C's `!number(0,5)`.

**Cite:** C — `src/mobact.c:210-213` (aggressive block): `if (AFF_FLAGGED(vict, AFF_PROTECT_EVIL) && IS_EVIL(ch) && !number(0,5)) continue;`. NOTE: this is the *aggressive* block, which is 1/6. C's separate *race-hate* block is 5/6 by design — do NOT touch the race-hate block (mobact.go ~361), it's already correct. Only lines 294 and 299.

**Regression Test:** none practical — RNG-gated branch. State in the PR body that the fix inverts the comparison to match C's 1/6 probability; a deterministic test would require injecting the roller, which is out of scope for a two-token fix.

---

## Fix 4: DP-1059 — `mold` registered for mortals; C is immortal-only

**File:** `pkg/session/commands.go` — line 277.

**Problem:**
```go
cmdRegistry.Register("mold", wrapSkill(command.CmdMold), "Mold a clay item.", 0, combat.PosStanding)
```
`mold` is an immortal object-creation command registered at level 0 / standing — every mortal can invoke it.

**Fix:**
```go
cmdRegistry.Register("mold", wrapSkill(command.CmdMold), "Mold a clay item.", LVL_IMMORT, combat.PosResting)
```
Use whatever `LVL_IMMORT` symbol is in scope in this file (grep the file; if none is imported, use the literal `31` and add a `// LVL_IMMORT` comment, matching how other immortal commands in this registry express their min level). Minimum position becomes `combat.PosResting`.

**Cite:** C — `src/interpreter.c:551`: `{ "mold", POS_RESTING, do_mold, LVL_IMMORT, 0 }`.

**Regression Test:** see Fix 5's shared test below.

---

## Fix 5: DP-1060 — `do_detect` command word should be `search`, not `detect`

**File:** `pkg/session/commands.go` — line 273.

**Problem:**
```go
cmdRegistry.Register("detect", wrapSkill(command.CmdDetect), "Detect hidden exits.", 0, combat.PosStanding)
```
C has no `detect` command. `do_detect` is bound to the word **`search`**. Players type `search`; today that word doesn't exist and `detect` is a word C players never used.

**Fix:** register the handler under `search`, and keep `detect` as an alias so nothing that already assumes `detect` breaks:
```go
cmdRegistry.Register("search", wrapSkill(command.CmdDetect), "Search for hidden exits.", 0, combat.PosStanding)
cmdRegistry.Register("detect", wrapSkill(command.CmdDetect), "Detect hidden exits (alias for search).", 0, combat.PosStanding)
```
`search` is NOT currently registered anywhere in `pkg/session/` (verified) — no conflict. If the registry rejects duplicate handlers or you find `search` already taken, stop and flag it rather than guessing.

**Cite:** C — `src/interpreter.c:411`: `{ "search", POS_STANDING, do_detect, 0, 0 }`.

**Regression Test:** `pkg/session/commands_test.go` (or wherever command registration is tested — grep for existing registry assertions; if none exist, add a small table test)
- Add `TestCommandRegistry_QABatch1`: after building the registry, assert:
  - `search` is registered at level 0, `PosStanding`, resolving to the detect handler
  - `detect` alias still resolves
  - `mold` is registered at `LVL_IMMORT` (31), `PosResting`
- If the registry has no lookup API to assert min-level/position, add the minimal accessor or assert via whatever inspection the existing tests use — do not expand the registry's public surface beyond what a test needs.

---

## Execution Order

Order doesn't matter — all five are independent. Suggested: Fix 2 (constant) → Fix 1 (cap) → Fix 3 (roll) → Fix 4 + Fix 5 (same file, do together). Run the build gate after all edits.

## After All Fixes

```bash
git checkout -b fix/qa-batch1-oneliners main
# ... make the five edits + tests ...
go build ./... && go vet ./... && go test ./...
git add -A
git commit -m "fix: QA batch 1 one-liner fidelity fixes (DP-1055, DP-1057, DP-1058, DP-1059, DP-1060)"
git push -u origin fix/qa-batch1-oneliners
gh pr create --title "fix: QA batch 1 one-liner fidelity fixes (DP-1055/1057/1058/1059/1060)" \
  --body "Five one-to-two-token fidelity fixes from the 2026-07-12 Fable QA pass. See docs/briefs/BRIEF-2026-07-12-qa-batch1-oneliners.md. Fixes DP-1055, DP-1057, DP-1058, DP-1059, DP-1060."
```

Then STOP. Do not merge. Claude reviews the PR against `origin/main`, runs the build gate, and merges.
