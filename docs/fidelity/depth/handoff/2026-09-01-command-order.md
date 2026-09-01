# Depth-fidelity handoff — 2026-09-01 — `order`

## Frontier

- Clean-main start: `Cases: 2679 total, 2609 proven/delegated, 22 blocked, 48 excluded`.
- Clean-main finish after the merged slice: `Cases: 2690 total, 2620 proven/delegated, 22 blocked, 48 excluded`.
- Actionable completion: `2620/2642 = 99.2%`.
- The special-procedure inventory remains exhausted. The one permitted
  `objmagic.sleep-entry-gates` attempt remains split: the cast-sleep vehicle is
  proven, while the unreachable remainder stays blocked with its existing note.

## Source path and queue decision

The next source-order command row was:

```c
src/interpreter.c:586: { "order" , POS_RESTING , do_order , 1, 0 }
```

I read the authoritative C path first: `src/act.offensive.c:294-357` calls
`half_chop`, resolves a visible room character or the `followers` pseudo-target,
rejects self and charmed actors, sends the named-target order and room
announcement, then either emits the indifferent-look room text or re-enters
`command_interpreter`. The `followers` arm announces to the room, dispatches
same-room charmed followers in the C follower list, and acknowledges only when
one was found. `comm.c:2392-2555` supplies the audience topology; the forced
player command reaches the ordinary `say` path in `act.comm.c:1146-1305`.

The next genuinely unclaimed interpreter family is now `orgasm` at
`src/interpreter.c:587` (`do_otouch`, `POS_RESTING`, `LVL_IMMORT`). `offer` is
covered by the existing `do_not_here` generic-family manifest, and `open` is
covered by the existing door-family manifest. The next branch must use
`glm/depth-orgasm` after a fresh `main` checkout/pull and frontier confirmation.

## Proof and implementation

Initial RED on main showed that Go's old handler only approximated a following
mob: it missed self and player target resolution, the C `followers` arm, exact
actor/victim/room audiences, and the master-plus-`AFF_CHARM` condition. The
fill-word probe also confirmed that C `half_chop` treats the first token as the
target rather than applying `one_argument` fill-word skipping.

The confirmed fix ports only that `do_order` path in
`pkg/session/cmd_combat_special.go`: canonical room resolution, exact C bytes,
`game.Act` audiences, charm/master gates, both player and mob dispatch, and the
session command callback for re-entry. The intentionally reversed C
`is_abbrev(name, "followers")` direction is expressed through an explicit
helper to preserve behavior and pass lint. No C or oracle file was edited.

Added evidence:

- `cmd/dp-oracle-diff/scenarios/order-depth.txt` — no-argument, missing target,
  self, visible NPC, non-charmed player, `followers` with no loyal subject, and
  `half_chop` boundary.
- `cmd/dp-oracle-diff/scenarios/order-charmed-depth.txt` — outlaw cast-charm
  setup, saving-throw fallback, and named charmed-player success.
- `pkg/session/order_depth_test.go` — registration gate, exact charmed-actor
  refusal, named charmed-player re-entry, follower dispatch, and audience text.
- `docs/fidelity/depth/order.tsv` — 11 manifest rows, with shared command
  execution delegated only where already covered by the focused boundary test.

Both oracle vehicles returned `no normalized divergence` for seeds
`1,2,3,5,8`; `--show-oracle --seed 2` showed the successful charmed order and
forced `say` output. The local gates all passed:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
```

## Review state

- Commit: `830e7d822 fix: deepen order fidelity (R1 R2 R3 R5e)`.
- PR: #1051, `fix: deepen order fidelity`.
- The initial hosted workflow did not fire; the one permitted retry was run:
  `gh workflow run "Dark Pawns CI/CD" --ref glm/depth-order`.
- Hosted lint, security, and test checks passed; build/deploy were reported as
  skipped. The PR was squash-merged to `main` as `3563b71aa`.

This handoff records R1/R2/R3/R5e and the shared-boundary treatment under
R5b/R5c. Continue from a clean `main`; do not repick `order`.
