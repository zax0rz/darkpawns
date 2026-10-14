# Acknowledged Gap Sweep — Train 3

Base: rebased from `9b3ecaae3d02cb1f2b066e9dd27c81bb5ec350fc` onto fresh
`origin/main` at `d8e8024ddbfe52fbde36d16e0b02e2d7606fac71` after PRs #1943,
#1944, and #1945 merged.
Scope: `do_visible` Kai-zai affect lookup.

## Finding and change

`pkg/game/other_character.go` previously searched the affect's display source
for the substring `zai`. The C handler calls `affected_by_spell(ch,
SKILL_KK_ZAI)`, and `src/handler.c:460-469` compares the numeric affect type
exactly. `src/act.other.c:563-566` confirms the live command gate; `src/interpreter.c:844`
registers `zai`, and `src/new_cmds.c:1552-1739` creates the affect through the
same command path.

The Go handler now uses `HasSpellAffect(skillNumKkZai)`. Unit controls prove
that affect ID 164 blocks regardless of source text and another affect whose
source text contains “zai” does not block. The live oracle vehicle
`visible-zai-depth` performs `zai` followed by `visible` and is named by the
depth manifest row `visible.zai-gate`.

## R5h control

The pre-fix handler from `origin/main` failed the focused test in both
directions:

- ID 164 with unrelated source text emitted `You are already visible.`
  instead of C's zai refusal.
- ID 165 with source text containing “zai” emitted the zai refusal instead of
  `You are already visible.`.

Restoring the numeric-ID lookup made both focused cases pass. No RNG draw was
added; the `visible` branch remains draw-free, and the live vehicle's prior
`zai` command consumes matching C/Go draws.

## Verification

The branch was rebased onto `origin/main` at
`d8e8024ddbfe52fbde36d16e0b02e2d7606fac71`; all gates below were rerun
separately on the integrated tree and passed.

- `make fmt`: PASS.
- `go build ./...`, `go vet ./...`, `go test ./...`,
  `go test ./pkg/game/...`: PASS.
- `golangci-lint run ./...`: PASS, 0 issues.
- `make deadcode-ratchet`: PASS, 52 baselined and 0 new.
- `make fidelity-depth`: PASS; 5,521 cases, 5,431 proven/delegated, 10 blocked,
  68 excluded, 12 divergent-approved.
- `make fidelity-units`: PASS; 1,424 rows, 1,313 symbols, 1,607 claims.
- `make string-census`: PASS; 94 Go-only, 98 known segments, 0 unreviewed;
  generated reports remain intentionally untouched.
- `gosec -severity=high -quiet ./...`: PASS.
- `go test -race ./pkg/game/...`: PASS.
- Focused `TestDoVisiblePreservesCEntryGateOrder`: PASS after restoring the fix;
  pre-fix control failed as recorded above.
- Scenario parser test: PASS.

The seed-1 `--show-oracle` run at `6fe56c822b747e86af965ac3d72e5c8cdcb78b18`
had no normalized divergence. Its C blocks show the `zai` cast and the
`visible` refusal while the affect is active; the output and run manifest are
retained under
`/home/zach/Archives/darkpawns/oracle-runs/2026-10-13/dp-1371-train3-show-oracle/`.
The `visible-zai-depth` targeted census is clean: one passed, zero expected,
unpinnable, stale, failed, infra, timeout, or unstable; elapsed 16.868s. The
`results.tsv` row is `PASS visible-zai-depth 10`. Its summary and manifest are retained under
`/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/dp-1371-train3-visible-zai/`.
The `visible.zai-gate` manifest row is now `oracle-green`. The final integrated
combined census on tip `94758e499` completed clean:
`oracle-combined: pairs=3402 deduplicated=1018 full=CLEAN claims=CLEAN elapsed=1956.788s verdict=CLEAN`.
Its `results.tsv` rows for `visible-zai-depth` and `lua-bind-actions` both
pass. The retained run is at
`/home/zach/Archives/darkpawns/oracle-runs/2026-10-14/dp-1371-train-3-integrated/`.

## Sweep continuity

Train 1 contains the complete inventory A/B, candidate disposition table, and
the exact deferred implementation issue texts in
`docs/fidelity/depth/handoff/2026-10-13-acknowledged-gap-sweep.md` (merged PR
#1943). Train 2's Lua object-target work is recorded in
`2026-10-13-acknowledged-gap-sweep-train-2.md` (open PR #1953). Train 4's
mobile and Lua binding changes remain in its own worktree pending census and
PR. Lua mobile following is one remaining actionable `e` candidate. Jail-guard
visibility remains a separate candidate; only the shared attack/message/RNG
divergence is owned by `mob.teleport-victim-combat-transcript`, as clarified
in the Train 2 handoff. Other pending candidates and triage remain in the Train
1 inventory and Train 4 handoff.
