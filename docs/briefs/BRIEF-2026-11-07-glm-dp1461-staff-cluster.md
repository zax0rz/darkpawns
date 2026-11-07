# Brief for GLM: the run-varying staff cluster (DP-1461)

Linear: DP-1461. Rulebook: R1, R3a/R3b (draw parity and order), R5e, R5h, R5j.

## Ground truth (verified by Claude at `origin/main` b7fdd6ef5)

Five staff scenarios are pinned **`run-varying`**. Their first divergent block moves from run to run, which usually means an order or draw difference, not a fixed byte:

| Scenario | Spells on the staffs |
|---|---|
| `spellmatrix-staff-olc-08-depth` | group recall, lycanthropy, mass heal, sobriety, vampirism |
| `spellmatrix-staff-olc-09-depth` | enchant armor, group invisibility, hellfire, identify, mind poke |
| `spellmatrix-staff-olc-12-depth` | adrenaline boost, cell adjustment, change density, dominate, psychic shield |
| `spellmatrix-staff-olc-15-depth` | calliope, flame strike, haste, protection from good, slow |
| `spellmatrix-staff-olc-16-depth` | dream travel, drowning, glyph of summoning, psiblast, zen |

DP-1461's original note on staff-olc-15: flaming arrow hits ItemPeer in C and ItemView in Go. That's either area target order (R3b) or a draw difference (R3a).

**C's staff fan-out:** `mag_objectmagic`, `ITEM_STAFF` case (`src/spell_parser.c:579-608`). It walks `world[ch->in_room].people` and calls `call_magic` for every occupant except the user, in **room list order** (`:596-607`). It saves `next_tch` **before** each call (`:598`), so a victim who dies or moves doesn't break the walk; check Go does the same. `char_to_room` **prepends** (`src/handler.c:537-538`), so the newest arrival comes first, and players and mobs share that one list. Go has helpers that already sort newest-first (`COrderedRoomMobs`, `GetPlayersInRoom`). Check whether the staff path uses them, and whether it interleaves players and mobs the way C's single `people` list does.

**Not yours:** mirror image, divine intervention and every `create_mobile` path are DeepSeek's today (DP-1487). Group member order (`mag_groups`) and `damage()`'s `stop_follower` are MiMo's (DP-1489). If a staff scenario's divergence turns out to be one of those, say so and move on.

## Do

1. **Measure the variation.** Run the five with `--draw-parity` at least three times, or use the rehearsal run plus two targeted runs. For each scenario, report whether its first divergent block is fixed, and what `draw_divergent` reports, with the draw position.
2. **Trace each scenario (R5e).** Find each one's root cause, with C and Go cites. Expect one or two shared causes: the staff fan-out order is the first suspect. Write it up in `docs/fidelity/evidence/dp-1461/staff-cluster.md`.
3. **Fix the shared causes at the source.** If a cause is a spell-specific bug outside the fan-out, fix it if it's small; otherwise list it for Claude to file.
4. **Tests.** A unit test for room-order fan-out with at least three occupants, players and mobs mixed. It needs a revert proof.
5. **Scenarios.** Each scenario that goes green gets its pin removed, its rows set to `oracle-green-multiseed` once it's green on seeds 1, 2, 3, 5 and 8, and a C witness line. Name any that stay red, with their first divergent block.
6. **Census.** Every `spellmatrix-*staff*` scenario plus every scenario using `use` on a staff (grep). Name the set.

## If you're low on usage

Step 1 plus the trace of staff-olc-15 alone is a useful stopping point. Report it.

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
