# Depth handoff — 2026-08-30 — `exits`

## Frontier and queue position

- Started from clean `main` at `0aea77ada` after the merged `escape` slice,
  ran `git pull --ff-only`, confirmed `make fidelity-depth`, and read
  `docs/fidelity/DEPTH_TESTING.md` plus the newest handoff,
  `2026-08-30-command-escape.md`.
- The frontier before this slice was 1,648 total, with 1,593
  proven/delegated, 14 blocked, and 41 excluded. This slice adds nine proven
  cases; the post-slice frontier is 1,657 total, 1,602 proven/delegated, 14
  blocked, and 41 excluded, with actionable completion 1,602/1,616 (99.1%).
- The source-order command gap was `exits`, registered at
  `src/interpreter.c:435`. The next session must rescan the table and all
  manifests from clean `main` before taking the next gap.

## C call path and branch inventory

`src/interpreter.c:435` registers `exits` with `POS_RESTING` and
`do_exits` in `src/act.informative.c:683-725`. The handler first rejects
`AFF_BLIND` with `You can't see a damned thing, you're blind!`. It then scans
all six directions in C order, accepting only exits whose destination is not
`NOWHERE` and whose door is not closed. Immortals receive the direction,
destination vnum, and destination name; mortals receive the destination name
unless that destination is dark and the viewer lacks infravision/holy light,
in which case C emits `Too dark to tell`. An empty visible list emits the
leading-space ` None.` line. The command ignores its argument, and the
POS_RESTING gate makes it callable while asleep.

The clean-main baseline vehicle was GREEN before any change: the existing Go
`cmdExits`/`World.DoExits` path matched the ordinary mortal list, including
trailing arguments. No production divergence was found or fixed. The blind
early return is pinned by `TestDoExitsBlindnessGate`; infravision visibility is
pinned by `TestDoExitsInfravisionSeesDarkDestination`.

## Coverage proof and result

The slice adds nine manifest rows in `docs/fidelity/depth/exits.tsv`:

- ordinary open-exit rendering and ignored arguments;
- no-open rendering and the sleeping POS_RESTING entry;
- closed-exit filtering;
- dark-destination concealment for mortals;
- immortal destination vnum/name rendering;
- blindness early return;
- infravision access to dark destination names.

The live vehicles are `exits-depth`, `exits-no-open`, `exits-filtering`,
`exits-dark`, and `exits-immortal`. Each was run with `--show-oracle` at least
once and all five vehicles are GREEN for seeds 1, 2, 3, 5, and 8. The two
focused tests cover the non-live affect branches without inventing a fragile
spell vehicle. Shared observation/state rendering remains in the existing
look/observation coverage; no new movement or transport behavior is claimed.

No `src/` or `darkpawns-c-oracle/` file was edited. This follows R1/R2/R3/R4
and R5e: the C loop and bytes remain authoritative, direction ordering and
visibility branches were checked against the actual call path, RNG has no
consumer in this handler, and no production behavior was invented. The
blind/infravision branch ownership is explicit under R5b/R5c.

## Gates

The baseline before this evidence-only slice was green. The branch must pass
the complete gates before commit and hosted checks before merge:

- `make fidelity-depth` — expected 1,657 total / 1,602 proven-or-delegated /
  14 blocked / 41 excluded
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...`
- `gofumpt -l .` clean
- `git diff --check` clean

The required hosted PR checks must be green before self-merging. If CI does
not fire, retry once with the prescribed `gh workflow run "Dark Pawns CI/CD"
--ref glm/depth-exits`; if it remains non-green, leave the PR open and move
on.
