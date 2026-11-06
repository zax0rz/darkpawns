# Brief for Luna: close the census's blind spots, then measure two more packages

2026-11-06, by Claude for Zach to hand to Luna (extra-high effort). This is a long goal with **no time limit**. Finish both parts, or stop at a real blocker and report it.

Read first: `AGENTS.md`, `docs/fidelity/RULEBOOK.md` (R1–R5, including R5h, R5i and R5j, which landed 2026-11-06) and `docs/fidelity/DEPTH_TESTING.md`. Read them from `origin/main`, not the shared checkout, which lags.

## Why

Your round-two experiment (PF-102, `docs/research/idiom-experiment/round2/`) changed one letter in each of 40 randomly drawn player-output functions. The full census missed **17 of the 40**. In five of those misses, a scenario was named, before the run, as covering the function, and it still stayed green.

- **Part 1** makes those 17 misses either catchable or explained.
- **Part 2** runs the same measurement on `pkg/session` and `pkg/spells`, which round two didn't sample.

Part 2 goes second so it isn't measured against scenarios we already know are blind.

## Ground truth (checked by Claude at `origin/main`, 2026-11-06)

The 17 misses are the rows with `caught = no` in `docs/research/idiom-experiment/round2/run/probes.tsv`. The probes were run against `5e9556a87`. The mutation rule: replace the literal's first ASCII letter with `Q`, or with `Z` if it is already `Q`.

| i | Function | Site | Scenario named before the run |
|---:|---|---|---|
| 0 | `findDoor` | `pkg/game/act_movement.go:601` | `door-basic` |
| 4 | `CmdSerpentKick` | `pkg/command/skill_commands.go:1289` | `serpent-depth` |
| 15 | `weatherChangeLocked` | `pkg/game/weather.go:574` | `weather-sky-transitions` |
| 23 | `CmdSkills` | `pkg/command/skill_commands.go:80` | `skills-practice` |
| 37 | `executeShoot` | `pkg/command/shoot.go:56` | `shoot-mob-hit-depth` |
| 2, 3, 8, 10, 11, 16, 18, 19, 21, 27, 38, 39 | see `probes.tsv` | | none |

Line numbers are from `5e9556a87`; re-find each literal at `origin/main` before you work on it.

C sources Claude grepped for the five named probes:
- `findDoor`: `src/act.movement.c:401`.
- `CmdSerpentKick` ("Kick who?"): three C sites, `src/act.offensive.c:606`, `:655` and `src/new_cmds2.c:711`. Find which one serpent kick takes.
- `weatherChangeLocked`: `src/weather.c:215`.
- `executeShoot`: `src/act.offensive.c:947`.
- **`CmdSkills` is probably not a port at all.** No file in `src/` contains its box-drawn "Your Skills" table. In `pkg/`, nothing outside its own definition names `CmdSkills`. Confirm both, including registration by string, before you classify it.

## Part 1: each of the 17 misses gets a verdict

Branch `luna/pf102-blind-probes`, in worktree `~/wt/luna-pf102`, cut from `origin/main`.

For each probe, first decide which of these it is (R5e: verify the actual call path, don't trust a summary):

- **A. Reachable, and the scenario misses it.** A player can reach the function by a C path, but no scenario drives that branch. Fix or add a scenario, not the game. It's done when both of these hold:
  - under the same one-byte mutation, a targeted census over that scenario goes **red**;
  - unmutated, it is green, with a C witness line showing the branch fired (R5j).
- **B. Unreachable in Go.** No production caller reaches the literal; dead code the ratchet can't see is the usual case. Report it with the grep that proves it. Don't delete it.
- **C. Go-only or invented output** (R4): no C source has the bytes, or C reaches them by another path. Report it with the C you checked.
- **D. Reachable, but the harness can't drive it** (for example, it needs a mudlog-only channel, linkless timing or a ban file). Report what blocks it, and leave it unproven.

**Rules for Part 1:**
- Never change game code. If a repaired scenario exposes a real divergence (it goes red unmutated), stop work on that probe. Keep its census run, leave the scenario out of the commit or demote its manifest row to `blocked`, and report it. Claude files the issue.
- When you change an existing scenario, it must keep its own witness for what it already claimed. Run it under its original manifest seeds before and after.
- **Census:** one targeted run per repaired scenario under its mutation (it must go red) and one without (it must be clean). Use `scripts/census.sh` only, `--name luna-pf102-<i>-mut` and `luna-pf102-<i>-clean`. Never commit a mutation.
- **Evidence:** `docs/fidelity/evidence/pf-102/verdicts.tsv`, one row per probe, with these columns: `i`, `function`, `verdict` (A/B/C/D), `scenario`, `mut_run`, `clean_run`, `witness`, `c_cite`, `notes`.
- Update the manifest rows the repaired scenarios prove, at their real depth.

**Gates:** run each on its own and check its exit status:
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...`
- `make fmt`
- `make fidelity-depth`
- `make expected-divergences-check`

Then open the PR with a "Decisions for Zach" section, and **go straight on to Part 2. Don't wait for the merge.**

## Part 2: random mutation sample, `pkg/session` and `pkg/spells`

Folder: `docs/research/idiom-experiment/round3/`. Branch `luna/round3-mutation-sample`, cut from **your Part 1 branch tip**. Record that commit; it is the base for every probe.

Use round two's method unchanged (`round2/BRIEF.md` stages 1–4), with these changes:

1. **Fix the population script first.** Round two's `population.go` read the message from argument index 1 for one-argument selector calls, so it missed `DreamContext.SendToChar` and `SendToRoom` (see `round2/RESULTS.md`).
   - Fix it, and add a self-test covering one selector call and one ordinary call.
   - Then regenerate **round two's** population at `5e9556a87`. Report how many rows it gains, and leave round two's frozen sample and denominator alone.
2. **Populations:** player-output functions in `pkg/session` and in `pkg/spells`, frozen and hashed separately, one per package. Record each sha256 in `round3/PREDICTIONS.md` before drawing.
3. **Predictions before drawing:** your expected catch rate per package, with an 80% interval. Commit them, then draw.
4. **Sample:** 40 per package, with `random.Random(20261106)` for session and `random.Random(20261107)` for spells. The probe-literal seed is the package seed + `i`, as in round two.
5. **Predicted scenarios:** write them before each run, as in round two.
6. **Each probe runs a full census:** `scripts/census.sh start --name luna-r3-<pkg>-<i>`. **Run at most one at a time.** About 80 censuses share the host with other agents' targeted runs. No wall-clock limit; it will take days, which is fine.
7. **Results:**
   - `round3/RESULTS.md` holds the catch rate per package with a Wilson interval, your predictions against the results, and every miss with a first-pass A/B/C/D verdict as in Part 1. Don't repair Part 2 misses; that's a later brief.
   - Add a field note and an `EVIDENCE_LEDGER.tsv` row.

## Hard rules

- Work only in your worktrees under `~/wt/`.
- Never switch branches in `/home/zach/darkpawns`, never run `git stash`, and never stop processes by name.
- Run censuses only through `scripts/census.sh`. Cancel only your own runs. Never poll in a loop: call `wait --run`, or end your turn.
- Cite C from `src/`, grepped in your session (R5g).
- Don't edit `RULEBOOK.md`, `north-star-1to1.md` or `AGENTS.md`. If a rule is needed, propose the exact text in your PR.
- Don't invoke another agent, model or paid service (`claude`, `codex`, `ultrareview` or other CLIs and APIs).

## Report

Stop and report after each PR opens; the Part 1 report doesn't wait for Part 2. Include:

- the verdict counts (A/B/C/D) and every divergence found;
- for Part 2, the two catch rates against your predictions;
- every census run directory;
- "Decisions for Zach".

Claude reviews in Zach's session, and Zach merges.
