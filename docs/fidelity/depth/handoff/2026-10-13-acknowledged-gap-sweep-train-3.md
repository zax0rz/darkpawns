# Acknowledged Gap Sweep — Train 3

Base: fresh `origin/main` at `9b3ecaae3d02cb1f2b066e9dd27c81bb5ec350fc`.
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

- `make fmt`: PASS.
- `go build ./...`, `go vet ./...`, `go test ./...`,
  `go test ./pkg/game/...`: PASS.
- `golangci-lint run ./...`: PASS, 0 issues.
- `make deadcode-ratchet`: PASS, 52 baselined and 0 new.
- `make fidelity-depth`: PASS; 5,513 cases, 11 blocked.
- `make fidelity-units`: PASS; 1,419 rows, 1,306 symbols, 1,600 claims.
- `make string-census`: PASS, 98 known Go-only segments and 0 unreviewed;
  generated reports remain intentionally untouched.
- `gosec -severity=high -quiet ./...`: PASS.
- `go test -race ./pkg/game/...`: PASS.
- Focused `TestDoVisiblePreservesCEntryGateOrder`: PASS after restoring the fix;
  pre-fix control failed as recorded above.
- Scenario parser test: PASS.

The `visible-zai-depth` oracle case remains `blocked` in the manifest until the
shared census slot is available. DP-1419 currently owns the slot; no second
census was started. Start a targeted census for this case, change its manifest
row to `oracle-green` only after a clean result, then run the train's combined
census as required by the goal brief.

## Sweep continuity

Train 1 contains the complete inventory A/B, candidate disposition table, and
the exact deferred implementation issue texts in
`docs/fidelity/depth/handoff/2026-10-13-acknowledged-gap-sweep.md` on PR #1943.
Train 2's Lua object-target work is recorded in
`2026-10-13-acknowledged-gap-sweep-train-2.md`. The remaining actionable
`e` candidates from the train 1 inventory are Lua mobile following, Lua
player-target `set_hunt`, the extra wounded/charmed mobact return, and the
scavenger `CAN_SEE_OBJ` gate. Jail-guard visibility remains assigned to the
combat transcript owner per train 1 triage; the blocked and out-of-scope rows
remain unchanged.
