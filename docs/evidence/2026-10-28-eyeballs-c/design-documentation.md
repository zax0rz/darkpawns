# Equipment and inventory design record

2026-10-28. Follow brand-voice §8.5. This is an ordinary extension of the
existing Haunted Paperback console. No visual-system change was requested or
needed. `admin-ui/DESIGN.md`, `admin-ui/PRODUCT.md`, and
`admin-ui/.impeccable/design.json` were read and left unchanged.

## Evidence checked

The named source files exist at `admin-ui/src/pages/PlayersPage.tsx`,
`admin-ui/src/components/PlayerItemsEditor.tsx`, and
`admin-ui/src/lib/playerItems.ts`. The direction contract exists at
`admin-ui/.impeccable/surfaces/admin-ui-src-pages-playerspage-tsx.md`.
This pass sampled those files, `admin-ui/src/index.css`, and the shared button,
input, and dialog components. It also read brand-voice §8.5 and the evidence
[README](README.md).

Visual comparison used [desktop equipment](equipment-desktop.jpg),
[mobile inventory](inventory-mobile.jpg), and the [mobile item picker](picker-mobile.jpg).
These are the retained browser captures described in the README, using synthetic
fixtures. They are evidence, not shipping raster assets. This documentation pass
did not run a browser or independently exercise the interactions. The parent's
fresh finish review reported **ship**, with no material fixes at its screenshot
and source-sample scope; that report is distinct from this documentation check.

## Fit with the incumbent system

- **Palette:** the source retains paper (`#EFE7D6`), deeper paper (`#E5DAC1`),
  charcoal ink (`#1A1614`), muted ink (`#56504A`), and sparse oxblood (`#A8201A`).
  The sampled views use the two paper tones and ink rules. Primary Save and focus
  use oxblood; item rows and secondary actions remain quiet.
- **Type:** global headings use DM Serif Display; reading and controls use Source
  Serif 4; item numbers and stack counts use JetBrains Mono. Equipment section
  titles use the existing 1.25rem scale, controls 0.875rem, and item metadata
  0.75rem. No new font or type scale was introduced.
- **Form and depth:** the extension uses shared buttons, inputs, tabs, and Radix
  dialogs. Shared controls retain 2px corners, ink borders, oxblood focus, and
  disabled opacity. The picker retains the ink-at-45% scrim and paper panel.
  The sampled extension adds no shadows, gradients, blur, or decorative cards,
  consistent with the Flat-Ink Rule.
- **Layout:** equipment renders every supplied wear slot, including Empty rows,
  as a ruled list with two columns from the existing 640px breakpoint and one
  below it. Inventory uses ruled rows, wrapping actions, and indentation of
  1.25rem per level, capped visually at four levels. Parent names provide another
  cue for nested items. These are local composition choices, not new global
  tokens. The mobile captures show the intended stacked layout and bounded
  picker; no new overflow or interaction proof is asserted here.
- **Save and copy:** the source places one sticky Save/Discard strip above stats
  or items, with explicit pending edit counts. Both drafts are passed to the same
  save call. Flags, identity, and password retain separate actions and explain
  when pending changes prevent them. The picker distinguishes carried items
  from new copies, labels name/vnum search, and explains that changes wait for
  Save. Removal says the item and its contents will be deleted when saved and
  explains how to bring them back beforehand. This follows §8.5's emphasis on
  concrete outcomes and plain verbs. Source inspection supports those controls;
  the README supplies the reported browser exercise and save verification.

## Drift left outside this extension

The existing product file is prose rather than a current structured product
schema. Existing design prose includes a separate “Character activity detail”
section beyond the canonical eight-section design format; the sidecar's overview
and component samples do not capture every later surface detail. These are
incumbent documentation limitations, not authority to rewrite the system.

The parent also reported tooling notices for stale sidecar metadata,
surface-path resolution, and an unset unrelated landing build path. This pass
confirmed the actual target source and contract paths above; it did not reproduce
or repair those tooling notices. No token, component rule, or prohibition was
added to legitimize metadata drift. The One Oxblood Rule and Flat-Ink Rule remain
the existing authority. No new visual defect was canonized.
