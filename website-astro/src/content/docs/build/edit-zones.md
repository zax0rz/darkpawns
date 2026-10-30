---
title: "Edit zones"
description: "Edit zone settings or room reset commands with the admin console, then commit each draft and save the zone file that stores it."
type: guide
audience: builders
verified: v0.1.7
---

Use webOLC in the admin console to change a zone’s settings or edit reset
commands for one room. Zone settings and room reset commands have separate
editor paths.

## Before you start

- Sign in at `/admin/` with a character that can build in the zone.
- Open the zone from **Zones & rooms**.

:::caution
The editor separates committing from writing the zone file. Commit applies the
changes to the running world; save the file before you finish. Script controls
on a room editor page apply at once, and **Discard** won’t undo them.
:::

## Change zone settings

1. Open the zone and choose **Edit zone**.
2. Change its name, lifespan, reset mode, or top room. The whole-zone reset
   timeline on this page is read-only.

![Zone name, lifespan, reset mode, and top-room controls in the zone editor.](/media/docs/admin-zone-editor.png)

3. Choose **Commit draft**, then **Save zone file**.

## Change a room’s reset commands

1. Open the room from its zone page.
2. Choose **Edit room reset commands**.
3. Change the commands in that room’s timeline, then choose **Commit draft**.
4. Choose **Save zone file**. The page also shows the whole-zone timeline for
   context; that timeline is read-only here.

## Did it work?

The zone detail reflects the committed settings. The zone no longer appears in
Workshop & saves after its file has been saved.

## If it didn’t work

- If a room or zone is claimed, wait for the current editor to finish.
- If **Save zone file** is disabled, commit the draft first.
- If the reset timeline is read-only, open the room’s **Edit room reset
  commands** page to change its commands.

## Related

- [Edit rooms](/docs/build/edit-rooms/)
- [Edit objects](/docs/build/edit-objects/)
- [Edit mobs](/docs/build/edit-mobs/)
