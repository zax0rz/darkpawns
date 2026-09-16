# Phase 7 OLC REDIT handoff — 2026-09-16

Branch: `glm/phase7-olc-redit`

Draft PR: [#1473](https://github.com/zax0rz/darkpawns/pull/1473)

Base: `origin/main` at `19abd4cbb0aed` (includes merged tedit PR #1469 and
the archived blog PR #1472).

Commits:

- `9f8b2b1c1` — `feat: implement faithful redit room editor`
- `ed03d281b` — `chore: refresh redit divergence pins`

## Delivered bounded slice

This slice ports the reachable C `redit` path and only the shared support it
needs:

- C entry gate, zone lookup/authorization, duplicate-editor guard, and
  no-argument current-room selection;
- descriptor-owned working copies with C menu/prompt/error bytes;
- room name/description, room flags, sector, all six exits, extra
  descriptions, room copy, and existing-room scripts;
- improved editor save/abort/help/blank-line behavior for room strings;
- internal save confirmation, discard, disconnect cleanup, and activity
  audience transitions;
- new-room insertion, parsed-world publication, C-format zone disk save, and
  parse-back restart proof;
- copy-on-write publication for the existing room web-admin setters so active
  snapshots cannot be mutated underneath readers.

The C transition inventory is [redit-transition.md](../redit-transition.md).
The depth manifest marks the 19 proven bounded cases in [redit.tsv](../redit.tsv).
Other OLC families (`medit`, `oedit`, `zedit`, `sedit`) remain outside this
slice and remain explicitly blocked.

## Proofs and gates

- `redit-entry-depth`: clean at seeds 1, 2, 3, 5, and 8.
- `redit-boundaries-depth`: clean at seeds 1, 2, 3, 5, and 8, including a
  co-located observer for start/stop audience behavior.
- `go test -race ./pkg/game ./pkg/session -run 'TestRedit|TestReditWorldPublication'`: clean.
- `go build ./...`: clean.
- `go vet ./...`: clean.
- `go test ./...`: clean.
- `go test ./pkg/game/...`: clean.
- `PATH=/usr/local/go/bin:$PATH golangci-lint run ./...`: clean.
- `make check-fmt`: clean.
- `make fidelity-depth`: clean; 4,873 total cases, 4,758 proven/delegated,
  64 blocked, 51 excluded.
- `make expected-divergences-check`: clean; stale redit expected-divergence
  rows and pins were removed after the focused oracle proof.

`make oracle-regression` was intentionally not run, per the bounded Phase 7
objective. No merge, deployment, full census, or other OLC/admin work was
performed.

## Review frontier

Review the C transition table and the copy-on-write publication boundary first.
If review changes player-visible bytes, rerun both redit vehicles across the
five seeds before updating the manifest. After review corrections, the next
owner may run the required broader census and decide whether this bounded
slice is ready for merge; this handoff does not authorize either merge or
deployment.
