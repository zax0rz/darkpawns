# 06 — Modernization Roadmap

Derived from reports/01–05 (same session, same data). Ranking formula per the brief: **(estimated lines removed × oracle coverage) ÷ risk**. Every item is a pure-refactor PR claiming byte-equivalence, judged by the oracle, one target per PR. Nothing here mixes cleanup with behavior change.

## The ordering test — verdict against the brief's hypothesis

The brief predicted "duplication collapse first (table-driven socials could delete thousands)." **The data inverts it**: the socials table-driven collapse *already happened at port time* (reports/03), so the duplication phase is smaller than hypothesized (~6.4–7.1K lines total, and ~2.7K of that non-social). What the data actually supports:

1. **Zero-risk deletions first** (~3.3K lines, no oracle dependency at all) — faster than any oracle-gated work and banked immediately.
2. **Socials loader-ization second** (−3,760) — the single biggest bite, near-zero risk, gated on the existing social corpus and the verified C command-surface exclusions (the former “15 missing scenarios” count was stale).
3. **The rest of duplication** (~2.7K) is real but modest; C-isms mechanical waves (~2–4K) are comparable in size.
4. **The spine stays last-or-never**, and one item (combat-ticker unification) is "never" absent a dedicated multi-week R3 project.
5. **Honest total: the census supports ~15–20K lines of identified removals ≈ 7–9% of the module** — not the 30–50% a generic C→Go cleanup narrative promises. The port already paid most of that debt. Further shrink requires architectural decomposition of pkg/game (91K lines), which the fidelity law prices at high risk for low byte payoff; the roadmap recommends against it except the one M-04/M-07 extraction below.

---

## Phase 0 — Prerequisites (no deletion; ~2–3 days)

| # | Item | Why | Verifies |
|---|---|---|---|
| 0.1 | Add `make oracle-regression` and record the full-corpus wall-time on the box where `darkpawns-c-oracle` lives | "The oracle is the throttle" (ground rule 3) and the BASELINE section is incomplete without it | full scenario corpus |
| 0.2 | Correct the stale “15 missing social scenarios” request: the social queue through `yuball` is already covered; `hiss`, `kneel`, and `mutter` are present only in `lib/misc/socials` and absent from C's command table, so they are verified-excluded under R2/R4/R5e | Unblocks loader-ization without inventing commands | existing social scenarios and C command table |
| 0.3 | Fidelity ticket and bugfix: `main.go:212` wires `systems.ShopManager`, while world accessors previously asserted `*game.ShopManager`; bridge parsed `.shp` records to the live session lookup and preserve the larger economy surface as blocked | Resolves the confirmed live-broken shadow boundary before consolidation | `shop-stack-list-live`; remaining shop rows |
| 0.4 | Stale-doc corrections: AGENTS.md now records the measured 8 non-production `#nosec G104` annotations; current handoffs and this roadmap use the corrected 4,758-case baseline | Prevents future agents from cleaning up phantom work | n/a |

## Phase 1 — Zero-risk deletions (measured −2,840 lines in merged #1385; no oracle gate)

| # | Target | Δ lines | Risk | Oracle |
|---|---|---|---|---|
| 1.1 | Delete `pkg/game/socials.json` (2,492) + `socials.txt` (187) — zero code references (one rg sweep first) | −2,679 | GREEN | none needed |
| 1.2 | Correct the stale commented-code heuristic (628 lexical matches; no wholesale deletion) — the largest cluster is API/source-mapping commentary above live handlers, with other matches being C formula transcriptions and test/golden notes | 0 | GREEN | `docs/fidelity/depth/handoff/2026-09-04-modernization-phase1.md`; R4 |
| 1.3 | Delete 3 dead funcs/files (`cmdNotBuy`, `cmdInfo`, dp-goat `applyAuthFormat`) — after confirming registration | −161 | GREEN | none needed |

**Phase delta: −2,840 lines of source deleted by merged PR #1385.** Note: 1.1 landed before 2.1 so the loader-ization diff is not entangled with artifact deletion.

## Phase 2 — Socials loader-ization (−3,760; the biggest bite)

| # | Target | Δ lines | Risk | Oracle / deps |
|---|---|---|---|---|
| 2.1 | Replace `pkg/game/socials.go` table (1,148) with `//go:embed lib/misc/socials` + ~60-line parser — precedent `lib/misc/messages_embed.go`; guard with existing `TestSocialTableMatchesCData` + 187 social-led depth scenarios | **≈ −3,760** (1,148+2,678 artifacts → ~60+embed) | GREEN-after-0.2 | all social scenarios; dep 0.2, 1.1 |

## Phase 3 — Shop stack resolution (−1,250; gated on 0.3)

| # | Target | Δ lines | Risk | Oracle / deps |
|---|---|---|---|---|
| 3.1 | Consolidate the remaining shop duplication (systems stack 1,402 L or the legacy path) to ONE shop engine. The confirmed live lookup bugfix is already isolated; do not claim the broader economy rows until their C-byte/state proof exists | **≈ −1,250** | YELLOW→GREEN after remaining proof | blocked cluster `show.shops-list`, shop-adjacent scenarios; dep 0.3 |

## Phase 4 — Mechanical handler dedup (~−930; all DEPTH-PROVEN handlers)

One PR per cluster; each re-verified by its units' named scenarios.

| # | Target | Δ lines | Risk | Verifying cases |
|---|---|---|---|---|
| 4.1 | Clan command family: route 9 `doClan*` through existing `resolveClanForImmortal` (clans.go:411) | −250 | GREEN | clan* scenarios (depth-proven) |
| 4.2 | Skill-command prologue: shared nil-player / CanUseSkill / OneArgument+FindTargetInRoom helpers (pkg/command) | −275 | GREEN | skill_commands.go's 45 proven units |
| 4.3 | Small pairs: tell/reply, sneak/stealth, SendToAll/Outdoor, buildRoomMobs/Items, parse×3, spike/stake, mail read/write, Sprintbit/sprintnbit | −180 | GREEN | per-unit scenarios |
| 4.4 | Channel wrappers: one parameterized cmdChannel (gossip/shout/holler/gratz/auction/newbie) | −58 | GREEN | channel scenarios (15/17 proven) |
| 4.5 | Position commands: table-drive DoStand/Sit/Rest/Sleep | −50 | GREEN | position scenarios |
| 4.6 | Verbatim dupes: `FindExp`≡`findExp` (57 L), `CalcLevelDiff`≡`calcKillXPShare` (~30) — dedupe via export, fix the import cycle that caused the LVL_IMMORT literal | −85 | GREEN (trivial) | xp/level scenarios + unit tests |
| 4.7 | FireMobFightScript/FireMobDeathScript shared spine (world_scriptable.go) | −37 | GREEN | mobprog fight/death scenarios |

## Phase 5 — Spec-proc families (~−460; per-family oracle-gated)

| # | Target | Δ lines | Risk | Verifying cases |
|---|---|---|---|---|
| 5.1 | Elements×7 shared teleport spine | −90 | YELLOW per proc | each element spec's scenario rows |
| 5.2 | Tattoo×4, castle-guards×3, undead-knights×2, fighter/paladin gate+picker, combat-gate idiom (~25 procs) | −370 | YELLOW per proc | spec-procs.tsv rows (241 scenario-proven, 243 unit) |

## Phase 6 — C-isms mechanical wave (~−2,300; subset of reports/02)

| # | Target | Δ lines | Risk | Verifying cases |
|---|---|---|---|---|
| 6.1 | Giant switches → data tables where mechanical: `wiz_set` toggle majority (51 cases), `findExp` class/level ladders, equipment slot↔name maps, small lookup switches | −1,200–1,800 (mechanical subset only) | GREEN for the listed ones; spell-dispatch consolidation is **NOT here** | affected proven units |
| 6.2 | String cleanup in proven files: 94 nested Sprintf → flatten; 57 loop-concats → Builder | −200–400 | GREEN | per-file unit scenarios |
| 6.3 | Keyed tables + constants: key the unkeyed data tables (spellDB 509 L, THAC0, saving-throws 1,943 L), name the literal ladders, single-source LVL_IMMORT | ~0 net (churn) | GREEN | unit tests; **highest bug-class value per line in the census** |
| 6.4 | Global→struct injection: mail.go (8 globals), weather.go, merge_bridge banManager, spec_assign registries | −250–400 | YELLOW (behavior-adjacent) | weather/mail/ban scenarios + unit tests |
| 6.5 | Production `_ =` → handle-or-slog (AGENTS.md:63) | ~0 (adds lines) | GREEN | build + vet |

## Phase 7 — YELLOW promotions (case-writing waves; enables nothing by itself but enlarges every later bite)

Priority order by downstream unlock: shoot state machine (9 cases) → shared combat/breed transcript (6) → show report surfaces (6) → OLC-family decision (15 blocked: *decide* whether Go keeps emitting `Huh?!?` — a deliberate divergence ticket — rather than porting OLC) → persistence-dependent last/wizlock (2) → staging gaps. **Every case written here converts YELLOW files to GREEN and is reusable proof forever.**

## Phase 8 — Spine: last or never (estimated net −400–800, mostly churn)

| # | Target | Verdict |
|---|---|---|
| 8.1 | Actor-interface expansion (kill the 35 Player/MobInstance dual type-switches, 17 files) | LAST. Only after Phase 7's combat/breed cases are green; behavior-risk on every case site |
| 8.2 | Spell-dispatch consolidation (≥6 fragmented `switch spellNum` blocks → one table) | LAST. Per-spell oracle cases needed; dice order inside dispatch is sacred |
| 8.3 | Combat-ticker → heartbeat unification ("Phase 2") | **NEVER** as a cleanup. Only as a dedicated R3 project with draw-parity proof (DP_DRAW_LOG diff vs C) — it changes draw interleaving vs the oracle by design |
| 8.4 | M-04 command-registration extraction (init() → explicit registry) + M-07 App-struct wiring | Cheap, behavior-invisible, but touches cmd/server + dispatch: YELLOW→GREEN once 0.1's regression cadence exists; do it early in this phase, not with 8.1–8.3 |

---

## Ground rules (encoded)

1. Pure-refactor PRs only; each claims byte-equivalence, judged by oracle re-run of the affected units' scenarios.
2. Baseline before everything (below); every phase proves its delta.
3. The oracle is the throttle: pace = oracle runtime (to be measured, 0.1), not token count. During long oracle runs, work Phase 7 case-writing (it needs no oracle).
4. Honesty: named above — **~15–20K identified lines ≈ 7–9%**; the 30–50% shed narrative does not survive contact with this census because the port already paid the debt. What the census cannot quantify: whether pkg/game *should* be split for human comprehensibility (91K lines / 364 files) even though bytes don't demand it — that is a maintainability judgment above the oracle's pay grade, and this roadmap's recommendation is: don't, beyond the shops/systems seams, until a concrete team need appears.

## BASELINE

Current (2026-09-04, corrected terminal snapshot):

| metric | value | source |
|---|---|---|
| Go files / lines (total incl. dp-goat) | 1,098 / 229,275 | `reports/00-baseline.md` |
| Main module lines / packages | 219,777 / 48 | `go list` |
| Build time (warm cache) | 6s | measured |
| Binary size (cmd/server, linux amd64) | 25,924,974 bytes | measured |
| Oracle runtime | **recorded by `make oracle-regression`** | 934 scenario files exercise the 4,758 modeled cases; 934/934 passed with 0 failed, 0 infra, and 0 timed out in 7,291.902s (2:01:31.902), 2026-09-04 16:40:06–18:41:38 EDT; see the dated terminal handoff |
| Ledger state | 4,758 cases / 4,653 proven / 54 blocked / 51 excluded | `make fidelity-depth` |

Track per phase (record after each phase merges, in this table's continuation in the real repo's `docs/modernization/`):

| phase | Δ lines (cum) | build time | oracle runtime | binary bytes |
|---|---|---|---|---|
| 0 | 0 | 6s | *fill from 0.1* | 25,924,974 |
| 1 | −2,840 (merged #1385) | | | |
| 2 | ≈ −7,100 | | | |
| 3–8 | *fill as landed* | | | |

## Dependencies (critical path)

0.1 ∥ 0.2 ∥ 0.3 (parallel) → 1.x (no deps) → 2.1 (needs corrected 0.2, after 1.1) → 3.1 (needs the 0.3 bugfix plus remaining shop proof) → 4.x/5.x/6.x (independent of each other; 4.2 wants its units' cases, 5.x per-proc) → 7 (parallel whenever oracle is busy) → 8 (needs 7's combat cases for 8.1/8.2; 8.4 anytime).

**First PR recommendation:** 0.1, the corrected 0.2 disposition, and the isolated 0.3 bugfix establish the evidence boundary; then land 1.1 and 2.1 as separate deletion/loaderization PRs only after their changed-file coverage lookups are proven.
