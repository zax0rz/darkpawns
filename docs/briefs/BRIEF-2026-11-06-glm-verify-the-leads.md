# Brief for glm-5.3: verify four leads, fix the ones that are confirmed

2026-11-06, by Claude for Zach to hand to GLM. Four suspected divergences came out of the port-completion survey (`docs/research/port-completion-survey-2026-11-05.md` §2) and the test audits. Each is found statically and is unproven. Your job:

1. prove or refute each one;
2. fix the three small ones if they're confirmed;
3. stop before fixing the fourth (PF-105), which needs a design decision.

Read first: `AGENTS.md` and `docs/fidelity/RULEBOOK.md`, from `origin/main`. Note R5h (a green must be able to fail) and R5j (a refused setup step is never neutral; your #2280 rule, landed 2026-11-06).

Branch `glm/verify-leads`, in worktree `~/wt/glm-verify-leads`, cut from `origin/main`. One PR, with one commit series per lead.

## Ground truth (grepped by Claude at `origin/main`, 2026-11-06)

### Lead 1: `give <obj> me` (DP-1457) — confirmed statically, fix it

**C:**
- `give_find_vict` finds the target with `get_char_room_vis`, in `src/act.item.c`. Find the exact line yourself.
- `get_char_room_vis` returns the caller for "me" or "self" (`src/handler.c:1276-1300`). So C replies "What's the point of that?".

**Go:**
- `doGive` (`pkg/game/item_transfer.go`) calls `FindMobInRoom` (`:543`, `:603`), then `FindPlayerInRoom` (`:503`). Those helpers are in `pkg/game/item_helpers.go`.
- The helpers ignore "me" and "self", skip the visibility check, don't handle `2.guard`, and search mobs before players instead of in room order.
- The faithful resolver already exists: `World.ResolveCharInRoom` (`pkg/game/target.go:71`).

**Do this:**
- Route `give`'s target lookup through `ResolveCharInRoom`.
- Then audit the class (R5b, R5c): list every in-room character lookup in `pkg/game`, `pkg/command` and `pkg/session` that doesn't go through `ResolveCharInRoom`, with the C function each one ports.
- Fix the sites whose C counterpart is `get_char_room_vis` itself.
  - **If that's more than 10 sites, fix `give`, list the rest, and stop.** Claude splits the remainder.
  - Leave alone any site whose C uses a different finder (`get_char_vis`, `get_player_vis`, and so on); just list it.

**Prove it:**
- An oracle scenario where `give <obj> me` returns "What's the point of that?".
- A `2.<name>` case, and an invisible-target case.
- Revert the fix, and the scenario must go red (R5h).

### Lead 2: the `combat.AttitudeLoot` duplicate

**C:** `damage()`'s kill tail calls `attitude_loot` when the killer is `MOB_AGGR24`, `MOB_LOOTS`, **or vnum 19650** (`src/fight.c:1690-1697`; the function is at `src/fight.c:1108`).

**Go has two copies:**
- **The faithful one:** `World.attitudeLootAfterKill` (`pkg/game/attitude_loot.go:60`). `death.go` calls it at `:378`, `:390` and `:410`.
- **The duplicate:** `combat.AttitudeLoot` (`pkg/combat/fight_core.go:1087`), called at `:632` only when `onDeath == nil`. It has no vnum-19650 arm, and it loots by issuing `get all corpse of <name>`.
- This is the same shape as the `counter_procs` duplicate that #2277 removed.

**Do this:**
- Decide whether any production kill reaches `takeDamage` with `onDeath == nil` and a looting killer. Trace every caller of `TakeDamage`, `TakeDamageWithDeath` and `TakeDamageAfterGate`.
- **If it's reachable:**
  - prove it with a scenario: a `MOB_LOOTS` mob kills something by the path in question, and the loot bytes differ;
  - route that path through the game layer's `attitudeLootAfterKill`;
  - delete the duplicate;
  - keep R3 draw order: compare the draw log before and after, with `DP_DRAW_LOG` if the scenario uses it.
- **If it isn't reachable:**
  - delete the duplicate and its tests, as in #2277;
  - show the grep and call-graph trace that proves it unreachable;
  - name the remaining test that covers the faithful copy.

### Lead 3: the strength-zero override

**C:** in the hit code, `if (GET_STR(ch) == 0) dam = 1;` runs **only in the normal weapon arm** (`src/fight.c:1880-1885`). The `SKILL_DISEMBOWEL` arm beside it (`:1875-1878`) doesn't apply it.

**Go:** `CalculateDamage` (`pkg/combat/formulas.go:522`) sets `dam = 1` whenever strength is 0, for every attack type (`:552-554`).

**Strength 0 is reachable in C:** `affect_total` clamps with `MAX(0, GET_STR(ch))` (`src/handler.c:357`), so strength-draining affects can drive it to 0.

**Do this:**
- Find which Go attack types reach `CalculateDamage`, and which C arm each one corresponds to.
- **If any non-normal type reaches it, it's confirmed.** Move the check into the normal-weapon path only, matching C's placement exactly.
- **Prove it:**
  - a unit test citing the C lines, at strength 0, for one normal and one non-normal type, that fails before the fix;
  - an oracle scenario if a player can reach strength 0 by a drivable path. If not, say why, and the unit test is the proof.

### Lead 4: PF-105, the merged wait counters — verify only, don't fix

**C has two counters:**
- `WAIT_STATE` always writes `ch->wait`. Its `else` branch is dead, because `ch` is never null (`src/utils.h:462-464`).
- `GET_MOB_WAIT` is written only by the Lua bridge (`src/scripts.c:2017`).
- Mob flee checks only `GET_MOB_WAIT` (`src/act.offensive.c:377`), and so does mob casting (`src/spell_parser.c:845`).
- The violence pass honours `GET_MOB_WAIT` and decrements it (`src/fight.c:1977-1982`).

**Go has one field,** `MobInstance.WaitState`, that stores both:
- player skills write it on their mob target (`pkg/command/skill_commands.go:1792-1794`);
- flee reads it (`pkg/game/flee.go:62`), and so does mob casting (`pkg/game/world_bridge.go:1193`).

**The suspicion:**
- a mob a player has just bashed can't flee or cast in Go, but can in C;
- Go's combat engine never honours a script-set mob wait;
- `TestProcessCombatPair_MobWithWaitStillAttacks` (`pkg/combat/engine_test.go:1115`) pins one side of that.

**Do this:** write the scenarios and don't fix anything.
- Scenario 1: a player bashes a wimpy mob, and the mob tries to flee within the bash's wait.
- Scenario 2: a mob caster is bashed and its spec tries to cast.
- Scenario 3: a Lua script sets a mob's `timer`, and the mob then fights.
- A known oracle blind spot applies: `DP_CLOCK` can't gate commands *you* inject under an active wait state (DP-1202). These are mob actions, so they should be drivable. If one isn't, say why.

**Report** for each scenario:
- red or green;
- its run directory;
- the C witness;
- a proposed field split: which Go reads and writes go to which counter, and every call site you'd change.

**Don't commit red scenarios.** A scenario that diverges goes in the PR only as `blocked`, with its run retained, and Claude files the issue. Then it can cite `divergence: DP-nnnn` and be pinned.

## Rules for every lead

- **Census:** targeted, through `scripts/census.sh`, for each new or changed scenario, plus the existing scenarios for the area you touch:
  - give: `grep -l '^give' cmd/dp-oracle-diff/scenarios/*`;
  - loot: `grep -l -i 'loot'`;
  - every `flee-*`, `bash-*` and mob-cast scenario for Lead 4.
  - Name what you ran and why. The final full census is batched; Claude runs it.
- **C citations:** grep them from `src/` in your session (R5g). The lines above are Claude's; verify them, don't copy them.
- **Proof:** every fix has a revert proof.
- **Gates:** run each on its own and check its exit status:
  - `go build ./...`
  - `go vet ./...`
  - `go test ./...`
  - `golangci-lint run ./...`
  - `make fmt`
  - `make deadcode-ratchet`
  - `make fidelity-depth`
  - `make expected-divergences-check`
  - `make db-migrate-test`, if you change an exported `pkg/game` symbol.
- **Repository hygiene:** never switch branches in `/home/zach/darkpawns`, never `git stash`, and never stop processes by name.
- **Governing docs:** don't edit `RULEBOOK.md`, `north-star-1to1.md` or `AGENTS.md`. Propose any rule text in the PR.
- **No outside help:** don't invoke another agent, model or paid service.

## Report

Stop and report. Include:
- a verdict per lead: confirmed and fixed, confirmed and stopped, refuted, or unreachable, each with its proof;
- the class list for Lead 1;
- the Lead 4 field-split proposal;
- every census run directory;
- "Decisions for Zach".

Claude reviews in Zach's session.
