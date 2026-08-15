---
title: "BOARDS"
description: "Bulletin boards are the forum of inter-player communication on the MUD."
date: 2026-04-28
draft: false
section: "help"
aliases: ['/help/boards']
---

Bulletin boards are the forum of inter-player communication on the [MUD](/database#item-5810).
There are different bulletin boards for different purposes -- for example,
a standard mortal board, a board for immortals, a board for fun "social"
messages, etc.  Naturally, not all players may be allowed to read all
types of boards.

Type "[LOOK](/help/commands/look/) [BOARD](/database#item-8099)" to see the messages already posted on a board.  Type
"[WRITE](/help/commands/write/) &lt;subject&gt;" to post a message to a board; terminate a message with
a '@' as the first character on a line.  Type "[READ](/help/commands/read/) &lt;number&gt;" to read a
post.  Type "[REMOVE](/help/commands/remove/) &lt;number&gt;" to remove your own messages.

Example:

  > look at board
  > write Am I using these boards correctly?
  [writes the message; terminates with a '@']
  > look at board
  > read 6
  > remove 6

See also: [MAIL](/help/commands/check-mail-receive/), [READ](/help/commands/read/), [WRITE](/help/commands/write/)