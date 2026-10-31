---
title: "Edit objects"
description: "Edit an object prototype in your building zone with the admin console, commit it to the running world, and save the zone file."
type: guide
audience: builders
verified: v0.1.7
---

Use webOLC in the admin console to build or tune an object prototype in your assigned zone. Changes stay in your draft until you commit them to the live world and write the zone's `.obj` file.

## Before you start

- Sign in at `/admin/` with a character assigned to build in the zone.
- Find the item in **Objects**, or grab the next free [vnum](/docs/reference/glossary/#vnum) on its zone page.

:::caution
**Commit draft** updates the prototype in the running world. **Save zone file** writes that change to disk so it survives a restart. The **Live script controls** apply immediately, and **Discard** won't undo them.
:::

## Edit an object

1. Open **Objects** and select the item you want to change.
2. Choose **Edit** (or open a free [vnum](/docs/reference/glossary/#vnum) from the zone page to create a new item). Opening the editor claims the object and starts your draft.
3. Edit its keywords, short and long descriptions, action description, item type, flags, wear slots, weight, cost, values, stat applies, or extra descriptions.

![The object editor's fields for a synthetic steel sword.](/media/docs/admin-object-editor.png)

4. Review your changes and choose **Commit draft**.

The object prototype updates in the running world and releases your claim.

5. Choose **Save zone file**, or save from [Workshop & saves](/admin/workshop).

:::tip[Load chances and weapon rules from help OEDIT]
In Dark Pawns, `cost-per-day` controls how often an item loads on zone reset (`help OEDIT`). If `cost-per-day` is above `10000`, the amount over `10000` is its percentage chance to load: `10100` loads 100% of the time (use `10100` for shopkeeper stock like bread so the baker never runs dry), while `10050` loads 50% of the time. For weapons, backstabbing requires type `pierce`, and clerics cannot wield slashing weapons even without a `!CLERIC` flag.
:::

## Did it work?

The object detail page shows your committed values, and saving the zone file clears the zone from pending saves in **Workshop & saves**.

## If it didn't work

- **The object is already claimed.** Another builder has a draft open; wait for them to finish.
- **Save zone file is disabled.** Commit your object draft first.
- **Edit buttons are unavailable.** Check your zone permissions in the [builder guide](/admin/workshop/help).

## Related

- [Edit rooms](/docs/build/edit-rooms/)
- [Edit mobs](/docs/build/edit-mobs/)
- [Edit zones](/docs/build/edit-zones/)
