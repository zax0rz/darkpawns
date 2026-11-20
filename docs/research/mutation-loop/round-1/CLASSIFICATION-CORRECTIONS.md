# Round 1 post-census classification corrections

The Round 1 sample and measured denominator remain frozen. These post-census
audits identify two sampled Go literals whose matching C branches are
unreachable from their player-facing callers.

| Row | Frozen class | Post-census finding | C branch cite | Basis |
|---:|---|---|---|---|
| 3 | c-reachable | c-unreachable after caller audit | `src/modify.c:131-139` (9 lines) | The mutated Go literal belongs to the first-line mail overflow branch. C mail uses `string_write(..., MAX_MAIL_SIZE, ...)` (`src/mail.c:544`, `src/mail.h:47`) while each accepted input line is bounded by `MAX_INPUT_LENGTH` (`src/structs.h:644`); the first-line guard cannot be reached through mail. C's later-line overflow branch emits different text. |
| 18 | c-reachable | c-unreachable after field-table audit | `src/act.wizard.c:2696-2700` (5 lines) | This refusal requires a PC target and a selected field without the `PC` bit. Every C `do_set` field is either `PC` or `BOTH`; the existing audit is recorded at `docs/fidelity/depth/wizard.tsv:209`. |

These corrections do not rewrite the frozen sample classifications, kills, or
Round 1 headline.
