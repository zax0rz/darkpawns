# 2026-10-05 — Combat body identity and layered verification

## Scope and sources

- **Tracks:** legacy-port fidelity; agent-assisted engineering; research operations.
- **Question:** what evidence distinguishes two equal-description runtime bodies,
  and what did fixture construction, review, and live play reveal separately?
- **Window:** the approved combat-identity prerequisite, its four implementation
  PRs, post-review repair, and local playtest/merge on 2026-10-05.
- **Revisions:** final tested tip `ded5392c3f9ca07738629e3a77ea125706eb66da`;
  merged main `aae4312544cd0b9b1de7c0409b53c91ba6491fa9`. Both have tree
  `71830b3d0abe826aab2f4aabe0d1c60c9465fa16`.
- **Provenance:** implementation/capture by the Codex assistant; the owner supplied
  review text attributed to DeepSeek. Exact model/provider versions, token usage,
  wall-clock authoring duration and reviewer independence are not established.
  This is an interactive field note, not a scheduled experiment or model comparison.
- **Public primary artifacts:** [design #1796](https://github.com/zax0rz/darkpawns/pull/1796),
  [membership #1798](https://github.com/zax0rz/darkpawns/pull/1798),
  [callbacks #1800](https://github.com/zax0rz/darkpawns/pull/1800),
  [cleanup #1801](https://github.com/zax0rz/darkpawns/pull/1801), and
  [fixture/review repairs #1803](https://github.com/zax0rz/darkpawns/pull/1803).
- **Retained local evidence:** `~/Archives/darkpawns/oracle-runs/2026-10-05/`
  contains `dp-1371-combat-body-identity-proofs/` and
  `dp-1371-combat-body-review-integrated/`. These artifacts are not bundled in
  this public note; source tests, controls and the scenario are in the repository.

No raw human conversation, player database, credentials, or unredacted review
transcript is published here. The playtest discussion is summarized by behavior.

## Runtime identity and a deliberately asymmetric fixture

The engine migration keys combat membership/pairs/order and its callbacks by
concrete bodies rather than display names or unqualified numeric IDs. The scope
included retained targets, output, protections, following inputs, stop/retarget,
and death/extraction cleanup. The approved design explicitly retained other
lifetime/navigation frontiers; this was not a second character registry.

The new scenario creates two actual shipped trainee mobs with equal descriptions
in one room and two ordinary PC peers. Only the newest trainee receives a
zone-equipped cutlass. Both fights advance, an observer reports their exact
opponents, and the observer kills only the armed duplicate. Subsequent pulses
expose the survivor's continued fight and the former opponent's stopped state.
The weapon asymmetry makes wrong-body selection visible in slash versus bare-hit
output, while ordered pulses expose draw-stream consequences (R1/R3/R5h).

**Important chronology:** the fixture was added after the membership, callback
and cleanup steps. Its initial red did not independently discover the original
engine identity problem. It exposed two remaining entry/death boundaries: Go's
canonical room lookup sorted prototype/ID rather than C's latest room arrival,
and the immortal kill path supplied attack type 0 instead of `TYPE_SLASH`.
These were repaired before the first complete-tip census.

C authority: `src/handler.c:532-556,1276-1300`,
`src/act.offensive.c:101-161`, `src/fight.c:207-254,1898-2032`.
Repository proof: `cmd/dp-oracle-diff/scenarios/combat-duplicate-bodies-depth.txt`,
`pkg/game/combat_body_ordinal_test.go`,
`pkg/session/combat_body_kill_corpse_test.go`. The initial paired diagnostic and
oracle ordinal green/revert/restored-green triple are retained in
`dp-1371-combat-body-identity-proofs/step4-fixture-diagnostic/` and
`step4-oracle-triple/`.

## Review widened the boundary audit

The supplied review identified residual name comparisons in breath enrollment,
post-death mobile loot, literal skill audience exclusions, and spell grouping.
The repair passed actual bodies through all three literal-output phases and used
the existing World `FollowingBody` for all five spell-group callers. A real World
group-heal test distinguishes PCs held under separate same-description mobile
leaders, then includes them once the retained leader is actually shared.

The expanded death audit also reproduced an NPC killer being resolved to an
unrelated same-name PC for kills, PK credit and outlaw flags. The repaired live
path retains the supplied killer body; names remain output/event labels. This
extra finding was not in the initial reviewer list. NPC counters were not expanded.

C authority: `src/fight.c:1367-1445,1671-1705`,
`src/comm.c:2485-2547`, `src/utils.c:655-674`.
Primary repair commit: `ded5392c3`. Tests:
`TestCombatBodyBreathDuplicateEnrollment`, `TestCombatBodyLootNameCollision`,
`TestCombatBodySkillAudienceCollision`, `TestCombatBodyGroupDuplicateLeaders`,
`TestCombatBodyLiveGroupSpellLeaders`, and
`TestCombatBodyDeathCreditNameCollision`.

Nine stack-4 mutation controls exist: three original controls and six review
controls. Each restores a specific old decision, must compile, must fail an
assertion, and must pass after restoration. The script excludes build failures
from mutation evidence. The oracle ordinal control separately requires an actual
normalized byte divergence. Older proof HEAD files label the precommit lab base;
the final census records the committed source, not that older lab HEAD.

Reproduce in a disposable worktree, without another editor/build running:

```sh
DP_ORACLE_BIN="$HOME/darkpawns-c-oracle/bin/circle" python3 docs/fidelity/depth/handoff/2026-10-05-combat-body-step4-controls.py --output /tmp/combat-step4-triples --oracle
```

Retained receipts: `review-integrated-triples/`, `review-death-triples/`,
`step4-oracle-triple/`; classifications and lock order are in
`docs/fidelity/depth/handoff/2026-10-05-combat-body-step4.md`.
Legacy standalone adapters, hunting/memory, mobile-rider support and global
fallback ordering remain explicitly bounded frontiers, not new parity claims.

## Census, CI and merge provenance

The final integrated census at `ded5392c3` executed **3,289 distinct scenario/seed
pairs**, eliminating **998 duplicate executions** from the two gate sets. Full
and claims verdicts were both **CLEAN_AFTER_RECHECK**, elapsed **1,912.520 seconds**
(31m 52.520s). The retained summary identifies `position-gates` as rechecked;
all five new duplicate-body seeds (1,2,3,5,8) passed. This is a dated result for
that corpus/workstation, not a throughput comparison or whole-game parity claim.
Sources: `dp-1371-combat-body-review-integrated/{go-head.txt,summary.txt,combined-results.tsv,combined-recheck-results.tsv}`.

The CI workflow filters pull requests by base branch (`main`/`master`). Stacks
2–4 originally had successful SHA-specific workflow_dispatch runs but no PR
checks. Retargeting to main alone did not trigger the default PR event set;
closing/reopening each remaining PR triggered real attached PR checks. Those
checks passed before its merge. No workflow/tooling change was made.

Attached CI runs: stack 1 [37382609197](https://github.com/zax0rz/darkpawns/actions/runs/37382609197),
stack 2 [37398395007](https://github.com/zax0rz/darkpawns/actions/runs/37398395007),
stack 3 [37398441771](https://github.com/zax0rz/darkpawns/actions/runs/37398441771),
stack 4 [37398446562](https://github.com/zax0rz/darkpawns/actions/runs/37398446562).
The four PRs merged in order with merge commits. The recorded main tree exactly
matches the census/playtest tip; `MERGE-RESULT.json` and
`merge-step{2,3,4}-attached-ci.txt` retain the final receipts. No production deploy
was performed in this session.

## Live validation and a separate uncovered path

The owner used an isolated local server with the retained validated binary,
live ticks, a fresh SQLite store, one immortal observer and two ordinary PCs.
Two equal-description trainees fought the PCs independently. Selective death,
purge, successful flee and transfer ended only the affected pair while the other
fight kept ticking. Distinct PC weapons produced pound/pierce independently;
returning a transferred PC did not restart stale combat. The owner declared the
playtest complete/PASS. The server was subsequently stopped at the owner's request.

Limits: the archived full transcript is the observer's first pass; subsequent
purge/flee/transfer results are operator reports in the conversation and summarized
in `LIVE-PLAYTEST-RESULT.md`. They are not paired C transcripts. Manual reconnect
was not supplied and is not claimed; existing automated session/race evidence
covers separate cleanup paths. Corpse decay was observed but this session did
not establish a new differential proof for it.

The owner also found numbered `restore` targets refused while unnumbered restore
worked. Source inspection shows Go's `cmdRestore` still bypassing the canonical
resolver through `findSessionByName`/`GetMobByName`, whereas C
`src/act.wizard.c:1584-1593` calls ordinal-aware `get_char_vis`. The same lookup is
present on pre-stack main `7f9999464`; this is a retained pre-existing finding.
A new executable/paired proof and fix were not produced in this session. The
existing restore scenario's green does not cover that numbered-target path.

## Claims, corrections and limits

- **PF-056 (verified):** the asymmetric duplicate-body scenario passes at five
  claimed seeds on the committed integrated tip and can fail under the retained
  ordinal mutation. This certifies that fixture, not every combat topology.
- **PF-057 (verified):** six review mutation controls discriminate the repaired
  name-based consumers, including the additional kill/PK/outlaw misattribution.
- **PF-058 (verified):** the final integrated full/claims union and merged tree
  have the recorded verdicts, pair count, elapsed time and exact source identity.
- **PF-059 (partially-verified):** numbered restore refusal is an operator-observed
  pre-existing uncovered path supported by source inspection; paired reproduction
  and repair remain open.
- **RO-013 (verified):** sibling-base PRs lacked attached checks despite successful
  dispatch runs; main-base reopen produced attached PR CI before merge.

These findings complement PF-053/PF-054: review and live use exposed paths beyond
existing fixtures, while adding a sharper fixture did produce genuine byte reds.
They do not establish detection rates, model rankings, reviewer independence,
cost savings, or completeness of the port. This note records new artifact-grounded
observations; the earlier notes remain unchanged.
