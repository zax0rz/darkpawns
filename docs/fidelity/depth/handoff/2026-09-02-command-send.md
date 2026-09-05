# Depth handoff — 2026-09-02 — command `send`

The `send` depth slice is complete. PR #1149 (`glm/depth-send`) passed hosted
lint, security, and test checks and was self-merged to `main` as
`a5c6c84f2` under R1/R2/R3/R5e. The implementation now follows the C path from
`src/interpreter.c:681` through `half_chop`, `get_char_vis`, player/NPC
delivery, and the `PRF_NOREPEAT` confirmation branch. It preserves exact C
line endings, empty-message behavior, raw internal/trailing spacing, room and
global character lookup, self/abbreviated targets, NPC confirmation, and
`NOPERSON` bytes. The proof set is `send-depth`, `send-global-depth`,
`send-mob-depth`, and `send-gate-depth`, each green on seeds 1, 2, 3, 5, and 8;
focused session tests pin the byte/order contracts.

The frontier after the merge is:

- 3163 total cases
- 3085 proven/delegated
- 26 blocked
- 52 excluded

The source-order interpreter sweep rechecked existing manifest command fields:
`sell` is already claimed by `docs/fidelity/depth/do-not-here.tsv:4` as part of
the shared `list/buy/sell` fallback, and `scream` is covered by shared social
evidence. Do not re-pick either. The next genuinely un-manifested command
family is `set` at `src/interpreter.c:682`, registered as `POS_DEAD` with
`do_set` and `LVL_GOD`. Begin it only after the fresh main checkout/pull,
`make fidelity-depth`, depth-guide read, newest-handoff read, and C-table/
manifest audit required by the loop. The blocked
`objmagic.sleep-entry-gates` row remains pending its one cast-sleep vehicle.

Fidelity rules for the next slice remain R1/R2/R3/R4/R5e, with shared ownership
bounded by R5b/R5c. Never edit `src/` or the C oracle tree.
