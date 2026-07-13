# BRIEF 2026-07-10 — Linear backlog reconcile & finding-linking (Kimi)

## Who / what / why

You are **Kimi** (K2.7-code). This is a **read-and-reconcile** task, **not** a code-writing
task — do **not** modify source or open PRs. GLM is concurrently editing code from a separate
brief; stay entirely in Linear + read-only source inspection so you never collide.

Goal: reconcile the **Dark Pawns (DP) Linear backlog** (88 open issues) against **current
`main`** and against a fresh clawpatch-findings reconcile, so the backlog reflects reality —
close what's already done (with evidence), link the confirmed findings to their tickets, and
flag the rest for a human/Claude call.

**Source of truth:** current `main`. Read code at a hard-mirrored clone (`/Users/zach/darkpawns`,
or the dev clone). The C source at `src/*.c` is truth for fidelity questions.
**Linear:** team **DP**, via your Linear access.

### Rules (proven on the 2026-07-08 pass that closed 19 stale tickets cleanly)
1. **Close a ticket ONLY with cited evidence** — a specific commit/PR, file:line showing the
   fix, or a coverage figure. Put the evidence in a closing comment. No evidence → leave open.
2. **When unsure, do NOT close.** Add a comment with what you found and tag it `needs-decision`
   (or note it in your final report). A flagged ticket is a success.
3. **Do not run destructive git/DB commands.** Reads only.
4. **Do not touch source files, branches, or PRs.**
5. **EXCLUDE these tickets** (already being handled — leave them alone):
   - GLM's mechanical brief: **DP-809, DP-817, DP-818, DP-867, DP-788, DP-816, DP-1011**.
   - Just-filed / mine: **DP-1014** (moderation hardening).

---

## Part A — Close tickets whose fix is already merged to `main`

Verify against current `main`, then close with a comment citing the PR/commit + file:line.

- **DP-1013** ("Duration-0 affect contract mismatch") — resolved by **PR #126** (merged).
  Verify `pkg/engine/affect.go`: `IsExpired`/`Tick` now treat `-1` as permanent and `0` as
  expired (matching `src/magic.c affect_update` and `game.AffectUpdate`). Close citing #126.
- Then **scan the other 87 open DP issues** for any whose described bug/feature is already
  present/fixed in `main` (recent merges include the batching/conn/race fixes in #122, the
  RoomCache/whisper fixes in #124, decision-log/dp-goatd/dp-agent/parser fixes in #127, and
  privacy-middleware DP-787 at commit `451c9f2`). For each that's genuinely done, close with
  evidence. Typical candidates: small bug tickets that a merged PR already covers.

## Part B — Re-triage the `[Reek]` coverage-gap tickets

These claim low test coverage for a package (e.g. **DP-608, DP-658, DP-661, DP-662, DP-870,
DP-871, DP-881, DP-892, DP-867**). For each:
- If you can **run `go test -cover ./<pkg>/...`** in a clone, do so; if coverage now meets the
  ticket's bar, close with the coverage number as evidence. If you **cannot run tests**, do
  **not** guess — add a comment `needs coverage re-run` and leave open.
- Several of these also appear as still-open clawpatch findings (see Part C), so if a finding
  confirms the gap is real, keep the ticket open and link them.

## Part C — Link confirmed findings to their tickets

A fresh reconcile of the 55 open clawpatch findings (verdicts by Reek + Claude) confirmed these
are **STILL REAL** in `main` and map to existing DP tickets. For each, add a short comment on
the ticket: "clawpatch confirms still-real in main @ <file:line>", keeping it open. (Do the
NON-GLM ones; GLM is commenting on its own set.)
- **DP-785** ← `Makefile` (deploy-site default creds)
- **DP-881** ← `pkg/game/systems/shop_test.go` (concurrent-txn coverage)
- **DP-702** ← `pkg/scripting/engine.go:167` (mutex held across Lua DoFile/PCall — DoS)
- **DP-759** ← `pkg/storage/interface.go` (Store methods lack context.Context)
- **DP-870** ← `pkg/storage/sqlite.go` (no tests for pkg/storage)

## Part D — Note duplicate findings (informational)

The reconcile found ~7 duplicate finding groups (same bug, different clawpatch signature):
`ratelimit.go` ×3, `test_onboarding.py` ×3, `engine.go` ×3, `cache.go` ×2 (both stale),
`test_emotion_classifier.py` ×2 (DP-1011), `dp-agent/main.go` ×2, `test-race/main.go` ×2,
`test_direct_memory.py` ×2. No Linear action required unless two tickets duplicate each other —
if you find duplicate DP tickets, mark one `duplicate of` the other (don't close blindly).

## Also confirmed STALE (already fixed in main, no ticket — informational only)
`pkg/game/act_comm.go` lastTellers race (code gone), `pkg/optimization/cache.go` Cache.Get
TOCTOU (already write-locked), `pkg/agentcli/config.go` PlayerName validation (added). No action.

---

## Deliverable
A final report (post it as a Linear comment on a tracking issue, or return it):
1. **Closed** — ticket ids + one-line evidence each.
2. **Linked/kept-open** — ticket ids you commented on.
3. **Flagged (needs-decision / needs-coverage-run)** — ids + why.
4. Count summary: closed / linked / flagged, out of 88.
Accuracy over volume. It is far better to flag 20 and close 10 correctly than to close 30 and
be wrong on 5.
