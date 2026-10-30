---
title: "Edit rooms"
description: "Edit a room in your building zone with the admin console: change fields in a draft, commit to the running world, and save the zone file."
type: guide
audience: builders
verified: v0.1.7
---

Use webOLC in the admin console to change a room in your assigned building
zone. You’ll work in a draft first, then commit it to the running world and save
the zone file.

## Before you start

- Sign in at `/admin/` with a character that can build in the zone.
- Know which zone or room you need to edit.

:::caution
**Commit draft** changes the running world. **Save zone file** is a separate
step that keeps the change across a server restart. The room’s **Live script
controls** apply at once, and **Discard** won’t undo them.
:::

## Edit a room

1. Open **Zones & rooms** and select your zone.
2. Open an existing room, then choose **Edit room**. For a new room, use the
   free [vnum](/docs/reference/glossary/#vnum) offered on the zone page.
3. Change the name, description, flags, sector, exits, or extra descriptions.
   The fields are held in your draft while you work.

![The room editor’s room fields for a synthetic test room.](/media/docs/admin-room-editor.png)

4. Review the room, then choose **Commit draft**.

The room changes in the running world, and the claim is released.

5. Choose **Save zone file**. You can also save the zone from
   [Workshop & saves](/admin/workshop).

## Did it work?

The room detail shows the committed fields. After the zone file saves, that zone
no longer appears under pending saves in Workshop & saves.

## If it didn’t work

- If another builder holds the room, wait for them to finish and reopen it.
- If **Save zone file** is disabled, commit your draft first.
- If an action is unavailable, check the current account’s capabilities in the
  [builder guide](/admin/workshop/help) or ask staff to check your zone assignment.

## Related

- [Edit objects](/docs/build/edit-objects/)
- [Edit mobs](/docs/build/edit-mobs/)
- [Edit zones](/docs/build/edit-zones/)
