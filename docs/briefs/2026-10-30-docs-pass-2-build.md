# Docs pass 2: Build section evidence

## Page purpose and plan

- `build/edit-rooms.md`: for a builder editing a room; focused capture `admin-room-editor.png`.
- `build/edit-objects.md`: for a builder editing an object prototype; focused capture `admin-object-editor.png`.
- `build/edit-mobs.md`: for a builder editing a mob prototype; focused capture `admin-mob-editor.png`.
- `build/edit-zones.md`: for a builder editing zone settings or one room's reset commands; focused capture `admin-zone-editor.png`.
- `build/edit-lua-scripts.md`: for a builder editing Lua files in the separate Workshop editor; focused capture `admin-lua-scripts.png`.
- `build/local-server.md`: developer flow for building and connecting to a disposable checkout; Mermaid diagram is clearer than a terminal image for the sequence.

## Running UI evidence

Verified against a disposable `TestConsolePreview` server and the Vite admin UI at ports 4350 and 5187 with Playwright. No production server was used. Relevant source and screen evidence:

- `admin-ui/src/App.tsx` registers `/admin/game/rooms/:vnum/edit`, `/admin/game/objects/:vnum/edit`, `/admin/game/mobs/:vnum/edit`, `/admin/game/zones/:zone/edit`, and `/admin/workshop/scripts`.
- `admin-ui/src/pages/RoomEditorPage.tsx` and `admin-ui/src/pages/EntityEditorPage.tsx` open a draft when their editor page loads. `pkg/admin/olc_entity_write.go` claims the corresponding resource and opens or recovers the owner's draft.
- `admin-ui/src/components/olc/ClaimSaveFrame.tsx` shows **Discard**, **Commit draft**, lease and claim status, and a separate **Save zone file** action. That save stays disabled until there are committed changes pending for its zone. Committing updates the live world and releases the claim; the save endpoint writes the pending zone data (`pkg/admin/olc_save.go`).
- `admin-ui/src/pages/ZoneEditorPage.tsx` edits zone settings. On this route the whole-zone reset timeline is read-only. `admin-ui/src/pages/RoomDetailPage.tsx` links to **Edit room reset commands**; the room-anchored zone editor exposes that room's timeline for editing and keeps the whole-zone view read-only.
- `admin-ui/src/pages/FileEditorPage.tsx` labels the Lua editor separately, displays **Applies immediately**, and says the next trigger runs the saved bytes. Save is parse-checked by the file-edit API; there is no OLC draft or undo. The running fixture's `guard.lua` was listed as selected.
- Room, object, mob, zone, and Lua editor fields were read from the running UI, not inferred from screenshots or memory. The real preview permits one open OLC draft per owner, so the capture script discards the unchanged synthetic draft and calls the fixture-only `/preview-reset` after each OLC image. Captures never commit edits.
- The Workshop UI is a real file editor, not an OLC script-draft page. The guide states that boundary explicitly.

## Captures

The captures come from `make media`'s `admin-ui/scripts/media-capture.mjs` flow. Crop targets avoid the live lease clock; the zone editor crop is limited to the stable zone settings section. The object and mob images crop the stable top of their field sections while highlighting the relevant field. Alt text is kept beside the capture descriptors and included in `docs/images/generated/manifest.json`.

Two consecutive admin-only capture runs produced the same SHA-256 for each focused Build capture:

- `admin-room-editor.png`: `da6d47df4c675ad00cdcf7f1e3956e5ac30dc6053d54f250fb8d51a1adf9bcdd`
- `admin-object-editor.png`: `47058cad3ca12a3d6c7a891726a84d04a6d20f126bb6ce50b4fe039f07b9e7f5`
- `admin-mob-editor.png`: `0889e55bdc1198e14e567df7e84aaec62ababe9ee6e208704303a21839999897`
- `admin-zone-editor.png`: `774a8e756558498b6bdc278291e1a371310d3a882f6651582367e7e8139bd45d`
- `admin-lua-scripts.png`: `a3645fce231a19f3f58a63407e546f9fee08a91de7806182edb68898dcf2a21f`

The routes, preview data and image crops are confined to synthetic entities: First Room, a steel sword, a guard, Test Zone, and `scripts/guard.lua`.

The Host configuration guide now includes `admin-configuration.png` next to the setting-mode explanation. Its source alt text is: “The Sessions & limits configuration card in the local checkout preview, with editable login and connection limits.” The capture targets the actual **Sessions & limits** card on `/admin/staff/settings`. The Playwright descriptor waits for the settings card, then verifies **Read-only on this server** and an enabled **Login idle timeout (s)** input before taking the screenshot. This records the checkout preview's actual state without implying that the Network card is managed or editable. Two admin-only capture runs produced the same SHA-256: `69766008b284f5030faff4392ab34b1a437051f26e23e2d57017ec55ae5a279a`. The descriptor sources `SettingsPage.tsx`, `NetworkCard.tsx`, the shared input/layout styles, admin preview fixture, and settings implementation; root's final full media run will refresh the combined manifest.

## Mudlet capture preference

The Mudlet dock capture keeps the live map but hides its room-information overlay through the native profile preference. The capture-only profile in `scripts/fixtures/mudlet-host.xml` no longer includes `<mapInfoContributor>Short</mapInfoContributor>`. Mudlet's `XMLimport::readMapInfoContributor()` adds each such element to the profile's enabled contributor set, and `XMLexport` writes enabled contributors back as these elements. The built-in `Short` contributor formats a room as `name / ID (area)`, which produced the unwanted player-facing room number. The independent `mShowRoomIDs="no"` attribute controls map room labels and does not disable this information overlay. Mudlet also provides the documented `disableMapInfo("Short")` API (available since 4.11), but the saved profile setting avoids adding any capture-time Lua behavior.

Verified against Mudlet's upstream source: [`XMLimport.cpp`](https://github.com/Mudlet/Mudlet/blob/development/src/XMLimport.cpp) (`readMapInfoContributor`), [`XMLexport.cpp`](https://github.com/Mudlet/Mudlet/blob/development/src/XMLexport.cpp) (profile serialization), and [`mapInfoContributorManager.cpp`](https://github.com/Mudlet/Mudlet/blob/development/src/mapInfoContributorManager.cpp) (`shortInfo`). The [Mudlet Mapper Functions manual](https://wiki.mudlet.org/w/Manual:Mapper_Functions) documents `disableMapInfo(label)`.

Two consecutive `python3 scripts/capture_mudlet.py` runs confirmed the UI behavior: the map remained visible and the top room-name/room-ID information panel was absent in both screenshots. Both Mudlet dock images have SHA-256 `63deb30766b559a2234c58d222b8ffbe0305f3ae298f1bb6629d9aa725fc0b6f`; both accompanying `web-client-first-steps.png` images have SHA-256 `53646551f2f23101eb008963fa78be0b40386b549acff934caab44d29c6a31db`. The two pairs are retained under `/home/zach/Archives/docs-pass-2/mudlet-short-overlay/`.
