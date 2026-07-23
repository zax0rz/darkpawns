# BRIEF (glm) — faithful input-timing: queue mid-combat commands, kill the invented wait gate (DP-1201)

**Owner:** glm-5.2. **Gate:** Claude runs the differential oracle on the *gateable*
slices (see Oracle gate) + reviews the unit tests; CI green.
**Git:** branch off `main` as `glm/input-timing`, commit, push, open a PR. Do NOT
merge. Sized to one PR (M).
**Closes:** DP-1201. **Related:** DP-1202 (why mid-combat injection is NOT
oracle-gateable — read it), DP-1170 (same family).
**Cite:** `src/comm.c:599-637` (game_loop command-drain + `--wait` gate),
`comm.c:617` (the `wait=1` throttle), `comm.c:712` (`if (!dp_clock) heartbeat`);
rules **R1**, **R3**, **R4** (`docs/fidelity/RULEBOOK.md`).

> ⚠️ Read DP-1201 and DP-1202 in full before starting. DP-1202 explains why the
> oracle cannot byte-verify mid-combat command injection — your gate is
> `combat-death` + opener scenarios + **unit tests**, not a mid-combat diff.

## The C truth (comm.c game_loop)

Per game-loop pass, in order: read input into a per-descriptor queue → **drain
commands** → later, `perform_violence` (inside `heartbeat`). The drain
(comm.c:603) is:

```c
if (... ((d->character ? --(d->character->wait) : 0) <= 0) && get_from_q(&d->input, comm, ...)) {
    ...
    command_interpreter(d->character, comm);
}
```

Four load-bearing facts:

1. **A command issued while `wait > 0` is NOT rejected.** The `&&` short-circuits
   before `get_from_q`, so the command **stays queued** and drains later when
   `wait` reaches 0. **No message is emitted.** C *delays*; it never says
   "too busy" from the input gate. (The `"You're too busy…"` strings that DO
   exist in C are emitted by *specific handlers* like do_bash/do_rescue checking
   fight state — never by the game loop.)
2. **`wait` is in PULSES.** `WAIT_STATE(ch, PULSE_VIOLENCE * n)` sets
   `wait = PULSE_VIOLENCE * n` pulses; it decrements **once per pulse**.
3. **One command drains per pulse** (it's an `if`, not a `while`).
4. **Command drain happens BEFORE `perform_violence`** within a pulse.

## The Go bug — delete it (R4/R1)

`pkg/session/commands.go:686-706` — the "C-10 WAIT_STATE enforcement" block is a
wholesale invention on three axes:

- emits `"You're too busy!\r\n"` (line 704) — **that exact string does not exist
  in the C source**;
- a `waitBypass{look,inv,say,score,tell,who,…}` allowlist — C's game loop has
  **no such carve-out**; the wait gate applies to every command uniformly;
- it **discards** the command (`return nil`) — C queues and delays it.

**Delete the whole block (686-706).** The wait gating moves to the drain loop.

## The Go replacement — queue wait>0 commands, drain on the pulse

Same code path serves interactive play and the oracle (the pump drives pulses
under DP_CLOCK, the ticker under wall-clock). No oracle-only special-casing.

**1. Per-session input queue.** Add a small FIFO of pending commands
(`cmd string, args []string`) to `Session`, guarded by the session mutex.

**2. Enqueue decision at the single player-input funnel.** In
`handleCommand` (`pkg/session/session_login.go:367`), *replace* the unconditional
`ExecuteCommand(s, cmd.Command, cmd.Args)` with:

- `if s.player != nil && s.player.GetWaitState() > 0` → **enqueue** `(cmd, args)`
  and return nil (no output, no error — the C delay);
- else → `ExecuteCommand` now (the `wait==0` fast path — preserves today's green
  behavior for openers and rapid no-pulse scenarios).

Do this *after* the existing PLR_WRITING / board / rate-limit intercepts (leave
those, and `sendCharInput`/`sendPagerInput`, untouched — C routes menus/pager
through their own game-loop branches, not the command gate). Internal
`ExecuteCommand` callers (order at `manager.go:590`, wiz-force) bypass
`handleCommand` and must stay immediate — do not touch them.

**3. Per-pulse drain callback.** Add `OnDrainInput func()` to
`GameLoopCallbacks` (`pkg/engine/gameloop.go:52`) and invoke it at the **TOP of
`heartbeat`** (gameloop.go:272), **before** `OnPerformViolence` (fact 4). Wire it
in the manager to iterate sessions and, per session:

- decrement the player's wait by 1 (per-pulse — fact 2);
- if `wait <= 0` and the queue is non-empty, dequeue **one** command (fact 3)
  and `ExecuteCommand` it.

**4. Player wait → pulse units.** Today `SetWaitState(n)` stores `n`
PULSE_VIOLENCE *rounds* and `DecrementWaitState` runs once per round via
`OnRoundEnd` (`manager.go:574`). Convert the **player** side to pulses so the
per-pulse drain gate is faithful: `SetWaitState(n)` stores `n * PULSE_VIOLENCE`
pulses; decrement happens in `OnDrainInput` (per pulse). **Remove the player
`DecrementWaitState` from `OnRoundEnd`** (a PULSE_VIOLENCE wait then still
expires in exactly one round — `PULSE_VIOLENCE` pulses — matching C).
Verify `PULSE_VIOLENCE`'s value in the engine constants and use the named
constant, not a literal. **Do NOT touch the mob wait-state** (`MobInstance`,
`pkg/combat/engine.go:429-433`) — that's the separate fight.c mob-lose-round
mechanic and stays round-granular in `OnRoundEnd`.

**5. Intentionally NOT modeled: the `wait=1` per-command throttle** (comm.c:617).
Under DP_CLOCK, C clears it live on loop spins (which is why rapid no-pulse
scenarios like `act-obj-sweep` stay green), and in real play it's a sub-100ms
"one command per pulse" limit that no human and no oracle can observe. Modeling
it would *break* those green scenarios. Leave player `wait` at 0 after a
non-WAIT_STATE command, exactly as today. Note this decision in the PR.

## Tests (this is the real verification — DP-1202)

Since the oracle can't byte-gate mid-combat injection, **unit tests carry the
proof.** In `pkg/session` (or wherever the manager/heartbeat is testable):

- **queue-and-delay:** player with `wait = 2*PULSE_VIOLENCE`; enqueue a command;
  pump pulses; assert it does NOT execute until pulse `2*PULSE_VIOLENCE`, then
  executes exactly once.
- **one-per-pulse:** enqueue two commands at `wait==0`-becomes-eligible; assert
  they drain on consecutive eligible pulses, not both at once.
- **wait==0 fast path:** a command at `wait==0` executes immediately (no queue).
- **drain-before-violence ordering:** a queued command that becomes eligible on a
  PULSE_VIOLENCE pulse executes before that pulse's combat round.
- **no invented string:** assert no code path emits `"You're too busy!"` (grep-
  style or behavioral) — the gate is gone.
- **unit conversion:** `SetWaitState(1)` → expires after exactly `PULSE_VIOLENCE`
  drain pulses.

## Oracle gate (Claude authors/runs; sketch in the PR)

- **`combat-death` stays GREEN** — the regression anchor (engine + a final `look`
  that must still drain at `wait==0`). If this goes red, the fast path broke.
- **Opener scenario (NEW), GREEN:** a thief backstabs as the *opening* move
  (`wait==0`, drains live and deterministically exactly like the green `hit`),
  then pulses drive the aftermath. This is how we gate the *skill* layer without
  hitting the DP-1202 race. Claude will author the fixture (thief + piercing
  weapon per `src/act.offensive.c` do_backstab); you don't need to.
- **NOT gated:** mid-combat command injection (e.g. `combat-kick-isolation`) —
  non-reproducible on the C side per DP-1202. Leave that scratch scenario alone;
  Claude will remove it at reconciliation.

## Guardrails

- **Never** edit `src/` or `darkpawns-c-oracle/` — reference only.
- `make reachability` zero regressions; **run `golangci-lint` if available** and
  `gofumpt -w` every Go file you touch (worktree pushes bypass the pre-push
  hook — this has cost two CI failures; don't be the third).
- Don't stage `website/static/map/world-sphere.json` or `docs/reports/reek/*`.
- If you find existing tests asserting player `WaitState` in round-units, update
  them to pulse-units **faithfully** (the C `WAIT_STATE` value), and say so in
  the PR — do not re-skin a test to hide a behavior change.

## Deliverable

Per-session queue + enqueue-at-`handleCommand` + `OnDrainInput` heartbeat step +
player wait→pulse units + deletion of the invented gate (commands.go:686-706) +
the unit tests above. Claude reconciles, authors the opener fixture, and runs the
oracle gate (`combat-death` + opener green).
