# Depth-fidelity handoff — deferred `~dpclock` heartbeats — 2026-09-27

Branch `fix/dpclock-deferred-heartbeats` from `origin/main` at `f84e93be6`
(PR #1678 merged). Companion to
[`2026-09-27-idle-lifecycle-expansion.md`](2026-09-27-idle-lifecycle-expansion.md),
which recorded the force-rent vehicle as deferred behind harness-invalid
oracle behavior. This round fixes the seam and lands the vehicle.

## What changed

`tools/oracle-seam/dp-determinism.patch` (and the oracle's `dp-oracle-seam`
branch, commit `98bf46cb9f89f31a3e22fee86c7374df023889a5`) no longer runs
`heartbeat()` inside `process_input()`. The control validates and consumes
`~dpclock pulse N` exactly as before but only adds to `dp_pending_pulses`
(capped at the documented 100,000 maximum), and the game loop drains the queue
at its existing heartbeat slot:

```c
    /* Now execute the heartbeat functions */
    if (dp_clock) {
      while (dp_seam_take())
        heartbeat(++pulse);
    } else {
      while (missed_pulses--)
        heartbeat(++pulse);
    }
```

That is the production position for a heartbeat (`src/comm.c:679`): after the
exception/`close_me` sweep, `process_input`, command dequeue and dispatch,
output flush, the prompt sweep and the `CON_CLOSE` sweep. So a pumped hour may
still close a descriptor — the idle force-rent does — but never one that the
input iteration is holding in `d`/`next_d` (`src/comm.c:577-583`). The
non-`DP_CLOCK` path keeps the original loop verbatim.

`DP_SEAM_SELFTEST=1` runs the seam's own contract checks before boot and exits
(`internal/oraclediff/seam_test.go` drives it).

## The regression

`cmd/dp-oracle-diff/scenarios/lifecycle-idle-force-rent-close.txt`: two
connections, the actor idles 31 pumped hours into the connected force-rent
while the driver pumps, `compare-close` compares the server-initiated close as
`<CLOSE>`, and a closing `<RELOGIN>` shows the post-disconnect state. The
actor's descriptor is the driver's `next` in the input iteration, i.e. exactly
the previously fatal shape.

Before/after, same scenario, same seed:

- **before** (seam as shipped in #1678): the oracle log ends
  `Closing link to: Idleforcerent.` / `Idleforcerent force-rented and
  extracted (idle).` / `*** bit out of range 0 - FD_SETSIZE on fd_set ***:
  terminated`, and the harness fails with a connection reset.
- **after**: `no normalized divergence`, with the actor's terminal block
  `<CLOSE>` on both engines; green at seeds 1, 2, 3, 5, 8 (6/6 runs including
  the `--show-oracle` capture).

## F2 disposition: unchanged, still open

The scheduling analysis was right about the *ordering* and wrong about the
outcome. C's terminal-pulse ordering is now production's: the hour's outdoor
line is queued and then discarded by `close_socket`, so the actor receives
`<CLOSE>` and nothing else. The port still flushes that queued line to the
socket before it retires the session, so at a weather-broadcast hour the C and
Go blocks still differ — C `<CLOSE>`, Go `The suns slowly disappear in the
west and south.` + `<CLOSE>`. That byte is a Go-side divergence (a session
that is about to close should not flush), not a harness one, and this round
does not touch production Go.

The committed vehicle therefore keeps its terminal pulse free of weather text
by raising the void room's sector to `SECT_INSIDE` (`set-room-sector 1 0`),
which both engines honor identically (`OUTSIDE(ch)` is
`!ROOM_INDOORS || sector != SECT_INSIDE`). The vehicle proves the harness
contract, not that byte; the natural behaviour is recorded in the retained
archive of this round and as the blocked row
`idle.force-rent-terminal-weather-byte`. Removing the sector fixture from the
scenario reproduces the divergence in one run.

## Reference binary and its optimization level (read before rebuilding)

The oracle binary in this round's runs was rebuilt from the branch tip plus this
seam at **`-g -O2 -fcommon`**, because that is the arm the deployed reference
artifact was built with (DWARF `DW_AT_producer`; `docs/DEV-SETUP.md` documents
`-O0` instead). The two levels disagree observably through latent UB: a
self-overlapping `sprintf` in `SPECIAL(start_room)` (`src/spec_procs.c:2219`)
appends the newbie vision text to its own buffer, and at `-O0` the oracle emits
three lines the port does not:

```text
A Burning Hut
...
[ Exits: None! ]

   Suddenly the hairs on the back of your neck stand up as if lightning had
struck nearby. A keen wailing fills the air, and an ethereal image appears
before you.
   'Ccreator, now is not your time to die,' speaks the figure.
```

Rebuilding at the documented `-O0` therefore turns `character-creation` and
`character-creation-name-retry` red with no port change (`-DP_SEED=1`, evidence
in this round's archive under `logs/character-creation-*`). It is not caused by
the seam: a fresh `-O0` build of the *previous* seam, and a fresh `-O0` build of
the pre-draw-log tree with only this seam, are both red the same way; the seam
change is byte-neutral for these scenarios (verified by rebuilding the old seam
from the same tree). Per R1a the durable fix is to patch the oracle to the
intended (tolerant) semantics and update the port, then rebuild `-O0` and
unpin; until then, build the reference at the level the corpus was calibrated
against and record which one it was.

## Rows this round changes

- `idle.force-rent-closes-transport`: blocked → `oracle-green-multiseed`
  (`lifecycle-idle-force-rent-close@1,2,3,5,8`).
- `idle.force-rent-no-menu`: blocked → `oracle-green-multiseed` (same proof:
  the terminal block carries `<CLOSE>` and no menu bytes on either engine).
- new `idle.force-rent-pump-deferred`: the harness contract itself.
- new `idle.force-rent-terminal-weather-byte`: blocked (F2, above).
- `idle.force-rent-lost-link`: still blocked (F1, out of scope here).
- `idle.rent-roundtrip-inventory` / `idle.rent-roundtrip-equipped-norent`: the
  harness obstacle is gone (the sector fixture excludes the terminal weather
  byte), so these are now buildable; they are left blocked for a follow-up
  round rather than mixed into this harness PR.

## Not done here

- F1 (rnum `3` vs vnum `3` force-rent destination) and F3 (DP-1340's mudlog
  sites): untouched.
- No production Go change of any kind.
- The reconfigured oracle must be rebuilt from the seam patch; the reference
  checkout's `bin/circle` was rebuilt in place and the previous binary is kept
  in this round's archive (`binaries/circle-before-old-seam`).
