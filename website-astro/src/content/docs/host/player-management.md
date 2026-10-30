---
title: "Saved character management"
description: "Edit a disconnected character’s equipment, inventory, flags, name, and password in the admin console, with clear save and delete boundaries."
type: guide
audience: operators
verified: v0.1.7
---

The **Players** page manages saved characters. A character must be fully
disconnected before you can edit it. This includes linkdead characters and
characters at the login screen. The editor refuses online changes so a late
save can't overwrite them.

## Equipment and inventory

Choose a character and open **Equipment & inventory**. Every wear slot appears,
including empty ones. Choose **Equip** beside an empty slot to move a carried
item there or add a **New copy**. Search by name, keywords, or vnum. Choose
**Give item** to add a new copy to carried inventory.

Carried containers show their contents underneath. **Hide contents** folds a bag;
**Show contents** opens it again. A container and everything inside it travel
together when you equip or unequip it. To take one item out, choose **Carried
inventory** from that item's destination menu and choose **Take out**.

**Unequip** moves an item back to carried inventory. **Remove** queues deletion
of the item and anything inside it. Removing an item can be undone with
**Discard changes** until you save. After saving, it can't be undone here.

Stat and item edits wait for **Save changes**. You can switch between those two
tabs and save them together. The save bar shows how many edits are waiting.
**Discard changes** restores both tabs to the last loaded character. Save or
discard first before changing flags, names, or passwords.

If the character changed elsewhere or connected while you were editing, Save
refuses the whole change and keeps your draft open. Review it before choosing
**Reload character**; reloading discards the draft after confirmation. A failed
item edit never leaves some stat or item changes saved and others unsaved.

Staff assignments don't check whether a character could wear the item in game.
Choose suitable items for each slot. A stack can't fill a single slot; a filled
slot must be unequipped first. You can queue up to 100 item edits per save, with
at most 2,000 saved item entries per character. **Not saved after logout** is a
shared item definition, changed from **Objects**, rather than a per-copy switch.

## Rename

Open **Players**, choose a character, and select **Identity**. Enter a new name
that follows the character creation rules. Names already in use, reserved for
staff, or blocked by the invalid-name list are rejected.

The player must be offline, including from linkdead sessions and the login
screen. The rename changes the name throughout the character's saved records
and board posts. Mail, clan membership, and house access stay with the same
character.

## Delete

Choose **Delete character…** on the **Identity** tab and review the confirmation.
For characters at level 31 or higher, type the character's name to confirm.
You can't delete your own character or a level 40 character. The delete action
works at every other level, including for frozen characters. The character must
be fully disconnected.

Deletes this character. Their saved items and aliases are removed, and the name
becomes available again. This can't be undone.

Old mail and house access stay with the old character. Clan membership is
cleared, and old board posts remain as history. A new character using the same
name starts fresh and doesn't inherit the old mail or house.

## Flags

Open **Flags** to turn character switches on or off. Each switch saves right
away, and the level shown beside it's the minimum staff level needed to change
it. The **Deleted** status is read-only; use **Delete** on the **Identity** tab
to remove a character.

The **Cryo-saved** switch changes the character's cryo marker only. It doesn't
store or restore saved items. An item's save setting is shared
by every copy; open **Objects** to change its definition.

![The player editor moves from character stats to equipment and flags.](/media/docs/admin-player-editor.gif)

## How it works

Stat and item edits are revision-checked and saved together. The server writes
the updated character and item records in one SQLite transaction. Renaming
preserves the character's stable ID; mail and house access are attached to that
ID, while board headings and aliases are updated with the new name. The admin
delete action is separate from the in-game self-delete command. It doesn't
change game commands, wear rules, random draws, or the save format.

For the original character creation and self-delete rules, see
[`src/interpreter.c`](https://github.com/zax0rz/darkpawns/blob/main/src/interpreter.c).
Mail and house ownership use character IDs in [`src/mail.h`](https://github.com/zax0rz/darkpawns/blob/main/src/mail.h)
and [`src/house.h`](https://github.com/zax0rz/darkpawns/blob/main/src/house.h).
