# Depth-fidelity handoff — `notitle`

Date: 2026-09-01

## Frontier

This session began from clean `main` after the `nod` correction handoff at:

```text
Cases: 2670 total, 2600 proven/delegated, 22 blocked, 48 excluded
Actionable completion: 2600/2622 = 99.2%
```

The `notitle` slice is merged to `main` in PR #1049 (`05170c90b`). The
post-merge frontier is:

```text
Cases: 2679 total, 2609 proven/delegated, 22 blocked, 48 excluded
Actionable completion: 2609/2631 = 99.2%
```

The special-procedure inventory remains exhausted. The intentionally blocked
row `objmagic.sleep-entry-gates` remains blocked; this session did not alter
the cast-sleep vehicle or its sharp note. The interpreter sweep continues.

## Slice proof

The next source-order unclaimed row was `src/interpreter.c:580`:

```c
{ "notitle"  , POS_DEAD    , do_wizutil  , LVL_GOD, SCMD_NOTITLE },
```

The C call path was read first: `do_wizutil` in `src/act.wizard.c:2077-2140`,
including its `one_argument` target parse at `src/act.wizard.c:2082` and the
standalone parser in `src/interpreter.c:1265-1283`. Reachable player-facing
branches are the LVL_GOD/POS_DEAD entry gate, no-argument response, missing
target, visible-NPC refusal, higher-immortal guard shared with the wizutil
family, mortal target toggle-on and toggle-off, actor-only acknowledgement,
and the C fill-word/trailing-token boundary. The target state transition is
`PLR_NOTITLE`; the command emits no victim or room bytes.

Added:

- `cmd/dp-oracle-diff/scenarios/notitle-depth.txt`
- `pkg/session/notitle_depth_test.go`
- `docs/fidelity/depth/notitle.tsv` (9 rows)

The initial seed-1 run was RED on the final fill-word probe: C skipped the
leading `the` and toggled `Godmortal`, while Go looked up `the` and emitted
`There is no such player.` The confirmed Go divergence was fixed by rejoining
the tokenized arguments before the shared `wizutilDispatch` applies
`game.OneArgument`. The corrected scenario's `--show-oracle --seed 1` run
confirmed the intended C blocks, and seeds 1, 2, 3, 5, and 8 all returned
`no normalized divergence`.

Focused tests passed, and all local gates passed:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
```

The feature branch was `glm/depth-notitle`, commit `28fd0c430`; PR #1049 was
merged only after hosted lint, security, and test checks were green. Hosted
build and deploy were skipped by the workflow. No source or C-oracle files
were edited.

## Fidelity rules applied

- R1: preserved exact CRLF bytes, branch messages, and actor-only toggle
  acknowledgement.
- R2: preserved the registered `notitle` command and its LVL_GOD/POS_DEAD
  command surface.
- R3: repeated the complete live vehicle across deterministic seeds 1, 2, 3,
  5, and 8.
- R4: changed Go only after the seed-1 transcript proved the parser mismatch;
  no speculative behavior was added.
- R5b/R5c: reused the already-proven shared wizutil target gates and audited
  the affected parser call class rather than duplicating unrelated behavior.
- R5e: verified the actual interpreter registration, `do_wizutil` call path,
  and C `one_argument` implementation.

## Next queue item

The fresh source-order sweep confirms `noogie` and `nudge`/`nuzzle` are
already covered by the existing social-family claim, and `nowiz` is covered
by the generic-toggle family. The next genuinely unclaimed command family is
`order` at `src/interpreter.c:586`:

```c
{ "order"    , POS_RESTING , do_order    , 1, 0 },
```

Map `do_order`'s actual C call path first, then use branch
`glm/depth-order`, one family PR, and one dated handoff. Do not re-pick
`notitle`, `newbie`, `nibble`, or `nod`; preserve the existing social and
generic-toggle claims.
