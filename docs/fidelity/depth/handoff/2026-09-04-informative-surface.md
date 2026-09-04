# Informative surface audit — 2026-09-04

This slice audits the remaining registered player-visible handlers in
`src/act.informative.c`: `do_coins`, `do_users`, and `do_toggle`. The audit
started from `origin/main` after the prior surface inventory and checked the
open-PR queue before running the vehicle. The dedicated scenario is
`cmd/dp-oracle-diff/scenarios/informative-residual-depth.txt`.

## Call-path findings

- `gold` is the command-table registration at `src/interpreter.c:469`, which
  calls `do_coins` at `src/act.informative.c:2743-2754`.
- `toggle` is registered at `src/interpreter.c:779` and calls `do_toggle` at
  `src/act.informative.c:2500-2579`.
- `users` is registered at `src/interpreter.c:797`, gated at `LVL_IMMORT`, and
  calls `do_users` at `src/act.informative.c:1948-2115`.
- `do_skills` is declared in `src/interpreter.c:237` and defined at
  `src/act.informative.c:2689-2741`, but has no command-table registration;
  it is therefore not part of the registered player-command surface under
  R2/R4/R5e.

## Oracle evidence

Scenario `informative-residual-depth`, seeds 1 and 2, both completed within
the 240-second per-scenario budget with no timeout or transport error.

- `gold` was an exact normalized match on both seeds. The three C lines are
  the carried, banked, and net coin totals.
- `users` was a stable content red on both seeds. C emitted its faithful
  descriptor table (`Num Class ... Site`), including descriptor number 1,
  `[40 Wa]`, `Playing`, the fixed harness host, and `1 visible sockets
  connected.`. Go emitted its separate security-oriented `Name / Level /
  Remote Addr` table, reported `unknown`, and ended with `1 player(s)
  connected.`.
- `toggle` was a stable content red on both seeds. The crowned first player
  has C's `init_char` defaults: hit-point, move, and mana displays `OFF`, and
  wimp `OFF`. Go reported those three displays `ON` and wimp `5`; its
  auto-exit line already matched `OFF`.

These are deterministic output/default-state divergences, not flaky reds.
The next commit will correct only the Go path, then rerun seeds 1 and 2 (and
the existing mortal toggle vehicle) before promoting the corresponding
manifest and surface-inventory rows.

## Fidelity ruling

No C source or oracle file was edited. The current rows remain unproven until
the Go output and first-player default state match. The `do_skills` handler is
an explicit command-table exclusion, not an invented command to port.
