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

### Phase 5.1 measured result (2026-09-11)

Phase 5.1 is already present on current `origin/main` through merged PR #1399
(`ecafde5302596e29d525617513bbd033d2c78727`, based on
`38c4f1b35875a28fb5fcf93fb5257951fa9ad1a2`). The measured delta is the
reviewed source diff, not the roadmap estimate:

| scope | before | after | measured delta |
|---|---:|---:|---:|
| production `pkg/game/spec_procs3.go` | 1,885 physical lines / 1,748 nonblank | 1,887 physical lines / 1,748 nonblank | **+127 / −125, net +2 physical lines** |
| tests and oracle scenarios | unchanged | unchanged | **0** |
| Phase 5.1 handoff documentation | absent | 48 lines | **+48** |

The production change shares the player-only transfer spine across
`elements_master_column`, `elements_platforms`, and `elements_galeru_column`,
and shares only the proven player ordering helper in `elements_galeru_alive`.
`elements_load_cylinders`, `elements_minion`, and `elements_guardian` have no
shared teleport branch and remain local. The estimate of −90 is retained as a
planning estimate only; line reduction is not an acceptance criterion.

### Phase 5.2 audit result (2026-09-11; [PR #1442](https://github.com/zax0rz/darkpawns/pull/1442))

Merged PR #1400 (`ad3a9e7db70169fd02e4be51183b6f33ab97976f`) is present on
current `origin/main` at `18b5a911ce75702e5ab71df4aa46e87baa00c076`, with no
later changes to its production paths. The original measured production delta
is `pkg/game/spec_procs.go` +34/−41 and `pkg/game/spec_procs2.go` +77/−193,
for **+111/−234, net −123 physical lines**. Tests and oracle scenarios were
unchanged; #1400 added 46 handoff-documentation lines. This is separate from
the roadmap estimate of −370.

The active registered tattoo ×4, castle-guard down/up/north ×3, and
fighter/paladin shared gate/target-picker behavior are verified within the
manifest, focused-unit, and named-seed oracle boundaries recorded in the
[`2026-09-11 Phase 5.2 audit handoff`](../fidelity/depth/handoff/2026-09-11-modernization-phase5-2-audit.md).
The two extracted undead-knight procedures remain **audit-blocked, not
excluded**: C declares them without an assignment, Go has no VNum assignment,
and the moved Go helper uses 11470/11471 and `number(0,3)` where C uses
18401/18402 and `!number(0,2)`. No synthetic registration or oracle case was
added. The smallest next action is to establish authoritative assignment
intent, then add C-first proof and a separate fidelity correction if the
procedures are meant to be live. Phase 5.2 is not advanced to verified status
until that boundary is resolved.

## Phase 6 — C-isms mechanical wave (~−2,300; subset of reports/02)

| # | Target | Δ lines | Risk | Verifying cases |
|---|---|---|---|---|
| 6.1 | Giant switches → data tables where mechanical: `wiz_set` toggle majority (51 cases), `findExp` class/level ladders, equipment slot↔name maps, small lookup switches | −1,200–1,800 (mechanical subset only) | GREEN for the listed ones; spell-dispatch consolidation is **NOT here** | affected proven units |
| 6.2 | String cleanup in proven files: 94 nested Sprintf → flatten; 57 loop-concats → Builder | −200–400 | GREEN | per-file unit scenarios |
| 6.3 | Keyed tables + constants: key the unkeyed data tables (spellDB 509 L, THAC0, saving-throws 1,943 L), name the literal ladders, single-source LVL_IMMORT | ~0 net (churn) | GREEN | unit tests; **highest bug-class value per line in the census** |
| 6.4 | Global→struct injection: mail.go (8 globals), weather.go, merge_bridge banManager, spec_assign registries | −250–400 | YELLOW (behavior-adjacent) | weather/mail/ban scenarios + unit tests |
| 6.5 | Production `_ =` → handle-or-slog (AGENTS.md:63) | ~0 (adds lines) | GREEN | build + vet |

### Phase 6.1/6.2 audit tracking (2026-09-11)

The original Phase 6 table and estimates above are preserved. The dated audit
[`2026-09-11-modernization-phase6-1-6-2-audit.md`](../fidelity/depth/handoff/2026-09-11-modernization-phase6-1-6-2-audit.md)
finds both phases **partially complete** on current `origin/main`:

- 6.1's known lookup-table and wizard-flag slices landed, but six direct
  wizard flags, exceptional `nohassle`/`frozen`/`loadroom` boundaries, and
  equipment-map entry/reachability proof remain. The original `reports/02`
  candidate inventory is not present in the repository.
- 6.2's six known builder slices landed and passed their focused scenarios,
  but the `94`/`57` figures are aggregate heuristics rather than a closure
  checklist. `DoScan` and `roomObjectLines` remain concrete output-loop
  candidates pending inventory and proof review.
- The bounded wizard-set follow-up is recorded in
  [`2026-09-11-modernization-phase6-1-wizset-proof.md`](../fidelity/depth/handoff/2026-09-11-modernization-phase6-1-wizset-proof.md): six direct binary
  fields now have named live on/off transcript vehicles paired with C-derived
  bit and Go-storage assertions, and `nohassle`/`frozen` have separate
  authority/self-target cases. The clanless `deleted` fixture proves only its
  bit and acknowledgement; its C clan and crash/alias-file side effects remain
  an explicit debt. This does not close the equipment-map or `loadroom`
  boundaries and does not claim whole-Phase 6.1 completion.

Do not interpret the eight 2026-09-06 slice handoffs as whole-phase completion
records. The audit recommends closing the named 6.1 wizard-set proof slice
before Phase 6.3.

### Phase 6.3 spellDB slice tracking (2026-09-11)

The bounded spellDB slice is implemented on `glm/modernize-spelldb-keys` and
is awaiting review in its unmerged PR. The current representation was already
`map[int]*spellData`; the useful risk reduction was replacing raw numeric map
keys and unkeyed `spellData` literals with the existing named spell constants
and named fields. The map container, lookup behavior, `SpellNum` field, all 103
records, the 28 absent indices in the `0..130` range, and all zero/default
effects remain unchanged. An independent exhaustive pre-keying fixture covers
every index and field; the dated handoff records the C comparison and reader
coverage.

This slice does not advance or claim the rest of Phase 6.3. THAC0,
saving-throws, literal ladders, `LVL_IMMORT`, and spell-dispatch consolidation
remain outside this change and require separate audits and proofs.

### Phase 6.3 THAC0 keying slice tracking (2026-09-12)

The bounded THAC0 slice is implemented on `glm/modernize-thac0-keys` from
`origin/main` commit `2ea3474002e5f055395ac9d90251e22bb84eabf6`. Both live Go
tables remain `[12][41]int`; the 12 rows are now keyed by the existing class
identifiers in `pkg/combat/formulas.go` and `pkg/game/player.go`. No values,
dimensions, sentinel/default entries, initialization effects, or consumers
changed. The actual readers are `getTHAC0` → `CalculateHitChance` and
`newCharacter`'s level-1 THAC0 initialization. NPC/file THAC0 handling and
derived combat formulas remain separate.

The complete pre-edit baseline covers 984 Go cells (two copies of 12×41) at
the source commit above. An independent numeric fixture and the existing C
golden fixture cover the level-0 sentinel and levels 1–40; semantic checksums
are recorded in the dated evidence and handoff. The focused oracle matrix
covers 20 unique scenario/seed runs across `combat-swing`,
`combat-hit-weapon`, `combat-hit-sleeping`, and `hit-depth`, with seeds
1/2/3/5/8 and no missing, duplicate, failed, infrastructure, timeout, or
stale results. Seed-1 `--show-oracle` blocks were inspected for each vehicle.

This slice removes only row-position risk. It does not claim saving throws,
`LVL_IMMORT`, other literal ladders, spellDB, spell dispatch, combat-ticker
work, or all of Phase 6.3. See
[`2026-09-12-modernization-phase6-3-thac0.md`](../fidelity/depth/handoff/2026-09-12-modernization-phase6-3-thac0.md)
and the [THAC0 evidence](../fidelity/evidence/2026-09-12-thac0/README.md).

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
