# L1 per-file audit: `improved-edit.c` (2026-11-14)

The frozen list `docs/research/frozen-lists-2026-11-11/L1-uncovered-sites.tsv`
holds 25 player-facing `improved-edit.c` rows. This audit names, for every row,
the live depth case that emits the site's bytes. An item settles by a green that
can fail (R5h), or by a C cite showing the arm cannot run.

Vehicle: the C improved editor is reached through REDIT, TEDIT, boards, luaedit
and `string`. The older manifest ranges covered several arms but not every
sibling: `tedit.edit-overflow` and its companion rows now drive the remaining
arms through REDIT's room description (MAX_ROOM_DESC 1024, `src/olc.h:276`).

| C site (`src/improved-edit.c`) | Emitted text / arm | Depth case | Proof |
|---|---|---|---|
| 22 | non-improved help line | `tedit.help-non-improved-unreachable` (**C-unreachable**) | `CONFIG_IMPROVED_EDITOR` is 1 (`src/improved-edit.h:6,8`); line 20 is the live branch |
| 47 | `/c` on an empty buffer | `tedit.clear-on-empty` | `improved-editor-edit-overflow-depth@1` |
| 59 | `/f` empty buffer | `editor.buffer-empty-arms` | `editor-improved-actions-depth` |
| 65 | `/i` empty buffer | `editor.buffer-empty-arms` | `editor-improved-actions-depth` |
| 74 | `/l` empty buffer | `editor.buffer-empty-arms` | `editor-improved-actions-depth` |
| 80 | `/n` empty buffer | `editor.buffer-empty-arms` | `editor-improved-actions-depth` |
| 140 | `/r` invalid format | `tedit.invalid-replace-input` | `tedit-depth` / `tedit-boundaries-depth` |
| 159 (two rows) | `String '%s' not found.` | `tedit.replace`; `tedit.invalid-replace-input` | `tedit-depth` |
| 161 | replacement-overflow ERROR line | `tedit.replace-overflow-in-loop` (**C-unreachable**) | in-loop overflow returns 0 via `if (i <= 0)`, so C prints `String not found` (`src/improved-edit.c:575-625`) |
| 163 | pre-check has no space | `tedit.replace-no-space` | `improved-editor-edit-overflow-depth@1` |
| 175 | delete range invalid | `tedit.invalid-delete-input`; `tedit.delete` | `tedit-depth` / `tedit-boundaries-depth` |
| 193 | delete out of range | `tedit.invalid-delete-input`; `tedit.delete` | `tedit-depth` |
| 241 | normal-list range invalid | `tedit.list-range-invalid` | `improved-editor-edit-overflow-depth@1` |
| 256 | normal-list out of range | `tedit.list-out-of-range` | `improved-editor-edit-overflow-depth@1` |
| 304 | numbered-list range invalid | `tedit.numbered-range-invalid` | `improved-editor-edit-overflow-depth@1` |
| 317 | numbered-list out of range | `tedit.numbered-out-of-range` | `improved-editor-edit-overflow-depth@1` |
| 347 | `/i` missing line number | `tedit.invalid-insert-input` | `tedit-depth` |
| 356 | `/i` NULL buffer | `tedit.insert-empty-buffer` | `improved-editor-edit-overflow-depth@1` |
| 366 | insert out of range | `tedit.insert-out-of-range` | `improved-editor-edit-overflow-depth@1` |
| 373 | insert buffer overflow | `tedit.insert-overflow` | `improved-editor-edit-overflow-depth@1` |
| 395 | `/e` missing line number | `tedit.invalid-edit-input` | `tedit-depth` |
| 404 | `/e` NULL buffer | `tedit.edit-empty-buffer` | `improved-editor-edit-overflow-depth@1` |
| 420 | edit out of range | `tedit.invalid-edit-input` | `tedit-depth` |
| 459 | edit buffer overflow | `tedit.edit-overflow` | `improved-editor-edit-overflow-depth@1` |

The pre-fix gaps this audit found in the existing ranges:

- `tedit.clear` cited 41-47 but only the cleared arm (45) ran; line 47 had no
  vehicle. The range is now 41-46 and `tedit.clear-on-empty` owns 47.
- `tedit.short-replacement` cited 434-500, which swept in PARSE_EDIT's overflow
  arm without exercising it. The range is now 152-164; the overflow arms have
  their own rows.
- The `/l`, `/n`, `/i` and `/e` refusal arms (241, 256, 304, 317, 356, 366, 404)
  were inside ranges whose scenarios never sent the triggering input. They now
  have arm-named rows backed by one scenario (356 itself is C-unreachable).
- Line 161 is unreachable too: a replacement overflow found inside
  `replace_str`'s loop becomes return 0 (`if (i <= 0)`), so C prints
  `String not found`; the only -1 return duplicates the caller's pre-check.
- Line 22 is the non-improved help branch and cannot compile into a reachable
  path, so it settles as C-unreachable rather than as a witness.

The vehicle's oracle transcript is retained at
`docs/fidelity/evidence/l1-2026-11-14/improved-editor-edit-overflow-show.txt`.
