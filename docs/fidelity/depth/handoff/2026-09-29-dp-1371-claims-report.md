# DP-1371 phase 1 — classified oracle claims

Phase 1 revalidated all 2,929 claimed (scenario, seed) pairs on the promoted reference oracle. Results: **2,913 PASS, 12 EXPECTED, four FAIL**; no unresolved INFRA/TIMEOUT, STALE or UNPINNABLE. The four failures classify as **two known-red over-claims and two false claims**. The >50 false-claim stop threshold was not reached. No game code, scenarios, manifest statuses, seed lists, pins or expected-divergence rows were changed.

The two false pairs are the same already-filed DP-1370 finding, Smackheads XP/death-cry ordering at seeds 3 and 5. They are false *manifest* claims despite being known from the earlier DP-1368 handoff: their manifest notes still assert green without acknowledging those seeds' red. This phase reports the issue and does not repair it.

## Inventory and boundaries (R5g)

The shared loader reads **5,188 rows**: 2,217 oracle-green-multiseed, 1,053 oracle-green, 1,144 unit-green, 648 delegated, 73 blocked, 53 excluded. The 3,270 oracle rows expand to 2,929 unique pairs, including 2,002 beyond seed 1. Every oracle case ID is represented. The shared loader reproduces every main row and its rendered depth report exactly, adding only source-line provenance. `make fidelity-depth` still reports 5,062/5,135 = 98.6%; this is the existing declared-status metric, not a new certification of completion. Unit-claim integrity remains phase 2.

| Seed | Claimed pairs | PASS | EXPECTED | FAIL |
|---|---:|---:|---:|---:|
| 1 | 927 | 923 | 4 | 0 |
| 2 | 524 | 520 | 2 | 2 |
| 3 | 498 | 495 | 2 | 1 |
| 5 | 489 | 486 | 2 | 1 |
| 8 | 491 | 489 | 2 | 0 |

Full run: `dp-1371-claims-baseline`, **1,659.757 seconds**, verdict **NOT_CLEAN**. Run HEAD: `ca1051a5d`; base/main: `f95479b53`. Reference sha256: `49a0799cd7bb107ea846bdd2768a85c100a75803fa2d9d8ded9337aa427bb76b`. Commands: `GOFLAGS=-buildvcs=false scripts/census.sh start --claims --name dp-1371-claims-baseline`, then `scripts/census.sh wait` repeatedly. All seed groups ran at up to 36 workers in ascending seed order. There were no infrastructure rows requiring recheck in this full run.

Before running, `git diff --stat origin/main...HEAD` showed only ten paths under `scripts/`. A direct diff of `pkg`, `cmd`, `lib`, and `docs/fidelity/depth` against the base was empty. This PR additionally includes this handoff and a stronger fixture assertion. The server, harness, world, scenario corpus and case manifests tested are main's.

## Non-PASS pairs

Case IDs and manifest rows below identify every claim that names the failing/expected vehicle. They are associations, not assertions that every named case's own block failed: a scenario can contain green blocks alongside an explicitly blocked branch. The first divergent block is the first retained fingerprint label; see the attempt logs for the full diff.

| Scenario | Seed | Kind | Case IDs | Manifest row(s) | First divergent block | Class |
|---|---:|---|---|---|---|---|
| accuse-depth | 1 | EXPECTED | accuse.self-target,accuse.target-not-found | docs/fidelity/depth/accuse.tsv:8,docs/fidelity/depth/accuse.tsv:9 | accuse the Ragnar trailing words are ignored [actor] | expected |
| gsay-act-depth | 1 | EXPECTED | gsay.act-actor-echo,gsay.act-complete-color,gsay.act-follower-order,gsay.act-invisible-speaker,gsay.act-sleeping-follower | docs/fidelity/depth/gsay.tsv:16,docs/fidelity/depth/gsay.tsv:17,docs/fidelity/depth/gsay.tsv:18,docs/fidelity/depth/gsay.tsv:19,docs/fidelity/depth/gsay.tsv:20 | gsay testing group act [actor] | expected |
| gsay-depth | 1 | EXPECTED | gsay.ansi-controls,gsay.group-audience,gsay.no-argument,gsay.norepeat,gsay.not-in-group,gtell.alias-audience,gtell.no-argument,gtell.not-in-group,gtell.spacing | docs/fidelity/depth/gsay.tsv:6,docs/fidelity/depth/gsay.tsv:7,docs/fidelity/depth/gsay.tsv:8,docs/fidelity/depth/gsay.tsv:9,docs/fidelity/depth/gsay.tsv:10,docs/fidelity/depth/gsay.tsv:11,docs/fidelity/depth/gsay.tsv:12,docs/fidelity/depth/gsay.tsv:14,docs/fidelity/depth/gsay.tsv:15 | gsay sleeping peer [peer] | expected |
| wizard-valid-reports-depth | 1 | EXPECTED | stat.mob-success,stat.object-fallback-success,stat.object-success,stat.player-default-position,stat.player-success,stat.room-success,vnum.mob-success,vnum.obj-success | docs/fidelity/depth/wizard.tsv:61,docs/fidelity/depth/wizard.tsv:62,docs/fidelity/depth/wizard.tsv:63,docs/fidelity/depth/wizard.tsv:64,docs/fidelity/depth/wizard.tsv:65,docs/fidelity/depth/wizard.tsv:67,docs/fidelity/depth/wizard.tsv:68,docs/fidelity/depth/wizard.tsv:126 | stat file Wizardvalidpeer [actor] | expected |
| accuse-depth | 2 | EXPECTED | accuse.self-target,accuse.target-not-found | docs/fidelity/depth/accuse.tsv:8,docs/fidelity/depth/accuse.tsv:9 | accuse the Ragnar trailing words are ignored [actor] | expected |
| dragon-depth | 2 | FAIL | dragon.failure,dragon.no-argument,dragon.one-argument,dragon.self,dragon.target-missing | docs/fidelity/depth/dragon.tsv:3,docs/fidelity/depth/dragon.tsv:4,docs/fidelity/depth/dragon.tsv:5,docs/fidelity/depth/dragon.tsv:6,docs/fidelity/depth/dragon.tsv:10 | ~dpclock pulse 20 [actor] | known-red over-claim |
| gsay-depth | 2 | EXPECTED | gsay.ansi-controls,gsay.group-audience,gsay.no-argument,gsay.norepeat,gsay.not-in-group,gtell.alias-audience,gtell.no-argument,gtell.not-in-group,gtell.spacing | docs/fidelity/depth/gsay.tsv:6,docs/fidelity/depth/gsay.tsv:7,docs/fidelity/depth/gsay.tsv:8,docs/fidelity/depth/gsay.tsv:9,docs/fidelity/depth/gsay.tsv:10,docs/fidelity/depth/gsay.tsv:11,docs/fidelity/depth/gsay.tsv:12,docs/fidelity/depth/gsay.tsv:14,docs/fidelity/depth/gsay.tsv:15 | gsay sleeping peer [peer] | expected |
| wizard-valid-reports-depth | 2 | FAIL | stat.mob-success,stat.object-fallback-success,stat.object-success,stat.player-default-position,stat.player-success,stat.room-success,vnum.mob-success,vnum.obj-success | docs/fidelity/depth/wizard.tsv:61,docs/fidelity/depth/wizard.tsv:62,docs/fidelity/depth/wizard.tsv:63,docs/fidelity/depth/wizard.tsv:64,docs/fidelity/depth/wizard.tsv:65,docs/fidelity/depth/wizard.tsv:67,docs/fidelity/depth/wizard.tsv:68,docs/fidelity/depth/wizard.tsv:126 | stat file Wizardvalidpeer [actor] | known-red over-claim |
| accuse-depth | 3 | EXPECTED | accuse.self-target,accuse.target-not-found | docs/fidelity/depth/accuse.tsv:8,docs/fidelity/depth/accuse.tsv:9 | accuse the Ragnar trailing words are ignored [actor] | expected |
| gsay-depth | 3 | EXPECTED | gsay.ansi-controls,gsay.group-audience,gsay.no-argument,gsay.norepeat,gsay.not-in-group,gtell.alias-audience,gtell.no-argument,gtell.not-in-group,gtell.spacing | docs/fidelity/depth/gsay.tsv:6,docs/fidelity/depth/gsay.tsv:7,docs/fidelity/depth/gsay.tsv:8,docs/fidelity/depth/gsay.tsv:9,docs/fidelity/depth/gsay.tsv:10,docs/fidelity/depth/gsay.tsv:11,docs/fidelity/depth/gsay.tsv:12,docs/fidelity/depth/gsay.tsv:14,docs/fidelity/depth/gsay.tsv:15 | gsay sleeping peer [peer] | expected |
| smackheads-outcome-depth | 3 | FAIL | smackheads.damage-death-pipeline,smackheads.failure,smackheads.failure-audience,smackheads.success,smackheads.success-audience | docs/fidelity/depth/smackheads.tsv:14,docs/fidelity/depth/smackheads.tsv:15,docs/fidelity/depth/smackheads.tsv:16,docs/fidelity/depth/smackheads.tsv:17,docs/fidelity/depth/smackheads.tsv:20 | smackheads generic recruit [actor] | false claim |
| accuse-depth | 5 | EXPECTED | accuse.self-target,accuse.target-not-found | docs/fidelity/depth/accuse.tsv:8,docs/fidelity/depth/accuse.tsv:9 | accuse the Ragnar trailing words are ignored [actor] | expected |
| gsay-depth | 5 | EXPECTED | gsay.ansi-controls,gsay.group-audience,gsay.no-argument,gsay.norepeat,gsay.not-in-group,gtell.alias-audience,gtell.no-argument,gtell.not-in-group,gtell.spacing | docs/fidelity/depth/gsay.tsv:6,docs/fidelity/depth/gsay.tsv:7,docs/fidelity/depth/gsay.tsv:8,docs/fidelity/depth/gsay.tsv:9,docs/fidelity/depth/gsay.tsv:10,docs/fidelity/depth/gsay.tsv:11,docs/fidelity/depth/gsay.tsv:12,docs/fidelity/depth/gsay.tsv:14,docs/fidelity/depth/gsay.tsv:15 | gsay sleeping peer [peer] | expected |
| smackheads-outcome-depth | 5 | FAIL | smackheads.damage-death-pipeline,smackheads.failure,smackheads.failure-audience,smackheads.success,smackheads.success-audience | docs/fidelity/depth/smackheads.tsv:14,docs/fidelity/depth/smackheads.tsv:15,docs/fidelity/depth/smackheads.tsv:16,docs/fidelity/depth/smackheads.tsv:17,docs/fidelity/depth/smackheads.tsv:20 | smackheads generic recruit [actor] | false claim |
| accuse-depth | 8 | EXPECTED | accuse.self-target,accuse.target-not-found | docs/fidelity/depth/accuse.tsv:8,docs/fidelity/depth/accuse.tsv:9 | accuse the Ragnar trailing words are ignored [actor] | expected |
| gsay-depth | 8 | EXPECTED | gsay.ansi-controls,gsay.group-audience,gsay.no-argument,gsay.norepeat,gsay.not-in-group,gtell.alias-audience,gtell.no-argument,gtell.not-in-group,gtell.spacing | docs/fidelity/depth/gsay.tsv:6,docs/fidelity/depth/gsay.tsv:7,docs/fidelity/depth/gsay.tsv:8,docs/fidelity/depth/gsay.tsv:9,docs/fidelity/depth/gsay.tsv:10,docs/fidelity/depth/gsay.tsv:11,docs/fidelity/depth/gsay.tsv:12,docs/fidelity/depth/gsay.tsv:14,docs/fidelity/depth/gsay.tsv:15 | gsay sleeping peer [peer] | expected |

## Classification and shared causes

**False claims — death-pipeline ordering (two pairs, one scenario, five associated case IDs).** `smackheads-outcome-depth`@3 and @5 differ at `smackheads generic recruit [actor]`: C prints “You receive 24 experience points.” before the death cry; Go prints it afterward. The claiming rows are `smackheads.tsv:14-17,20`, with the owning pipeline row at :20. The earlier `2026-09-29-dp-1368-review.md:93` already records these same seeds under DP-1370, but the case notes do not acknowledge the red, so these are classified as false claims rather than known-red over-claims under this brief's manifest-note definition. The live Go call at `pkg/command/skill_commands.go:1816` reaches `pkg/game/damage_stubs.go:254-256`, which calls DeathCry before HandleDeath. C's `damage()` emits XP at `src/fight.c:1646-1647` before `die_with_killer()` at :1691 and `raw_kill()`'s death cry at :573. These lines and the call path were read during this report (R1/R3b/R5e/R5g). No ordering fix was made.

**Known-red over-claim — following-round combat (one pair).** `dragon-depth`@2 first diverges at `~dpclock pulse 20 [actor]`: Go adds “A guard trainee scrambles to his feet!”, then different combat/death/corpse output. `dragon.tsv:10` explicitly says seed 2's following-round divergence is still open and that 1,3,5,8 are the green set, yet the proof field still includes 2. This is DP-1370, also recorded in the DP-1368 handoff above. Its five associated rows remain unchanged.

**Known-red over-claim — fresh-save snapshot/pin coverage (one pair).** `wizard-valid-reports-depth`@2 differs only at `stat file Wizardvalidpeer [actor]`. `wizard.tsv:66` already blocks this fresh-save mismatch and explicitly describes the creation-time C save versus the later Go runtime snapshot. Seed 2 exhibits those same fields (hometown 1/0, practices 2/4, hit points 1/10, original constitution 0/13). Its fingerprint differs from the seed-1 pin, so it remains FAIL rather than being called ledger-consistent EXPECTED. The acknowledgement is for the same blocked branch; it is not a newly discovered save defect. The eight associated green rows test other blocks in the shared vehicle. No save behavior or pin was changed.

**Expected (12 pairs).** `accuse-depth`@1,2,3,5,8 is backed by `expected_divergences.tsv:2-4` and `accuse.tsv`'s blocked target/argument branches. `gsay-depth`@1,2,3,5,8 is backed by ledger :10 and the blocked sleeping-recipient branch. `gsay-act-depth`@1 is backed by ledger :8-9 and blocked act framing. `wizard-valid-reports-depth`@1 is backed by ledger :19 and `wizard.tsv:66`. The worker verified the complete divergence fingerprint set against the existing pins, not just the first block. These expected vehicles contain other green claimed cases and are not new findings.

**Infra:** none after the census; no pairs were dropped from the report.

## Main confirmation

A separate, clean worktree at main/base `f95479b53` ran the same reference binary through the original `census.sh`:

- `dp-1371-seed2-reference`: dragon-depth and wizard-valid-reports-depth, both FAIL.
- `dp-1371-seed3-reference`: smackheads-outcome-depth, FAIL.
- `dp-1371-seed5-reference`: smackheads-outcome-depth, FAIL.

Every candidate failure has the same kind on main; there is no candidate-FAIL/main-PASS regression. Both false claims are therefore pre-existing, independently of the claims driver. Full candidate diffs are retained; the prior DP-1368 handoff records the same Smackheads first block and ordering difference on its paired baseline. Current main confirmations establish kind parity; this report does not claim current main full-diff fingerprints were retained by the old wrapper.

## Tooling proof and retained evidence (R5h)

Each new census test was run alone, broken and restored: wrong seed/list (`claims_groups`); swapped merge columns (`claims_merge`); wrong recheck seed (`claims_recheck`); FAIL retried (`claims_no_fail_retry`); failure ignored (`claims_verdict`); aborted runner ignored (`claims_aborted`); seed progress omitted (`claims_progress`). The fixture test covers default seed, multi-seed lists, status filtering, duplicate-pair case IDs, and composite oracle/unit proofs; changing the default seed to 2 fails it. Both CLI output and imported helper output are checked against the fixture. Mutation caches were isolated for the final fixture proof. Full stub suite: **19.41 seconds**, 53 assertions passing, below 30 seconds.

All required gates passed independently: make fmt, go build ./..., go vet ./..., go test ./..., golangci-lint cache clean, golangci-lint run ./..., git diff --check, make fidelity-depth. The known DP-1363 test did not require a rerun.

Evidence root: `/home/zach/Archives/darkpawns/oracle-runs/2026-09-29/`:

- `dp-1371-claims-baseline/`: wrapper manifest, frozen claims enumeration, initial four-column results, recheck file, per-seed summaries, overall summary, per-seed attempt logs/dumps, and **classified-claims.tsv** (all 16 non-PASS pairs, including full notes).
- `dp-1371-seed{2,3,5}-reference/`: original main wrapper manifests, summaries and result rows.
- `dp-1371-development/`: inventory/parity, scope stat, gates, break/restore logs and supplemental manifest.
- `dp-1371-claims-aborted-vcs/`: failed initial startup, excluded from claim findings. Go found an empty parent `/home/zach/.git` directory and failed VCS stamping before any probes ran. That directory was left intact. The subsequent runs used `GOFLAGS=-buildvcs=false`; actual Git provenance is captured separately, and the promoted C binary is unchanged. The driver now distinguishes an abort before results from content failure.

Report preparation also exposed timestamp-valid stale Python bytecode left by a same-second mutation. The census CLI had executed source directly, and its enumeration/results match all source pairs. Removing the ignored reporting cache restored import parity; no census data or manifest changed. The fixture now checks helper output as well as CLI output, with isolated cache directories for its mutation proof.

Phase 1 stops here for review. No game fixes, claim relabeling, seed deletion, ledger/pin changes or governing-document edits were made. Phase 2 starts only after this PR merges.
