---
title: "Edit Lua scripts"
description: "Edit a Lua script in the builder workshop, let the server parse-check the file, then publish it for the next game trigger."
type: guide
audience: builders
verified: v0.1.7
---

The builder workshop has a separate editor for Lua files. It checks a script
before saving, then the saved script runs the next time its trigger fires.

## Before you start

- Sign in at `/admin/` with an account allowed to write scripts.
- Know the existing script name, or the folder where you want to create one.

:::caution
Saving publishes the script immediately. There’s no draft or undo, so keep a
copy of the old text if you may need to restore it.
:::

## Edit or create a script

1. Open **Workshop & saves**, then choose **Lua scripts**.
2. Choose a file from the list. To create one, enter its name under **New script
   here** and choose **Open**.

![The selected guard.lua file with Save, Delete, and the live-apply notice.](/media/docs/admin-lua-scripts.png)

3. Edit the file and choose **Save**. The server parse-checks the text before it
   writes the file.

The saved script runs on its next trigger. The editor shows which world records
refer to an existing file under **Used by**.

## Did it work?

The editor reports that the file was saved, and it appears under **Files**.
If the script is already used by a mob, object, or room, its record appears under
**Used by**.

## If it didn’t work

- If the editor is read-only, the account needs script write access.
- If the parse check refuses the save, fix the reported Lua error and save again.
- If another editor saved first, review the diff and apply your changes to the
  current server copy before saving.

## Related

- [Edit rooms](/docs/build/edit-rooms/)
- [Edit objects](/docs/build/edit-objects/)
- [Edit mobs](/docs/build/edit-mobs/)
