# Brief for glm-5.3: DP-1473, split the mob wait counters

2026-11-06, by Claude for Zach to hand out. **Zach approved your proposed split** from PR #2300. Implement it, and ship your three PF-105 scenarios as the green proof instead of pinning them red.

Read first: `AGENTS.md` and `docs/fidelity/RULEBOOK.md`, from `origin/main`, noting R3 (draw parity), R5b/R5c and R5h. **Start after PR #2300 merges**, because its blocked rows (`flee.bashed-mob-can-flee`, `cast.bashed-mob-can-cast`, `lua.mob-wait-zeroes-attacks`) are this brief's targets. Work in `~/wt/glm-dp-1473`, on branch `glm/dp-1473-mob-wait-split` cut from `origin/main` once #2300 is in.

## Ground truth (your #2300 findings, checked by Claude)

**C has two counters:**
- `WAIT_STATE` always writes `ch->wait`, because its `else` arm is dead (`src/utils.h:462-464`). Only descriptors decrement it (`comm.c`), so for a mob nothing reads or decrements it.
- `GET_MOB_WAIT` is written only by the Lua bridge's `timer` (`src/scripts.c:2017`).
- `GET_MOB_WAIT` gates:
  - mob flee (`src/act.offensive.c:377-381`);
  - mob casting (`src/spell_parser.c:844-848`);
  - the violence pass, which decrements it and zeroes the mob's attacks (`src/fight.c:1977-1981`);
  - the NPC scramble to its feet (`:1982`).

**Go has one field,** `MobInstance.WaitState` (`pkg/game/mob.go:70`, commented "C GET_MOB_WAIT"), with `GetWaitState`, `SetWaitState` and `DecrementWaitState` (`mob.go:1001-1023`). Both counters write it, and the gates above read it.

**Your three vehicles,** red in `~/Archives/darkpawns/oracle-runs/2026-11-06/glm-verify-leads-targeted`. Their texts are frozen at `…/glm-verify-leads-targeted/source/cmd/dp-oracle-diff/scenarios/pf105-*.txt`:
- `pf105-bash-mob-flee-depth`
- `pf105-bash-mob-cast-depth`
- `pf105-mob-timer-fight-depth`

## Do this

1. **Split the field.**
   - `WaitState` becomes C's `ch->wait`. Add `MobWait` as `GET_MOB_WAIT`, with its own getter and setter.
   - Classify **every** caller of the mob's `GetWaitState`, `SetWaitState` and `DecrementWaitState`, and every interface that reaches them (`pkg/scripting/types.go:96` documents `GetWaitState` as `GET_MOB_WAIT`, for one). Each caller is either `ch->wait` or `GET_MOB_WAIT`, by its C counterpart.
   - Put the table in `docs/fidelity/evidence/dp-1473/callsites.tsv`, with columns `site`, `c_counterpart`, `counter` and `change`.
   - The ones you named in #2300:
     - **`ch->wait`:** player-skill target waits (`pkg/command/skill_commands.go:1792-1794`).
     - **`GET_MOB_WAIT`:**
       - the Lua timer (`pkg/game/world_bridge.go:251`, and the readback at `:136`);
       - flee (`pkg/game/flee.go:62`);
       - the cast gate (`world_bridge.go:1193`);
       - the engine's NPC arm in `processCombatPair`, which honours and decrements it each `PULSE_VIOLENCE` and handles the stand-up block.
   - There are more callers across `pkg/game`, `pkg/session` and `pkg/combat`. Classify all of them, not just these.
2. **Keep C's quirks:**
   - A mob's `ch->wait` is never decremented.
   - Lua `me.*` writebacks from mob `oncmd` scripts don't land (`src/scripts.c:1805-1813`). Don't "fix" that.
3. **Draw parity (R3):** zeroing attacks removes draws. Run every vehicle, plus every `flee-*`, `bash-*` and mob-cast scenario, with `DP_DRAW_LOG`, and compare C's and Go's draw streams. Any mismatch is a FAIL, even when the transcripts match.
4. **Tests:**
   - Rewrite `TestProcessCombatPair_MobWithWaitStillAttacks` (`pkg/combat/engine_test.go:1115`) to C's behaviour: a script timer zeroes attacks, and a skill wait doesn't.
   - Add a unit test for each counter's reader.
5. **Ship the three vehicles** as green proofs.
   - Restore their manifest rows from `blocked` at the depth and seeds you prove.
   - Each needs a C witness line (R5j).
   - Revert the split, and all three must go red (R5h).
6. **Census:** targeted, through `scripts/census.sh`: the three vehicles at their claimed seeds, every `flee-*`, `bash-*`, mob-cast and `lua-*` scenario, and every scenario whose mob uses a Lua `timer` (`grep -l timer lib/world/scripts` for the scripts, then the scenarios that load them). Name what you ran and why.

## Rules

- **Gates:** run each on its own and check its exit status:
  - `go build ./...`
  - `go vet ./...`
  - `go test ./...`
  - `golangci-lint run ./...`
  - `make fmt`
  - `make deadcode-ratchet`
  - `make fidelity-depth`
  - `make expected-divergences-check`
  - `make db-migrate-test`, since this changes exported `pkg/game` symbols.
- **C citations:** cite C from `src/`, grepped in your session (R5g).
- **Repository hygiene:** never switch branches in `/home/zach/darkpawns`, never `git stash`, and never stop processes by name.
- **Governing docs:** don't edit `RULEBOOK.md`, `north-star-1to1.md` or `AGENTS.md`. Your stand-in rule proposal from #2300 is with Zach; don't repeat it here.
- **No outside help:** don't invoke another agent, model or paid service.

## Report

Open a PR, ending its description with "Decisions for Zach". Then stop and report:
- the call-site table summary;
- the three vehicles' results and revert proof;
- the draw-parity result;
- the census run directories;
- anything that turned red that wasn't one of the three.

Claude reviews in Zach's session.
