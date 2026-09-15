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
| 6.3 | Keyed tables + constants: key the unkeyed data tables (spellDB 509 L, THAC0, saving-throws 1,943 L), reconcile the recovered literal-ladder candidates, single-source LVL_IMMORT | ~0 net (churn) | BOUNDED GREEN for the four named slices; recovered formula candidates remain separately bounded; no unconditional whole-phase closure | unit tests; **highest bug-class value per line in the census** |
| 6.4 | Global→struct injection: mail.go (8 globals), weather.go, merge_bridge banManager, spec_assign registries | −250–400 | YELLOW (weather lock re-entry proof required before any injection) | weather/mail/ban scenarios + unit tests |
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

### Phase 6.3 spellDB slice tracking (2026-09-11; corrected 2026-09-12)

The bounded spellDB slice merged as PR #1445 (merge
`2ea3474002e5f055395ac9d90251e22bb84eabf6`, implementation checkpoint
`e59809f5d`). The current representation was already `map[int]*spellData`;
the useful risk reduction was replacing raw numeric map keys and unkeyed
`spellData` literals with the existing named spell constants and named fields.
The map container, lookup behavior, `SpellNum` field, all 103 records, the 28
absent indices in the `0..130` range, and all zero/default effects remain
unchanged. An independent exhaustive pre-keying fixture covers every index and
field; the dated handoff and [evidence](../fidelity/depth/evidence/2026-09-11-spelldb/README.md)
record the C comparison and reader coverage.

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

Validation at source checkpoint `e38120cd38ecc8df2edc96665280626752e4013d`
passed the repository build, vet, test, formatter, and lint gates, plus the
focused and full fidelity checks. The current full corpus census was 940
scenarios: 930 passed, 9 expected, 1 permitted human-cleared unpinnable
`accuse-noarg-depth`, with `failed=0`, `infra=0`, `timed_out=0`, and `stale=0`.

### Phase 6.3 saving-throw table-keying slice tracking (2026-09-12)

The bounded saving-throw table-keying slice is implemented on
`glm/modernize-saving-throw-keys` from `origin/main` commit
`e54301fb4abfea6bf0c4811111d244c4ae141912e`. The table remains
`[12][5][41]int`; its class rows now use the existing class identifiers and
its five category rows use the existing `SavingThrowType` identifiers. No
values, dimensions, element types, initialization effects, sentinels,
defaults, lookup clamps/fallbacks, NPC handling, formulas, or draw order
changed. The actual readers are `GetSavingThrow` and `CheckSavingThrow`, with
focused live coverage through poison, sleep, and medusa save paths.

The complete pre-edit baseline covers 2,460 cells and matches the independent
C fixture at SHA-256
`caf57e8dc021b352253db65c00274194fe1cf1ba393101aa008b1403ad9a6b80`.
Focused oracle coverage is 20 unique scenario/seed runs: poison 12, sleep 5,
and medusa 3, with missing 0, duplicates 0, and no normalized divergence.
Repository code gates, depth, and expected-divergence checks pass.

The full `make oracle-regression` was started at implementation checkpoint
`f9e3178d5`; the merged runner fix was integrated at `d3871f69f` from PR
#1448 (`d580f1389`). With the fixed runner and frozen inputs, the complete
parallel census produced `scenarios=940 passed=930 expected=9 unpinnable=1
stale=0 failed=0 infra=0 timed_out=0`. The aggregate exit was 2 solely for
the existing human-cleared `accuse-noarg-depth` UNPINNABLE baseline. The
reconciliation found 940 unique scenario names, no missing or unexpected
scenarios, and one duplicate aggregate display of that baseline—not a
duplicate run. `medit-entry-depth` is now correctly classified as pinned
EXPECTED after infrastructure recovery. The durable run log and frozen-input
manifest are under
`/home/zach/saving-throw-evidence-2026-09-12/full-census-d3871f69f/`.

This slice removes only saving-table row-position risk. It does not claim
`LVL_IMMORT`, other literal ladders, spellDB, THAC0, spell dispatch,
saving-throw formulas, broader combat work, or all of Phase 6.3. See the
[`saving-throw evidence`](../fidelity/evidence/2026-09-12-saving-throws/README.md)
and [dated handoff](../fidelity/depth/handoff/2026-09-12-modernization-phase6-3-saving-throws.md).

### Phase 6.3 `LVL_IMMORT` single-source slice tracking (2026-09-12)

The bounded `LVL_IMMORT` slice is complete on
`glm/modernize-lvl-immort`, based on `origin/main` at
`dd2e4638a583d5c876f59af3aca732f37f90e7b9`. PR #1447 (saving-throw keys) was
verified merged before the branch was created and is present in that base.
The canonical compile-time value remains `31` in `pkg/combat`; the duplicate
Go definitions in `pkg/game` and `pkg/session`, the game-local `LVLImmort` and
`lvlImmort` aliases, the Lua export, and the confirmed `cmdLevels` threshold
now refer to that owner. Existing spell aliasing was already satisfied.

The C value is independently pinned at `src/structs.h:620`. The focused proof
covers the 30-row `levels` output, strict `fly_exit_up` above/equal/below
boundaries, mortal XP cap at level 30, compile-time arithmetic/array use, and
the Lua numeric export. The 13-run focused oracle matrix and the full 940-
scenario census are recorded in the
[`LVL_IMMORT handoff`](../fidelity/depth/handoff/2026-09-12-modernization-phase6-3-lvl-immort.md).

This entry advances only the named `LVL_IMMORT` candidate. It does not claim
the remaining Phase 6.3 named ladders, other constants, spell dispatch,
privilege-policy work, or whole-Phase 6.3 completion.

### Phase 6.3 bounded audit disposition (2026-09-12; provenance corrected 2026-09-13)

The four explicitly listed implementation slices are complete within their
named scope: spellDB keys/fields (#1445), THAC0 class rows in both runtime
tables (#1446), saving-throw class/category rows (#1447), and the canonical
`LVL_IMMORT` with compatibility aliases/consumers (#1449). Their production
implementation checkpoints are all ancestors of current `origin/main`
(`3b131e940b8cece457044d9895ff8ba45d942092`), and the audited production files
have no later changes. PR #1448 is the independent oracle retry-classification
repair; it is part of the tested baseline where noted, but is excluded from
the Phase 6.3 implementation delta.

The whole roadmap sentence is not closed as an unbounded claim. The original
audit reports were absent from the tracked tree, reachable history, and the
bounded unreachable-object search when #1450 was written, so its
“unrecoverable” conclusion was evidence-bounded and correct at that time.
The original reports have now been recovered at `/home/zach/dp-modernization`.
Their relevant excerpts, exact source hashes, and the distinction between
explicit candidates and inferences are preserved in the
[`2026-09-13 provenance evidence`](../fidelity/evidence/2026-09-13-phase6-3-provenance/original-excerpts.md).
The finite reconciliation and actual C/Go call-path verification are in the
[`2026-09-13 provenance handoff`](../fidelity/depth/handoff/2026-09-13-modernization-phase6-3-provenance.md).

The recovered source changes the review boundary. In particular, the
structural report’s historical `formulas.go` examples explicitly recover
`GetAttacksPerRound` NPC attack-count thresholds and player extra-attack
thresholds/chances. Source verification under R5 shows these are combat
formula/RNG candidates, not authority constants. The report’s line-111
“weapon-skill learn thresholds” label is not supported by the cited functions:
the actual practice path is `src/spec_procs.c:203-249` plus
`src/class.c:261-267`, and its Go counterpart is `pkg/game/practice.go` and
`pkg/game/class_spells.go`. No practice slice is inferred from that label.

The finite reconciliation review boundary is:

1. Review the remaining C authority-level family from `src/structs.h:610-624`:
   `LVL_GOD=34`, `LVL_LEGEND=35`, `LVL_HIGOD=36`, `LVL_GRGOD=38`,
   `LVL_IMPL=40`, and `LVL_FREEZE=LVL_GRGOD`, including the remaining Go
   package-local definitions and live consumers. This is a candidate inventory,
   not an authorization to change privilege behavior.
2. Retain two separately bounded future slices for `GetAttacksPerRound`:
   NPC bands/random bonus and player gates/chances, tied to C’s combat formula
   (`src/fight.c:1898-1947`) and its live violence call path. Do not extract
   or rename the thresholds in this documentation PR. Keep spell IDs, class
   IDs, dimensions, sentinels, defaults, formulas, authored values, and
   authority levels distinct unless a separate C-backed scope proves shared
   meaning. The current unit/golden and delegated combat coverage is not yet
   a complete C-derived threshold/draw-order proof.
3. Track the reachable board fidelity debt separately: Go
   `pkg/boards/boards.go:545` uses `59` where C
   `src/boards.c:403-405` uses `LVL_IMPL-1` (`39`). The smallest repair is a
   boundary test at the C threshold plus a dependency-neutral or injected
   authority value; it is behavior-changing fidelity work, not part of the
   behavior-preserving keying slice. The correction and its proof are recorded
   in [`2026-09-13-board-remove-authority.md`](../fidelity/depth/handoff/2026-09-13-board-remove-authority.md).

The recommendation is **accept a bounded reconciliation disposition**: record
the four slices as complete within scope; retain `getTHAC0`’s defensive
`1..40` guard as a literal with its documented rationale; retain the two
recovered attack-count formula candidates for separately authorized future
work; and defer the inferred practice label, unnamed literal ladders, and
broader `LVL_GOD`–`LVL_IMPL` consolidation for insufficient original scope or
proof. The board defect remains separate under #1451. Do not claim
unconditional whole-Phase 6.3 completion or start Phase 6.4 from this audit.
Full details, proof boundaries, deltas, the corrected prior audit, and the
fresh current-main validation are in the dated provenance handoff and the
[`2026-09-12 audit`](../fidelity/depth/handoff/2026-09-12-modernization-phase6-3-audit.md).

### Phase 6.4 global→struct audit tracking (2026-09-13; documentation-only)

The original Phase 6.4 row and its −250–400 estimate remain historical
planning metadata. A fresh audit from origin/main at
57414fb998f2d1257afd17a62eecb5894a79a482 verified the four named families
against current Go/C call paths under R5. The result is YELLOW, not
implemented and not complete:

- mail state/hooks require a lifecycle and storage proof first; the current
  production path does not call InitMailSystem, and the recovered C mail
  contract differs from current Go constants/paths;
- weather weatherWorld has a clear back-pointer seam, but the live heartbeat
  re-enters `weatherMu.RLock()` from event helpers while holding
  `weatherMu.Lock()` at hours 5 and 21; the canonical weather/tick state and
  conditional RNG path remain one protected boundary, and selected command
  rows do not prove the blocked weather/time lifecycle;
- merge_bridge.go banManager and World.Bans are two live authorities with
  different login/admin callers and file paths, so consolidation is deferred
  as a separate fidelity decision; and
- spec assignment maps and handler registries have different startup/runtime
  roles, direct production readers, and exported test mutation seams, so they
  are retained pending a bounded proof task.

No family is currently ready for an unqualified implementation slice. The
recommended next task is a bounded weather lock-reentry proof/triage task for
the scheduled hour-5/hour-21 paths, ahead of the mail lifecycle proof, because
the defect can block the live heartbeat. It must not become a weather/RNG,
scheduler, or injection refactor. After that defect has a reviewed
disposition, the mail proof remains the next candidate: boot scan, fixed-block/
restart behavior, postmaster output, composition cancellation, and concurrent
access. Only after those proofs should a human authorize any injection slice.
The full inventory, exact original excerpts and hashes, corrected focused
coverage, proof gaps, separate fidelity defects, and stop conditions are in
[2026-09-13-modernization-phase6-4-audit.md](../fidelity/depth/handoff/2026-09-13-modernization-phase6-4-audit.md)
and its [evidence package](../fidelity/evidence/2026-09-13-phase6-4/README.md).

This tracking entry does not authorize implementation, change save/storage
formats, reopen Phase 6.3 deferred combat/RNG or authority candidates, or
close Phase 6.4.

### 2026-09-14 mail initialization and lifecycle proof boundary

The bounded mail proof follows the reviewed weather repair in #1457 and does
not reopen the broad modernization audit. On fresh `origin/main` at
`d5328ce1b`, the R5 call-path sweep confirms that `InitMailSystem` has no
production caller. The live server reaches room 1204's assigned postmaster
and dispatches `mail`, but the disposable fresh character stops at the current
Go stamp-affordability gate before recipient lookup or composition. An
explicitly initialized helper vehicle proves one short message can complete,
be stored, be checked, be received, and be consumed once in the same process.
The real child-process restart/reopen then reads the 512-byte file but indexes
zero messages because `scanFile` reads into a marshaled temporary buffer and
does not unmarshal `nextBlock`.

Disposition: **blocked**, with production and helper evidence intentionally
separate. C's boot/no-mail/postmaster lifecycle remains the comparison source,
and C/Go fixtures remain native and separate: `etc/plrmail`/100-byte C blocks
versus `data/mail`/512-byte Go blocks. This proof does not wire initialization,
change constants or storage, claim C/Go format compatibility, or claim Phase
6.4 completion. The dated result table and durable outputs are in
[`2026-09-14-mail-lifecycle.md`](../fidelity/depth/handoff/2026-09-14-mail-lifecycle.md)
and its [evidence package](../fidelity/evidence/2026-09-14-mail-lifecycle/README.md).

The concrete next task is one bounded initialization/fidelity repair: add the
reviewed production `InitMailSystem` owner/identity hooks and repair Go header
decode in `scanFile`, without changing the current Go format or attempting
C-format convergence. Only after that repair proves this exact production
send→restart→receive vehicle should any human consider a further mail slice;
no injection readiness is established here.

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

### 2026-09-13 weather lock re-entry repair boundary

The specific weather lock re-entry defect is repaired within the tested
boundary in `eac85ac30` (`fix: prevent weather lock re-entry`). `WeatherAndTime`
retains one `weatherMu.Lock()` across the combined tick; its lock-held
`anotherHourLocked` and `weatherChangeLocked` bodies preserve the existing
callback/event/clock/draw order. Exported `AnotherHour` and `WeatherChange`
are now synchronized direct-entry wrappers, and all six event helpers use an
owner-captured `weatherWorld` pointer in the combined tick while retaining
synchronized pointer snapshots for direct helper calls.

Completion regressions cover hour 4→5 and 20→21 with nil/live targets,
non-event and `mode=false` controls, hour/sunlight progression, exact
existing Go event output with outdoor-before-event ordering, qualifying and
nonqualifying moon-condition boundaries, all six helper paths, the direct
entry points, focused race coverage, and independent weather draw plans.
The exact test/output boundary is recorded in
`docs/fidelity/depth/handoff/2026-09-13-weather-lock-reentry-repair.md` and
`docs/fidelity/evidence/2026-09-13-weather-lock/README.md`.

This does not claim full C weather fidelity: the known event output/state
differences and separate C/Go lunar/ghost-ship/gate RNG debts remain open.
Existing named weather/time command cases may not reach event hours, so their
oracle results do not replace the focused lifecycle regressions. Phase 6.4
weather-world injection remains unimplemented, and mail lifecycle proof stays
downstream of this reviewed locking repair.

### 2026-09-14 bounded mail initialization/restart repair

PR #1461 was verified merged into `origin/main` before this dependent repair.
The isolated branch `glm/fix-mail-initialization` started at merge commit
`1dd4794ac`; the implementation checkpoint is `411ad4b20`. The primary
checkout's unrelated edit was preserved. This repair is one production PR and
does not begin Phase 6.4 ownership injection.

The persistent identity authority is the existing PostgreSQL player record
(`players.id` plus canonical `players.name`), matching Go returning-player
login and the C `player_table`/`get_id_by_name`/`get_name_by_id` call paths.
`cmd/server/mail_identity.go` uses the existing `ListPlayerNames` and
`GetPlayer` APIs, with case-insensitive name matching delegated to the DB,
validated boot enumeration, reverse-map refresh on ID misses, and fail-closed
lookup errors. The boot owner is `cmd/server/main.go` immediately after
`session.NewManager` and before listener acceptance. If identity or storage
initialization fails, `cmd/server/mail_boot.go` disables mail and the boot
owner logs the failure while continuing the server, matching C's explicit
`no_mail` availability disposition; this is covered by the server boot test.
No-DB mode remains an explicit non-persistent/oracle configuration and does
not receive an online-only mail identity hook.

`pkg/game/mail.go` now decodes each complete existing Go 512-byte header with
`unmarshalMailHeader` before inspecting its marker/recipient, returns failure
for unusable storage, and preserves the current Go path, markers, block size,
encoding, and written bytes. The Go helper restart regression and the
production telnet vehicle now prove one short send, real SIGTERM/restart on
the same disposable storage, offline recipient check/receipt, preserved
sender ID/name and body, a second no-mail check/receive, and direct reading of
the delivered note with exactly one mail inventory item. The full evidence
and remaining gaps are recorded in
[`2026-09-14-mail-initialization-repair`](../fidelity/evidence/2026-09-14-mail-initialization-repair/README.md)
and the dated handoff
[`2026-09-14-mail-initialization-repair`](../fidelity/depth/handoff/2026-09-14-mail-initialization-repair.md).

C and Go mail remain separate native formats (`etc/plrmail`/100-byte C blocks
versus `data/mail`/512-byte Go blocks), with distinct markers and current
level/price behavior. The host C ABI size guard still limits C runtime
comparison. Multi-block/corruption/free-list/concurrency coverage, C mail
repair, output/object fidelity, and Phase 6.4 injection remain open; no C,
format, constant, normalization, pin, or exclusion was changed.

The fresh full oracle census completed with
`scenarios=941 passed=931 expected=9 unpinnable=1 stale=0 failed=0 infra=0
timed_out=0`; seven bounded infrastructure-shaped retries recovered to PASS.
The dated evidence explicitly reconciles the one final-result preservation
race (`yuball-depth`) with a tight-poll same-input recovery; execution
coverage was complete and no unexpected or duplicate scenario identity
remained.

### 2026-09-14 recipient-save failure proof boundary

The post-#1462 lifecycle audit found that receipt was proven without proving
recipient persistence. A fresh isolated proof now establishes the exact
failure: `readDelete` copies the full fixed 488-byte Go header text into
`Runtime.MailText`; a short message carries 473 NULs into the inventory state,
`encoding/json` emits `\u0000`, and PostgreSQL rejects `players.inventory`
JSONB with `22P05 unsupported Unicode escape sequence`. The recipient reload
after the failed statement remains unchanged, while a terminated-value control
saves and reloads the mail body.

This is **cause established; repair deferred**, not Phase 6.4 completion. The
smallest next repair is a C-string conversion at the mail fixed-block read
boundary for header and sibling data text, preserving the existing file bytes,
locks, identity, JSON/schema/save format, and once-only receipt. It must prove
receipt, save success, reload, byte preservation, and a second empty receive.
The characterization evidence and dated handoff are in
[`2026-09-14-mail-save-failure`](../fidelity/evidence/2026-09-14-mail-save-failure/README.md)
and
[`2026-09-14-mail-save-failure`](../fidelity/depth/handoff/2026-09-14-mail-save-failure.md).
No production repair, schema change, mail migration, or ownership injection
is included in this proof boundary.
### 2026-09-14 mail ownership-injection readiness decision

This documentation-only follow-up starts from fresh `origin/main` at
`f51eb840a`, containing merged #1462 (`4af6d9fad`). The durable census record
was verified at implementation checkpoint `d6b64449b4f66c48b2567f22f241f778bab9357c`:
941 result identities match 941 tested scenario inputs, with `931 PASS / 9
EXPECTED / 1 UNPINNABLE`, `failed=0`, `infra=0`, `timed_out=0`, and `stale=0`.
The seven infrastructure-shaped attempts were manually inspected; six
recovered to PASS and `redit-entry-depth` recovered to two identical pinned
expected results. The only unpinnable identity is the established
human-cleared `accuse-noarg-depth` baseline. Checkpoint-to-`origin/main`
equivalence is documentation-only outside the tested implementation, so the
repair's production, tests, scenarios, fixtures, runner, and oracle inputs
remain equivalent.

The finite #1455 mail inventory was rechecked under R5. #1462 proves the
production boot/identity/disabled/header/restart-delivery paths and preserves
the current Go persistence contract, but its passing receive vehicle records
a PostgreSQL `22P05` recipient-save error during shutdown. The supported
disposition is **PROOF-FIRST**, not READY: first run one future test-only
disposable-PostgreSQL proof that saves/reloads the recipient before receipt,
delivers one short message, attempts the post-receipt save, captures the exact
serialized payload/offending bytes and SQL error, checks reload persistence,
and compares a minimal no-mail-object control. The ownership-boundary
experiment remains deferred until that result is reviewed; it is not launched
here.

The concrete owner task, exact production callers and compatibility boundary,
unchanged identity/persistence contracts, lock ordering, remaining globals,
acceptance tests, known C/Go debts, and separate recipient save-error symptom
are recorded in the [dated ownership-readiness handoff](../fidelity/depth/handoff/2026-09-14-mail-ownership-readiness.md)
and [preserved evidence README](../fidelity/evidence/2026-09-14-mail-ownership-readiness/README.md).
Phase 6.4 remains open; no ownership implementation is part of this change.

### 2026-09-14 bounded mail text-conversion repair

The fixed-block NUL-padding/save defect established after #1464 is repaired
within its tested boundary on the unmerged branch
glm/fix-mail-text-conversion. Fresh origin/main was f3c1bfbf0 and the
implementation checkpoint was ec34175862827fec6d497354b1228de44f4b6ec6.
The four-file production/test delta adds a first-NUL C-string conversion for
both header and continuation text in readDelete; the Go writer, native file
format, written bytes, markers, offsets, identity resolution, locks,
serializer, schema, and save representation are unchanged. No src/ or
darkpawns-c-oracle/ file changed.

Native conversion regressions cover empty, padded short, embedded-terminator,
full no-terminator, short single-block, and bounded multi-block text. The
explicit disposable PostgreSQL proof used isolated native storage, a
persistent sender, an offline recipient, real composition, server restart
before receipt, live read, empty second check/receive, and the actual
shutdown cleanup save path. The persisted database row contains exactly one
mail object with exact sender/body and no NUL or \u0000 escape. This
distinguishes live receipt proof from database persistence proof.

Fresh full oracle regression on frozen inputs completed with
941 scenarios: 931 PASS, 9 established EXPECTED, 1 established
human-cleared UNPINNABLE (accuse-noarg-depth), and zero failed, infra,
timed-out, or stale scenarios. Nine bounded infrastructure-shaped attempts
recovered to PASS. The complete output and available attempt logs are
preserved under
/home/zach/dp-mail-text-conversion-evidence-2026-09-14/full-census-ec3417586.
The dated evidence is in
docs/fidelity/evidence/2026-09-14-mail-text-conversion/README.md and the
dated handoff is in
docs/fidelity/depth/handoff/2026-09-14-mail-text-conversion.md.

The diagnostic fresh relogin remains a separate blocked proof: PostgreSQL
persists the synthetic VNum -1 mail object, but RecordToPlayer skips it
because it has no world prototype. A separately authorized mail-object
rehydration/ownership-boundary slice is required for reload proof. Known
mail-object, C/Go format/output, broader lifecycle, and Phase 6.4 ownership
debts remain open. Phase 6.4 ownership injection is deferred.

### 2026-09-15 bounded persisted synthetic mail-object reload repair

The separate reload failure is repaired on the unmerged branch
`glm/fix-mail-reload`. PR #1465 was verified merged before implementation;
fresh `origin/main` was `29aa29d96e95fe224489bdf366bf2aa7efd166bd`, and the
implementation checkpoint is `cbe87c8b7e67b000d0c116b22e87d094b8946294`.
The repair changes exactly three files (`pkg/db/convert.go`,
`pkg/db/convert_test.go`, and `tests/e2e/mail_lifecycle_test.go`) for a
measured delta of 235 insertions and 0 deletions. No source/oracle, ordinary
prototype path, unrelated synthetic path, mail file, JSON/schema/save format,
or ownership code changed.

Under R5, returning-player login reaches `RecordToPlayer`, whose prototype
lookup skipped persisted synthetic mail at VNum -1. The narrow reconstruction
contract requires existing non-empty string `state.mail_text`; it then uses
`World.CreateMailObject`, preserves the text and canonical identity/type
fields, restores through `Inventory.RestoreItem`, and establishes
`LocInventoryPlayer`. Missing/malformed mail state is explicitly skipped;
VNum -1 alone does not identify mail, no prototype is invented, and ordinary
or unrelated synthetic objects remain unchanged.

The dedicated production proof now establishes exactly one receive → actual
recipient save → server restart → recipient login → read path. It used
disposable PostgreSQL and isolated mail storage, a persistent sender, an
offline recipient, real session/server/persistence paths, exact sender/body,
one readable inventory object, and empty subsequent check/receive. The
recipient save result and receive/reload server logs report no DB-save or
linkdead-save errors. A separate race-backed production run repeats the
lifecycle. Durable artifacts and the exact proof boundary are in the
[`2026-09-15 mail reload evidence`](../fidelity/evidence/2026-09-15-mail-reload/README.md)
and [dated handoff](../fidelity/depth/handoff/2026-09-15-mail-reload.md).

The full frozen-input oracle census completed with
`scenarios=941 passed=931 expected=9 unpinnable=1 stale=0 failed=0 infra=0
timed_out=0`; it exited 2 only for the established human-cleared
`accuse-noarg-depth` unpinnable baseline. Five infrastructure-shaped first
attempts recovered to PASS, and all 941 identities reconciled with no missing,
unexpected, duplicate, stale, failed, final-infra, or timed-out result.

Remaining debts are the C/Go format, output, object-field, free-list,
corruption, and concurrency gaps; broader synthetic-object reload and mail
lifecycle coverage; and Phase 6.4 mail ownership injection. Phase 6.4
ownership injection is explicitly deferred. The next recommended non-mail
slice is the separately evidenced Phase 6.3 board authority candidate
`boards-remove-authority-depth` under #1451; it is only a recommendation and
is not started here. Stop for human review; do not merge, deploy, or expand
this PR.
