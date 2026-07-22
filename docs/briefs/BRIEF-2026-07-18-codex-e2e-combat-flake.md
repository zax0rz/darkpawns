# BRIEF (codex) — close the e2e CI blocker on PR #401 (combat-round flake + full e2e)

**Owner:** codex. **Branch:** `glm/reset-time-port` (PR #401), build on the committed bundle (see below). **This is test-infra hardening — do NOT weaken the faithful `reset_time` port.**

## Context (the fidelity work is DONE and gated)
PR #401 ports C's `reset_time`/`mud_time_passed` (glm) + a `DP_FIXED_TIME` seam. Claude added the matching C-oracle side (oracle branch `dp-oracle-seam`, commit 5b6b484: under `DP_FIXED_TIME`, C pins the calendar and skips `date_record`) and wired `DP_FIXED_TIME=1770838461` into the oracle harness. **`character-view`'s `time` block is GREEN; full oracle sweep green.** The architecture is settled: **`DP_FIXED_TIME` (pin the clock) is orthogonal to `DP_CLOCK` (freeze pulses)** — e2e uses only `DP_FIXED_TIME` so pulses keep firing.

## The one blocker: `TestTelnetSmoke_Combat` flake
The faithful port exposed a **pre-existing** e2e fragility: outdoor rooms go dark on night MUD-hours, hiding the NPCs the tests engage. glm's fix (pin a daytime clock via `DP_FIXED_TIME` + `DP_SEED=1` in `launchAndDial`) fixed `TestTelnetSmoke_SkillKick`, but `TestTelnetSmoke_Combat` still flakes ~3/5 with `"engaged a target but observed no combat rounds — engine stalled?"` (`tests/e2e/telnet_smoke_test.go:162`). Clean `main` passed it 10/10; the change causes it. Full diagnosis: `docs/reports/REPORT-2026-07-18-glm-reset-time-e2e-blocker.md`.

## Your task
Make the **full e2e suite** (`go test ./tests/e2e/...`) reliably green under the seeded + fixed-time env, without seed-cherry-picking a fragile magic seed and without touching the port.

1. **Diagnose the flake first (instrument the observable, per glm's lesson).** After engagement (line 140-151 gets a first combat marker), the test waits 12s for a *further* round (line 160). Determine what actually happens at `DP_SEED=1`+daytime: does the engaged NPC **die or flee** before a second observable round (combat legitimately ends → not a stall), or do rounds genuinely stop firing? Log the telnet stream for the 12s window. `combatOutputMarkers` already includes death markers (`"dies"`, `"is dead"`) — if the mob dies, that should be caught, so a bare timeout implies the fight ended *silently* or the second round lands late.
2. **Fix robustly, preferring test-correctness over luck:**
   - If it's a **fast-kill / fight-ends** case: restructure so a completed engagement (first-round marker already seen at line 148) is sufficient, or re-engage a fresh NPC and accept any of {round marker, death marker} — don't require a *second* round from the *same* mob.
   - Widening the 12s window is acceptable only as a secondary measure and only if you show a round genuinely lands late (it won't help a silently-ended fight).
   - Do **not** hard-code a "lucky" seed; the test must be robust across the pinned daytime clock.
3. **Run the FULL e2e suite** under the env (glm did NOT finish this) — confirm `TestTelnetSmoke_CastEligibility` and every other e2e test pass with `DP_SEED=1` + `DP_FIXED_TIME`. Fix any others the port exposed the same way (darkness/day-night).
4. Keep `go build ./... && go vet ./... && golangci-lint && gofumpt` clean.

## The committed bundle you're building on
The PR branch (after Claude's commit) contains: glm's `weather.go` port + `ConfigureNowFromEnv` (reads `DP_FIXED_TIME`), `cmd/server/main.go` calling it, `tests/e2e/telnet_smoke_test.go` seeding (`DP_SEED=1`, `DP_FIXED_TIME`), and `cmd/dp-oracle-diff/main.go` passing `DP_FIXED_TIME=1770838461` to both engines. Note: the e2e test currently uses `DP_FIXED_TIME=650337471` (noon) — fine to keep (any daytime value works); don't churn it unless your fix needs a specific hour.

## Housekeeping (do in this PR)
- Remove the stale **untracked** `cmd/dp-oracle-diff/scenarios/combat-round.txt` if present (a leftover; not part of the committed suite — verify `git ls-files` doesn't track it before deleting).

## Acceptance (Claude-gated)
1. `go test ./tests/e2e/...` green **10/10 runs** for `TestTelnetSmoke_Combat` (no flake) and all other e2e tests, under `DP_SEED=1`+`DP_FIXED_TIME`.
2. Full oracle sweep stays green (Claude re-runs; the port must not regress any scenario — `character-view` already green).
3. The `reset_time` port is unchanged (fidelity intact); the fix lives in test infra only.
4. CI (build/vet/lint/test) green → PR #401 mergeable.
