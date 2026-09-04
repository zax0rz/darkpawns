# Depth-fidelity triage handoff — `redit` OLC family

Date: 2026-09-03

Branch: `glm/depth-redit-triage`

Base: `origin/main` at `9070a56ee`

## Verdict

The `redit-entry-depth` and `redit-session-depth` reds remain blocked. They
confirm the missing descriptor-driven OLC subsystem. No Go behavior change was
made.

The two honest family attempts were run against current main with
`--show-oracle`:

1. `redit-entry-depth`, seed 1: `redit abc` and `redit 999999` diverged. C
   returned `Yikes!  Stop that, someone will get hurt!` and `Sorry, there is
   no zone for that number!`; Go returned `Huh?!?` for both.
2. `redit-session-depth`, seed 1: empty `redit` selected the actor's current
   room (room 1204) and C emitted the complete 24-line `REDIT_MAIN_MENU` while
   entering the editor state; Go returned `Huh?!?`.

Both are stable content divergences, not timeout or contention failures.

## Call-path audit

The C registration is `src/interpreter.c:656`:

```c
{ "redit"    , POS_DEAD    , do_olc      , LVL_BUILDER, SCMD_OLC_REDIT},
```

`src/olc.c:80-275` owns the shared OLC entry path: NPC rejection, `save`,
argument parsing, the empty-input current-room selection, numeric validation,
zone resolution and permission checks, state allocation, and transition to
`CON_REDIT`. The valid path calls `redit_setup_existing` or
`redit_setup_new` in `src/redit.c:65-92`, emits `REDIT_MAIN_MENU`, and routes
subsequent descriptor input through `redit_parse` (`src/redit.c:673-709`,
`src/interpreter.c:1723-1731`).

The Go repository has command-gate metadata for the C row but no registered
`do_olc`/REDIT handler or equivalent descriptor state machine. A message-only
shim would not cover current-room selection, menu transitions, edits, save,
or cleanup, and would invent an incomplete surface. Keep all four rows in
`docs/fidelity/depth/redit.tsv` blocked until a complete OLC subsystem is
intentionally implemented and audited as a class (R1/R2/R4/R5b/R5c/R5e).

## Checks

`make fidelity-depth` passed on the base before this triage, reporting 4,111
cases: 4,013 proven/delegated, 45 blocked, and 53 excluded. No `src/` or
`darkpawns-c-oracle/` file was edited.

This handoff advances after the two honest attempts required by the objective
and cites R1/R2/R4/R5b/R5c/R5e.
