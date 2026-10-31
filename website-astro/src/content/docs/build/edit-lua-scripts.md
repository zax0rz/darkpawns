---
title: "Edit Lua scripts"
description: "Edit a Lua script in the builder workshop, let the server parse-check the file, then publish it for the next game trigger."
type: guide
audience: builders
verified: v0.1.7
---

Unlike room, mob, and object drafts, Lua scripts in the builder workshop have no staging step. When you click **Save**, the server syntax-checks your code and writes the file to disk, and the new script runs the very next time its trigger fires.

## Before you start

- Sign in at `/admin/` with a HIGOD (level 36) or higher character allowed to write scripts.
- Know the existing script path, or the folder where you want to create one.

:::caution
Saving publishes the script immediately. The server's syntax check catches Lua parse errors so a missing `end` won't break the parser, but logic bugs are all yours on the next trigger. Keep a copy of the old text if you might need to roll it back by hand.
:::

## Edit or create a script

1. Open **Workshop & saves**, then choose **Lua scripts**.
2. Pick a script from the file list, or enter a filename under **New script here** and choose **Open**.

![The selected guard.lua file with Save, Delete, and the live-apply notice.](/media/docs/admin-lua-scripts.png)

3. Edit the script and choose **Save**. The server parse-checks the Lua source before writing the file.

The saved script runs on its next trigger. The **Used by** panel lists every room, mob, or object prototype currently wired to that file.

## Did it work?

The editor confirms the save, the file appears under **Files**, and any room, mob, or object using the script is listed under **Used by**.

## If it didn't work

- **The editor is read-only.** Builders below HIGOD (level 36) can read scripts to see how triggers work, but saving requires level 36 or higher.
- **The save is rejected with a syntax error.** Fix the line number reported by the Lua parser and save again.
- **Another editor saved first.** Review the diff between the server copy and your buffer, merge your changes, and save again.

## Related

- [Edit rooms](/docs/build/edit-rooms/)
- [Edit objects](/docs/build/edit-objects/)
- [Edit mobs](/docs/build/edit-mobs/)
