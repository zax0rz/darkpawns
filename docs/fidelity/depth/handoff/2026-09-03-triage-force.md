# Depth-fidelity triage handoff — `force` NPC command interpreter

Date: 2026-09-03

Branch: `glm/depth-force-triage`

Base: `origin/main` at `6a2d7ff9c`

## Verdict

The `force-mob` family remains blocked. The red is a confirmed Go coverage
gap, but it requires the shared NPC command-interpreter architecture; no
sit-only or stand-only adapter was added.

The two required isolated attempts were run against current main with
`--show-oracle`, at seeds 1 and 2:

```text
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle \
  /usr/local/go/bin/go run ./cmd/dp-oracle-diff \
  --scenario force-mob --show-oracle --seed 1
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle \
  /usr/local/go/bin/go run ./cmd/dp-oracle-diff \
  --scenario force-mob --show-oracle --seed 2
```

Both runs reported the same normalized divergences:

```text
force trainee sit
C: Okay.\nA guard trainee sits down.
Go: No-one by that name here.

force trainee stand
C: Okay.\nA guard trainee clambers to his feet.
Go: No-one by that name here.
```

The repeated result is stable content evidence, not a timeout or contention
failure.

## Call-path audit

The registered C command is `src/interpreter.c:438`:

```c
{ "force"    , POS_SLEEPING, do_force    , LVL_GOD, 0 },
```

`src/act.wizard.c:1856-1906` parses the target and command with `half_chop`,
resolves the visible character through `src/handler.c:1276-1325`, sends
`Okay.`, and calls `command_interpreter(vict, to_force)` for an ordinary
target. The NPC path is therefore the same generic interpreter as the player
path; the C `sit`/`stand` handlers produce the observed mob room actions.

The Go path in `pkg/session/wiz_communication.go` resolves force targets as
descriptor-backed `*Session` values and invokes `forceSessionCommand`. There
is no generic `*MobInstance` command session/interpreter for this path, so an
NPC target cannot be resolved and Go emits its missing-target response.

The divergence is confirmed under R1/R2/R5e. However, adding special cases
for the two vehicle commands would violate R4 and leave the rest of C's
`command_interpreter` surface unimplemented. Keep both
`force.visible-npc-target` and `force.npc-command-interpreter` blocked until
the shared architecture is deliberately addressed and audited as a class
(R5b/R5c).

## Checks

`make fidelity-depth` passed on the base before this triage, reporting 4,111
cases: 4,013 proven/delegated, 45 blocked, and 53 excluded. No `src/` or
`darkpawns-c-oracle/` file was edited, and no Go behavior change was made.

This handoff advances after the two honest attempts required by the depth-loop
objective and cites R1/R2/R4/R5b/R5c/R5e.
