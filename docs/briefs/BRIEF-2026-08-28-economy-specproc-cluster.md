# Codex Goal — Economy-Specproc Cluster

**Date:** 2026-08-28
**Frontier before this slice:** 587 cases, 569 proven/delegated, 5 blocked, 13 excluded (`make fidelity-depth`).
**Why this slice:** Command-surface breadth is effectively complete for load-bearing
commands (~289/298 probed once cosmetic socials are excluded). The last real
untouched command-surface gaps cluster in the economy specprocs. Close them, then
return to spec-proc depth — the true remaining frontier.

## Process (unchanged)

- Add manifest rows under `docs/fidelity/depth/` with statuses
  `oracle-green` / `oracle-green-multiseed` / `unit-green` / `delegated` / `blocked`.
- Every claim must carry a real proof artifact. `scripts/gen_fidelity_depth.py`
  enforces this: `oracle-green` rows must point to an existing scenario `.txt`
  carrying the matching depth-case annotation; `unit-green` rows must name a symbol
  that actually appears in a `*_test.go`. Dangling claims fail the gate.
- Record genuine blockers as `blocked` with the C-site reason. Do **not** force-green.
- One slice = one PR, green hosted checks before merge. Source order A → B → C.

## Slice A — Bank triad

`SPECIAL(bank)` at `src/spec_procs.c:2345`. Commands `balance` / `deposit` / `withdraw`
are `do_not_here` at the interpreter level and dispatch to the specproc when a banker
mob is present. Prove against a banker-mob scenario:

- `balance`: deposited-balance readout vs the zero-balance text (`spec_procs.c:2349-2355`).
- `deposit`: missing-amount prompt (`:2363`), gold→balance mutation, confirmation text (`:2373`).
- `withdraw`: missing-amount prompt (`:2382`), insufficient-funds gate (`:2387`),
  balance→gold mutation, confirmation text (`:2392`).
- The `TO_ROOM` "makes a bank transaction" act on both deposit and withdraw (`:2375`, `:2394`).

## Slice B — Recharger

`SPECIAL(recharger)` at `src/new_cmds2.c:415`. Command `recharge` (plus `list`/`help`).

- Command-gate: the proc only fires on `list` / `help` / `recharge` (`new_cmds2.c:421`);
  prove non-matching commands fall through.
- Recharge economy path and the recharger's spoken line (`:427`).

## Slice C — Rent pair

- `offer` — `src/objsave.c:1139` (rent-quote path; note the `!CMD_IS("offer") && !CMD_IS("rent")` guard).
- `retrieve` — `src/spec_procs3.c:830` (item retrieval from rent/pawn).
- Probe against the relevant rent/pawn specproc mob.

## Out of scope / already recorded — do not touch

- **Socials** — already done; the coverage tool's "never-probed" socials are
  delegated-by-mechanism, not a gap. Do not add per-emote probes.
- **`dig`** — known honest gap (implemented-unwired, `src/new_cmds2.c:818`, LVL_BUILDER,
  DP-1225). Leave it recorded as-is; do not paper over it.
- **The 26 `missing` commands** — unbuilt. Do not fabricate probes for unimplemented commands.

## After the cluster

Resume the source-ordered spec-proc depth march. `dragon_breath` was the paused
next-up before this pivot.
