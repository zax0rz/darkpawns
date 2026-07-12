# Brief: Route Kill XP Through gain_exp — 2026-07-12

**Workspace:** `/Users/zach/.openclaw/workspace/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.

---

## Fix 1: DP-1023 — Kill XP Bypasses gain_exp (High)

**File:** `pkg/game/party.go` — `AwardMobKillXP()`

**Problem:**
Kill XP distribution calculates a share, then mutates player XP with `Player.AddExp()`. That bypasses the faithful `gain_exp()` port, so combat kills do not apply the C per-kill cap and do not trigger level advancement.

**Fix:**
Replace direct `AddExp()` calls in `AwardMobKillXP()` with `World.GainExp()`. Also remove the extra `level * 1000` cap from `World.GainExp()` because C only applies `max_exp_gain` and the one-level cap.

**Cite:** C source — `src/fight.c:688-705` (`perform_group_gain`) sends the XP share message, then calls `gain_exp(ch, share)`. `src/limits.c:297-315` (`gain_exp`) caps positive gains at `max_exp_gain`, caps to one level, adds XP, and advances one level when the threshold is crossed.

**Regression Test:** `pkg/game/party_test.go`
- `TestGainExpUsesCLimitsWithoutPerLevelCap`: proves `GainExp()` accepts a 50,000 XP gain for a level-10 player instead of applying the non-C `level * 1000` cap.
- `TestAwardMobKillXPUsesGainExpCap`: proves combat kill XP is capped by `max_exp_gain`.
- `TestAwardMobKillXPCanAdvanceOneLevel`: proves combat kill XP can trigger the `gain_exp()` level-up path.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Execution Order

1. Fix `World.GainExp()` to match C caps.
2. Route solo and grouped kill XP through `World.GainExp()`.
3. Add regression tests for the cap and level-up behavior.

## After Fix

```bash
git add pkg/game/limits_exp.go pkg/game/party.go pkg/game/party_test.go docs/briefs/BRIEF-2026-07-12-dp1023-kill-xp-gain-exp.md
git commit -m "fix: route kill XP through gain_exp (DP-1023)"
git push -u origin fix/dp1023-kill-xp-gain-exp
gh pr create --title "fix: route kill XP through gain_exp (DP-1023)" --body "Fixes DP-1023. See docs/briefs/BRIEF-2026-07-12-dp1023-kill-xp-gain-exp.md for details."
```
