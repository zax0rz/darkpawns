---
title: "Edit objects"
description: "Edit an object prototype in your building zone with the admin console, commit it to the running world, and save the zone file."
type: guide
audience: builders
verified: v0.1.7
---

Use webOLC in the admin console to change an object prototype in your assigned
building zone. The editor keeps your changes in a draft until you commit them.

## Before you start

- Sign in at `/admin/` with a character that can build in the zone.
- Find the object in **Objects**, or use the free VNUM on its zone page.

:::caution
Committing changes the running world. Save the zone file afterward to keep the
change across a server restart. The **Live script controls** apply at once, and
**Discard** won’t undo them.
:::

## Edit an object

1. Open **Objects** and choose the object you need to change.
2. Choose **Edit**. For a new object, open its free [vnum](/docs/reference/glossary/#vnum) from the zone page.
   Opening the editor claims the object and starts or restores its draft.
3. Change its keywords, descriptions, item type, flags, wear slots, weight,
   cost, values, applies, or extra descriptions.

![The object editor’s fields for a synthetic steel sword.](/media/docs/admin-object-editor.png)

4. Review the changes and choose **Commit draft**.

The object is updated in the running world, and its claim is released.

5. Choose **Save zone file**. You can also save the zone from
   [Workshop & saves](/admin/workshop).

## Did it work?

The object detail page shows the committed values. After the file save, the
zone disappears from pending saves in Workshop & saves.

## If it didn’t work

- If the object is already claimed, wait for its current editor to finish.
- If **Save zone file** is disabled, commit the object draft first.
- If a control is unavailable, check the account’s actions in the
  [builder guide](/admin/workshop/help).

## Related

- [Edit rooms](/docs/build/edit-rooms/)
- [Edit mobs](/docs/build/edit-mobs/)
- [Edit zones](/docs/build/edit-zones/)
