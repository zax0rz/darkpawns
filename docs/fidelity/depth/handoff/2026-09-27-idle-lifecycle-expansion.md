# Depth-fidelity handoff — idle lifecycle (DP-1311) — 2026-09-27

Branch `fix/idle-lifecycle-oracle-expansion` from `origin/main` at `fb784f84b`
(PR #1676 / DP-1311 merged). This round turns the DP-1311 idle-lifecycle fixes
into permanent black-box regressions. Two vehicles landed; the force-rent half
is deferred behind real C/Go divergences found while spiking it (see below).

## Landed vehicles

- `cmd/dp-oracle-diff/scenarios/lifecycle-idle-observer.txt` — the void
  departure and the return are room-visible and per-audience: no departure
  before the ninth tick, `$n disappears into the void.` to the peer only,
  `You have been idle, and are pulled into a void.` to the actor only,
  `$n has returned.` to the peer only. Green at seed 1.
- `cmd/dp-oracle-diff/scenarios/lifecycle-idle-command-reset.txt` — an accepted
  command resets the idle timer: eight silent hours after the command leave the
  actor in the original room (the immortal peer's own `look` proves it without
  touching the actor's counter) and the ninth hour voids them. Green at seeds
  1, 2, 3, 5, 8.

Both need an immortal observer. With `empty-players` the first character is the
God, so the actor self-demotes (`set player <name> level 1`) and the peer is
promoted (`set player <name> level 31`) and placed with `teleport <name> <vnum>`
— all ordinary God commands, all oracle-covered. An immortal's timer still
ticks, but `check_idling`'s thresholds are level-gated
(`src/limits.c:424-425`), so the observer can stand in the room for the whole
run. Note that C's `Valid_Name` rejects any name containing a substring of
`lib/misc/xnames` (`src/ban.c:257-285`): `Rentwatch` contains `twat`, so the
scenario names avoid that list.

## Harness support added: `compare-close`

A server-initiated disconnect was invisible to the diff: an empty block and a
closed block compared equal. `internal/oraclediff` now records a server EOF on
`TCPConn` (`CloseReporter.ObservedClose`), `RunAudienceProbe` marks the block
whose read saw it, and a scenario that declares the `compare-close` fixture
renders `<CLOSE>` (`oraclediff.CloseMarker`) into that block. Audience reads now
accept EOF in the same positions as the actor read (last step, or the step
before a relogin/restart) instead of only at the last step — a server can close
an audience's transport (the C idle force-rent closes the actor) on a step the
actor does not own. Nothing changes for scenarios that do not opt in.

## Feasibility matrix

| Candidate | State reached in C? | Player-visible proof? | Implement / defer reason |
|---|---:|---:|---|
| command reset | yes (`~dpclock` + 9 hours each side of a real command) | yes, actor room + peer `look` | landed: `lifecycle-idle-command-reset` |
| observer void/return | yes (9 hours of silence) | yes, peer act + actor line, both audiences | landed: `lifecycle-idle-observer` |
| force-rent close/lost-link | yes (31 hours; `close_socket` runs, mudlog fires) | partly | deferred: `<CLOSE>` matches, but the same block carries a byte the port emits and C drops (F2), and the lost-link act's room differs (F1) |
| rent relogin inventory/equipment | yes (rent file written, re-entry loads it) | yes | deferred: the vehicle must force-rent while connected, so it inherits F2 |
| wounded combat void | not attempted | — | deferred: the force-rent half it depends on is blocked, and the combat setup needs hostile-mob RNG and HP arithmetic the harness cannot pin deterministically; no masking was added to fake it |

## The DP_CLOCK pump can free the descriptor it is looping over

`~dpclock` is oracle-only instrumentation: `process_dpclock_control` runs
`heartbeat()` inside `process_input` (`darkpawns-c-oracle/src/comm.c:1836-1856`,
wired at `comm.c:2027`). Production C runs the heartbeat from the game-loop tail
(`src/comm.c:679`), after the descriptor loops. So a pumped hour that closes a
descriptor — the idle force-rent — frees it while the input loop still holds it
in `next_d` (`src/comm.c:577-583`), and the next iteration dereferences freed
memory:

```text
*** bit out of range 0 - FD_SETSIZE on fd_set ***: terminated
```

The abort happens whenever the freed descriptor is exactly the pumping
connection's `next`, which is why every first attempt at this vehicle died. The
workaround is ordering, not masking: the actor `quit`s and `<RELOGIN>`s before
the pump, so the actor is the newest (head) descriptor and the pump's captured
`next` is a live peer. `lifecycle-idle-force-rent` (kept out of the commit) is
the recipe; the C oracle itself cannot be patched (R1/R5a — `src/` and the
checkout are read-only).

## F1 — the idle force-rent moves the character to a different room

C: `char_to_room(ch, 3)` (`src/limits.c:441`) takes an **rnum**. With the
shipped zone order rnum 3 is vnum 4 (`Frontline's Sphere`). The port:
`PlayerTransfer(p, 3)` (`pkg/game/limits_misc.go:73`) takes a **vnum**, so the
port's character lands in vnum 3 (`A Totally Empty Room`).

Proof (both directions, seed 1): a peer teleported to vnum 3 sees
`Idleforce has lost his link.` on the port and nothing on C; the same scenario
with the peer in vnum 4 sees it on C and nothing on the port. The act's audience
is the only player-visible surface of that room choice (free_rent is YES, so the
rent pass takes the objects instead of dropping them in the room).

## F2 — the disconnect pulse delivers a byte C drops

At the pulse where the idle timer passes `IDLE_DISCONNECT`, both engines run
`weather_and_time` before `point_update`. C's `close_socket` then closes the
descriptor inside `check_idling` (`src/comm.c:2092`), and the C game loop's
output pass (`src/comm.c:636`) never runs for a descriptor that is already gone:
the outdoor weather line queued to that player earlier in the same heartbeat is
discarded and the player never receives it. The port retires the session after
the pumped heartbeat returns, so the actor's socket receives that line before
the close.

Evidence: the force-rent probe block is `The suns slowly disappear in the west
and south.` + `<CLOSE>` on the port and `<CLOSE>` alone on C, at seed 1 and
again in the confirmation run.

## F3 — mudlog surface differences seen while spiking (DP-1340's class)

With an immortal peer at `syslog complete`, the same vehicles also exposed:

- `[ Losing player: <name>. ]` — C logs it when a descriptor in the menu (not
  `CON_PLAYING`) drops (`src/comm.c:2134-2138`); the port emits nothing.
- `[ <name> un-renting and entering game. ]` — C logs it from `Crash_load`
  (`src/objsave.c:517-520`); the port emits nothing.
- `[ <name> [127.000.000.001] has connected. ]` vs the port's `[127.0.0.1]` —
  a host-format difference in the connect line itself.

These are not part of the idle lifecycle; they are recorded here because any
depth vehicle that shows the disconnect to an immortal will keep hitting them
until DP-1340's remaining sites are ported.

## What the next round needs

1. Decide F1 (does the port reproduce C's rnum destination? if not, pin the
   intended room and prove the act's audience) and F2 (flush/close ordering at
   an extracted session) — both are production changes.
2. Then `lifecycle-idle-force-rent` and the rent round trip promote from
   `blocked` to `oracle-green` with no harness change: the recipe, the
   `compare-close` support and the `<CLOSE>` match already exist.
3. Vehicle E (wounded combat void) still needs a deterministic damaging-mob
   fixture; `advance`/`set` can place stats, but no scenario yet pins a combat
   round's HP arithmetic without masking output.

## Evidence

`/home/zach/Archives/darkpawns/oracle-runs/2026-09-27/idle-lifecycle-expansion/`
holds the focused run logs (including `--show-oracle`), the per-seed multiseed
logs, the force-rent spike logs for F1/F2/F3, and `MANIFEST.md` with the
revision, commands, seeds, statuses and file sizes.
