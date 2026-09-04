# Wizard activity surface slice — 2026-09-04

## Scope

This slice continues the source-file-order surface inventory at
`src/act.wizard.c`. The file has 352 weighted `act()`/`send_to_char()` call
sites. Existing depth manifests already own the communication, movement,
loading, level, visibility, gecho, force, OLC-adjacent, and other command
families; this slice starts the residual registered commands and keeps the
file-level inventory unproven until every call-site family has an explicit
owner.

The C command table confirms the residual registered names are `transfer`,
`teleport`, `vnum`, `stat`, `switch`, `vstat`, `wiznet`, `zreset`, `syslog`,
`sysfile`, `wnewbie`, `zlist`, and `wizlock`, with `tick` as a lifecycle/time
pulse entry. The source call path was checked in `src/interpreter.c`; no
handler was treated as reachable from its C definition alone (R2/R4/R5e).

## Vehicles and findings

The new `wizard-residual-depth` vehicle probes the safe no-argument paths for
the residual command names. The existing `god-system-residuals` vehicle owns
the `wizlock` and `tick` probes. Seed 1 initially exposed these concrete
differences, all now corrected in the working tree:

- `transfer` was absent from the Go command registry, so the registered C
  command reached `Huh?!?` in Go.
- `wiznet` used an invented one-line usage string and one fewer leading space
  on its continuation line.
- `syslog` reported an invented default and did not use the C PRF_LOG1/2
  state bits.
- `sysfile` used an invented empty-argument usage line instead of C's
  `That isn't a file!` branch.
- `zlist` synthesized a Go zone summary instead of paging the selected C
  `world/zon/<vnum>.zon` file bytes.

After those corrections, `wizard-residual-depth` is byte-green at seeds 1 and
2. The follow-up `transfer-depth` vehicle then covered target-miss, successful
player transfer audience/state, and one-argument parsing at seeds 1 and 2.
The `teleport-depth` vehicle covered target-miss, self-target, missing and
invalid destinations, successful audience/state, and two-argument parsing at
seeds 1 and 2. The `wizard-report-depth` vehicle covered missing and invalid
branches for `vnum`, `stat`, and `vstat` at seeds 1 and 2. These probes exposed
and corrected the absent `transfer` registration, teleport movement/gating and
audience drift, and the invented wizard-report diagnostics.

The newly covered branches are recorded in `docs/fidelity/depth/wizard.tsv`.
The valid report bodies, `switch`, `wizlock` mutation, `wiznet` message paths,
`zreset`, `syslog` state-setting, `sysfile` file selection, `wnewbie`, `zlist`
variants, and the remaining shared/admin call-site families remain explicitly
unproven. No entry or error-path green is promoted to full file coverage.

At this handoff, `make fidelity-depth` reports 4,199 cases: 4,102
proven/delegated, 46 blocked, and 51 excluded. The next source-order frontier
is the valid wizard report/state branches, followed by the still-unowned
families in `src/act.wizard.c` and then the next C source file in the inventory.

## Follow-up implementation

The follow-up commit contains the transfer registration/handler, faithful
teleport movement and gates, the `wiznet`, `syslog`, `sysfile`, and `zlist`
corrections, wizard report diagnostics, the four two-seed vehicles, and their
depth-manifest rows. No C or oracle-tree files are changed.
