# Brief: Restore C XP Share Formula — 2026-07-12

**Workspace:** `/Users/zach/.openclaw/workspace/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.

---

## Fix 1: DP-1024 — XP Share Formula Fabricated (High)

**File:** `pkg/game/party.go` — `AwardMobKillXP()`

**Problem:**
The live kill-XP path used proportional scaling: higher-level killers got `xp * victimLevel / killerLevel`, and lower-level killers got a fabricated fight-up bonus. The C source does neither. C starts from `MIN(max_exp_gain, MAX(1, base))`, only penalizes higher-level attackers, gives solo players two levels of slack, and applies a flat over-level-20 penalty.

**Fix:**
Add `calcKillXPShare()` as a direct port of `src/fight.c calc_level_diff()` with explicit `inGroup` input from the already-resolved party path. Use it for solo and grouped `AwardMobKillXP()` awards. Also align `pkg/combat.CalcLevelDiff()` with C's integer truncation after the full percentage subtraction.

**Cite:** C source — `src/fight.c:659-685` (`calc_level_diff`). `share = MIN(max_exp_gain, MAX(1, base))`; if attacker is higher level, solo attackers get `level_diff -= 2`, then `>15` loses 70%, `>10` loses 50%, `>5` loses 30%; attackers over level 20 lose another 20%; no fight-up bonus exists.

**Regression Test:** `pkg/game/party_test.go`, `pkg/combat/fight_core_test.go`, `pkg/command/kill_payout_test.go`
- Pin no fight-up bonus for lower-level killers.
- Pin solo two-level slack.
- Pin grouped higher-level penalties.
- Pin over-level-20 penalty stacking.
- Pin C percentage truncation after the full subtraction.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Execution Order

1. Add a C-faithful XP share helper in `pkg/game`.
2. Route solo and grouped kill awards through that helper.
3. Align the combat helper's truncation behavior.
4. Add focused regression tests.

## After Fix

```bash
git add pkg/game/party.go pkg/game/party_test.go pkg/combat/fight_core.go pkg/combat/fight_core_test.go pkg/command/kill_payout_test.go docs/briefs/BRIEF-2026-07-12-dp1024-xp-share-formula.md
git commit -m "fix: restore C XP share formula (DP-1024)"
git push -u origin fix/dp1024-xp-share-formula
gh pr create --title "fix: restore C XP share formula (DP-1024)" --body "Fixes DP-1024. See docs/briefs/BRIEF-2026-07-12-dp1024-xp-share-formula.md for details."
```
