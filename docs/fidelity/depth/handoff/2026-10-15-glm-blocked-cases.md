# 2026-10-15: the nine blocked depth cases (GLM train, DP-1371 finish line)

Worktree `~/wt/glm-blocked-cases` (branch `glm/blocked-cases`), base `f1bf62126`.
Brief: `~/GLM-BRIEF-2026-10-15-blocked-cases.md`. This handoff covers Family A
(the NPC command interpreter, four blocked rows), the two R3 draw extras
(DP-1436/DP-1437), the mudlog and zreset aggregate closures, and the decision
memo for the two equipment rows.

## 1. Family A design: how a mobile reaches the real `do_*` handlers

The port already had the faithful `command_interpreter` skeleton:
`pkg/session/character_dispatch.go` (`dispatchCharacterCommand`) ports the
leading `number(0,3)` hide draw (`interpreter.c:889`), the ordered-table
prefix resolution with the level filter, "Huh?!?", the frozen/NPC-immortal/
position gate ladder in C's order (`interpreter.c:915-946`), the `special()`
ladder (`interpreter.c:1393-1479`), and then hands off to the command's
registered `ActorHandler` (`pkg/command` registry) — the session-independent
`do_*` body. A mobile enters that dispatcher through three production lanes,
all of which now converge on it:

| C call site | Go lane | State before |
|---|---|---|
| `act.wizard.c:1880,1892,1905` (do_force) | `Manager.executeCharacterCommand` (wiz_communication.go) | already faithful |
| `act.offensive.c:332,348` (do_order) | `executeOrder` → `executeCharacterCommand` | **was** the VNum adapter |
| `scripts.c:135` (lua_action) | `WorldScriptableAdapter.Command` → `World.NPCCommand` → `CharacterCommandExecFunc` | already faithful |

What this train added:

- **`performMoveMob`** (`pkg/game/flee.go`): the port of `perform_move`
  (`act.movement.c:311-358`) for a mobile mover — the exit refusals
  ("Alas...", the three closed-door branches), the `do_simple_move` leg, and
  the follower drag (a player follower of the mover receives "You follow $N."
  and is moved by the recursive `perform_move`). `doFleeMoveMob` was split so
  both callers share one `doSimpleMoveMob(ch, dir, needSpecialsCheck)`;
  `do_move` passes `needSpecialsCheck=false` because the interpreter already
  offered the command to `special()` (`interpreter.c:947` — C's
  double-spec-proc guard).
- **Actor handlers registered**: the six direction rows (`do_move`), `flee`
  (`do_flee` via `DoFlee`), and the DP-1440 channel helpers for
  `gossip`/`shout`/`auction` (`do_gen_comm` via `mobGenComm`). Together with
  the existing say/tell/emote/give/stand/sit/socials lanes these cover every
  command the live lanes exercise.
- **The parallel imitation is gone**: `executeMobCommand` (world.go, the
  hand-rolled ~180-line switch), `World.ExecMobCommand`,
  `WorldScriptableAdapter.ExecuteMobCommand` and the scripting-bridge method
  are deleted. Its invented `social <name>` prefix form (C registers no such
  command, R4) is deleted with it. `NPCCommand`'s session-less fallback keeps
  the previously ported say/tell/emote/give/stand/social behavior and now
  also carries gossip/shout/auction so unwired worlds (pkg/game tests) see
  the same sinks as the registry.
- **Order now draws**: the order lane previously skipped the interpreter's
  `number(0,3)` draw; C's `command_interpreter` consumes it for every
  ordered command. `TestNPCOrderUsesSharedExecutorAndInterpreterDraws` pins
  one draw per ordered command.

**Explicit-Unsupported is the design, not a gap**: a command whose `do_*`
body has no actor lane (get, drop, open, follow, kill, …) is reported
`CharacterCommandUnsupported` and logged — never guessed. Each such command
needs its own actor-lane port and depth row; that inventory is the remaining
long-tail of the surface, owned per-command, not by these rows.

An ordering note discovered on the way: **C's charm gate makes ordered
movement unreachable** — a charmed follower sharing the orderer's room
refuses to leave ("The thought of leaving your master makes you weep.",
`act.movement.c:121-125`), and an ordered follower is by definition charmed
and co-located. The movement proofs therefore ride the force lane
(`force-mob-move`, `force-mob-flee`).

## 2. lua.nested-writeback-slot — the reference build refutes the premise

The blocked row claimed that in a nested `run_script`, `table_to_char`'s
`lua_pushvalue(L, 1)` (`scripts.c:1979`) reads "the calling binding's first
argument", so C's write-back applies a stale table while the port writes
`ch` back. A live experiment (`lua-nested-writeback` scenario) shows the
`-O0` reference build does no such thing: a greet script fired inside
`action(me, "north")` (the outer script's `lua_call` frame live) has its
`ch.pos = 6` **applied** — the arriving mob lands sitting in the destination
room's listing — and the port matches byte-for-byte. Row flipped to
oracle-green; the premise is recorded as refuted by observation
(R1a/R5a — observation over static analysis; the vendored Lua 4.0.1 stack
layout evidently leaves the freshly pushed table at slot 1 in this build).
Sibling nested geometries (object oncmd, enter script) remain owned by their
own manifests.

While building the vehicle I also confirmed `interpreter.c:1460` gates the
mob oncmd script lane on `!IS_NPC(ch)` — a mob's own `action()` command is
never offered back to mob oncmd scripts (Go's `runSpecials` already mirrors
this).

## 3. mudlog.unported-sites (DP-1340) and zreset.reset-zone-state — closures

- **mudlog**: the two scenario-pointer residuals now name their landed unit
  proofs (`olc.save-info` → `TestOLCDispatchAndSaveInfo`;
  `game-spike-stake` → `TestGameDiagnosticSpikeStake`). The aggregate row is
  `delegated` to the 133-case inventory per
  `handoff/2026-10-12-dp-1371-mudlog-closure.md` (which this implements).
- **zreset**: every `reset_zone` command case now maps to a green sub-row
  (M/O/P/G/E/R/D/L, if_flag, comments, unknown, zone clock/age/occupancy,
  mobile-equipment effects/readers/removal). The aggregate is `delegated`
  to those rows; the named residuals (R's broader Lua extraction, the
  specials NPC lanes) live in their own rows.

## 4. DP-1436 / DP-1437 — palm and peek entry draws

- `do_palm` draws `number(1,101)` before `one_argument` and every gate
  (`new_cmds.c:2110-2122`): the no-argument and item/weight refusals now
  burn the draw (`TestPalmEntryDrawPrecedesEveryGate`); `CmdPalm` no longer
  returns before `DoPalm`.
- `do_peek` consumes `number(1,101)` before its immortal test
  (`act.other.c:1688-1692`): immortals burn the draw too
  (`TestPeekImmortalConsumesPercentDraw`). Go previously short-circuited
  `ch.GetLevel() < LVL_IMMORT &&` before drawing.
- Census `glm-dp-1436-1437-palm-peek-draws`: 6 scenarios (mortal-obj-batch,
  peek-depth, peek-success-depth, peek-higher-immortal-depth,
  peek-kender-depth, spec-proc-no-get-palm) CLEAN, `draw_divergent=0`.

## 5. Equipment rows — decision memo for Zach (not implemented)

Both rows sit at the persistence/session-lifecycle stop tier; per the brief
("If the faithful fix touches load or save, stop and write it up") these are
recorded, not fixed:

1. **`equipment.saved-zero-stats-lethal-nowhere-reads`** — C reads
   `world[-1].name` and `ROOM_FLAGGED(world[-1], ROOM_NEUTRAL)` at NOWHERE
   (`fight.c:1673-1680,1493`); both reads are undefined and the port already
   made Zach's delegated choice (empty room-name segment; the neutral-room
   rescue never fires, so the death branch always runs). The row is blocked
   only on ratifying that delegated decision (R1a). Nothing further is
   implementable without changing the decision.
2. **`equipment.saved-zero-stats-dex-suffering-message`** — C delivers the
   set-399 `skill_message` suffering line to the entering character's own
   descriptor at every DEX firing (`fight.c:1534-1535`; messages 868-882),
   before the character is registered. Go's combat delivery
   (`Manager.combatSession` → `attachedBodyLocked`) drops bytes for a
   `charCreating` session (switch_ownership.go:26). Delivering combat output
   to a creation-state session is a session-lifecycle design decision: the
   options are (a) a pre-registration delivery sink for TYPE_SUFFERING at
   entry only, or (b) a `divergent-approved` row with the dropped bytes
   documented. Neither should land without Zach's choice.

## 6. Evidence and controls

- Census `~/Archives/darkpawns/oracle-runs/2026-10-15/glm-family-a-npc-interpreter`:
  16 scenarios (force-mob, force-mob-move, force-mob-flee, lua-bind-actions
  with the new `actiongossip` branch, lua-nested-writeback, order ×3,
  flee ×8) CLEAN, `draw_divergent=0`, dirty tree at `f1bf62126` (this
  branch's changes).
- Census `~/Archives/darkpawns/oracle-runs/2026-10-15/glm-dp-1436-1437-palm-peek-draws`:
  CLEAN, `draw_divergent=0`.
- R5h copy-based revert controls (never `git stash`), all observed failing
  with the fix reverted and passing restored: the movement actor-handler
  registration, the order dispatcher lane, the closed-door refusal bytes,
  the palm entry draw, the peek immortal draw.
- Gates (each run on its own, exit statuses checked): `make fmt`, `go build
  ./...`, `go vet ./...`, `go test ./...`, `go test ./pkg/game/... -count=1`,
  `golangci-lint run ./...` (0 issues), `make deadcode-ratchet` (52
  baselined, 0 new), `make fidelity-depth` (5589 cases, 6 blocked — the two
  equipment rows above, the two narrowed turn/flee residuals, and the two
  login/identity rows that are Zach's), `make fidelity-units` (1678 PASS),
  `make string-census`, `gosec -severity=high -quiet ./...`,
  `go test -race ./pkg/game/...`.

## 7. Residuals handed forward

- NPC actor lanes for get/drop/open/follow/kill (each needs a faithful
  `do_*` port + row; the dispatcher already routes them once registered).
- The specials inventory offered to a moving NPC (`turn.npc-flee-special-lane`
  narrowed note): an unported special blocks the move by design; each
  special's NPC lane needs its own proof.
- Mounted-mover and greet/entry-script branches of the NPC `do_simple_move`
  lane (the unproven tail of `flee.transitive-movement-failures`).
- The player-path follower drag omits C's `get_rider(k->follower) != ch`
  check (`act.movement.c:347`): a rider following their own mount is not
  skipped. Found during the port (R5c class finding); the new mobile path
  implements the check, the player path still lacks it — one-line fix plus
  an R5h test, left for review rather than smuggled into this train.
