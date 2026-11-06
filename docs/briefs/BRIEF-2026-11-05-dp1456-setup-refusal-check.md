# Brief (glm-5.3): DP-1456, make a refused setup fail its scenario, then re-prove the hollow rows

**Owner:** glm-5.3. **Written by:** Claude, 2026-11-05, for Zach to deliver.
**Linear:** DP-1456. Read the issue; this brief adds a root cause and pre-made decisions.
**Tier:** harness and manifests, which is stop tier. Claude reviews in Zach's session and Zach merges. Don't merge.
**Rules:** R5h, R5c, R5b, R5e, R5g. Read `AGENTS.md` and `docs/fidelity/RULEBOOK.md` first.

## Why

An oracle scenario passes when the C and Go transcripts agree. If a setup step is refused on both servers, the two refusals agree, and the scenario passes without ever reaching what it claims to test. This has now happened twice: peer logins in September (RO-012), then the spell-item matrix in October.

In October, 52 `spellmatrix-*` scenarios shipped in v0.1.8 as `oracle-green-multiseed` proof for 269 manifest rows, and none of their spells ever fired. `transfer-depth` went hollow the same way on 2026-10-16. Fixing scenarios one at a time doesn't stop a third time. A harness check does.

## Ground truth (checked by Claude, 2026-11-05, `origin/main` `c85ab41be`)

**The root cause of the spell-matrix cluster is warmup ordering, not OLC access.**

- With `empty-players`, the first character created is the God (level 40). In these scenarios that's the actor, for example `ItemMage`.
- Early in `[warmup]`, the actor demotes itself: `set player ItemMage level 35`.
- C's `set level` field needs `LVL_GRGOD`, which is 38 (`src/act.wizard.c:2577`; `src/structs.h:612`). So every later `set player … level` line is refused on both servers.
- That includes `set player ItemBuilder level 35`, so the builder stays at level 1. `oedit` needs `LVL_BUILDER` (= `LVL_IMMORT`, 31: `src/olc.h:54`, `src/interpreter.c:592`), so `oedit 4399` then gets `Huh?!?` (`src/interpreter.c:916`).
- The actor ends warmup at level 30, so `load obj 4399` in the probe is also below `LVL_IMMORT` (`src/interpreter.c:544`).
- See `cmd/dp-oracle-diff/scenarios/spellmatrix-damage-potion-group-04-depth.txt` for the full pattern. Verify it yourself with `--show-oracle` before you change anything (R5e).

**The class is bigger than 52.** This static scan finds **86** scenarios where the actor demotes itself below 38 and a later warmup line sets a level:

```python
import glob, re
for f in sorted(glob.glob('cmd/dp-oracle-diff/scenarios/*.txt')):
    sec = actor = None; demoted = hit = False
    for l in open(f):
        l = l.strip(); m = re.match(r'\[(.+)\]$', l)
        if m: sec = m.group(1); continue
        if sec == 'setup:oracle' and actor is None and l: actor = l.lower()
        if sec == 'warmup':
            mm = re.match(r'set player (\S+) level (\d+)', l, re.I)
            if mm and demoted: hit = True
            if mm and mm.group(1).lower() == actor and int(mm.group(2)) < 38: demoted = True
    if hit: print(f)
```

- 85 of the 86 are `spellmatrix-*`; the other is `call-lightning-skill-message-miss-depth`.
- All 52 that the witness audit flagged are among them. The audit's list is in `~/.local/share/openresearch/files/darkpawns/research-scoping/audits-v1/witness-sweep.jsonl`, rows with `candidate_group == "C"`.
- The other 34 have the same ordering, but nobody has checked whether the refusal hollows their claim. Your check will tell.

**Other facts:**

- **Warmup runs through the probe machinery.** `RunWarmup` (`internal/oraclediff/scenario.go:874`) calls `RunAudienceProbe`, then throws the output blocks away. So warmup already supports `send:<peer>`, and the output is available to check.
- **Transfer.** `transfer-depth` hollowed at `4c9e21d26`, which added a `low` character. The issue says the actor then lacks the level for `transfer`, which needs `LVL_IMMORT+1` (`src/interpreter.c:784`). That's the witness audit's reading; confirm the mechanism yourself.
- **Eleven more weak candidates are listed in DP-1456 §3.** They lack the claimed event in C's output, for reasons that may not be a refusal.

## Decisions already made (don't reopen them; raise disagreements in the PR)

1. **What's checked.** Check every `[warmup]` step, on both engines, against the **sender's** output for that step. The sender is the actor, or the named peer for `send:<peer>`. Don't check the audience's output.
2. **What counts as a refusal.**
   - Always: `Huh?!?` (`src/interpreter.c:916`).
   - Also C's own refusal strings for the commands that actually occur in warmup lines across the corpus: `set`, `load`, `teleport`, `goto`, `transfer`, the OLC editors, `force`, `cast`, and whatever else your inventory turns up. Grep each from `src/` and cite each string with `src/file:line` in one table in the code (R5g).
   - Match whole normalized lines. Never substrings of ordinary text.
3. **The verdict.** A refusal on **either** engine fails the scenario, with the message `setup-refused: warmup step <n> "<line>" engine=<c|go>: <matched line>`.
   - It must land in `results.tsv` as **FAIL**, never INFRA or TIMEOUT, because those get rerun and could hide it. Find how `scripts/census_runner.sh` classifies a harness error, and prove which verdict the new failure gets.
   - A refusal on Go's side only is also a real finding (setup diverged), so it fails too.
4. **The opt-out.** `[fixture] allow-setup-refusal`, valid only when the scenario also has a `# setup-refusal-reason: …` comment. The parser rejects the directive without the reason. Expect almost no uses, and list every one in the PR.
5. **Configuration belongs in warmup.** When you repair a scenario, move its configuration steps out of `[probe]` and into `[warmup]`, so the check covers them. That means the OLC `send:builder oedit …` lines, `load`, and `set`. The probe keeps only what the scenario claims to test. Fix the ordering so that any self-demotion is the **last** level change in warmup.
6. **Rows go to `blocked` first.** In your first manifest commit, demote every row that cites an affected scenario from `oracle-green*` to `blocked`, with `DP-1456` in its notes. Restore a row's status only after its scenario is re-proven with a witness (step 5). A row that can't be re-proven stays `blocked`, with the reason.
7. **Scope stops at the harness and the scenarios.** If a repaired scenario turns red because the game really diverges, don't touch game code. List it under "Divergences found"; it gets its own brief.

## Steps

1. **Worktree:** `git worktree add ~/wt/glm-dp-1456 -b glm/dp-1456-setup-refusal origin/main`. Work only there.
2. **The check, with unit tests.**
   - Implement decisions 1–4.
   - In `internal/oraclediff`, add tests that use a fake connection. A warmup step answered with `Huh?!?` must produce the error. The same step answered normally must not. The opt-out without a reason must be rejected by the parser.
   - Revert the check and watch those tests fail (R5h). Record that you did.
3. **Prove it red on the known-hollow set, before any repair.** Run a targeted census over the 86, plus `transfer-depth` and the 11 candidates from DP-1456 §3:
   ```bash
   scripts/census.sh start --name dp1456-check-before-repair --scenarios <comma list>
   ```
   - Expected: every one of the 52 known-hollow scenarios is FAIL with `setup-refused`.
   - Report the other 34 and the 12 others as they come out.
   - This run is the revert proof for the repairs: it shows what the unrepaired scenarios do under the check.
4. **Blast radius.** The check changes every scenario, so the smallest census that can fail on it is the full one. Run one full census (not `--release`) with the check on, before you repair anything outside the 86:
   ```bash
   scripts/census.sh start --name dp1456-check-full
   ```
   - Every new FAIL is either a hollow scenario (repair it, or block its rows) or a legitimate refusal (opt-out, with a reason).
   - Put them all in `docs/fidelity/evidence/dp-1456/refusals.tsv` with columns `scenario	step	line	engine	matched	disposition`.
5. **Repair and witness.**
   - Fix each affected scenario following decision 5.
   - Run each repaired scenario with `--show-oracle` and confirm that C's output shows the claimed event: the spell's message, the transfer arrival, and so on.
   - Record one row per scenario in `docs/fidelity/evidence/dp-1456/witness.tsv` with columns `scenario	claimed_event	c_witness_line	seeds`. The witness line is copied from the C transcript, not written from memory.
   - A spell's message comes from `lib/misc/messages` or `src/magic.c`, so cite where you found it.
6. **Re-prove the rows.**
   - Rerun the targeted census from step 3 on your repaired branch, with the seeds each row claims (multiseed rows need their seeds).
   - Restore each row's status only where its scenario is CLEAN and has a witness row.
   - Then run `make fidelity-depth` and `python3 scripts/gen_fidelity_depth.py`, and report the new blocked count.
7. **Gates.** Run `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `make deadcode-ratchet` and `make fmt` (gofumpt), each on its own with its exit status checked. Never chain them with `;`.
8. **PR.** Push the branch and open a PR. Retain every census run directory and cite each one in the PR.

## Report (PR description), then stop

- **Mechanism:** what you confirmed or corrected in the root cause above, and the same for `transfer-depth`.
- **Refusal strings:** the table, with its C cites.
- **Census runs:** each run directory, and its verdict counts.
- **Counts:** scenarios that failed the check before repair; how many were repaired, opted out, or blocked; how many rows were restored and how many stay blocked.
- **Divergences found:** real red after repair. List them; don't fix them.
- **Proposed rule text, if you think one is warranted:** the exact text, with this incident as its "Taught by" line. Don't edit the rulebook; Zach approves rule text.
- **Decisions for Zach:** anything you had to decide that this brief didn't settle.

Then stop and report. Claude reviews in Zach's session.

## Hard rules

- Never edit `src/` or `darkpawns-c-oracle/`. Never edit `docs/fidelity/RULEBOOK.md`, `docs/fidelity/north-star-1to1.md` or `AGENTS.md`.
- Never switch branches in `/home/zach/darkpawns`, and never `git stash`.
- Run censuses only through `scripts/census.sh`. Don't poll them. Call `wait --run <dir>`, or end your turn with the run's status. Cancel only your own runs.
- Never stop processes by name. Stop only the PIDs you started.
- Don't invoke another agent, model or paid service (`claude`, `codex`, ultrareview, other CLIs or APIs).
- Never invent a commit, line, string or count. If you can't cite something, put it in open questions.
