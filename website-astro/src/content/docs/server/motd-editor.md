---
title: "MOTD editor"
description: "Edit the welcome messages, preview terminal colours and wrapping, and save without overwriting another editor."
section: "server"
audience: "operator"
order: 17
sourcePath: "website-astro/src/content/docs/server/motd-editor.md"
updated: 2026-10-28
draft: false
---

Choose **Staff tools → MOTD** in the admin console, or open the **MOTD** card on
the dashboard. Builders can read both messages. Saving requires GRGOD access
(level 38); staff below that level see a read-only editor.

## Edit a welcome message

1. Choose **MOTD** for mortal characters or **Immortal MOTD** for immortal
   characters. Immortals see their message instead of the regular one.
2. Edit the text. Each tab keeps its own draft while you switch between them.
3. Check the **Telnet preview**. Choose **40**, **80**, or **100 columns** to
   see where a terminal wraps each line, including long words. On a narrow
   screen, scroll sideways inside the preview.
4. Toggle **Colour** to compare rendered colours with the text containing the
   colour codes. Preview controls do not change your draft.
5. Choose **Save MOTD** or **Save Immortal MOTD**. Save each message separately.
   The saved message takes effect at once. Players see it at their next login;
   no reboot is needed.

Each message allows **2,400 bytes**. The counter includes two bytes for each
line break and counts text as UTF-8, so some characters take more than one byte.
Shorten an over-limit message before saving.

Open **Colour codes** for the available markers. Lowercase codes use ordinary
colours, uppercase codes use bright colours, `&n` resets the colour, and `&&`
prints an ampersand.

## Insert a snippet

Place the cursor where you want the snippet, or select text to replace.

- **Discord block** asks for an invite. Enter your own HTTPS invite from
  `discord.gg` or `discord.com/invite`, then choose **Insert Discord block**.
  There is no preset invite. Check the address before saving.
- **Gossip notice** inserts: “What's said on gossip echoes to our Discord.”

Snippets only change your draft. They do not save it or change Notifications.

Check [Notifications](/docs/server/notifications/) before inserting the gossip
notice. Its **Show the automatic gossip notice in the MOTD** option can already
add that sentence while the gossip mirror is active and ready. Automatic notices
are absent from the preview. Leave the snippet out if Notifications adds it, or
turn the automatic notice off if your own message already explains the mirror.
The game does not remove duplicate notices for you.

## Keep or discard a draft

**Load saved copy** replaces the selected tab's draft with the saved message.
If that tab has unsaved changes, confirm before discarding them.

Leaving the page or closing the browser warns about unsaved changes in either
tab. Drafts last while this editor stays open; they are not a backup. Copy any
text you need before leaving.

If someone else saves first, your save stops and your draft stays here. The
comparison marks the saved copy with **−** and your draft with **+**. Copy the
changes you want to keep, choose **Load saved copy**, then make your changes
again and save. If saving fails for another reason, check your connection and
access, then try again.

## How it works

The page lives at `/admin/staff/motd`. It uses the existing `fileEditApi` with
the `tedit` root and the `motd` and `imotd` fields. Reads return an ETag; saves
send it with `If-Match`. A stale ETag returns 412 without overwriting the saved
text. Successful saves write the existing text file, refresh the live message,
and record a `file_edit_save` audit entry.

The shared web `tedit` save floor is GRGOD (38), matching the command gate in
`src/interpreter.c:764` (R2/R5e). Fields requiring IMPL (40) keep that higher
requirement. Lua file writes still require HIGOD (36). Builder reads remain
available. The byte limit counts CRLF line endings; saved text uses LF endings.

The preview uses xterm with the game's ampersand colour markers. It changes
neither saved text nor game output. This editor adds no default player text and
does not change existing login behaviour (R1/R4).
