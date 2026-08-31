# Depth handoff correction — 2026-08-31 — `fwap` queue order

The preceding `2026-08-31-command-fwap.md` named `gag` as the next command
because it skipped the blank source-table row. A fresh source-order check
corrects that statement: `src/interpreter.c:457` is `fwap`, line 458 is blank,
line 459 is the un-manifested `get`, and line 460 is `gag`. The checked-in
`pkg/session/command_order.tsv` confirms the same order (`fwap`, `get`, `gag`).

No frontier or code changes are made by this correction. The next depth
session must start from clean `main` and take `get`, in accordance with R2,
R5e, and the standing queue's source-order rule; `gag` remains unclaimed.
