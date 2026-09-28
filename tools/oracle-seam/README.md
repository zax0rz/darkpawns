# C-oracle determinism seam

This directory preserves the Dark Pawns C oracle's test-only determinism seam
outside the oracle clone. The patch applies to pristine base `d2cb13e`; the
oracle's `dp-oracle-seam` branch carries the same seam on top of upstream.

The seam adds four environment-gated behaviors:

- `DP_SEED=<uint32>` seeds the CMWC stream; when unset, the oracle still seeds
  from `time(0)`.
- Presence of `DP_CLOCK` freezes wall-clock heartbeats.
- With `DP_CLOCK` present, `~dpclock pulse <n>` **queues** `n` deterministic
  heartbeats. The game loop runs them at its normal heartbeat phase — after
  descriptor input, command dispatch, output, prompt and `CON_CLOSE` processing
  (`src/comm.c` game loop, the `/* Now execute the heartbeat functions */`
  block) — which is where production C runs them. The control line itself is
  consumed before command queuing and interpretation: it does not enter command
  history or the input queue, alter wait-state, expand aliases, or pass through
  `command_interpreter`. Only the heartbeats it queues may draw or mutate game
  state.
- `DP_SEAM_SELFTEST=1` runs the seam's own contract checks before boot and then
  exits, printing `dp-seam-selftest: …` lines for the queue, the drain, the
  queue ceiling, normal-mode refusal, and invalid-control refusal. The Go suite
  drives this (`internal/oraclediff/seam_test.go`).

## Why the pump is deferred

Running heartbeats inside `process_input` — the seam's original form — lets a
pumped hour act while the game loop's input iteration still owns the descriptor
in `d`/`next_d`. An idle force-rent is exactly such an hour: `check_idling`
calls `close_socket()`, which frees that descriptor, and the loop's next
iteration dereferences freed memory:

```text
*** bit out of range 0 - FD_SETSIZE on fd_set ***: terminated
```

Deferring the pump to the heartbeat phase removes the hazard without changing
what a pulse does, because production C also runs heartbeats there.
`lifecycle-idle-force-rent-close` is the black-box regression: it aborts the
oracle on the old seam and matches the port on this one.

## Limits

A control accepts 1 through 100,000 pulses, and the pending queue is capped at
the same 100,000. A control that would push the queued total past the cap is
refused the same way a malformed control is — it is not consumed as a control,
so the line stays ordinary player input — instead of wrapping the counter.

## Build flags: match the reference binary, not just the docs

The lap-top reference oracle on the workstation is a **`-g -O2 -fcommon`**
build (DWARF `DW_AT_producer`), while `docs/DEV-SETUP.md` mandates `-O0` for the
UB tolerance it buys. The two disagree today, and the difference is observable:
the DikuMUD C carries latent undefined behavior (here a self-overlapping
`sprintf` in `SPECIAL(start_room)`, `src/spec_procs.c:2219`, that appends the
vision text to its own buffer). At `-O0` the oracle emits the full vision; at
`-O2` the optimizer drops the first three lines — the bytes the Go port was
calibrated against. A `-O0` rebuild therefore turns
`character-creation` and `character-creation-name-retry` red without any port
change. Until the oracle is patched to the intended semantics and the port
follows (R1a), rebuild a reference oracle at the same optimization level as the
one the corpus was calibrated against, and say which one it is in the run
manifest.

## Rebuild

From a clean C-oracle checkout at `d2cb13e`:

```bash
git apply /path/to/darkpawns/tools/oracle-seam/dp-determinism.patch
cd src
make
```

With `patch`:

```bash
patch -p1 < /path/to/darkpawns/tools/oracle-seam/dp-determinism.patch
cd src
make
```

`make` installs the rebuilt executable at `../bin/circle`, which is
git-ignored; the oracle's `lib/` tree must stay beside `bin/`. A checkout that
is already at `dp-oracle-seam` needs only the delta from the previously
released seam, not the whole patch. To verify the patch without changing a
checkout:

```bash
git apply --check /path/to/darkpawns/tools/oracle-seam/dp-determinism.patch
```

Rebuild after any change here: the harness runs whatever binary
`DP_ORACLE_BIN` points at, and a stale binary reintroduces the force-rent
abort above.
