# Eyeballs C: equipment and inventory

2026-10-28. Follow brand-voice §8.5. This extends the saved-character editor;
network/ports (#10) remain with GLM. Zach confirmed one Save for pending stat
and item edits. Flags, identity and password retain their existing actions.

## Browser proof

Chrome was not connected in this session. CUA used the in-app browser and the
actual built admin bundle against a disposable `adminpreview` fixture. Every
name and item is synthetic. No production credentials, character or world data
were used. Native browser JPEG captures are evidence, not shipping UI assets.

| View | Desktop | Mobile |
| --- | --- | --- |
| All wear slots, worn or empty | [top](equipment-desktop.jpg) | [top](equipment-mobile.jpg) |
| Nested inventory | [tree](inventory-desktop.jpg) | [tree](inventory-mobile.jpg) |
| Search and choose an item | [carried item](picker-desktop.jpg) | [new copy by vnum](picker-mobile.jpg) |
| Empty search | | [no matches](picker-empty-mobile.jpg) |
| Remove confirmation | | [pending deletion](remove-mobile.jpg) |
| Revision conflict, draft kept | | [conflict](conflict-mobile.jpg) |
| Discard and reload | [confirmation](reload-desktop.jpg) | |

[Before](before.jpg) captures the incumbent editor at the browser's normal
972 × 828 viewport. Desktop evidence is 1440 × 1000; mobile is 390 × 844. The
console scrolls its fixed-height main area, so equipment and inventory have
separate captures. The document and main area fit at 390px and the 1024px sidebar
breakpoint without horizontal overflow. Temporary viewport overrides are reset.

The real UI was used to:

- Equip a carried sword after name search, change Health, save both together,
  reload and verify the equipped slot.
- Give a new ring by vnum; cancel removal, then confirm removal. A net-zero
  give/remove save cleared the draft instead of replaying it.
- Equip a satchel containing a pouch and a stack of stones, save, unequip and
  save again; descendants followed the container. Collapse hid descendants.
- Take a nested pouch out into carried inventory, then exercise reload Cancel
  (draft retained) and Discard and reload (draft cleared).
- Save a change in a second editor, then refuse the first editor's stale save
  while keeping its pending item edit visible.
- Check the mobile picker, empty results, focus and removal confirmation. The
  final browser error log was empty.

The old native dirty-reload confirmation blocked browser automation. Zach
dismissed it; the updated shared Radix confirmation was then tested through
Cancel and Discard and reload. No bypass or raw browser protocol was used.

## Save boundary and tests

The existing PATCH operation accepts at most 100 ordered item edits alongside
numeric fields. Indices address each preceding operation's result. Original
instance state remains on the server. Every operation is checked against a copy
under the existing offline reservation and revision guard; one database
transaction writes character and object saves. A failed operation writes neither.
The old single-item endpoint still works. Slots remain staff assignments, with
occupied-slot and stack checks, rather than new in-game wear rules.

`TestSavedPlayerItemBatch` exercises all 22 slots, container labels, stat/item
rollback after a later failed grant, ordered reindexing, nested instance-state
preservation, object-save persistence, stale and connected-character refusal,
filled slots, stacks, invalid actions, the 100-edit limit, subtree removal and
audit recording. The existing editor suite exercises permissions and prior
single-item behavior. The draft projection test covers subtree remapping,
nonmutation, stacked/filled slot errors, sequential grant/move/remove and search.

## Verification

- `make fmt`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...`: zero issues
- `make deadcode-ratchet`: 52 baselined, zero new
- `make admin-api`; admin `npm run build`, `npm run lint`
- `node --experimental-strip-types admin-ui/scripts/test-player-items.mjs`
- Focused race tests for saved-player item batching and the existing editor
- Pinned gosec v2.29.0 HIGH scan
- `make build-site` including voice, Markdown, inventory, palette/wordmark checks
- Impeccable manual detector (one pass): `[]`

Each Go gate ran separately. No engine command, RNG, player-facing output or
save format changed (R1–R4). No census was started: this is an admin-only
extension, and transport/storage tests are the relevant proof boundary (R5f).
Claude owns the final combined release gate for the reviewed batch.

## Decisions for Zach

The requested one-Save model covers stats and items. Other editor actions stay
separate and are disabled while those edits are pending. An item assignment may
use any single item, matching the existing staff editor. Filled slots require
Unequip first; stacks can be carried or removed, but not assigned to one slot.
Giving creates one new copy; moving uses the existing saved instance. Container
contents are visible and can be taken out, but adding an item inside a container
is outside this pass.

The design contract is in the Players surface brief. Existing design tokens and
components remain authoritative. Impeccable reported older product/schema and
sidecar metadata, plus surface-path resolution drift; the named source files
exist. A separate `init`/`document` refresh can reconcile the metadata. The
unrelated landing-page build-path preference was not changed.

## Finish review

A fresh Impeccable finish reviewer returned **ship**, with no material fixes.
It checked all ten required captures, the direction contract, design system and
source samples; it did not claim a complete backend audit. TYPE, MATERIAL,
GROUND, THESIS, STORY, FIRST VIEWPORT, FORM, protected actions and voice matched.
A separate [documentation check](design-documentation.md) preserved the incumbent
design files and recorded pre-existing metadata drift without repairing it.

The preview was stopped by its exact PID after QA. No stash or process-name
kills were used. Claude's review in Zach's session remains the merge boundary.
