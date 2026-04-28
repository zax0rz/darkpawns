---
title: "STRING"
description: "Usage: string <type> <name> <field> [<string> | <keyword>]"
date: 2026-04-28
draft: false
section: "help"
aliases: ['/help/string']
---

Usage: string <type> <name> <field> [<string> | <keyword>]

For changing the text-strings associated with objects and characters.  The
format is:

Type is either 'obj' or 'mob'.

Field is one of the following(the names may be abbreviated):

Name                  (the call-name of an obj/char - kill giant)
Short                 (for inventory lists (obj's) and actions (char's))
Long                  (for when obj/character is seen in room)
Title                 (for players)
Description           (For look at.  For obj's, must be followed by a keyword)
Delete-description    (only for obj's. Must be followed by keyword)

Where a string is expected as the last argument, a newline will make you enter
a string mode, and the game will prompt you for new lines to the string until
you exceed the maximum length for the string or you end a line with a '@'.

Players currently cannot be strung.
wizonly