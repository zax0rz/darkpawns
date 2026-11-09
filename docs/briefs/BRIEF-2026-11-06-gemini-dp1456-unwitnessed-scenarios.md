# Brief for Gemini: 22 scenarios whose C transcript never shows the event they claim

Linear: DP-1456 (follow-up). Rulebook: R1 (bytes), R5e (verify the call path), R5g (cite C from `src/`, grepped in session), R5h (a green must be able to fail), R5i (a fixture that silences a surface), R5j (refused setup steps are never neutral).

## Orientation (read first)

Dark Pawns is a 1:1 port of a 1990s C MUD (`src/`, read-only, the ground truth) to Go (`pkg/`). The **oracle harness** (`cmd/dp-oracle-diff`) runs each scenario file (`cmd/dp-oracle-diff/scenarios/*.txt`) against both servers and diffs the bytes players receive. A scenario that passes is only proof if **C's own transcript shows the claimed behavior happening**: that's the C witness (R5j). Otherwise both servers can agree on nothing at all, a refused command or an empty room, and still pass.

Read, in order:
1. `AGENTS.md`: the gates, worktrees, and census rules. Follow them exactly.
2. `docs/fidelity/RULEBOOK.md`, especially R5h, R5i and R5j.
3. `docs/fidelity/DEPTH_TESTING.md`: the scenario format and the manifest TSVs.
4. `docs/fidelity/evidence/dp-1456/`: the audit this brief continues.

Work in a worktree: `git worktree add ~/wt/gemini-dp-1456-witness -b gemini/dp-1456-unwitnessed-scenarios origin/main`. Never switch branches in `/home/zach/darkpawns`, never `git stash`, and never stop processes by name. Five other agents are working in `pkg/` right now. **This brief changes no game code**: only scenario files, the harness's fixture code if needed, and manifest rows.

## Ground truth (verified by Claude at `origin/main` 696f2d63c)

`docs/fidelity/evidence/dp-1456/blocked-no-witness.tsv` lists 22 scenarios: `set-name-cleanup`, `spellmatrix-scroll-olc-02-depth`, and 20 `spellmatrix-world-{wand,staff,scroll}-*-depth`. Each passes after DP-1456's repair, but the **C transcript lacks the claimed event: the item never reaches the actor**. 21 of them are `blocked` in the manifests.

They share one fixture shape. Example: `spellmatrix-world-wand-fireball-depth.txt`:
```
[fixture]
no-settle
spawn-obj 20014 99 8162 200
force-load 20014
...
[warmup]
...
get book
wear book
[probe]
equipment
use book ItemPeer
```
Object 20014 is "mage spells book volume" (`lib/world/obj/200.obj:149-151`). `force-load` rewrites the prototype's load percent so the zone reset can't skip it (`cmd/dp-oracle-diff/main.go`, `applyForceLoadFixtures`). Despite that, C's `get book` evidently finds nothing.

**Why that's unknown, and is your job:** possible causes include `spawn-obj` placing the object after the warmup ran, `no-settle` skipping the reset pulse that loads it, the room or count arguments, or `get` refusing for a reason C prints. Find it from the retained C transcripts. Re-run one scenario and read C's output line by line from `get book` onward.

## Do

1. **Diagnose the class (R5e).** Pick one scenario. Run it: `go run ./cmd/dp-oracle-diff` per `DEPTH_TESTING.md`, or a one-scenario census. Find exactly why the item never reaches the actor in C, citing the harness code (`cmd/dp-oracle-diff/*.go`) and any C involved (`src/`, grepped). Write it up in `docs/fidelity/evidence/dp-1456/no-witness-diagnosis.md`: one page, cause and evidence.
2. **Repair the fixture**, preferably once for the class, not 22 times. Change the scenario files, or the harness fixture if it's wrong. If you change harness code, keep every other scenario's behavior identical, and say how you checked that.
3. **For each of the 22, after the repair:**
   - **C now shows the event and Go matches:** set the manifest rows to their claimed green status, and record the C witness line in `docs/fidelity/evidence/dp-1456/witness.tsv`.
   - **C shows the event and Go differs:** that's a real divergence. Don't pin it, opt it out or edit it to pass. List it with its first divergent block, for Claude to file.
   - **Still no witness:** say why.

   Write `docs/fidelity/evidence/dp-1456/no-witness-resolution.tsv` with columns `scenario`, `outcome (green/divergent/still-no-witness)`, `C witness line or first divergent block`, `run dir`.

## Proof

1. **R5h for the harness:** show that before your repair C's transcript lacked the event, and after it, C shows it. Quote both lines for one scenario.
2. **A targeted census** of the 22: `scripts/census.sh start --name gemini-dp1456-witness --scenarios …`, then `scripts/census.sh wait --run <dir>`. Don't poll it in a loop, and don't `cat` or `tail` a running `census.log`. If you changed harness code, also run every other scenario that uses `spawn-obj`, `force-load` or `no-settle` (grep for them).

## Budget

Your usage is limited. If you're running low, stop after step 1 plus repairing and proving **three** scenarios, and report. The diagnosis alone is valuable.

## Rules

- Cite C only from `src/`, grepped in this session, never from memory or from `darkpawns-c-oracle/`.
- Don't edit `docs/fidelity/RULEBOOK.md`, `docs/fidelity/north-star-1to1.md` or `AGENTS.md`.
- Never invoke another model, agent or paid service.
- Run every gate, each on its own, and check its exit status; never chain them with `;`: `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `make fmt` (gofumpt, not gofmt), `make deadcode-ratchet`, `make fidelity-depth`, `make expected-divergences-check`. CI `test` must be green before you report.

## Done means

A PR against `main`, **with `DP-1456` in its title**, containing the diagnosis, the repair, both TSVs, the census directory, and a **Decisions for Zach** section. Then stop and report. Claude reviews in Zach's session, and Zach merges.
