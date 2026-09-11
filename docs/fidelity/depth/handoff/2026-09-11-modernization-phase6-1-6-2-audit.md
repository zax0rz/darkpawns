# Modernization Phases 6.1 and 6.2 — completion audit

Date: 2026-09-11
Base: `origin/main` at `18b5a911c`
Audit branch: `glm/audit-phase6-1-6-2`
Scope: documentation and evidence only; no production, oracle, fixture, save-format, or Phase 6.3 changes

## Verdict

Phase 6.1 is **partially complete**. The two known mechanical slices landed,
but the phase cannot be called complete: the direct wizard-set table has six
entries without a named live case, its two exceptional binary fields and
`loadroom` need their own boundary proof, and the equipment parse/name tables
do not have a table-contract test. The original report inventory is also not
present in the repository, so the roadmap's “small lookup switches” estimate
cannot be converted into a closed candidate list.

Phase 6.2 is **partially complete**. All six known builder slices landed and
remain on their intended call paths. Their focused scenario families are
green, but the historical “94 nested Sprintf / 57 loop-concats” numbers are
aggregate estimates, not a valid deletion checklist. A reproducible current
search finds two concrete player-output loop candidates (`DoScan` and the
room-object count suffix), while other matches are state assembly, AI/database
text, fixed small appends, or behavior-sensitive output with no worthwhile
builder simplification. The missing source inventory prevents a claim that
all original 6.2 candidates were located.

Do not start Phase 6.3 yet. Finish the bounded 6.1 wizard-set proof closure
first, then re-baseline the missing 6.2 inventory before selecting any
remaining builder slice.

## Evidence limitation and reconstruction method

The roadmap says it was derived from `reports/01–05` and identifies Phase 6 as
a subset of `reports/02` (`docs/modernization/06-roadmap.md:1-3,89-95`). Those
report paths and the report text are not present in current `origin/main`.
`rg --files`, the reachable Git history, and a scan of loose/unreachable Git
objects did not recover a report containing the concrete 6.1/6.2 inventory.
The only surviving phase-level sources are the roadmap, the eight merged PR
descriptions, and the dated 2026-09-06 slice handoffs. Therefore this audit
reconstructs the candidates that can be proven from those records and current
source, and marks provenance-unknown residuals explicitly rather than
inventing a checklist from the two aggregate counts.

The six 6.2 handoffs and the two 6.1 handoffs are historical slice records,
not whole-phase completion records. This note supersedes any stale reading of
those handoffs as “Phase 6.1” or “Phase 6.2” complete while preserving them
unchanged.

## Trace of merged work and later drift

The known commits are present in a linear sequence on current main:

| slice | commit / PR | landed change | current status |
|---|---|---|---|
| 6.1 lookup tables | `5e417ed1a` / PR #1401 | FindExp, equipment slot/name, and observation equipment-position tables | present; no later edits to these tables |
| 6.1 wizard flags | `cd5913f53` / PR #1402 | twelve direct `wiz_set` binary fields in `setBinaryFieldTable` | present; no later edits to this table |
| 6.2 where | `e5ecb149f` / PR #1403 | no-argument `cmdWhere` builder | present; argument branch was already builder-based |
| 6.2 clan | `02a3ac645` / PR #1404 | `doClanInfo` builder | present |
| 6.2 search | `6f5cd21f7` / PR #1405 | `DoDetect` builder | present; later setter call is state-only |
| 6.2 prompt | `b4fc0138e` / PR #1406 | playing prompt builder | present; early-return branches remain explicit |
| 6.2 infobar | `e7ddca549` / PR #1407 | update/on/off infobar builders | present |
| 6.2 bans | `0c4e7c239` / PR #1408 | two `ListBans` headers written into the existing builder | present |

Relevant additional history was checked rather than treated as phase work:

- `ce2919ee6` / PR #1397 removed the private `findExp` duplicate before
  Phase 6.1. All current production callers use `game.FindExp`; this is
  already satisfied, not an unlanded Phase 6.1 candidate.
- PR #1409 (`6ccad8efa`) changed depth infrastructure and wizard/creation
  paths, not a Phase 6 target.
- PR #1414 (`afb934da5`) added `Equipment.Snapshot` and changed a locked
  inventory transfer in `equipment.go`; it did not alter slot/name tables or
  output mapping.
- PR #1422 (`ad51f41bc`) made the `DoDetect` secret-mark setter unconditional.
  `SetRoomFlagBit` is idempotent; no bytes, RNG calls, or exit iteration order
  changed. The current call path is still the same builder path.
- PR #1427 (`7c3e925d4`) changed vitals reads to snapshots/getters and added a
  closed-send guard/login-password prompt branch. It did not reorder or
  reformat the audited prompt/infobar builder output.

The later affected-file diff is preserved at
`audit-logs/2026-09-11-later-modifications.diff`.

## Phase 6.1 candidate matrix

Source citations below refer to the surviving roadmap and 2026-09-06
handoffs; where the original reports would have supplied a narrower citation,
that absence is called out.

| original target / source | current file, function, and call path | merged commit | disposition | exact coverage boundary | remaining proof / smallest next action |
|---|---|---|---|---|---|
| `FindExp` class modifiers and levels 0–12; roadmap 6.1; handoff `2026-09-06-modernization-phase6-1.md` | `pkg/game/limits_exp.go:75-124`; `FindExp` is called by `cmdLevels`, score/info paths, `newInfobarState`, `ExpNeededForLevel`, and `GainExp` | `5e417ed1a` / #1401 | landed; duplicate already satisfied | `pkg/game/find_exp_golden_test.go` covers all fixed levels, all 12 class modifiers, and formula levels 13–40. `pkg/session/display_cmds_test.go` covers level 13, default class 99, levels 0/-1, fixed levels, and level 20. `levels-depth`, `levels-arguments-depth`, `levels-npc-depth`, seed 1 all match. | No production gap found. The current map lookup has no iteration or RNG effect. Keep spell dispatch out of this phase. |
| equipment slot ↔ canonical name and accepted input maps; roadmap 6.1; same handoff | `pkg/game/equipment.go:51-110`; `EquipmentSlot.String` and `ParseEquipmentSlot` are used by serialization/conversion and equipment-facing APIs; parsing lowercases but does not trim, and aliases remain intact | `5e417ed1a` / #1401 | landed, with table-entry proof gap | `TestDoEquipmentUsesCOrderAndLabels`, `TestDoEquipmentRendersEveryCWearSlot`, color/covered-position tests, and `equipment-glance-depth` exercise player-facing labels. No current named test enumerates every `String` and `ParseEquipmentSlot` key/alias. | Add one bounded table-contract unit test covering canonical names, aliases, unknown input, and no-trim behavior before declaring the mapping candidate proven. |
| equipment slot ↔ C `where[]` position map; roadmap 6.1; same handoff | `pkg/game/look.go:1010-1070`; `DoLookTarget` → `appendCharacterLook` → `appendPlayerEquipment` → `equipmentWhere`. `appendMobEquipment` is a separate sorted-position path and was not changed. | `5e417ed1a` / #1401 | landed, with reachability gap | `TestDoLookTargetPlayerEquipmentColorizesObjectFlags` reaches the player-equipment path for a wielded item. `equipment-glance-depth` and the equipment unit tests cover the separate `do_equipment` C-position loop, including all C positions and extended hover output. The player-look loop only iterates `slot < SlotMax`, so the extended 100+ slots in the map are not reachable there. | Add a direct map contract or an all-slot player-look vehicle if this candidate is required to claim every map entry. Do not infer coverage from the separate `do_equipment` loop. |
| wizard-set direct flag majority; roadmap says “51 cases”; handoff `2026-09-06-modernization-phase6-1-wizset.md` | `pkg/session/cmdSetText` → `findSetField` → `applySetField` → `applySetBinaryField` → `setCPlayerFlag`/`setCPrfFlag`; table at `pkg/session/wiz_set.go:386-421` has 12 direct entries: `brief`, `invstart`, `nosummon`, `outlaw`, `roomflag`, `siteok`, `deleted`, `nowizlist`, `quest`, `color`, `nodelete`, `chosen` | `cd5913f53` / #1402 | landed as a code slice; phase proof incomplete | `TestSetFieldTableMatchesCOrderAndGates` proves all 59 ordered C field rows, levels, PC/NPC masks, and types. `set-depth`, `set-extended-depth`, and `set-gate-depth`, seeds 1/2/3/5/8, cover the generic binary path and live commands for `brief`, `nosummon`, `outlaw`, `color`, `nodelete`, and `chosen` (six of twelve). | The six direct table entries `invstart`, `roomflag`, `siteok`, `deleted`, `nowizlist`, and `quest` have no named command case in the current set vehicles. The smallest follow-up is a bounded command matrix for those six plus a direct table/bit assertion; do not claim all twelve from the generic path. |
| `nohassle` and `frozen` binary cases | Same `cmdSetText` call path, but `applySetField` deliberately leaves both out of the table at `wiz_set.go:436-442` and handles them in the explicit switch because C applies additional authority/self-target behavior | no new commit; intentionally retained by #1402 | deferred: behavioral exception | `set-extended-depth` seeds 1/2/3/5/8 executes both fields, but does not isolate every authority and self-target boundary for these fields. Generic per-field gate coverage is `set-gate-depth`, but its explicit commands are gold and str, not these exceptions. | Preserve the switch. Add explicit authority/self-target cases only if Phase 6.1 is to claim closure. |
| `loadroom` binary/state case | `applySetField` explicit path; it combines the room flag with numeric room validation/state update | no new commit; intentionally retained by #1402 | deferred: behavioral exception | `set-extended-depth` exercises valid `loadroom 8162` and `loadroom off`; `set.special-errors` covers the live error path. | Preserve the explicit branch. A table-only rewrite would risk state transition and validation bytes; no mechanical deletion is justified without a separate proof. |
| “small lookup switches” from roadmap 6.1 | Current sibling scan found `getStatLocation` (`pkg/game/equipment.go:456-482`), `getWearFlags` (`:373-454`), and several `wiz_set` parsers/clamps | no attributable phase commit; original report candidate provenance missing | `getStatLocation`: still actionable only after inventory recovery; `getWearFlags` and parsers: deferred | `getStatLocation` has no named depth case found; `getWearFlags` is a bit-priority decoder with C-specific multiple-slot expansion; `parseSetSex`, `parseSetClass`, `parseSetRace`, and `clampSetValue` preserve validation/error behavior. | Reconcile these against the recovered `reports/02` inventory first. Do not turn the R5c sibling scan into an unapproved production refactor. `getWearFlags` is a behavioral exception, not a simple lookup. |
| spell-dispatch consolidation | `pkg/session/cast_cmds.go` and spell dispatch tables | explicitly excluded by roadmap 6.1 and the 2026-09-06 handoff | deferred: excluded scope | Spell matrix remains independently tracked in `surface-inventory.tsv`; no Phase 6.1 claim is made here. | Keep outside 6.1. |

### 6.1 reconciliation

The “51 cases” roadmap figure is an aggregate estimate for the `wiz_set`
toggle majority, not 51 current deletions. The landed table contains 12 direct
flag rows; `nohassle`, `frozen`, and `loadroom` are intentionally outside it.
The direct flag setter has no side effects beyond the existing flag-sync calls,
including no invented behavior for `deleted`, satisfying R4. The remaining
proof gap is coverage of six direct entries, not evidence of a production
defect.

The FindExp ladder is complete as a code/data slice and has stronger unit
coverage than its phase estimate. The old duplicate was removed by PR #1397,
which is why no second Phase 6.1 FindExp candidate remains.

The equipment maps preserve canonical names, aliases, case folding, fixed C
positions, shared finger/neck/wrist labels, and the unknown fallback. The
coverage boundary is real: all-C-position `do_equipment` proof does not prove
the player-look map, and the extended map entries are not traversed by that
player-look loop.
## Phase 6.2 candidate matrix

| original target / source | current file, function, and actual call path | merged commit / PR | disposition | exact coverage and fidelity checks | remaining proof / deferral |
|---|---|---|---|---|---|
| `where` no-argument output; handoff `...phase6-2-where.md`, C `src/act.informative.c:2253-2280` | `pkg/session/cmd_info.go:978-1042`; registered `where` → `cmdWhere`; sessions are copied under lock, stably sorted by connection time, then written to a builder; argument branch at `:996-1019` was already builder-based | `e5ecb149f` / #1403 | landed | `info-basic`, `info-where-immort`, `where-immort-zone-arg`, seed 1; `info.tsv` covers mortal gate, immortal listing, and argument path. Header, newline style, order, fallback, and 30-row early break are unchanged. | No changed branch is unproven within the named where cases. The file has other unrelated formatted output; do not count those as this slice. |
| clan list/detail output; handoff `...phase6-2-clan.md`, C `src/clan.c:723-785` | `pkg/game/clan_info.go:82-145`; `ExecClanCommand` → `doClanInfo`; list loop preserves clan index order, sparse fallback uses `Builder.Reset`, detail preserves plan/war/fee ordering | `02a3ac645` / #1404 | landed | Six seed-1 scenario runs: `clan-depth`, `clan-member-depth`, `clan-applicant-depth`, `clan-plan-depth`, `clan-plan-mortal-depth`, `clan-rename-depth`; `clan.tsv` covers list/detail, private visibility, applicant/member/plan, sparse fallback, and war/peace. | No byte/order/reset/early-return defect found. The six scenarios prove named branches, not every clan database topology. |
| search/detect output; handoff `...phase6-2-search.md`, C `src/new_cmds2.c:500-541` | `pkg/game/skills2.go:355-401`; command `search`/`detect` → `command.CmdDetect` → `game.DoDetect`; initial room line precedes RNG, failure suffix and wait are preserved, six exits use `dirs` order, setter marks secret exit | `6f5cd21f7` / #1405 | landed | `search-case-depth`, `search-depth`, `search-failure-depth`, `search-gates-depth`, `search-secret-depth`, each seeds 1/2/3/5/8; `search.tsv` covers no skill/Elf/blind/roll/found directions/up/down/secret marking. Seed 3 search-depth had one C-oracle bind collision; the bounded retry passed. | No changed bytes, RNG ordering, or exit-order defect found. The later unconditional idempotent setter is state-only. |
| playing prompt fields; handoff `...phase6-2-prompt.md`, C `src/comm.c:1055-1105` | `pkg/session/session_send.go:279-326`; transport prompt path → `promptText`; writing, inactive, and AFK early returns remain before the playing builder; later getter/snapshot edits preserve field order | `b4fc0138e` / #1406 | landed | Unit tests `TestInvisPromptIncludesCLevel`, `TestPromptTextPreservesCFieldOrdering`, `TestInfobarUpdateUsesCLayoutAndBitOrder`, `TestCmdPromptMatchesCDoDisplay`; `invis-depth` seeds 1/2/3/5/8 including `invis.level-prompt`. | The builder is only the playing branch; no claim is made that early-return/editor prompt paths were converted. Their explicitness is faithful behavior, not residual cleanup. |
| infobar update/on/off output; handoff `...phase6-2-infobar.md`, C `src/act.display.c:80-706` | `pkg/session/display_cmds.go`; `cmdInfoBarUpdate`, `cmdInfoBarOn`, `cmdInfoBarOff` call the existing field-format helpers in the same bit/VT100 sequence | `e7ddca549` / #1407 | landed | `infobar-depth` and `infobar-mortal-depth`, seeds 1/2/3/5/8; units `TestInfobarUpdateUsesCLayoutAndBitOrder` and `TestInfobarUnknownStateResetsToOff`. Covers frame, save/restore, update bit order, mortal display, and unknown-state reset. | The helper functions still use one-off `fmt.Sprintf`; they are not nested output-buffer candidates. Their fixed VT100 arithmetic and reset sequence should remain explicit unless a separate byte proof is supplied. |
| ban list headers; handoff `...phase6-2-ban.md`, C `src/ban.c:142-171` | `pkg/game/bans.go:208-230`; `merge_bridge.ListBans` delegates to `BanManager.ListBans`; empty fallback, two headers, and body loop remain ordered | `0c4e7c239` / #1408 | landed | `ban-depth` seeds 1/2/3/5/8; `ban.tsv` covers empty/list/add/newest-first/date/fixed-width output. `TestIsBannedLiteralAsterisk` is a separate ban predicate unit, not header proof. | No changed formatting/order defect found. Do not treat the unrelated literal-asterisk test as evidence for the builder. |

### 6.2 inventory reconciliation

The historical counts cannot be reproduced as candidate counts from current
source. The reproducible search recorded in
`audit-logs/2026-09-11-phase62-search-reproduction.txt` found:

```text
literal WriteString(fmt.Sprintf) matches: parent 0, current 0
broad plus/Sprintf matches:               parent 55, current 55
production fmt.Sprintf lines:              parent 896, current 895
production += lines:                       parent 316, current 315
```

The broad expression intentionally overmatches and therefore is not a
deletion checklist. Manual classification of the current matches is:

- **Landed:** the six listed PR slices. `cmdWhere`'s argument branch was
  already builder-based before #1403, so #1403 is only the no-argument path.
- **Still actionable, pending proof:** `pkg/game/skill_advanced.go:385-420`
  (`DoScan`) appends two formatted rows inside an exit/player loop and has no
  `scan` manifest/scenario in the current depth inventory; and
  `pkg/game/look.go:313-348` (`roomObjectLines`) appends a formatted count
  suffix in the reverse-order room-object rendering path. Their smallest next
  actions are a source-inventory decision followed by branch-specific depth
  proof; no code change is made by this audit.
- **Already converted or not a new target:** existing builders in the six
  slices and earlier refactors; one-off `fmt.Sprintf` calls written directly
  to a writer; and the six infobar field helpers, whose output is a fixed
  VT100 cell with arithmetic and reset semantics rather than a repeated
  concatenation buffer.
- **False positive or excluded scope:** `pkg/agentcli`, `pkg/db/narrative_memory.go`,
  and other AI/database prompt assembly are not MUD player-output paths;
  editor/persisted-text buffers, command reconstruction, numeric state
  accumulation, and fixed four-line appends are not equivalent builder
  targets. Spell identify output remains a separate spell/output surface and
  is not promoted into Phase 6.1 or 6.2 without its own inventory and proof.

This classification explains why the raw 94 and 57 numbers do not add up to
“94 remaining” or “57 remaining,” and why no phase-completion claim follows
from the aggregate estimates.

## C/Go fidelity review

The merged diffs were compared with their cited C call paths and reviewed for
R1 bytes, R3 ordering/draw behavior, and R5c/R5e call-path scope. The audit
found no confirmed production defect. In particular:

- no builder changed a literal, CRLF/newline convention, fallback, reset,
  early return, row cap, iteration order, or VT100 sequence;
- `FindExp` table lookup does not iterate and does not consume RNG; its formula
  and float/integer operations are unchanged;
- `wiz_set` table application keeps the existing player/preference flag
  setters and does not add `deleted` side effects;
- `DoDetect` still performs its RNG call after the initial line and scans
  `dirs` in order; the later room-flag setter change is idempotent and does
  not emit output;
- later vitals snapshots/getters affect synchronization only; prompt and
  infobar bytes remain in the same sequence.

The lack of a `scan` manifest and the incomplete direct-flag/map coverage are
proof gaps, not evidence that an untested branch is faithful. Per R5f, the
green boundary stops at the named cases.

## Measured landed deltas

These are non-overlapping adjacent Git ranges: `ad3a9e7db..cd5913f53` for
6.1 and `cd5913f53..0c4e7c239` for 6.2. They are measured diffs, not roadmap
estimates.

| phase / scope | additions | deletions | net |
|---|---:|---:|---:|
| 6.1 production (`equipment.go`, `limits_exp.go`, `look.go`, `wiz_set.go`) | 155 | 240 | **−85** |
| 6.1 tests | 0 | 0 | **0** |
| 6.1 handoff docs (2) | 135 | 0 | **+135** |
| 6.2 production (`bans.go`, `clan_info.go`, `skills2.go`, `cmd_info.go`, `display_cmds.go`, `session_send.go`) | 71 | 67 | **+4** |
| 6.2 tests (`prompt_render_depth_test.go`) | 53 | 0 | **+53** |
| 6.2 handoff docs (6) | 248 | 0 | **+248** |

The later PRs listed above are not included in these phase deltas, preventing
double-counting later race/transport changes as Phase 6 work.

## Focused validation

Environment: `PATH=/usr/local/go/bin:$PATH` and
`DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle`.

Focused unit command:

```text
go test ./pkg/game ./pkg/session                 PASS
```

Focused oracle execution used the manifest-relevant seeds 1, 2, 3, 5, and 8
for the multiseed families. Seed-1 runs included `--show-oracle` blocks; the
search-depth seed 3/5/8 retry logs also retain the oracle blocks.

The tally was computed from actual log lines matching
`result: no normalized divergence`, not from handoff totals:

| family | scenario-seed runs | pass | content failures |
|---|---:|---:|---:|
| 6.1 levels | 3 | 3 | 0 |
| 6.1 equipment | 1 | 1 | 0 |
| 6.1 wizard set | 15 | 15 | 0 |
| 6.2 where | 3 | 3 | 0 |
| 6.2 clan | 6 | 6 | 0 |
| 6.2 search | 25 | 25 | 0 |
| 6.2 prompt/invisibility | 5 | 5 | 0 |
| 6.2 infobar | 10 | 10 | 0 |
| 6.2 bans | 5 | 5 | 0 |
| **total** | **73** | **73** | **0** |

There are 25 unique scenario files and 73 valid scenario/seed executions.
One initial `search-depth` seed-3 run failed C readiness with
`bind: Address already in use`; the bounded retry passed. Four initial
attempt logs were interrupted while the batch was stopped and are excluded,
not counted as passes or failures. The durable computed list and per-scenario
tally are at:

- `audit-logs/2026-09-11-focused-oracle/computed-pass-log-list.txt`
- `audit-logs/2026-09-11-focused-oracle-tally.tsv`
- `audit-logs/2026-09-11-focused-oracle/search-depth-seed3.log`
- `audit-logs/2026-09-11-focused-oracle/search-depth-seed3-retry1.log`

## Repository gates

All required local gates passed on the audited `origin/main` source state:

```text
make fmt                                      PASS; no Go changes
gofumpt -l .                                  PASS; no output
go build ./...                                PASS
go vet ./...                                  PASS
go test ./...                                 PASS, including tests/e2e
golangci-lint run ./...                       PASS, 0 issues
make fidelity-depth                           PASS: 4798 total, 4679 proven/delegated, 68 blocked, 51 excluded
make expected-divergences-check               PASS: 26 expected rows; pins OK
git diff --check                               PASS
```

Logs are retained under `/home/zach/darkpawns-phase6-audit/audit-logs/`,
including `2026-09-11-make-fidelity-depth.log`,
`2026-09-11-make-expected-divergences-check.log`,
`2026-09-11-go-build.log`, `2026-09-11-go-vet.log`,
`2026-09-11-go-test-all.log`, `2026-09-11-golangci-lint.log`, and the
focused logs listed above.

No full `make oracle-regression` census was run. The material phase findings
(missing original inventory, six unproven direct flag entries, exceptional
wizard-set boundaries, equipment-map reachability, and residual output-loop
candidates) prevent a conclusion that either whole phase is complete. A full
census would not resolve those proof gaps and is intentionally deferred.

## Ranked follow-up and recommendation

1. **Phase 6.1 wizard-set direct-flag proof closure:** cover
   `invstart`, `roomflag`, `siteok`, `deleted`, `nowizlist`, and `quest` with
   exact command/state cases, and separately pin the `nohassle`/`frozen`
   authority/self-target exceptions. Keep `loadroom` explicit.
2. **Phase 6.1 equipment map proof closure:** add a table-level check for
   canonical names/aliases and decide whether the extended map entries need a
   player-look vehicle, since `do_equipment` coverage is not the same call
   path.
3. **Phase 6.2 inventory re-baseline:** recover `reports/02` or create a
   reviewed replacement manifest from the original data before selecting
   cleanup work. Review `DoScan` and `roomObjectLines` as separate, small
   output-loop slices; each needs its own exact branch proof.
4. **Phase 6.3:** defer until the above evidence is closed or explicitly
   accepted as a bounded phase exit.

Recommendation: finish item 1, the named **6.1 wizard-set direct-flag proof
closure**, before proceeding to Phase 6.3. This audit makes no implementation
changes and stops for review.
