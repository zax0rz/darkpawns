# Brief for MiMo: group scroll member order, and `damage()`'s `stop_follower` (DP-1489)

Linear: DP-1489. Rulebook: R1, R3b (order is behavior), R5b/R5c, R5e, R5h, R5j.

## Ground truth (verified by Claude at `origin/main` b7fdd6ef5)

**The divergence.** `spellmatrix-scroll-olc-02-depth` diverges at `recite book ItemPeer [actor]`, census `~/Archives/darkpawns/oracle-runs/2026-11-06/gemini-dp1456-witness`. It's pinned to DP-1489. The scroll carries the group and mind spells listed in the scenario header: group recall, group invisibility, mind poke, mind blast, acid blast, and others. Gemini read the divergence as two causes. Neither is verified yet: **verify both from the transcript and the source before fixing (R5e)**, and check the block again on today's baseline. #2325 (`damage()`'s default arm) has merged since Gemini looked, so part of the divergence may already be gone.

**C `mag_groups`** (`src/magic.c`, after `perform_mag_groups` at `:1431`) walks the group in this order:
1. Starting from `k = ch->master` (or `ch` itself), every **follower of k** in `k->followers` list order, skipping `ch`, anyone in another room, and anyone not `AFF_GROUP`.
2. Then `k` itself, if it's in the room and grouped.
3. Then `ch` last, if `k != ch`.

**List order matters.** Read `add_follower` and `add_follower_quiet` in `src/utils.c` to find where a new follower is inserted (head or tail), and cite it.

**`perform_mag_groups`** (`src/magic.c:1431-1450`): `SPELL_MASS_DOMINATE` has **no `break`**, so it falls through into `SPELL_GROUP_INVIS`. That's C's bytes (R1). Check that Go reproduces the fall-through.

**`damage()`** (`src/fight.c:1457-1458`): `if (victim->master == ch) stop_follower(victim);`. This runs on every `damage()` call, after the fighting setup and before the inline hide fade (`:1460-1464`). Check whether Go's spell-damage path (the arm #2325 made live) and its melee path both do this, in that position.

## Do

1. **Trace (R5e).** Run `spellmatrix-scroll-olc-02-depth` on today's main, and read C's transcript line by line from the `recite`. Write up which of the two causes is real, plus anything else, in `docs/fidelity/evidence/dp-1489/trace.md`, with cites.
2. **Fix it at the source.** Port the follower walk in C list order, the fall-through, and `stop_follower` at its C position in every Go twin of `damage()`. Grep for every place Go calls a damage entry point.
3. **Audit the class.** Check every Go group or area walk against C's list order: `mag_groups`, `mag_areas`, `mag_masses` (if present), group gain and split, and `perform_group` in `src/act.other.c`. Write it to `docs/fidelity/evidence/dp-1489/member-order.tsv` with columns `C walk (file:line)`, `C order`, `Go site`, `Go order`, `match?`. Fix any mismatch in this PR if it's small. If it isn't, list it for Claude to file.
4. **Tests.** A unit test for follower order, one for the fall-through, and one for `stop_follower` on damage. Each needs a revert proof.
5. **Scenarios.** Get `spellmatrix-scroll-olc-02-depth` green (remove its pin, set its rows to `oracle-green`, and add the C witness line). If no existing scenario has **three or more** grouped members, add one, so that order can be seen at all (R5h: a two-member group can't show an order bug).
6. **Census.** Every scenario that groups or follows (grep `group`, `follow`) plus every `spellmatrix-*scroll*` scenario. Name the set.

## Rules (all briefs)

- Read `AGENTS.md`, `docs/fidelity/RULEBOOK.md` (R1–R5j) and `docs/fidelity/DEPTH_TESTING.md` first.
- Work in your own worktree: `git worktree add ~/wt/<agent>-<issue> -b <agent>/<issue-slug> origin/main`. Never switch branches in `/home/zach/darkpawns`, never `git stash`, never stop processes by name. Other agents are working in `pkg/` today, so rebase on `origin/main` before you report.
- `src/` and `darkpawns-c-oracle/` are read-only. Cite C only from `src/`, grepped in this session (R5g), never from memory.
- Don't edit `docs/fidelity/RULEBOOK.md`, `docs/fidelity/north-star-1to1.md` or `AGENTS.md`. If a rule needs changing, propose the exact text in your report.
- Never invoke another model, agent or paid service.
- Every fix gets a unit test **and a revert proof**: revert the fix, show the test fail, and quote the failure (R5h). Every scenario that you turn green needs a C witness line, meaning C's own transcript shows the event happening (R5j).
- **Pins.** If you clear a divergence, its pin goes STALE. Remove the pin. Don't re-mint it. If you find a new divergence, don't pin it, opt it out or edit the scenario to pass: report it with its first divergent block, and Claude files it. If a PR legitimately needs new pins, mint them only with `scripts/pin_expected_divergences.py --from-run <run-dir>`, run from your worktree. Then run `make expected-divergences` and commit.
- **Census** only through `scripts/census.sh` (`start --name … --scenarios …`, then `wait --run <dir>`). Don't poll it, and don't `cat` or `tail` a running `census.log`. Attribute every changed row against the rehearsal baseline `~/Archives/darkpawns/oracle-runs/2026-11-07/claude-rehearsal-1107` (a full combined census of main at `b7fdd6ef5`, with draw parity).
- Run every gate on its own and check its exit status. Never chain gates with `;`. The gates: `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `make fmt` (gofumpt), `make deadcode-ratchet`, `make fidelity-depth`, `make expected-divergences-check`. CI `test` must be green before you report.

## Done means

A PR against `main` with **the Linear ID in its title** (GitHub→Linear closes issues only from the title). The PR body has the trace, the tests, the revert proofs, the census directory, and a **Decisions for Zach** section. Then stop and report. Claude reviews in Zach's session, and Zach merges.
