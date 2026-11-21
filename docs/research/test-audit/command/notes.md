# Test audit: pkg/command — notes

Audit run: 2026-11-20, branch `kimi/test-audit-command`, base `origin/main` (b2e884aca).
Auditor: Kimi. Method: `docs/research/test-audit/pilot-combat/BRIEF.md` with the three
changes from PROMPT-2026-11-20 (single agent self-checks non-R marks; fix-as-you-go;
output in-repo).

Baseline: `go test ./pkg/command/...` passes at b2e884aca.

Ground truth: 173 test functions in 34 files (5,025 test lines) against 14 production
files (7,385 lines). 37 tests born before the first oracle harness (2026-07-12,
ba05d9906): 6 registry/middleware (a767d832c, 2026-05-27), 1 rescue (7465d8aef,
2026-06-26), 3 bash (54cd125d9/02b111ae4, 2026-07-05/11), 27 kill-payout (8183a9039,
2026-07-06).

## Predictions (written before reading any test)

Both earlier audits (PF-103 combat, PF-107 spells) put nearly all weak marks on
pre-oracle tests: 93 of 94 later combat tests were keepers. I expect the same shape
here, slightly milder — pkg/command's later tests are mostly oracle-era depth tests
with C citations, and its one big pre-oracle block (27 kill-payout tests from a
single 2026-07-06 commit) is exactly the kind of go-generated-expectation suite the
pilot flagged as J1.

| Mark | Pre-oracle (37) | Later (136) | Total (173) |
|---|---|---|---|
| R keep | 24 | 127 | 151 (87%) |
| F fix | 7 | 5 | 12 (7%) |
| C merge | 4 | 3 | 7 (4%) |
| D delete | 2 | 1 | 3 (2%) |

Junk codes expected: J1 (Go-generated expected values, especially kill-payout XP
numbers and gold) dominates pre-oracle; J2 (assertions that cannot fail) and J8
(mock/fixture supplies the asserted behavior) next; a little J5 (implementation-shape
assertions) in the registry/middleware block. At least one live Go/C divergence
pinned by a test (J10) — both earlier audits found exactly one each.

Confidence: low on the F/C/D split between pre-oracle and later (the totals are a
firmer guess than the partition).

## Baseline

`GOEXPERIMENT=nojsonv2 go test ./pkg/command/...` → ok (2026-11-20).

## Batches

- Batch 1: the 37 pre-oracle tests. DONE (this PR).
- Later batches: remaining 136, oldest first, ~50–100 per batch.

## Batch 1 results (37 pre-oracle tests)

Marks: R 32, F 2, C 0, D 3.
Against prediction (R 24 / F 7 / C 4 / D 2): fewer weak tests than feared, but
the D marks became a finding the prediction missed — the entire `pkg/command`
middleware layer (`Middleware`, `Registry.Use`, `LoggingMiddleware`,
`WhitelistMiddleware`) had **zero production callers** (grep of pkg/, cmd/,
internal/ on 2026-11-20). Their tests protected dead code:
- D `TestMiddleware_Logging` (J2+J7): asserts nothing at all.
- D `TestMiddleware_Whitelist` (J7): no caller.
- D `TestRegistry_Middleware` (J7): `Use`/chain has no caller.

Deleting the tests tripped `make deadcode-ratchet` (the deleted tests were the
only thing marking those functions reachable). The ratchet's own remedy —
"wire, delete, or baseline" — so this PR also deletes the dead production code:
`pkg/command/middleware.go`, the `Middleware` type, `Registry.Use`, the
`middleware` field and the wrap loop in `Register`. It updates
`docs/architecture/ARCHITECTURE.md` (whose middleware section was already stale —
it documented a `RateLimitMiddleware` that never existed) and drops the two dead
rows from `docs/research/function-index/index.tsv` (regenerating wholesale
produced 414 lines of unrelated drift; the check's ratchet passes either way).
No census: the removed functions were unreachable, so no oracle scenario can
observe the change (proved by the ratchet report plus the caller grep, R5h — no
scenario exists that could fail on it).

F fixes (both proven by mutation, see mutations.tsv):
- `TestCmdBash_FightingTargetFallback` (J9): negative-only assertion; a no-op
  `CmdBash` stayed green. Fixed with a positive knockdown assertion
  (target `PosSitting`, src/act.offensive.c:493). The fixture also had to move
  above C's level-10 player-protection gate (src/fight.c:1345-1356) and gain
  move points (src/act.offensive.c:423) — the original fixture could never land
  a bash, another reason the negative check was hollow.
- `TestKillPayout_PlayerDeath_Idempotent` (J3): the second death invocation the
  name promises was never made. Fixed to call `HandleDeath` again and assert no
  double punishment (queued-extract guard, pkg/game/death.go handlePlayerDeath;
  the test's own "after respawn" comment was wrong — raw_kill only queues
  extraction, src/fight.c:534-577).

Cited-value checks that passed: 240 XP row reproduces C `calc_level_diff`
(src/fight.c:660-685) by hand; exp/37 loss matches src/fight.c:589;
`Bash who?`/`Backstab who?` match src/act.offensive.c:446/:181;
PULSE_VIOLENCE 20 matches src/structs.h:628-634. No J10 divergences found in
this batch (both earlier audits found exactly one each — not this time).

Attribution (git only): 5 introducing commits. 4 by git author BRENDA69
(agent unknown — no prefix/trailer): PR #19, #84, #113, #182. 1 by zach with
Co-Authored-By Claude Opus 4.8 (agent claude): PR #37.

Coverage: 68.4% before → 66.6% after the test deletions → 67.8% after deleting
the dead production code. The func-level coverprofile diff after the test
deletions showed the only newly-uncovered functions were the three dead ones;
after the production removal the denominator shrank to live code only, and no
remaining statement lost a test — behavior coverage only got stricter (two
contracts gained real assertions).


## Batch 2 results (74 post-oracle July tests)

Marks: R 69, F 5, C 0, D 0. Junk: 5×J3, all in one family.

The one weak family: the `_Executes` smoke tests from the DP-608/658 utility
coverage push (PR #414, claude). `TestCmdBerserk_Executes`,
`TestCmdDetect_Executes`, `TestCmdScrounge_Executes`, `TestCmdKujiKiri_Executes`
and `TestCmdPoint_Executes` asserted only `err == nil` — every one stayed green
when its handler was mutated to a no-op (mutations.tsv). Fixed by asserting the
exact C gate bytes each fixture input produces (src/new_cmds.c:2142,
src/new_cmds2.c:506, src/new_cmds2.c:73, src/new_cmds.c:1561,
src/new_cmds.c:2527); each fixed test goes red under the same mutation.

Everything else in the batch held up. The glm DP-1212/1213 draw-order and
enrollment tests are the strongest tests in the package: seed search that
accepts only seeds distinguishing C order from the old order (sharp by
construction, R5a), three independent observables per run, and the real
sendSkillResult path. The 33 gate byte pins from PR #414 (bearhug, behead,
bite-no-args, carve, compare, cutthroat, disarm, disembowel, dragon kick,
first aid, flesh alter, mindlink, palm, serpent kick, sharpen, slug,
smackheads, spike/stake no-args, strike, tag, tiger punch — plus the XPLower
formula row and the three-command thief utility test) were each verified
against grepped C lines. The byte pins dated later (turn, mold, compare blind,
carve exact-no-args, spike/stake NOPERSON gates, bite punctuation and the
silent-bite contract, strike's charmed-master gate) belong to batch 3; their
verification is recorded there. No J10 divergences found.

Attribution: glm 8 (PRs #469×4 tests, #474×2, #476, #460), claude 64 (PR #414,
Co-Authored-By trailer; branch prefix was clawpatch), codex 1 (PR #366, branch
prefix; git author BRENDA69), unknown 1 (PR #189, git author BRENDA69).

Coverage: 67.8% before → 67.8% after (assertions only; no tests added or
removed, so the denominator is unchanged and no live statement lost coverage).
