# Brief for Kimi: DP-1469, spell reagents are found by name; C matches by vnum

Linear: DP-1469 (touches DP-1459 item 3). Rulebook: R1 (bytes), R4 (no invention), R5b and R5c (find the class), R5g (cite C from `src/`, grepped in session), R5h (a green must be able to fail).

**Read first:** `AGENTS.md` (the gates, worktrees and census rules) and `docs/fidelity/RULEBOOK.md`. Dark Pawns is a 1:1 port of a C MUD to Go: the Go server must send players the same bytes as the C server. `src/` is the C original and is read-only.

Work in a worktree: `git worktree add ~/wt/kimi-dp-1469 -b kimi/dp-1469-reagents-by-vnum origin/main`. Never switch branches in `/home/zach/darkpawns`, never `git stash`, and never stop processes by name.

**Other agents are in `pkg/spells` at the same time** (DeepSeek in `call_magic.go` and `inflictDamage`, GLM on level reads in `affect_spells.go`, MiMo on `isGood`/`isEvil`). Stay inside the reagent code. Rebase onto `origin/main` before opening the PR. If a conflict lands in someone else's hunk, stop and report.

This brief is sized for one PR. If you run short of budget, stop after step 3 and report what's done.

## Ground truth (verified by Claude at `origin/main` 3298d1de9)

**C:**
- The reagent table: `reagents[][]` (`src/magic.c:465-477`), one object vnum per spell: sleep 1226, fireball 1225, disintegrate 1227, energy drain 1230, curse 1233, blindness 1234, charm 1231, metalskin 1229, waterwalk 1228, color spray 1236, magic missile 1239.
- `has_reagents` (`src/magic.c:480-492`) calls `mag_materials` (`:540-592`). That walks `ch->carrying` in order, matches `GET_OBJ_VNUM(tobj) == item0`, extracts the first match, and returns TRUE or FALSE.
- **How C uses the result differs by spell.** The damage spells use `reag = has_reagents(...) ? level : 0` (`src/magic.c:621,659,675,691,775`). The affect spells use the boolean, 0 or 1: blindness `:932`, curse `:978`, sleep `:1201`, waterwalk `:1274`, metalskin (grep it yourself). Waterwalk's duration is `af[0].duration = 4+reag?20:0;` (`src/magic.c:1278`): C's operator precedence makes it `(4+reag) ? 20 : 0`, so it's always 20. Keep that quirk (R1a) and check what Go does.

**Go:**
- `checkReagents` (`pkg/spells/affect_spells.go:1638`) finds the reagent **by name** through `Player.ConsumeSpellReagent(name)` (`pkg/game/spell_reagents.go:6`, `Inventory.FindItem`). Any object whose keywords match is consumed, and a real reagent with a different name is missed.
- It also returns an invented bonus, `level / 2` (minimum 1), or 1 with a `"flat:1"` argument (DP-1459 item 3).
- Callers:
  - waterwalk and metalskin, `pkg/spells/affect_spells.go:63,66`;
  - the five damage spells through `consumeReagent`, `pkg/spells/damage_spells.go:26-56`.
- **The faithful path already exists.** `Player.ConsumeSpellReagentVNum` (`pkg/game/spell_reagents.go:35`) is used by blindness, smokescreen, sleep and curse through `consumeSpellReagentVNum` (`pkg/spells/affect_spells.go:499`).
- `TestCheckReagents_ItemConsumedOnExactShortDescMatch` pins the name match.

## Do

1. **Route every reagent spell through the vnum path**, with C's vnums from the table. Make sure the carry-list order matches C's `ch->carrying` walk: say what order `Inventory.Snapshot()` returns and whether it's C's.
2. **Return C's value at each site:** `level` for the damage spells, 1 for the affect spells. Delete the `level / 2` bonus and the `"flat:1"` convention. Check each affect spell's use of `reag` against C, including waterwalk's always-20 duration.
3. **Remove `ConsumeSpellReagent` (by name) and `checkReagents`** if nothing else uses them. Rewrite or delete the tests that pinned the name match.
4. **Audit (R5b):** write `docs/fidelity/evidence/dp-1469/reagent-sites.tsv` with columns `spell`, `C file:line`, `C vnum`, `C reag value`, `Go file:line`, `Go before`, `Go after`. Include charm (1231), which C lists as "TBD": say whether C's charm path calls `has_reagents` at all.

## Proof

1. **Unit tests:** a decoy object carrying the reagent's keyword but a different vnum is **not** consumed, and the spell acts as if there were no reagent. A real reagent with an unrelated name **is** consumed. Do one damage spell and one affect spell. Revert proof: restore the name lookup, and the decoy test fails. Paste the failure.
2. **An oracle scenario** with a decoy: the caster carries an object whose keywords match the reagent but whose vnum doesn't. Use an existing world object if one fits; otherwise use OLC the way the `spellmatrix-*` scenarios do. Cast the spell and show the component message, or its absence. It must be red before the fix (R5h). Read `docs/fidelity/DEPTH_TESTING.md` and an existing `cmd/dp-oracle-diff/scenarios/spellmatrix-*` file for the format.
3. **A targeted census** of your scenario plus every scenario that casts a reagent spell (grep the scenario directory for the eleven spells, and name the set): `scripts/census.sh start --name kimi-dp1469-reagents --scenarios …`, then `scripts/census.sh wait --run <dir>`. Don't poll it in a loop.

## Rules

- Cite C only from `src/`, grepped in this session.
- Don't edit `docs/fidelity/RULEBOOK.md`, `docs/fidelity/north-star-1to1.md` or `AGENTS.md`.
- Never invoke another model, agent or paid service.
- Run every gate, each on its own, and check its exit status. Never chain them with `;`: `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `make fmt` (gofumpt, not gofmt), `make deadcode-ratchet`, `make fidelity-depth`, `make expected-divergences-check`, and `make db-migrate-test` if you change an exported `pkg/game` symbol (removing `ConsumeSpellReagent` counts). CI `test` must be green before you report.

## Done means

A PR against `main`, **with `DP-1469` in its title** (Linear closes issues from the PR title), containing the TSV, the fix, the tests and revert proof, the scenario, the census directory, and a **Decisions for Zach** section. Then stop and report. Claude reviews in Zach's session, and Zach merges.
