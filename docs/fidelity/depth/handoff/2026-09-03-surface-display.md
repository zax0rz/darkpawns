# Surface-family handoff — `act.display.c` — 2026-09-03

## Scope

This is the next source-file family in the off-command surface inventory after
the already-promoted `act.comm.c` family. The branch starts from merged
`main` at `713912d77`, with the case report at 4,117 total, 4,020
proven/delegated, 46 blocked, and 51 excluded.

## C ownership audit

The 48 `act()`/`send_to_char()` call sites in `src/act.display.c` belong to
the two registered handlers and their InfoBar rendering helpers:

- `do_lines` (`src/act.display.c:80-109`) is fully owned by
  `docs/fidelity/depth/lines.tsv`, including its registration/position gate,
  default query, atoi/rejection boundaries, valid state update, trailing
  input, and active-InfoBar redraw;
- `do_infobar` (`src/act.display.c:112-155`) and `InfoBarOn`, `InfoBarOff`,
  `InfoBarUpdate`, and all field helpers (`:158-706`) are fully owned by
  `docs/fidelity/depth/infobar.tsv`, including immortal/mortal frames, raw
  VT100 bytes, field order, transitions, unknown-state reset, and update
  state; and
- the existing tests and oracle vehicles in those manifests cover the
  audience, rendered-byte, state, and deterministic seed dimensions. No
  residual helper or alternate dispatch path remains in this file.

This is an ownership/delegation promotion, not a new oracle claim. No C or
oracle file was edited.

## Result

Promote `src/act.display.c act-send family` in
`docs/fidelity/depth/surface-inventory.tsv` to `proven-already`, citing
`lines.tsv` and `infobar.tsv`. The surface inventory remains free of
`excluded-with-C-reason` rows.

Rules applied: R1/R2/R3/R4/R5b/R5c/R5e — preserve the existing exact output
proofs, honor the registered surface, avoid inventing a second renderer,
delegate shared ownership, and verify every call-site family against its C
owner.

