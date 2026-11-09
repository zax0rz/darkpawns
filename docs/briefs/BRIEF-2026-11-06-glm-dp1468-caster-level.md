# Brief for GLM-5.3: DP-1468 + DP-1458, spells use the wrong level

Linear: DP-1468 (and DP-1458). Rulebook: R1 (bytes), R3 (determinism and draw parity), R5b and R5c (find one, find the class), R5g (cite C from `src/`, grepped in session), R5h (a green must be able to fail).

Work in a worktree: `git worktree add ~/wt/glm-dp-1468 -b glm/dp-1468-caster-level origin/main`. Never switch branches in `/home/zach/darkpawns`, never `git stash`, and never stop processes by name.

**Other agents are in `pkg/spells` at the same time.** DeepSeek's #2318 changes `call_magic.go` and `damage_spells.go`, MiMo is changing `isGood`/`isEvil` in `damage_spells.go`, and Kimi is changing `checkReagents` in `affect_spells.go`. Stay out of those functions. Rebase onto `origin/main` before opening the PR. If a conflict lands in someone else's hunk, stop and report.

## Why

A spell has two levels. One is the **cast level** passed to the spell routine; from a wand, staff, scroll or potion, that's the object's level. The other is the **caster's own level**, `GET_LEVEL(ch)`. For a player casting a spell they're the same, so every fixture passes. For an item cast they differ, and C reads the caster's level in many places where Go reads the cast level. The staff/wand oracle matrix can now reach those paths.

## Ground truth (verified by Claude at `origin/main` 3298d1de9)

**Oracle proof already exists for lay hands.** In `~/Archives/darkpawns/oracle-runs/2026-11-06/cline-dp1481b-cluster`, `spellmatrix-wand-olc-09-depth` and `spellmatrix-staff-olc-14-depth` diverge at draw 4186: C rolls `Dice(3,30)`, using ItemMage's level 30, and Go rolls `Dice(3,20)`, using the wand's spell level. The output matches only because the heal hits max HP.

**Known sites:**

| C | Go |
|---|---|
| lay hands `hit = dice(3, GET_LEVEL(ch))`, `src/magic.c:1823` | `magPointsFormula`, `pkg/spells/affect_spells.go:804` (called at `:689`) |
| circle of summoning `GET_OBJ_TIMER(obj) = GET_LEVEL(ch) / 2 + number(-2, 1)`, `src/spells.c:1018` | `pkg/spells/affect_spells.go:2097` |
| meteor swarm `dam = GET_LEVEL(ch)*6 + number(-10, (GET_LEVEL(ch)*3))`, `src/spells.c:1094` | `pkg/spells/affect_spells.go:2995` |
| mirror image `create_mobile(ch, MOB_CLONE, 1, FALSE)`, `src/spells.c:1136`: **a literal 1** | `SpawnMobWithLevelI(69, roomVNum, level)`, `pkg/spells/affect_spells.go:3917` (DP-1458) |

**The class is bigger.** Claude's grep finds 35 `GET_LEVEL(ch)` uses in `src/magic.c` and `src/spells.c`, including:
- most `mag_affects` durations (`src/magic.c:987-1333`);
- the metalskin modifier (`:1310`);
- the `create_mobile` levels at `src/magic.c:1726` and `src/spells.c:1212`;
- `src/spells.c:1049, 1072, 1178`;
- saving throws (`src/magic.c:415-417`).

Some of these are level comparisons where `ch` may not be the caster. Read each one in context.

## Do

1. **Audit (R5b).** For every `GET_LEVEL(ch)` and every `level` use in `src/magic.c` and `src/spells.c`, find the Go site and record which level it reads. Write `docs/fidelity/evidence/dp-1468/level-sites.tsv` with columns `C file:line`, `C reads (GET_LEVEL(ch) / level / literal)`, `Go file:line`, `Go reads`, `match (yes/no)`, `fix`. Also do every `create_mobile` call in `src/` against its Go spawn, for the level argument (DP-1458's R5c).
2. **Fix every mismatch.** Read the caster's level through whatever accessor the package already uses for `getLevel(ch)`; don't add a parallel one.
3. **Mirror image** spawns at level 1. Correct `TestCastMirrorImageOutputAndRoomAudience` to C's value.
4. **Draw parity (R3).** Meteor swarm's `number()` bounds change with the level, so check that the draw count stays the same and that only the bounds move.

## Proof

1. **Unit tests** for lay hands, meteor swarm, circle of summoning and mirror image, each with the cast level different from the caster's level. Revert proof for each: with the fix undone, the test fails. Paste the failures.
2. **Oracle:** `spellmatrix-wand-olc-09-depth` and `spellmatrix-staff-olc-14-depth` stop being `draw_divergent` under `--draw-parity`. If #2318 has merged, run on top of it; if not, say so and run on `main`.
3. **New scenarios** where it isn't covered yet: at least mirror image via an item (a scenario showing the clone's stats, for example `stat` on the clone, red before the fix) and one `mag_affects` duration from an item cast that's visible through `affects` or `stat`. Each must fail with the fix reverted (R5h).
4. **A targeted census** of every `spellmatrix-*` scenario plus your new ones: `scripts/census.sh start --name glm-dp1468-caster-level --scenarios …`, then `wait --run <dir>`; don't poll. Attribute every changed row. Re-mint any DP-1460/DP-1461 pin that changes shape, with `scripts/pin_expected_divergences.py --from-run`, and say which issue each remaining shape belongs to. A pin row that goes STALE means fixed: remove the pin, don't re-mint. Before you call a changed pin "smaller", compare its blocks with the old run.

## Rules

- Cite C only from `src/`, grepped in this session, never from memory or from `darkpawns-c-oracle/`.
- Don't edit `docs/fidelity/RULEBOOK.md`, `docs/fidelity/north-star-1to1.md` or `AGENTS.md`. Propose any rule text in the PR description.
- Never invoke another model, agent or paid service.
- Run every gate, each on its own, and check its exit status: `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `make fmt`, `make deadcode-ratchet`, `make fidelity-depth`, `make expected-divergences-check`, and `make db-migrate-test` if an exported `pkg/game` symbol changes. CI `test` must be green before you report.

## Done means

A PR against `main`, **with `DP-1468` and `DP-1458` in its title** (Linear closes issues from the PR title), containing the TSV, the fixes, the tests and their revert proofs, the scenarios, the census directory with attribution, and a **Decisions for Zach** section. If the audit finds more than about 15 mismatches, fix the four known sites plus the `mag_affects` durations, list the rest, and stop. Then stop and report. Claude reviews in Zach's session, and Zach merges.
