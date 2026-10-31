---
title: "Edit mobs"
description: "Edit a mob prototype in your building zone with the admin console, commit it to the running world, and save the zone file."
type: guide
audience: builders
verified: v0.1.7
---

Use webOLC in the admin console to edit a [mob](/docs/reference/glossary/#mob) prototype in your assigned building zone. Your changes stay in a draft until you commit them to the running world and write the zone file.

## Before you start

- Sign in at `/admin/` with a character assigned to build in the zone.
- Locate the mob in **Mobs**, or use the next free [vnum](/docs/reference/glossary/#vnum) on its zone page.

:::caution
**Commit draft** updates the running world. **Save zone file** writes the `.mob` file so the change survives a server restart. The **Live script controls** apply immediately, and **Discard** won't undo them.
:::

## Edit a mob

1. Open **Mobs** and select the mob you want to change.
2. Choose **Edit** (or pick the free [vnum](/docs/reference/glossary/#vnum) on the zone page to create a new mob). Opening the editor claims the prototype and opens a draft.
3. Change its alias keywords, short and long descriptions, level, hit points, damage dice, position, race, alignment, or behavior flags.

![The mob editor's fields for a synthetic guard.](/media/docs/admin-mob-editor.png)

4. Review your edits and choose **Commit draft**.

The mob prototype updates in the running world and releases your claim.

5. Choose **Save zone file**, or save from [Workshop & saves](/admin/workshop).

:::tip[Behavior flags and carveable corpses from help MEDIT]
In `help MEDIT`, flagging a mob `AWARE` prevents players from backstabbing it, `AGGR24` makes it trash-talk and attack characters at level 24 or higher (and occasionally brag on `gossip` when it kills someone), and `HUNTER` lets it track targets right through `!MOB` rooms while ignoring its own `SENTINEL` flag. To make a beast's corpse carveable for food (`carve`), add `carve_meat`, `carve_rabbit`, `carve_fish`, or `carve_bird` to its alias keyword list.
:::

## Did it work?

The mob detail page reflects your committed stats, and saving the zone file removes the zone from pending saves in **Workshop & saves**.

## If it didn't work

- **The mob is already claimed.** Wait for the other builder to commit or discard their draft.
- **Save zone file is disabled.** Commit your mob draft first.
- **Controls are read-only.** Check your zone permissions in the [builder guide](/admin/workshop/help).

## Related

- [Edit rooms](/docs/build/edit-rooms/)
- [Edit objects](/docs/build/edit-objects/)
- [Edit zones](/docs/build/edit-zones/)
