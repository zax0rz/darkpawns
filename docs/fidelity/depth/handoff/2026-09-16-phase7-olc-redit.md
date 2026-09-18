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

## Correction round — 2026-09-18

The first review returned four findings; all four are addressed here with
focused validation only (no census, no merge, no deployment).

1. **Menu colors.** All redit menus now interleave the `get_char_cols`
   strings (`grn/nrm/cyn/yel`; ANSI at color level ≥ 2, empty below) exactly
   as the C format strings place them, including `redit_disp_sector_menu`'s
   stale-global quirk. Proof is raw, not normalized: a new `keep-ansi`
   scenario mode compares probe blocks with ANSI intact
   (`NormalizeKeepANSI`), and two new vehicles — `redit-menu-color-on`
   (creation answers Y to the ANSI question) and `redit-menu-color-off` (N)
   — are clean at seeds 1, 2, 3, 5, 8. `-show-oracle` confirms the compared
   blocks really contain the escapes (color-on) and really are plain
   (color-off), so neither proof is vacuous. Because menu bytes changed,
   `redit-entry-depth` and `redit-boundaries-depth` were rerun at the same
   five seeds. Unit proof: `TestReditMenuColorsMatchGetCharCols`
   (byte-exact main/exit menus at levels 0/2/3).

2. **Atomic duplicate-editor admission.** The duplicate gate is now a
   manager-owned reservation (`claimRoomEdit`/`releaseRoomEdit`) instead of a
   check-then-install session scan, so two sessions racing through `cmdRedit`
   cannot both be admitted; C's message order (duplicate before zone/auth
   refusal) is preserved by claiming first and releasing on every refusal
   path. The reservation is released on all three exits: confirm-save
   (either answer) and disconnect cleanup. The save path keeps the read-only
   peek. Unit proof under `-race`:
   `TestReditConcurrentEntryAdmitsExactlyOneEditor` (8 concurrent contenders
   → exactly one editor, 7 refusals, re-admission after both disconnect and
   clean-quit releases).

3. **Routine mutations vs structural insertion.** `SetExitInfo` (door
   commands, zone-reset D, scripting) and `SetRoomFlagBit` (search's secret
   mark) now take a per-room copy-on-write path (`mutateRoom`) that clones
   one room, repoints the map entry, and republishes — no parsed-array copy,
   no rooms-map rebuild, no sorted-vnum resort, which stay on the
   `CommitEditedRoom` structural path. The split also fixes a latent hazard:
   a structural commit previously repointed every room into the fresh parsed
   array, silently reverting unrelated rooms' runtime door state (C never
   does); commits now preserve live room pointers and swap only the edited
   room. Measured on the real world (`lib/world`, ~10k rooms,
   `world_redit_bench_test.go`): door operation ≈ 246 µs / 297 KB / 40
   allocs; structural commit ≈ 1.28 ms / 1.85 MB / 75 allocs. The residual
   door-op cost is `SnapshotManager.Publish`'s full map copy — the price of
   the lock-free reader contract, not the rebuild. Unit proofs:
   `TestRoutineMutationSkipsStructuralRebuild`,
   `TestRoutineMutationConcurrentDoorWritersAndReaders` (`-race`).

4. **Real restart proof.** `TestReditSaveSurvivesRealRestart` boots a world
   from a disposable lib tree, edits an existing room through the full
   dialogue (name, description, exit target/keywords/key/pickproof flags,
   extra description), commits, saves the zone, then boots a FRESH world
   through the production loader (`parser.ParseWorld` + `game.NewWorld`,
   the `cmd/server` boot path) from the saved directory. A new session
   revisits the room: look's state payload plus direct snapshot asserts
   cover the edited fields and the untouched exit description. The earlier
   parse-back-only test remains as the new-room disk check.

Gates rerun after the corrections: `go build ./...`, `go vet ./...`,
`go test ./...`, `golangci-lint run ./...` (0 issues), `make check-fmt`,
`make fidelity-depth` (4,879 cases, clean), and the focused race set
(`TestRedit*`, `TestRoutineMutation*`) — all clean. The PR remains in draft
and stopped for review again, per the correction-round protocol.
