# Brief for Gemini: act()'s missing capital, and two DP-1470 leftovers (DP-1488, DP-1490)

Linear: DP-1488 and DP-1490. Put both IDs in the PR title. Rulebook: R1, R1a (the reference transcript decides undefined behavior), R5b/R5c (the class), R5h, R5j.

## Ground truth (verified by Claude at `origin/main` b7fdd6ef5)

### 1. Create water: act() capitalizes; Go doesn't (DP-1488)

- **C:** `act("$p is filled.", FALSE, ch, obj, 0, TO_CHAR)` (`src/spells.c:112`). `act` sends the finished line through `CAP(lbuf)` (`src/comm.c:2477`), which uppercases the first character. So "a golden chalice" becomes "A golden chalice is filled.".
- **Go:** `sendToCaster(ch, fmt.Sprintf("%s is filled.\r\n", desc.GetShortDesc()))` (`pkg/spells/affect_spells.go:2390`). That's a hand-built act() with no CAP.
- **Scenario:** `spellmatrix-world-wand-create-water-depth`, pinned to DP-1488.

**The class (R5b), which is the real job here.** Any Go site that hand-builds an act() line starting with a substituted name (`$n`, `$N`, `$p`, `$P`) instead of calling the shared act renderer drops the capital whenever the substituted text starts lowercase, as most object short descriptions and mob names do. Find every such site in `pkg/spells` and `pkg/game`:
- grep for `fmt.Sprintf("%s ` near a comment citing a C `act(`;
- grep for `GetShortDesc()` or `GetName()` used as the first verb of a send.

Write them to `docs/fidelity/evidence/dp-1488/act-cap-sites.tsv` with columns `Go site`, `C act() site`, `can the first char be lowercase? (cite the data)`, `fixed?`. Route each one through the existing act renderer where you can, rather than adding `strings.ToUpper` patches. If there are more than about 15, fix those in `pkg/spells` and list the rest.

### 2. Perception affect slot (DP-1490, item 1)

- **C:** `af[0].location = APPLY_SPELL` for greater perception (`src/magic.c:1344`) and lesser perception (`:1354`), with modifier 0.
- **Go:** stores `ApplyNone`.

**Find out if a player can see it.** Write a scenario that casts each spell, then sends `stat <char>`, `affects` and `score`. If C's bytes differ, fix it. If they don't, record the scenario as the proof that it's invisible.

### 3. Wand and staff identify spacing (DP-1490, item 2)

- **C:** `sprintf(buf, "This %s casts: ", ...)`, then `sprintf(buf, "%s %s\r\n", buf, spells[...])` (`src/spells.c:533-534`). That's a self-overlapping `sprintf`: undefined behavior, so **the reference oracle's bytes decide (R1a)**, not a reading of the source.
- **Go:** prints one space for wand and staff (`pkg/spells/affect_spells.go:2491-2493`). Its scroll and potion lines print two (`pkg/spells/identify_output_test.go:41,48`). Check those against the oracle too.
- **Scenario:** identify a wand, a staff, a scroll and a potion, and match all four to C byte for byte. The census normalizer may fold whitespace: check `cmd/dp-oracle-diff` and confirm a one-space difference would actually show up. If it wouldn't, add a byte-exact assertion and say how (R5h).

## Do

Fix, test with a revert proof, and get a scenario green with a C witness line for each item. Remove the create-water pin.

**Census:** the create-water scenario, your new scenarios, and every scenario that reaches a site you changed (grep). Name the set.

## Budget

If you're running low, do item 1 for create water plus the class table (unfixed rows are fine), and report.

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
