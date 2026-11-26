# Round 4 classification corrections

- **Row 64 — frozen `ask-zach`; corrected to `c-reachable` after the census.**
  The all-mutants census finished at 18:43 on 2026-11-25; the ruling reached
  `main` at 19:18 (`a48baa25c`). The frozen `sample.tsv` keeps `ask-zach`. Zach's 2026-11-25 ruling in
  `docs/research/mutation-loop/DECISIONS.md` says overlapping C
  `sprintf(buf, "%s...", buf)` appends are classified by the pinned reference
  build's output. C `do_auto` appends this exact `Exits ` fragment at
  `src/act.other.c:1308-1312`; the CLEAN combined run
  `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/nightly-20261125`
  records it in `census-dump/player-state-toggles.txt`. The one-byte mutation
  and full-run commit are unchanged. The row was already in the denominator as
  `ask-zach`, so this correction did not change the denominator.
- **Row 92 — `c-reachable` affirmed under the NULL `%s` ruling.** C
  `do_clan_plan` formats the old-plan response at `src/clan.c:1427-1432`.
  Zach's 2026-11-25 ruling establishes that a NULL `%s` in the reference
  prints `(null)`; no class or denominator change is needed for this row.

- **Row 0 — C-site attribution corrected; class remains `c-reachable`.** The
  cited line at `src/spell_parser.c:868` is in `cast_spell`, not `do_cast`.
  The `lua-cast-spell-gates-depth` fixture reaches the default-position branch
  with a stunned peer, so the pinned C path can print the sampled bytes. The
  Go sample is in the session command gate; its isolated check is recorded in
  `probes.tsv`.
- **Row 27 — post-census call-path correction: C-unreachable at the sampled
  site; the frozen class remains `c-reachable`.** `src/interpreter.c:374` sets
  carve's minimum position to `POS_STANDING`; `command_interpreter` returns
  for a fighting character at `src/interpreter.c:939-948`, before C `do_carve` can reach the literal at
  `src/new_cmds.c:124-127`. `src/new_cmds.c` has no other `do_carve` caller.
  In Go, the production `CmdCarve` registration is
  `pkg/session/commands.go:560`, and `executeCommandRaw` applies the checked-in
  C position gate before the handler (`pkg/session/commands.go:1138-1140;
  pkg/session/command_gates.tsv:58`). Switched-body commands and Lua
  `command()` calls on a mobile use `NPCCommand` and
  `dispatchCharacterCommand`'s position gate (`pkg/game/world_bridge.go:356-365`;
  `pkg/game/npc_command.go:20-26`; `pkg/session/manager.go:1137-1140`;
  `pkg/session/character_dispatch.go:149-167`). Their `carve` actor handler
  calls `doCarveActor`, not the sampled `CmdCarve` line. Lua `command()` calls
  on a player use `CommandExecFunc` and `ExecuteCommand`, which reaches the same
  regular command gate (`pkg/game/world_bridge.go:356-365`;
  `pkg/session/manager.go:1123-1135`; `pkg/session/commands.go:1138-1140`).
  A repository-wide caller search found no production direct caller of
  `CmdCarve` outside its registration. So no Go route reaches the sampled line
  without the gate.
  The depth manifest records the C gate proof in `docs/fidelity/depth/carve.tsv`
  (`carve.fighting-no-arg`). This correction does not revise the pre-census
  class or denominator; the row remains a non-kill in the frozen measurement.
- **Row 35 — post-census C-path correction: C-unreachable in the shipped
  world.** C `do_pour` emits the sampled error only if
  `read_object(20, VIRTUAL)` fails (`src/act.item.c:1249-1253`). The shipped
  world defines object vnum 20 as a puddle (`lib/world/obj/0.obj:126-132`).
  Go's `characterMobPourOut` creates vnum 20 at
  `pkg/game/npc_command_utility.go:196-200`; the sampled error is its nil
  creation branch, which is not reached when the shipped prototype is loaded.
  This post-census correction does not revise the frozen sample or denominator;
  the row remains a non-kill in the recorded measurement.
- **Row 80 — post-census call-path correction: C-unreachable at the sampled
  site; the frozen class remains `c-reachable`.** C `do_clan_set` dispatches
  `clan set applev` to `do_clan_application` at `src/clan.c:1533-1535`. The handler first returns
  through `send_clan_format` for an empty argument (`src/clan.c:1273-1283`).
  For immortals it then resolves the second word as an exact clan name;
  `find_clan` compares the supplied name with `str_cmp`
  (`src/clan.c:799-806`), so an empty name returns "Unknown clan." before the
  sampled `Set to which level?` branch (`src/clan.c:1308-1310`). A valid
  immortal call retains the first word as the level argument, so it does not
  reach that empty-value branch. Go follows the same barriers:
  the production clan command calls `doClanSet` at
  `pkg/game/clan_command.go:117`, which dispatches `applev` to
  `doClanAppLevel` at `pkg/game/clan_economy.go:126-128`. That function returns
  on empty input at `pkg/game/clan_economy.go:58-61`;
  `resolveClanForImmortal` rejects an empty immortal clan name at
  `pkg/game/clans.go:459-464`; after a successful lookup, Go retains the first
  word as the level argument (`pkg/game/clan_economy.go:63-78`). The checked
  path therefore cannot reach the sampled prompt. This correction does not
  revise the pre-census class or denominator; the row remains a non-kill in
  the frozen measurement. It also corrects the original evidence's function
  name: `do_clan_application`, not `do_clan_sp`.

Rows 0, 27, 35 and 80 have post-census evidence corrections. They do not revise
any pre-census class or denominator. The `sample.tsv` checksum in `SHA256SUMS`
remains the frozen draw checksum.
