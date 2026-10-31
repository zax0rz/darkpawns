---
title: "Edit zones"
description: "Edit zone settings or room reset commands with the admin console, then commit each draft and save the zone file that stores it."
type: guide
audience: builders
verified: v0.1.8
---

Use webOLC in the admin console to adjust a zone's reset timer and boundaries or to edit the reset commands for a specific room. Zone-wide settings and per-room reset commands have separate editor views so two builders don't step on each other.

## Before you start

- Sign in at `/admin/` with a character assigned to build in the zone.
- Open the zone from **Zones & rooms**.

:::caution
The editor separates committing from writing the `.zon` file. **Commit draft** applies your changes to the running world; **Save zone file** writes them to disk before you log off. Script controls on a room editor page apply immediately, and **Discard** won't undo them.
:::

![The zones list for the synthetic world used by the admin console preview.](/media/docs/admin-zones.png)

## Change zone settings

1. Open the zone and choose **Edit zone**.
2. Change its name, lifespan (minutes between resets), reset mode, or top room [vnum](/docs/reference/glossary/#vnum). The full-zone reset timeline on this page is read-only.

![Zone name, lifespan, reset mode, and top-room controls in the zone editor.](/media/docs/admin-zone-editor.png)

3. Choose **Commit draft**, then **Save zone file**.

## Change a room's reset commands

1. Open the room from its zone page.
2. Choose **Edit room reset commands**.
3. Add, reorder, or edit the mob, object, and door reset commands in that room's timeline, then choose **Commit draft**.
4. Choose **Save zone file**. The page also displays the whole-zone timeline for context; that timeline stays read-only here.

:::tip[Max-exist conventions from help ZEDIT]
Because an item's `cost-per-day` already controls its percentage load chance, `help ZEDIT` recommends setting `max-exist` to `999` for equipment loading on a mob unless the item is an artifact or rare boss drop. For fixtures loading directly on a room floor (like fountains or ATMs), set `max-exist` to `1` so the room doesn't fill up with duplicates every reset.
:::

## Did it work?

The zone detail view shows your committed settings, and saving the file clears the zone from **Workshop & saves**.

## If it didn't work

- **The room or zone is claimed.** Another builder has a draft open; wait for them to finish.
- **Save zone file is disabled.** Commit your draft first.
- **The reset timeline is read-only.** You're looking at the whole-zone overview; open the specific room and choose **Edit room reset commands**.

## Related

- [Edit rooms](/docs/build/edit-rooms/)
- [Edit objects](/docs/build/edit-objects/)
- [Edit mobs](/docs/build/edit-mobs/)
