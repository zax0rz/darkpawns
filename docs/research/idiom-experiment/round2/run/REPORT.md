# Idiomatic-Go experiment, round two: report

Run on `luna/idiomatic-go-round2`, based on `5e9556a8789c4125bed8913d34d16e6f2ed68fac`, the tip of draft PR #2275. The Stage 1 population and sample were committed before any source refactor; the preregistered `PREDICTIONS.md` was not changed.

## Work completed

- **Tier A:** changed the 88 in-scope empty-interface spellings to `any`, in one commit per package. The underscore-name audit found no eligible unexported rename; the remaining candidates were exported or deliberately C/Lua-facing. The required full non-combined Tier A census at `603005922` is `/home/zach/Archives/darkpawns/oracle-runs/2026-11-04/luna-r2-tier-a`, verdict `CLEAN_AFTER_RECHECK`.
- **Tier B:** kept 21 new one-target extraction commits across ranks 6–13, 15, and 17. Every first real targeted census was `CLEAN`; there were no abandoned targets. Nine of the 21 mutation probes stayed green and are described in `findings.md`. Rank-by-rank dispositions and all non-attempted subtargets are in `target-review.md`.
- **Verification:** per-target formatting, build, vet, touched-package tests, focused lint, and deadcode gates passed after fixes. Final `go test ./...` and `go test ./pkg/game/...` also passed. The focused four-linter count moved from `dupl 26, funlen 191, gocognit 111, gocyclo 4` to `26, 190, 107, 4`. No exported symbol changed, so `make db-migrate-test` was not required.
- **Final combined census:** not run; Claude owns the reviewed batch census as specified by the repository instructions.

The measured predictions are tabulated in [`../RESULTS.md`](../RESULTS.md). In brief: the frozen sample caught 23/40 mutations (57.5%), with 17 green probes; the agent's pre-census scenario predictions were 23 exact, 7 partly correct, and 10 incorrect. Eight unrelated infra rows occurred across seven sample-probe censuses and were rechecked; probe 23 required a full retry after its same infra scenario failed the recheck. All final sample outcomes are in `probes.tsv`.

## What surprised me

- The clarified RNG rule made several previously avoided extractions safe. Rank 17's `Dream` level switch and `huntVictim` evasion gate contain random draws; each remains in a helper with one call site at its original position, with condition order and early-return behavior retained.
- The Stage 1 green rate was 42.5%, above the 30% point prediction but inside its 15–50% interval. Several rank-17 probes also stayed green even though the function-level census was clean; the exact mutated branch did not appear in the retained normalized output for those runs.
- Call-path tracing corrected the initial rank-13 backstab fixture choice: `ApplySkillDamageWithMessage` bypasses `takeDamageFrom`. The ambush hit/miss/death fixtures exercise the actual `takeDamageFrom` path, and the corrected death-message probe was caught.
- A first cleric gate pass exposed that its target is `combat.Combatant`, not `*MobInstance`; the helper signature was corrected and all gates passed. A first lint pass on rank 11c found a redundant return; that was removed before the final gates.

## Population measurement limitation

Post-run review found a same-function argument-map defect in the frozen
`population.go`. `SendToChar` and `SendToRoom` each have a free-function or
two-argument selector form whose message is at index 1, and a one-argument
`DreamContext` interface-method form whose message is at index 0. The analyzer
keys only by `callName` and indexes `CallExpr.Args` without considering whether
the call is a selector, so the map's `{1}` misses literals in `Dream` and
`DreamTravelFn`. This means the recorded 402-row population undercounts at
least those two functions. The population hash, 40-function sample, and
denominator were left frozen; the 23/40 result describes that preregistered
syntactic population, not a repaired census of all interface-method output.

This is a research-instrument issue, not a discovered game-port bug. The
measurement script should distinguish selector-call argument positions from
package-function positions before the population is reused. The proposed
ledger row is deferred because open PR #2271 also edits
`docs/research/EVIDENCE_LEDGER.tsv`:

```tsv
idiom-r2-population-selector-args	legacy-port-fidelity	The frozen round-two population inventory omits Dream and DreamTravelFn message literals passed through DreamContext.SendToChar and SendToRoom because the analyzer maps these names to free-function argument index 1 while selector calls expose the message at CallExpr.Args[0].	verified	2026-11-05	docs/research/idiom-experiment/round2/run/population.go; pkg/game/dreams.go; population SHA-256 a790e7a2d87f334eb37495f1433714de54b388d8d7d9f865ce1f13c52451b631		Found after sample freeze; do not revise this run's denominator.
```

## Findings and rule proposals

There were no latent game-port bugs to log. Census blind spots, including the
green Stage 1 probes and the green rank 8, 12, 13, and 15–17 Tier B probes, are
recorded in `findings.md`. No R1–R5 rulebook amendment is proposed. No scenario
was added or edited.

## Protected-path check

Command run from the round-two worktree against base SHA
`5e9556a8789c4125bed8913d34d16e6f2ed68fac`:

```bash
git diff --name-only 5e9556a8789c4125bed8913d34d16e6f2ed68fac..HEAD | grep -E '_test\.go$|^cmd/dp-oracle-diff/|^internal/oraclediff/|^docs/fidelity/|^scripts/|^AGENTS\.md$|^src/|^pkg/(session|db|telnet|spells)/'
```

Output:

```text

```

No protected paths or open-PR package files were changed. The requested
`EVIDENCE_LEDGER.tsv` update is the one deferred file because PR #2271 owns it;
the proposed row above is ready for Zach to apply after that conflict clears.
