# `flee.six-invalid-attempts` proof repair — 2026-10-09

## Finding

The original `flee-no-exits` transcript matched C's terminal
`PANIC!  You couldn't escape!`, but `flee-death-exit` emitted the same bytes.
That transcript alone did not prove that all six sampled directions failed
`CAN_GO` (`src/act.offensive.c:392-415`; R1/R5e/R5h). The registered Go path is
`flee` → `cmdFlee` → `World.DoFleeMove` (`pkg/session/commands.go:93`,
`pkg/session/combat_cmds.go:244-330`, `pkg/game/act_movement.go:229-234`).

## Control

`docs/fidelity/depth/handoff/2026-10-09-flee-no-exits-control.py` captures
original fixture → compiling fixture mutation → restored fixture. The control
adds six safe exits from room 8162 to 8161; C then emits the movement success
line, invalidating the no-exits assertion. The original and restored fixture
emit PANIC. The script checks the oracle diff exit and rejects build failures.

Run:

```sh
python3 docs/fidelity/depth/handoff/2026-10-09-flee-no-exits-control.py \
  --output "$HOME/Archives/darkpawns/oracle-runs/2026-10-09/luna-flee-no-exits-control"
```

Result: green/control/restore all PASS. Retained C captures, logs, and HEAD are
in that output directory. The row is `oracle-green` for the original
`flee-no-exits` claim. No production code, RNG path, room peaceful flag, or
shared formatting changed. Reader/lock audit: only supported scenario fixture
replacement; runtime movement remains the existing command path.
