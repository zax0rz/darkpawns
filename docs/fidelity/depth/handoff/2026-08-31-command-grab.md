# Depth handoff — 2026-08-31 — `grab`

## Frontier and queue position

- Started from clean `main` at `59eaaa855` after merging the `group`
  handoff, ran `git pull --ff-only`, confirmed `make fidelity-depth`, and
  reread `docs/fidelity/DEPTH_TESTING.md` plus the newest prior handoff,
  `2026-08-31-command-group.md`.
- The starting frontier was 1,909 total, with 1,852 proven/delegated, 16
  blocked, and 41 excluded. The dedicated `grab` manifest adds 11
  proven/delegated cases. Immediately after the slice, the frontier was
  1,920 total, with 1,863 proven/delegated, 16 blocked, and 41 excluded;
  actionable completion was 1,863/1,879 (99.1%).
- During this session, the previously open glare evidence PR #896 acquired
  green hosted test/lint/security checks and was merged without repicking the
  slice. The current post-housekeeping frontier is 1,930 total, with 1,873
  proven/delegated, 16 blocked, and 41 excluded; actionable completion is
  1,873/1,889 (99.2%).
- A fresh source-order audit confirms `grab` at `src/interpreter.c:472` is
  covered. `grats` at line 473 is covered by `channels.tsv`; the next
  un-manifested command-table family is `greet` at
  `src/interpreter.c:474`. The next session must return to clean `main`,
  pull, rerun the frontier check, reread this handoff, and begin `greet`.

## C call path and branch inventory

`src/interpreter.c:472` registers `grab` as `POS_RESTING`, minimum level 0,
and `do_grab`; `hold` at line 497 shares the handler but has minimum level 1.
The C handler path is `src/act.item.c:1685-1709`:

- `one_argument` skips fill words, lowercases the selected token, and ignores
  trailing input. Empty input emits `Hold what?`.
- `get_obj_in_list_vis(ch, arg, ch->carrying)` restricts lookup to visible
  carried objects. A miss emits `You don't seem to have <an> <arg>.`.
- A carried object must have `ITEM_WEAR_HOLD`, unless its type is wand,
  staff, scroll, or potion; otherwise C emits `You can't hold that.`.
- The accepted object enters `perform_wear(ch, obj, WEAR_HOLD)`. The shared
  path uses `ITEM_WEAR_TAKE`, rejects an occupied hold slot with
  `You're already holding something.`, rejects a held object while a
  two-handed weapon is wielded, and on success calls `wear_message` with the
  exact actor and room `grab` lines before `equip_char`.
- Shared resolver, slot, and two-handed branches are recorded as delegated
  or unit-proven rather than duplicated; anti-alignment/class checks remain
  part of the shared `perform_wear` owner boundary.

The registered C command gate and the Go `DoGrab` path were compared directly
against `pkg/session/cmd_inventory.go`, `pkg/game/item_equipment.go`,
`pkg/game/item_transfer.go`, and the typed equipment/location helpers. No Go
behavior change was confirmed or needed.

## Coverage proof

The unchanged-main `grab-depth --seed 1 --show-oracle` run was GREEN and
showed the intended C blocks for empty input, a numbered inventory miss, a
numbered successful staff selection with trailing input ignored, an occupied
hold slot, non-holdable bread and tunic, and successful actor/room broadcasts.
The same vehicle reported no normalized divergence for seeds
`1,2,3,5,8`.

The fixture used two force-loaded registered `20302` war gongs (staffs), one
bread, and one tunic in room 8162. This reaches C's explicit consumable
exception and the numbered carried-object resolver without editing the source
world or oracle tree.

The slice follows R1/R2/R3/R4/R5e and R5c: C player-facing bytes and the
distinct command surface remain authoritative, deterministic state and
audiences are proven across five seeds, no behavior was invented, the actual
`do_grab` call path was traced before recording coverage, and shared owners
were delegated rather than duplicated.

## Changes and gates

- Added `cmd/dp-oracle-diff/scenarios/grab-depth.txt` with the direct branch
  vehicle and named room peer.
- Added `docs/fidelity/depth/grab.tsv` with 11 explicit rows.
- Added `pkg/session/grab_test.go` to pin the C `grab` entry gate.
- No implementation change was made: the existing Go path matched the C
  oracle on the RED-or-GREEN vehicle.
- Local gates passed: `make fidelity-depth` — 1,920 total /
  1,863 proven-or-delegated / 16 blocked / 41 excluded; `go build ./...`,
  `go vet ./...`, `go test ./...`, `golangci-lint run ./...` (0 issues),
  `gofumpt -l .`, and `git diff --check`.

PR #905 (`glm/depth-grab`) was self-merged only after hosted `lint`,
`security`, and `test` checks were green following the one permitted
workflow retry. Its `build-and-push` and `deploy` jobs were skipped by policy;
merge commit `4ccc542a0` is on `main`.

The earlier glare PR #896 was left open when its test was initially reported
as the unrelated retry-based `pkg/spells/TestMagAffects_Sleep`; after its
hosted checks later reported green, it was merged as housekeeping in this
session (merge commit `eabb2c253`).
