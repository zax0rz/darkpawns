---
version: 1
slug: "admin-ui-src-pages-playerspage-tsx"
primary_target: "admin-ui/src/pages/PlayersPage.tsx"
related_targets: ["admin-ui/src/components/PlayerItemsEditor.tsx"]
---

# Character equipment and inventory

Mode: Operate. Follow brand-voice §8.5. Extend the existing saved-character
editor and Haunted Paperback controls. Zach confirmed one Save for pending
stat and item edits; flags, identity and password keep their existing actions.

## Direction contract

THESIS: See every wear slot and the contents of carried containers, then make
specific changes without guessing item numbers or losing instance state.

OWN-WORLD: Inherit cream, charcoal rules, sparse oxblood, serif controls and
monospace item numbers. Use the existing buttons, tabs, inputs and dialogs.

STORY: Inspect worn and empty slots, search by name or vnum, choose a carried
item or a fresh copy, review pending changes, then save or discard together.

FIRST VIEWPORT: Character heading and tabs stay in place. A compact save strip
shows pending work. Equipment is a dense ruled two-column slot list on desktop,
one column on mobile. Inventory follows as an indented, expandable tree.
A focused picker opens from Give item or an empty slot; search results name the
item and number, and explicitly distinguish existing items from new copies.

FORM: Local extension, code-led; no concept roll applies. Empty slots are real
rows. Containers move with contents. Staff assignments retain existing slot
semantics rather than inventing game wear validation. One revision protects the
whole stat/item transaction; failed saves preserve the local draft.

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance
