# Depth-fidelity handoff — `mold`

Date: 2026-09-01  
Queue: un-manifested interpreter command families, source-table order  
Rules: R1, R2, R3, R4, R5b, R5c, R5e

## Frontier

Started from fresh `main` after the `moan` handoff.  `make fidelity-depth`
reported 2,532 cases: 2,468 proven/delegated, 18 blocked, and 46 excluded
(99.3% actionable).  After adding the `mold` slice and merging it, the
frontier is 2,544 total: 2,480 proven/delegated, 18 blocked, and 46 excluded
(99.3% actionable).

The next un-manifested command family is `mosh`, the registered C row at
`src/interpreter.c:552` (`POS_RESTING`, no command minimum level,
`do_action`).  No existing depth manifest row claims `mosh`.

## C path and proof

The queue item was `{ "mold", POS_RESTING, do_mold, LVL_IMMORT, 0 }` at
`src/interpreter.c:551`.  The implementation in `src/new_cmds.c:55-105`
performs these ordered operations:

1. `one_argument` selects the carrying object keyword, then `one_word`
   selects the new name, including a quoted multi-word name; the remainder is
   the raw description.
2. `get_obj_in_list_vis` searches `ch->carrying` only and returns
   `You don't have one of those.` on failure.
3. The object's keyword list must contain `halo`, `clay`, or `playdough`.
4. Name and description must both be present.
5. Success rewrites the live object `name`, `short_description`, and
   `description`, then emits the hardening sentence.

The clean-main RED vehicle found three confirmed divergences: Go emitted an
invented usage string for empty/partial input; Go stored mold metadata without
rewriting the live object fields, so inventory, drop, room look, and get could
not see the C mutation; and wizard-loaded inventory order was reversed because
the Go placement path appended while C `obj_to_char` prepends.  The Go fix
mirrors C parsing, uses inventory-only keyword lookup, updates typed runtime
display fields, and prepends immortal-loaded objects.  The source and oracle
trees were not edited.

Added:

- `cmd/dp-oracle-diff/scenarios/mold-depth.txt` — no argument, missing object,
  material gate, missing name/description, quoted-name success, inventory and
  room-description visibility, and renamed-keyword drop/get.
- `pkg/game/mold_depth_test.go` — C `one_word` parsing and live object fields.
- `pkg/session/mold_depth_test.go` — C command registration and entry gates.
- `docs/fidelity/depth/mold.tsv` — 12 manifest rows, including the delegated
  shared position gate.

The scenario passed at seeds 1, 2, 3, 5, and 8.  Seed 1 was run with
`--show-oracle`; all normalized blocks matched, including C inventory order,
room description, and subsequent renamed-keyword lookup.

## Gates and merge

Local gates passed on `glm/depth-mold`:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...` (0 issues)
- `gofumpt -l .` clean
- `git diff --check`

Feature commit `21e1a8358` was submitted as PR #1022 (`glm/depth-mold`).
Hosted `test`, `lint`, and `security` checks were green; `build-and-push` and
`deploy` were skipped by the workflow for this PR.  PR #1022 was self-merged
to `main` as `b49c727af`.

No C source or oracle-tree files were edited.
