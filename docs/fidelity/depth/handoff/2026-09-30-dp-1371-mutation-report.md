# DP-1371 Phase 2b: 100-symbol mutation audit

At main `0fadb33c6b9f8e1e31886ed0a027f45dd2bacc8e`, **94 sampled anchor
claims killed their registered mutation, five survived, and one timed out**.
All 100 selected tests passed before mutation and after exact source restoration.
No production behavior, tests, manifests, pins, oracle sources, or governing
documents are changed by this PR. Findings are left for separately approved fixes.

## Population and locked sampling design

A fresh `make fidelity-units` baseline executed all **1,144 row claims / 925
runnable symbols** across nine packages: all PASS, no skipped descendants.
Its wall time was 18.241 seconds. Phase 2 originally had 923 runnable symbols;
the two unresolved names have since been repaired. This audit uses the current
925, rather than silently dropping those two from the population.

Seed **20260930**, frame and selected anchors were written before inspecting
test bodies. The committed [selection.json](../../../../scripts/fidelity_mutations/selection.json)
contains only the seed, original lock timestamp, main HEAD, full-frame SHA-256,
and the 100 selected symbols with their anchor locators. The complete frame,
references and strata remain at
`/home/zach/Archives/darkpawns/proof-integrity/2026-09-30/dp-1371-2b/sampling-locked.json`,
with SHA-256
`8d4bcbeae99868a34ee1cf1ead47d67510a25fd1494c9ea5f1e9724095f77bf1`.
The sampler can regenerate that full file byte for byte using the recorded HEAD,
baseline, seed and `--created-utc` timestamp. This was verified during review;
the regenerated full frame and compact selection both match their originals.
No selected symbol or
anchor was replaced after observing tests or outcomes.

Each symbol's anchor was selected by SHA256(seed:symbol:case_id), retaining
its other references. Package × manifest-derived behavior category produced
25 nonempty strata. Each received one slot, then Hamilton apportionment
allocated the remaining 75 proportionally to residual population. Within a
stratum, SHA256(seed:symbol) determined selection. This covers all nine
packages and every nonempty package/category stratum, not every behavior arm.
Categories use manifest filenames, not test inspection; broad world-utilities
includes social registration tests and script manifests. See the exact classifier
and quotas in the archived frame rather than treating those labels as semantic coverage.

| Package | Selected |
|---|---:|
| boards | 1 |
| combat | 2 |
| command | 4 |
| game | 44 |
| parser | 1 |
| scripting | 2 |
| session | 42 |
| spells | 3 |
| telnet | 1 |

**Design correction before inspection:** the first allocator repeatedly favored
the largest remaining stratum. It was replaced with Hamilton apportionment
before any test-body review. `sampling-v1-before-test-inspection.json` and
`sampling-design-note.txt` retain that rejected frame and correction. This was
a sampling bug correction, not outcome-driven reselection.

## Execution and interpretation

One behavior-specific mutation per symbol was registered after inspecting the
anchor, selected test, and reachable production path. Detached worktrees at the
recorded main HEAD isolated packages; cases within each package ran serially,
with up to three independent package workers. Each selected test ran **alone**:

```text
go test -json -count=1 -timeout 120s -run '^ExactSelectedSymbol$' ./pkg/package
```

Every case retains PLAN.json, main HEAD, actual commands, execution exits and
elapsed times, clean/broken/restored JSON events, production diff, clean source
SHA-256, and empty tracked Git status after restoration. Source bytes are restored
in `finally`; later cases also retain the restored source hash explicitly.
The final source hashes and worktree status audit establish restoration for
all nine package worktrees. Mutations were never committed.

A kill requires a failed terminal event for the selected symbol and a matching
registered assertion in that symbol or its descendants. Go RUN/FAIL framing,
a failure in another test, panic, build error, skip, timeout, or failed clean/
restored run cannot earn a kill. The committed [results.tsv](../../../../scripts/fidelity_mutations/evidence/results.tsv)
contains **all 100 outcomes**, anchor rows, C sites, changed behavior, assertion
patterns, actual failing subtests and assertion output, exits, timings, and
archive pointers. The [patch bundle](../../../../scripts/fidelity_mutations/evidence/mutations.json)
contains every temporary production diff; these are evidence, not proposed fixes.

| Outcome | Count | Interpretation |
|---|---:|---|
| Killed by mapped assertion | 94 | R5h failure evidence for the named anchor and this mutation |
| Survived | 5 | Claimed behavior can change while the selected proof stays green |
| Timeout | 1 | Broken transport detected through a deadline; no assertion kill credit |
| Runtime crash / final compile failure | 0 | None in the final sample |
| No meaningful mutation possible | 0 | A reachable production mutation was found for every anchor |
| Clean/restored failure or skip | 0 | Every selected test was runnable on both sides |

There are **128 manifest references** to these 100 symbols. Only the **94
killed anchors** receive the failure evidence above. The other 28 references
are not certified by a kill elsewhere in their reused test. The remaining
825 symbols are unsampled. Neither 94% nor a passing unit run certifies all
1,144 row claims or every possible regression in a killed anchor.

## Five weak proofs, grouped by cause

All citations below refer to the recorded main revision. These are weaknesses
in proof vehicles, not claims that pristine production has these mutant bugs.
Each clean → broken → restored exit sequence is **0 → 0 → 0**.

| ID / class | Exact anchor / test | Mutation and why it survived |
|---|---|---|
| M010: permissive message assertion | `kick.tsv:15`, `kick.message-source`; `pkg/game/kick_skillmsg_test.go:181`, `TestDoKick_MissMessageFromSkillMessages` | File-backed `InitFightMessages` maps Kick attack 134 to Backstab 131. The test accepts any nonempty messages except two obsolete Kick phrases, so Backstab corpus bytes pass. The actual caller is `wireKickMessages` → `InitFightMessages`, not `InitSkillMessages`. C claim: `fight.c:1023-1092; lib/misc/messages:252-285`. |
| M021: missing success-arm proof | `spec-procs.tsv:83`, `mob.dragon-breath-combat-roll`; `pkg/game/spec_dragon_breath_test.go:88`, `TestSpecDragonBreath_CombatRollAndSharedReturn` | Changing combat number(0,3) to number(1,3) makes the direct breath-magic arm unreachable. The fixture asserts TRUE return and absence of an invented “breathes” act; both hold when every roll misses. It does not assert a successful call or the roll stream. C claim: `spec_procs.c:960-963; random.c`. |
| M025: gate masked by input | `spec-procs.tsv:114`, `room.pray-command-gate`; `pkg/game/spec_pray_for_items_test.go:40`, `TestSpecPrayForItems_FallsThroughOrdinarySocial` | Removing the pray-command gate still passes non-pray `look` with an empty argument because it falls through the no-item path. A meaningful argument could enter pray processing despite the wrong command. C claim: `spec_procs.c:2077; interpreter.c:1407-1456`. Owner-approved staff-name refusals remain intact. |
| M035: loose numeric bound | `charge.tsv:10`, `charge.mounted`; `pkg/game/skill_combat_test.go:758`, `TestDoCharge_MountedBonusDamage` | Mounted bonus 50 → 49 changes the claimed exact arithmetic. The test checks only total damage ≥50; base damage keeps the mutant above that lower bound. C claim: `new_cmds.c:917-925;939-952`. |
| M042: unchecked state contract | `spike.tsv:17`, `spike.success-player-state`; `pkg/game/skill_combat_test.go:148`, `TestDoSpike_KillsWerewolf` | Successful spike's RawKill flag true → false preserves success and the “drive” message. Those are asserted, but the kill contract and resulting player state are not. C claim: `new_cmds.c:1155-1175`. |

## Timeout and rejected attempts

**M100**, `entry.tsv:7`, `entry.telnet-boundary`,
`TestEntryTelnetSavedIdentityEntersWorld` (`pkg/telnet/entry_transport_test.go:85`):
changing `db.GetPlayer` from folded-name lookup to case-sensitive equality
routes `aiko` into new-character confirmation rather than the saved identity's
password prompt. The broken real TCP journey ends after five seconds with
`i/o timeout` while waiting for `Password: `, having received the fantasy-name
confirmation. The runner records **timeout**, not kill. Clean/restored pass.
This establishes a detecting deadline, not a precise assertion of the intended
identity-routing regression; a direct bounded protocol assertion is a future
proof-strengthening option.

**M030 rejected compile attempt:** the first implementation tried a type
assertion on `GetTarget()`, whose return is already concrete `*MobInstance`.
The compiler rejected it. `rejected-attempts/M030-type-error/` retains all
three runs and that diff. The same intended mob-target rejection was expressed
as a nil check, then killed by the selected target-resolution assertion. No
survivor was retried with a different mutation to chase a kill. The rejected
compile run is additional evidence, not a second sampled outcome.

One initial tooling break check bypassed a redundant skip branch: a passing
root was still classified as survived, so that change did not break its test.
The retained tooling-v1 attempt was replaced by disabling the skip predicate
itself, which breaks baseline/restored/descendant-skip checks. Seven final tooling
break/restore checks each produced **0 → 1 → 0**: Hamilton quotas, assertion
mapping, skips, crashes, timeouts, exact-byte restoration after interruption, and rejection
of tampered report verdicts.
Earlier snapshots and the final source are retained; no failed evidence was erased.

## Measured costs and extension proposal

The [machine summary](../../../../scripts/fidelity_mutations/evidence/summary.json)
records these observed times; they are wall durations, not CPU time or person-hours.

| Measurement | Observed |
|---|---:|
| Full nine-package unit baseline | 18.241 s |
| Sum of 100 clean/broken/restored focused run durations | 485.714 s (8.10 min) |
| Mean execution per symbol, three runs | 4.857 s |
| Median / nearest-rank 95th percentile per symbol | 4.605 s / 6.635 s |
| Sample lock to final restored event, including interleaved design/execution | 1,819.939 s (30.33 min) |

Compilation cache, existing fixtures, one agent's assisted source review, and
many simple registration gates materially affect these measurements. The last
wall window is **not isolated mutation-design labor** and does not include
final reporting, harness hardening or repository gates. No human design-time
measurement was made; the original proposal's person-hour assumptions have
not been validated. Do not substitute agent wall time for human labor cost.

At the observed focused-run mean, auditing 925 symbols would require roughly
**74.9 cumulative execution minutes**, or **66.8 additional minutes** for the
825 unsampled symbols, before design, review, retries, cold builds or extra
row mappings. Scaling the measured assisted campaign window would suggest
about 4.2 additional agent wall hours; this is a conditional planning estimate,
not a promised schedule or measured human cost. For robust planning, budget
separate reviewed design batches and log active design/review time explicitly.

**Proposal, not implementation:** first repair and re-prove the five weak
anchors in a separately approved batch, and strengthen M100's routing assertion.
Then extend the audit to the remaining 825 symbols in package/category batches
of 50–100, retaining one specific row-to-assertion mapping per mutation. Keep
reused rows in a separate mapping queue: one mutation cannot certify unrelated
arms. Start with damage/state, RNG, specials success arms and entry/persistence;
report each batch's survivors and non-kills before broadening. An exhaustive
symbol audit still leaves additional row behaviors and other possible mutations
unverified. No governing text change is needed or proposed by this PR.

## Evidence, scope and validation

Canonical local archive:
`/home/zach/Archives/darkpawns/proof-integrity/2026-09-30/dp-1371-2b/`.
It retains the baseline, sampling versions, test inspection snapshots, all 100
mutation plans and execution logs, rejected attempts, source restoration audit,
tooling break/restore proofs, and gate logs. The committed
[evidence checksums](../../../../scripts/fidelity_mutations/evidence/evidence-sha256.tsv)
identify retained files; the committed results and patches make findings
reviewable without access to that local archive. Credentials used by disposable
entry fixtures are synthetic. No production connection or player data is involved.

Reproduction: use `fidelity_mutation_sample.py` with a fresh passing
`fidelity-units` output at the recorded HEAD and seed; compare its selected
symbols/anchors; pass the original `--created-utc` from `selection.json` to
reproduce the full JSON hash. `--selection-out` emits the compact selection. Run
`fidelity_mutations/run.py --sample ... --plans ... --root ... --trees ... --out ...`
using the full archived or regenerated frame on that main revision with fresh
evidence paths; the runner refuses to overwrite
case evidence. `report.py` validates HEAD/frame/restoration and assertion mappings
before emitting the review artifacts. Committed plans abbreviate source context;
`compact-plan-equivalence.json` proves all 100 replacements produce identical
source to the original registered plans.

Scope is tooling, report and evidence pointers only. No oracle census was run:
this PR does not change game code, the oracle harness or scenario corpus, and
the approved measurement is the isolated unit-proof mutation sample. Baseline
and temporary mutants are explicitly anchored to main, rather than treating
mutant diffs as branch changes. Final gate outcomes are recorded below.

Validation: `make fmt`, `go build ./...`, `go vet ./...`, `go test ./...`,
`golangci-lint cache clean`, `golangci-lint run ./...` (zero issues),
`make fidelity-depth`, the complete Python script suite (14 tests, including
nine mutation-tool tests), and `git diff --check` passed. Seven isolated
tooling break/restore checks passed with 0 → 1 → 0. The scope audit found
only scripts, this report and its research evidence pointer; all 100 production
source hashes match the clean originals and all nine mutant worktrees are clean.

Review packaging: removed the full 1.56 MB frame from the PR in favor of the
compact selection. Full-frame regeneration reproduced the original SHA-256,
and the generated compact file matches the committed selection byte for byte.
