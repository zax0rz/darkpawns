# Depth-fidelity triage handoff — `medit` OLC family

Date: 2026-09-03

Branch: `glm/depth-medit-triage`

Base: `origin/main` at `6a2d7ff9c`

## Verdict

The `medit-entry-depth` and `medit-session-depth` reds remain blocked. They
confirm a missing descriptor-driven OLC subsystem, not a one-line Go message
translation. No Go behavior change was made.

The two honest family attempts were run against current main with
`--show-oracle`:

1. `medit-entry-depth`, seed 1: no argument, nonnumeric argument, and unknown
   zone all diverged. C returned respectively `Specify a mobile VNUM to edit.`,
   `Yikes!  Stop that, someone will get hurt!`, and `Sorry, there is no zone
   for that number!`; Go returned `Huh?!?` for each.
2. `medit-session-depth`, seed 1: valid VNUM `18305` entered C's full
   `MEDIT_MAIN_MENU` (24 normalized lines); Go returned `Huh?!?`. The next `q`
   was misrouted by Go to the unrelated quaff handler (`What do you want to
   quaff?`) while C exited the unchanged editor.

These are stable content divergences, not timeout or contention failures.

## Call-path audit

The C registration is `src/interpreter.c:553`:

```c
{ "medit"    , POS_DEAD    , do_olc      , LVL_BUILDER, SCMD_OLC_MEDIT},
```

`src/olc.c:80-275` owns argument parsing, numeric and zone validation,
permissions, duplicate-editor checks, OLC allocation, and dispatch to
`medit_setup_new`/`medit_setup_existing`. A valid entry transitions the
descriptor to `MEDIT_MAIN_MENU`; `src/medit.c:612-1121` and
`medit_parse` then own the menu, field editors, script/flag submenus, save,
and cleanup state machine.

The Go repository has command-gate metadata for the C row but no registered
`do_olc`/MEDIT descriptor handler or equivalent session state machine. The
correct fix is therefore a complete, separately scoped OLC architecture. A
message-only shim or a fabricated menu would violate R1/R2/R4 and would not
cover the follow-up input path (R5e).

The existing four rows in `docs/fidelity/depth/medit.tsv` remain blocked with
their exact C sources. This advances after the two honest attempts required by
the objective; the family should be revisited only when the shared OLC
subsystem is intentionally implemented and audited as a class (R5b/R5c).

## Checks

`make fidelity-depth` passed on the base before this triage, reporting 4,111
cases: 4,013 proven/delegated, 45 blocked, and 53 excluded. No `src/` or
`darkpawns-c-oracle/` file was edited.

This handoff cites R1/R2/R4/R5b/R5c/R5e.
