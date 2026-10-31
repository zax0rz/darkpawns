---
title: "Edit rooms"
description: "Edit a room in your building zone with the admin console: change fields in a draft, commit to the running world, and save the zone file."
type: guide
audience: builders
verified: v0.1.8
---

Use webOLC in the admin console to reshape a room in your assigned building zone. You edit a private draft first, commit it to the live world when it looks right, and then write the zone file to disk so your work survives the next reboot.

## Before you start

- Sign in at `/admin/` with a character assigned to build in the zone.
- Know which zone or room [vnum](/docs/reference/glossary/#vnum) you want to change.

:::caution
**Commit draft** updates the running world in memory. **Save zone file** is a separate step that writes `.wld` to disk across restarts. The room's **Live script controls** apply immediately, and **Discard** won't undo them.
:::

This flow shows how webOLC separates live world commits from disk saves.

```mermaid
flowchart LR
  A["Open editor and claim record"] --> B["Edit fields in draft"] --> C["Commit draft to live world"] --> D["Save zone file to disk"]
```

## Edit a room

1. Open **Zones & rooms** and select your zone.
2. Open an existing room and choose **Edit room**. To carve out a new room, pick the free [vnum](/docs/reference/glossary/#vnum) offered on the zone page.
3. Change the name, description, flags, sector, exits, or extra descriptions. Your edits stay isolated in your draft while you work.

![The room editor's room fields for a synthetic test room.](/media/docs/admin-room-editor.png)

4. Review the room, then choose **Commit draft**.

The room updates in the running world and releases your editing lock.

5. Choose **Save zone file**, or batch-save your pending zones from [Workshop & saves](/admin/workshop).

:::note[Legacy room flags from help REDIT]
In `help REDIT`, `BFR` means "Bad for recall" (`recall` fails in that room), `PEACEFUL` blocks violence and offensive magick, and `NEUTRAL` sends dying players back to the temple instead of making a corpse (so keep `NEUTRAL` rooms flagged `!MOB`). Leave internal engine flags (`ATRIUM`, `HCRSH`, `HOUSE`, `*`, `**`) untouched.
:::

## Did it work?

The room detail view shows your committed fields, and once you save the zone file, the zone clears out of pending saves in **Workshop & saves**.

## If it didn't work

- **Another builder holds the room.** Wait for them to commit or discard their draft, then reopen it.
- **Save zone file is disabled.** Commit your room draft to the running world first.
- **Edit controls are locked.** Check your account's zone permissions in the [builder guide](/admin/workshop/help) or ask an implementor to check `set <player> olc <zone>`.

## Related

- [Edit objects](/docs/build/edit-objects/)
- [Edit mobs](/docs/build/edit-mobs/)
- [Edit zones](/docs/build/edit-zones/)
