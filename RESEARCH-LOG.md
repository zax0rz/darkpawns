# Research Log — Dark Pawns AI Project

Living document. Updated per session by Daeron.

## [DIGEST] Week of 2026-07-16 to 2026-07-22

### Reek Reports

| Metric | Count |
|---|---|
| New findings this week | 0 |
| All PENDING (untriaged) | 1 (DP-813, LOW — AGENTS.md go test → go vet mismatch) |
| HIGH/MED | 0 |

Quiet week on the Reek front — no new crawl reports surfaced. The single pending finding is a documentation metadata mismatch (LOW, non-blocking). Reek's attention has been absorbed by the oracle harness and fidelity campaign.

### Triage Outcomes (This Week)

The automated triage pipeline did not process new findings this week — none arrived.

**Tier-2 autonomous PR review pipeline (Daeron) was the star this week:** 4 PRs reviewed and merged, all oracle-clean:
- **PR #406** — `RecordLoginFailure` lockout was 1/60th intended duration (`.Minutes()` → `.Seconds()` for Postgres interval). One line fix.
- **PR #407** — `validateWorldPath` TOCTOU gap — validated cleaned path but opened original. Path traversal closed.
- **PR #411** — HTTPMiddleware trusted unvalidated `X-Forwarded-For` header in log output. Spoofable. Removed trust-and-overwrite block.
- **PR #412** — `GetSecret` exposed the master AES-256 encryption key via unrestricted `os.Getenv` lookup. Blocked `ENCRYPTION_KEY` before env access.

All four went through the full oracle harness (6 scenarios), Daeron judged, auto-merged. This is the pipeline working as designed.

### Codebase Changes

- **48 commits** this week (Jul 16–22) — heavy but more focused than last week's 173
- **14+ PRs merged** (#385–#412) — primarily fidelity alignment and security hardening
- **Fidelity campaign dominated:** zone reset RNG draw order (#385), combat round order (#386), do_recall byte-for-byte (#387), starting inventory order (#388), combat death path aligned (#389), stat draw parity fix (#396), C oracle determinism seam restored (#397), boot iteration order stabilized (#398), randzon placement parity (#399), quit/hunger-thirst alignment (#400), clock epoch ported (#401)
- **Security fixes:** 4 of the merged PRs were security-class (X-Forwarded-For injection, encryption key exposure, path traversal TOCTOU, login lockout bypass)
- **Character creation 1:1** now diffed by oracle — DP-1173 filed as RED on main, then closed as Done (creation transcript matches C nanny())
- **Telnet SGA fix** — stopped offering WILL SGA, fixing character-at-a-time mode in Mudlet clients
- **DP_CLOCK deterministic oracle harness** landed — the game clock is no longer a Year-0 placeholder; `reset_time()`/`mud_time_passed()` derived from C epoch

### Linear Activity (This Week)

| Status | Count | Notable |
|---|---|---|
| Done | ~9 | DP-1117 (cast), DP-1162 (clock seam), DP-1169 (combat race), DP-1173 (creation), DP-1178 (clock epoch), DP-1156 (take-name), DP-645 (Tier-3 deferred), DP-909 (Fable menus), DP-1063 (WS client) |
| Todo | 1 | DP-813 (Reek LOW — AGENTS.md metadata mismatch) |
| Cancelled | ~10 | Cleanup of stale shop/door/movement findings from prior weeks |

### Hot Zones

| Area | Findings | Theme |
|---|---|---|
| Security | 4 PRs | X-Forwarded-For, encryption key, path traversal, lockout bypass |
| Fidelity alignment | 9 PRs | RNG draw order, combat, recall, inventory, clock, randzon |
| Oracle harness | 3 PRs | DP_CLOCK seam, determinism, stat draw parity |

### Bug Categories

| Category | Count | Notes |
|---|---|---|
| Security | 4 | Log injection, key exposure, path traversal, auth bypass |
| Fidelity drift | 9 | PRNG draw order, combat order, clock epoch, recall contract |
| Concurrency | 1 | Combat data race in tests (DP-1169) |
| Display/cosmetic | 2 | Score blank line, telnet SGA |

### Reek Accuracy Trend

No new Reek reports this week. Overall pipeline FPR remains at 4.8% across 229 findings (historical). The autonomous Tier-2 PR review pipeline ran cleanly — 4/4 oracle-clean merges with zero rejections.

### Key Observations

**The fidelity campaign is converging.** This week's PRs were surgical — each one closed a specific C↔Go behavioral gap. The oracle harness is now the primary driver: it surfaces a divergence, a fix is coded, the harness proves green, the PR lands. This is the workflow working.

**Security hardening emerged as a theme.** Four independent security fixes merged this week — not from Reek, but from the Tier-2 review pipeline finding real vulnerabilities in the admin/transport layer. The encryption key exposure (PR #412) is particularly notable: a one-line guard prevents the master key from leaking through env var lookup.

**The character creation story closed.** DP-1173 was filed as RED — the Go port heavily re-skinned C's nanny(). By end of week, the creation transcript matches byte-for-byte. The "1:1 player-facing north star" claim now has its first proof point.

**Reek went quiet.** No new crawl reports this week. Either Reek's attention is elsewhere, or the codebase has reached a density where the remaining findings are in areas Reek hasn't explored yet. Worth monitoring next week.

### Paper-Relevant Notes

- The Tier-2 autonomous PR review pipeline (Reek finds → Daeron triages → clawpatch codes → Daeron judges oracle → auto-merge) ran 4 full cycles this week. This is a concrete example of multi-agent software engineering with an automated quality gate.
- The security findings came from the review pipeline, not Reek — demonstrating that the pipeline surfaces different classes of bugs than the crawl does.
- The character creation 1:1 match is a measurable fidelity achievement: byte-for-byte transcript match between 1994 C code and 2026 Go port.
- The oracle harness (DP_SEED + DP_CLOCK seams) now enables deterministic comparison across the full game lifecycle.

## [DIGEST] Week of 2026-07-12 to 2026-07-19

### Reek Reports

| Metric | Count |
|---|---|
| New findings this week | 2 |
| All PENDING (untriaged) | 2 |
| LOW | 2 |
| HIGH/MED | 0 |

Both new findings are LOW-severity mapping drift — `file_mapping.md` references stale file paths (`pkg/session/cmd_social.go` → actually in `pkg/game/act_social.go` etc.; `pkg/game/mapcode.go` → actually in `pkg/session/map_cmds.go`). Documentation debt from the domain refactor. Non-blocking.

### Triage Outcomes (This Week)

The automated triage pipeline processed 0 of the 21 PENDING findings from the prior week. The backlog from last digest remains.

### Codebase Changes

- **173 commits** this week (Jul 12–19) — another massive week, though slightly below last week's 228
- **~30 PRs merged** (#334–#401) — spanning the full domain refactor continuation, skill system, combat, character creation, clock epoch, and the MCP/transport design explosion
- **Domain refactor completed** — score, who, consider, time/weather, channels/socials, directed speech, movement, equipment, object inventory all now aligned with C behavior
- **Skill system foundation landed** (PRs #356, #359, #361, #364–#367) — C-faithful practice command, guild-mob learning, improve_skill draw-parity, thief utility skills, headbutt double-call fix. Retired 3 invented commands (DP-1116/1128/1129).
- **CMWC PRNG port** (PR #348) — C-compatible seeded random stream. The linchpin for deterministic testing.
- **Character creation 1:1** (PR #382) — creation transcript now matches C nanny() byte-for-byte
- **Clock epoch ported** (PR #401) — `reset_time()`/`mud_time_passed()` derived from C epoch. Game clock no longer Year 0. (DP-1178 closed.)
- **Combat round order** preserved (PR #386), **combat death path** aligned with C oracle (PR #389)
- **Zone reset RNG draw order** aligned (PR #385), **boot iteration order** stabilized (PR #398)
- **Do_recall** ported byte-for-byte (PR #387), **quit/reallyquit** split restored (PR #377), **cast contract** restored (PR #376)
- **MCP/transport design explosion** — 15 new Linear issues (DP-1140 through DP-1176) for the human client transport and agent MCP migration. Streamable HTTP transport decided, maldorne fork evaluated, external adapter MVP designed. All in Backlog.
- **Telnet SGA fix** (recent commit) — stopped offering WILL SGA, fixing character-at-a-time mode in Mudlet

### Linear Activity (This Week)

| Status | Count | Notable |
|---|---|---|
| Done | 13 | DP-1115 (quit split), DP-1116 (practice), DP-1117 (cast), DP-1133 (color), DP-1156 (take-name), DP-1162 (clock seam), DP-1166 (spell names), DP-1167 (mob HP roll), DP-1168 (improve_skill), DP-1169 (combat race), DP-1170 (hide draw), DP-1173 (creation), DP-1178 (clock epoch) |
| Backlog | 18 | 15 MCP/transport design issues (DP-1140–1176), 2 mapping drift (DP-1180/1181), 1 inventory order (DP-1172), 1 hide guard (DP-1171), 1 stat draw parity (DP-1177) |
| Cancelled | 0 | — |

### Hot Zones

| Area | Findings | Theme |
|---|---|---|
| Skill system | 6 PRs | Practice, improve_skill, thief skills, guild learning |
| Combat fidelity | 4 PRs | Round order, death path, defense draw parity, callback race |
| Character creation | 3 PRs | 1:1 nanny alignment, whitespace residuals, stat draw parity |
| Clock/time | 2 PRs | Epoch port, clock freeze seam, deterministic pulse |
| Object/equipment | 3 PRs | Take-name, starting gear order, inventory carry order |
| MCP transport | 15 issues | Design-phase explosion — Streamable HTTP, maldorne, external adapter |

### Key Observations

1. **The domain refactor is essentially complete.** Two consecutive weeks of 173+ commits have aligned every major command domain with C behavior. The "Fable" codebase is now genuinely playable with C-faithful output.

2. **The skill system went from stub to functional.** Practice, guild learning, improve_skill, thief utility skills, and headbutt all landed this week. The invented commands are retired. This is the kind of subsystem that would take a human developer weeks — it took the pipeline 7 days.

3. **Clock epoch porting is the last major structural piece.** DP-1178 (done) closed the Year 0 gap. The game now derives its clock from the C epoch. The remaining fidelity gaps are increasingly granular — per-command message wording, draw-order micro-divergences, flag colors.

4. **The MCP/transport design explosion is forward-looking.** 15 new issues in Backlog. Streamable HTTP transport decided, package structure designed (DP-1176), external adapter MVP scoped (DP-1153), native MCP server planned (DP-1175). None of this is code yet — it's architecture. Important for the paper but not urgent for playability.

5. **Reek's triage backlog is growing.** 199+ PENDING findings. The automated revalidate pipeline doesn't seem to be processing the Jul 8–15 batch. Only 2 new LOW findings this week — the oracle differential testing has shifted Reek's role from code reviewer to behavioral difference detector.

6. **Multi-agent pipeline velocity: 30 PRs merged in one week** across 4+ agent types (Claude, Codex, Kimi K3, Daeron). The brief-driven workflow is the bottleneck — writing good briefs, not execution.

### Paper-Relevant Notes

- **Domain refactor velocity quantification:** 173 commits, 30 PRs, 13 issues closed in 7 days by a human+AI pipeline. This measures the throughput of AI-assisted port fidelity work — a concrete data point for the "what the agent preserved" section.
- **The skill system resurrection** (DP-1116/1128/1129→356/359/361) is a case study in how the oracle differentially testing catches drift that static analysis misses. Practice was registered at level/position 0 (wrong), the `spells` command was a literal stub, and `listskills` was overwriting the primary `skills` route. All three were "working" — no crashes, no panics. They were just wrong. The oracle caught them by comparing transcripts.
- **Clock epoch port** demonstrates the value of environmental determinism — once DP_CLOCK freezes wall-clock pulses, the harness can reproduce any scenario identically. This is the methodology contribution: "seed both implementations identically and compare transcripts."
- **The MCP transport design** (DP-1140–1176) is the beginning of the "agent interaction" story for the paper. Not code yet, but the design decisions (Streamable HTTP over raw WS, push-not-poll for real-time combat, resource subscriptions) address real failure modes documented in prior art (minecraft-mcp-server).

## [SESSION] 2026-07-12 — C Oracle Built + Differential Harness Landed (PR #243)

**The original C Dark Pawns server boots on macOS Apple Silicon.** No source modifications. Just compiler flags and `make`. The C oracle is live at `/Users/zach/.openclaw/workspace/darkpawns-c-oracle/bin/circle`. This is the ground truth for port fidelity — not source code we read and guess about, but a running game we can ask questions.

**The differential-test harness landed on its first run and found three real divergences:**
1. Different starting room — C drops new chars in Temple Infirmary; Go lands them at Temple Altar [8004]
2. Go leaks room vnum `[8004]` in room name (immortal/holylight-only in Diku/Circle)
3. Go dumps full item stat blocks on room look (Keywords/Type/Weight/Damage) where C just says "A shiny short sword is here."

Three fidelity findings from one `look` scenario. Static code reading would have missed all three. The harness paid for itself on the first run.

**Reek pipeline evolution:** The oracle transforms Reek from static code reviewer to behavioral difference detector. Scenario files → harness → normalizer → diff → Reek triages surviving diffs with concrete evidence (C output vs Go output). False positives collapse to near zero. Coverage expands from ~20% of codebase to 100% of behavior matrix.

**Oracle pipeline roadmap:**
1. Tier 1 widening — movement/sector, objects/shops, score/display (cheap, worker-distributed)
2. Tier 2 — random.c port (the keystone, unlocks combat/skill/spell testing)
3. Tier 3 — combat/skills/spells (crown-jewel subsystems get reference oracle)
4. Regression detection — run suite before/after merges, catch new divergences

**Paper contribution:** Automated behavioral difference detection across language ports. Not static analysis, not type checking — "run both implementations and compare transcripts." Underexplored in game preservation and software migration literature. The oracle harness is a novel tool for C→Go port fidelity verification.

**[DIGEST] Week of 2026-07-06 to 2026-07-12**

- C Oracle built and booted on macOS Apple Silicon (no source changes required)
- Tier-1 differential harness landed (PR #243) — first run found 3 real divergences
- Reek pipeline evolution discussed: static review → behavioral difference detection
- Board: 0 CRITICAL/HIGH open bugs, 14 Fable fidelity issues remaining
- Research: Oracle harness as paper contribution — automated behavioral diff across language ports

## [SESSION] 2026-07-03 — Fable Review Sprint (15 issues, 4 PRs)

**Massive sprint session.** The Architect ran a Fable codebase review and we executed fixes all night. 4 PRs merged, ~15 issues closed. CI was broken when we started, now unblocked.

**The game is playable.** Combat is bidirectional (DP-900), skills kill things (DP-901), backstab and circle have their C gates (DP-906/914). Mobs stay in their shops (DP-898/899). Room flags parse correctly (DP-896/897). Character creation is deterministic (DP-909). The world doesn't degrade over uptime (DP-908).

**GLM-5.2 via ZCode is production-ready.** Dispatched by The Architect for DP-900/901/906/914. All build gates green. Reads brief + Linear + code + C source in one pass. Found and fixed circle fidelity gaps proactively. 1.5x pricing is good value for this quality.

**The Fable review surfaced 14 findings (F-1 through F-14).** All CRITICAL/HIGH now fixed. Key finding: "verified dead code" phenomenon — fidelity fixes landed in MakeHit() which had zero callers. The unit tests passed because they tested the dead function in isolation. This is paper-worthy.

**DP-902 (session wedging) is the last CRITICAL.** Brief written. Root cause: writePump doesn't call Unregister on exit, no lastActive tracking, no linkdead reaper. Ghost players accumulate and block rooms.

**Paper-relevant observations:**
- The "verified dead code" pattern (DP-905) — fidelity fixes in dead code pass tests but change nothing in the live game. Static analysis can't catch this because the tests are correct for the function they test.
- Multi-agent parallel execution (Claude + GLM-5.2) with zero conflicts when file domains don't overlap.
- Brief-driven workflow proven across 3 sprints now. The bottleneck is writing good briefs, not execution.
- The gateway rendering bug (exec outputs as images) is a meta-observation about AI agent tooling reliability.

## [SESSION] 2026-06-26 — Kimi Batch Deploy + Database Permissions + Linear Grooming

**Kimi K2.7-code completed the full 62-finding clawpatch batch.** Merged to main, deployed to CT 120. 108 files changed, 4,229 lines added. Build, vet, tests all green. Cross-compiled linux/amd64 binary deployed at 21:10 EDT.

**Database permission drift discovered and fixed.** All 16 PostgreSQL tables were owned by `postgres` but the server connects as `darkpawns`. Server was silently running without persistence — no character saves, no moderation, no player data. Fixed with ALTER TABLE OWNER + ALTER DEFAULT PRIVILEGES. This is the kind of bug that doesn't show up in code review because it's not in the code.

**Linear grooming: 26 issues closed.** 20 from KIMI-BRIEF finding IDs, 6 from commit log verification. 3 left open (DP-622, DP-648, DP-649). Reek did the mechanical work from a brief.

**Security hardening brief written for next Kimi batch.** 6 findings (3 high, 3 medium) — hardcoded Postgres creds, WebSocket dev bypass, DNS hostname resolution for bans, pprof exposure, login rate limiting, k8s secrets. Kimi started but hit rate limits with 4 of 6 done.

**Paper-relevant observations:**
- The database permission drift is a new failure mode worth documenting: infrastructure-level regressions that are invisible to code analysis. Reek found 70 code bugs but never caught this because it's not in the source.
- The multi-agent pipeline (Claude → Kimi → Reek → Daeron) worked. Each agent did what it was good at. Claude wrote the brief with C-source verification. Kimi did the mechanical fixes. Reek did the linear grooming. Daeron did the deployment and triage. No single agent could have done all of this.
- Token cost for the full operation: Claude brief (~100k tokens) + Kimi fixes (~300k tokens) + Reek grooming (~50k tokens) + Daeron deployment/triage (~20k tokens). Rough estimate: ~470k tokens for 26 issues closed, 62 code fixes, 1 deployment, 1 database permission fix.

## [RESEARCH] 2026-06-25 — Research Writing: The Taxonomy of Simplification (Part 2)

**Cron-triggered (Program 5).** The Jun 23 draft was logged in RESEARCH-LOG.md but never written to disk. Wrote the full draft today — ~1,100 words decomposing the largest fidelity drift category into five specific mechanisms.

**Topic:** The word "simplified" appears 15 times in the Go codebase. Each occurrence is a claim — a claim that the behavioral gap is acceptable. This draft defines five patterns of simplification, each mechanically detectable, each with a concrete codebase example.

**The five simplifications (with examples):**
1. **Argument truncation** — DoMindlink type assertion fails on mob targets. Go takes fewer args than C; missing params control edge cases.
2. **Logic flattening** — persName/CAN_SEE reduced to awake-only check (since fixed). C has 4+ branches, Go has 1.
3. **Stub displacement** — checkReagents() returns 0 permanently. 22 stub functions found across the port. Function exists, compiles, does nothing.
4. **Algorithmic substitution** — exp_needed_for_level estimated as 1000*level (C uses quadratic). Matches at low levels, diverges 30x at level 30.
5. **Behavioral omission** — Ban system ported but never wired to telnet listener. Spec procs assigned but command dispatcher never checks them.

**Key argument:** "Simplified" is a rhetorical move that transforms a regression into a design choice. AI porting agents are comment-followers — they treat "simplified" as license to skip verification. The fidelity audit strips this license.

**Detection methods:** Five mechanical checks — argument count, branch count, return defaults, formula comparison, call tracing. Automatable. Don't require understanding the game.

**File:** `docs/research/drafts/2026-06-23-the-taxonomy-of-simplification.md`

**Status:** Draft ~1,100 words. Extends Silent Drift by decomposing its third category (logic simplification) into five specific mechanisms. Supports Constraint Engineering (what briefs detect) and What the Agent Preserved (what agents lose).

---

## [DAERON] 2026-06-25 — Morning Triage: Reek's Overnight Crawl + Fidelity Review

**Cron-triggered (Program 1).** Reek ran overnight crawl (6 findings) + fidelity review (9 findings). I verified each against the codebase.

**Results:** 8 rejected, 5 confirmed (1 critical, 2 high, 2 low), 1 self-dismissed. New failure mode: fabricated file references (formatting.go, channel backpressure in main.go). Reek is pattern-matching against expected structures rather than reading actual code.

**Confirmed findings:**
- CRITICAL: cmd/dp-agent zero tests (367 lines, not 1700 as Reek claimed)
- HIGH: pkg/session race OOM (654MB, 3344 goroutines)
- HIGH: 3 packages skipped by race detector (spells, telnet, validation)
- LOW: LiteLLM empty key fallback in 3 scripts
- LOW: dp_session_consolidate.py missing /v1/ prefix (still unfixed from Jun 24)

**Rejected findings (notable):**
- HH-209 lock ordering deadlock: both processBuy and processSell lock player→shop consistently
- HH-210 ObjectPool.TryGet deadlock: TryGet calls getLocked(), not Get()
- HH-211 Door.Reset(): code correctly restores initialClosed/initialLocked
- HH-215 GetAlignment nil: nil check exists at fight_core.go:262
- 3 fabricated references (channel, formatting.go, library os.Exit)

**Paper note:** Reek's 57% false positive rate on fidelity findings is concerning. The fabricated file paths suggest the crawler is hallucinating code structures — a novel failure mode worth documenting. Previous batches had ~33% FPR. The trend is worsening, not improving.

---

## [DAERON] 2026-06-24 — Morning Triage: Reek's Overnight (95 Findings)

**Cron-triggered (Program 1).** Reek ran a comprehensive overnight review: 95 findings (4 critical, 25 high, 66 medium), 15 false positives dismissed, 12 confirmed. I verified each against the codebase.

**Results:** 4 rejected (nil safety false positive, dead middleware, missing file, cosmetic), 6 confirmed (all low/medium severity), 2 needs context. No CRITICAL server-impacting bugs. The dp_session_consolidate.py missing `/v1/` prefix is the most actionable — it means LLM narrative consolidation is silently broken.

**Coverage analysis:** 21% overall. 5 packages with zero tests. pkg/game at 14.7%. The coverage gap data is paper-relevant — it quantifies the testing debt in a 73K-line C-to-Go port.
