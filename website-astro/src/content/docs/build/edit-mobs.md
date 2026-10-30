---
title: "Edit mobs"
description: "Edit a mob prototype in your building zone with the admin console, commit it to the running world, and save the zone file."
type: guide
audience: builders
verified: v0.1.7
---

Use webOLC in the admin console to change a [mob](/docs/reference/glossary/#mob)
prototype in your assigned building zone. The editor keeps your changes in a
draft until you commit them.

## Before you start

- Sign in at `/admin/` with a character that can build in the zone.
- Find the mob in **Mobs**, or use the free VNUM on its zone page.

:::caution
Committing changes the running world. Save the zone file afterward to keep the
change across a server restart. The **Live script controls** apply at once, and
**Discard** won’t undo them.
:::

## Edit a mob

1. Open **Mobs** and choose the mob you need to change.
2. Choose **Edit**. For a new mob, open its free [vnum](/docs/reference/glossary/#vnum) from the zone page.
   Opening the editor claims the mob and starts or restores its draft.
3. Change its descriptions, level, hit points, damage, position, race, flags,
   or other available fields.

![The mob editor’s fields for a synthetic guard.](/media/docs/admin-mob-editor.png)

4. Review the changes and choose **Commit draft**.

The mob is updated in the running world, and its claim is released.

5. Choose **Save zone file**. You can also save the zone from
   [Workshop & saves](/admin/workshop).

## Did it work?

The mob detail page shows the committed values. After the file save, the zone
disappears from pending saves in Workshop & saves.

## If it didn’t work

- If the mob is already claimed, wait for its current editor to finish.
- If **Save zone file** is disabled, commit the mob draft first.
- If a control is unavailable, check the account’s actions in the
  [builder guide](/admin/workshop/help).

## Related

- [Edit rooms](/docs/build/edit-rooms/)
- [Edit objects](/docs/build/edit-objects/)
- [Edit zones](/docs/build/edit-zones/)
