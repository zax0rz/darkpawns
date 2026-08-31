# Depth handoff correction — 2026-08-31 — `get` already complete

The previous queue correction identified C table line 459 as `get`, but a
fresh manifest audit shows `get` is not un-manifested: `docs/fidelity/depth/get.tsv`
already covers the complete `do_get` family (11/11), its `get_from_room` and
`get_from_container` branches, `can_take_obj`, money conversion, script hook,
shared matcher, and fill-word cases. `make fidelity-depth` reports every one
of those cases proven, delegated, excluded, or unit-green.

The authoritative next un-manifested command is therefore `gag` at
`src/interpreter.c:460`, after `get` at line 459. No frontier or code changes
are made by this correction. The next depth session must start from clean
`main` and take `gag`, in accordance with R2 and R5e.
