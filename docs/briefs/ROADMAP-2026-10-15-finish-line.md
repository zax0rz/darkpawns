# Roadmap: the finish line (week of 2026-10-15)

Baseline: `docs/research/field-notes/2026-10-15-finish-line-baseline.md` (PF-070), measured on main `fd327ae1a` = production.

**The definition of done doesn't change.** It's still R1–R5: every C surface unit proven or recorded as an exclusion or approved divergence, with byte parity *and* draw parity, on a clean census. What changed is how we measure progress: by C surface units, not authored cases.

## The scoreboard

| Measure | Baseline 2026-10-15 | Target |
|---|---|---|
| Depth-case rows not blocked | 5,588 / 5,659 (98.7%) | 100% (or recorded) |
| **C surface units blocked** | **4,031 / 4,929 (82%)** | 0 |
| Coincidence greens (DRAW rows) | DRAW_BASELINE | 0 |
| Blocked depth cases (non-surface) | 11 | 0 |
| Populated-world soaks | parked (DP-1439) | green, in the corpus |

## Workstreams

### 1. Luna: map the blocked surface (`~/LUNA-GOAL-2026-10-15-surface-mapping.md`)

Per C file: enumerate every `act()`/`send_to_char()` site, map it to the proofs that execute it, and classify it as proven, gap-reachable, gap-blocked or unreachable. Then fix the reachable gaps. The surface inventory was last audited on 2026-09-04, before DP-1371's thousands of proofs, so much of the 82% should flip without new code. Start with the biggest files: `act.wizard.c` 352, `new_cmds.c` 204, `spec_procs.c` 179, `clan.c` 165, `spec_procs2.c` 129, `spells.c` 102.

### 2. GLM: the nine blocked cases (`~/GLM-BRIEF-2026-10-15-blocked-cases.md`)

- **Family A, the NPC command interpreter:** one root under `force`, `lua` and `turn`.
- **Family B:**
  - flee transitive movement failures;
  - Lua nested writeback;
  - mudlog's unported sites (DP-1340);
  - zone reset state;
  - the two saved-zero-stats rows (persistence-adjacent).
- **Extras:** DP-1436 and DP-1437.

### 3. DeepSeek: draw parity to zero (brief to follow)

Every DRAW row in the baseline run is a coincidence green: bytes match on a drifted stream. Triage by first-mismatch cluster and fix by class. Known starting points:

- God-vs-mob round order (the five shop rows; GLM's DP-1438 follow-up);
- `carve-depth`'s warmup draw count;
- DP-1177 (newbie stat-roll draws, zones 150/165);
- DP-1345 (seed-2 `god-skill-keys`);
- DP-1369 (milestone reward `number(1,3)`).

### 4. Claude: harness, batching, triage

- **DP-1427 v2:** draw parity inside `--combined`, plus multi-seed, so a batch gets one run instead of two. Census harness, so Zach merges.
- **The surface map checker** (`make` target) after Luna's first PR. Tooling, so Zach merges.
- **Batched censuses and deploys** as PRs land. Deploys need Zach's explicit "go" in the turn.
- **Linear hygiene:** close-candidates are below. Confirm each against main before closing.

### 5. Zach + Claude together (stop tier)

- **DP-1439:** long clock pulses lose Go's actor output, which blocks the populated-world soaks. It could also mean real players lose output under load. **Highest-value stop-tier item.**
- **DP-1367:** the Go server keeps running when its telnet bind fails, so the harness can drive another worker's server. Census integrity.
- `entry.browser-name-routing` and `entry.identity-consumers`: login and identity.
- DP-1423, DP-1424, DP-1425: clan `is_playing`, `PLR_IT` untag at load, and `Crash_save_all` skipping editors.

## Linear: important open items, triaged

**High, active fidelity**
- DP-1439 (above)
- DP-1367 (above)
- DP-1177 (draw parity, DeepSeek)
- DP-1295: `canSee` ignores darkness and holylight (`LIGHT_OK`). Broad visibility fidelity, not yet assigned. A candidate for GLM after Family A, or for the next DeepSeek round.
- DP-1313: Temple of the Cross accumulated bulletin boards on the live server. **Check live**: Zach saw one board on 2026-10-15.

**Close candidates (verified 2026-10-15; confirm before closing)**
- DP-1394: the "deterministically failing" luaedit test passes on main.
- DP-1398: steal-depth coincidence greens in room 8162. Fixed by #1896 (2026-10-10). Re-confirm with the DRAW baseline.
- DP-1321: after flee, commands still answer as if fighting. Fixed by #1693 (2026-09-28).
- DP-1413: `mallory` never sends Mickey. The special is assigned to no mob in the 1997 world, so it's moot until post-launch content review.

**Medium, schedule after the streams above**
- DP-1319 (mob fight scripts get no target vs mobs; `snake.lua` errors every round in production)
- DP-1227 (pager for long listings)
- DP-1361 / DP-1348 (unported `track`, `olc`)
- DP-1341 (`enchant weapon` hold)
- DP-1331 (`mag_damage` floors)
- DP-1372 (group autogold names the killer)
- DP-1377 (vampire daylight drink)
- DP-1393 (`restore` numbered targets)
- DP-1352 (idle timer)
- DP-1407 (`dc` link-loss messages)

## Cadence

- **Agents:** targeted censuses while working, then PR and keep going.
- **Claude:** stacks ready PRs, runs one combined census per batch, merges on clean, and stages a deploy.
- **Measure weekly:** re-run the baseline measurement (surface units blocked, DRAW rows) and add a dated row to the field note's table.
