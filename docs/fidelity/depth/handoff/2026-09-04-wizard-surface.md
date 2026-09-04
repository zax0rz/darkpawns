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

## First vehicle

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
2. Valid target/report branches remain unproven and will be tested or sharply
blocked in subsequent slices; no no-argument green is promoted to full file
coverage.

## Working-tree changes after this handoff commit

The follow-up commit will contain the transfer registration/handler, the
`wiznet`, `syslog`, `sysfile`, and `zlist` corrections, the two-seed vehicle,
and depth-manifest rows for the entry probes. No C or oracle-tree files are
changed.

