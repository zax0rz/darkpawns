---
title: "USERS"
description: "Usage: users [switches]"
date: 2026-04-28
draft: false
section: "help"
aliases: ['/help/users']
---

Usage: users [switches]

USERS gives a list of all sockets (i.e., connections) currently active on the
MUD.  The multi-column display shows the socket number (used by DC), class,
level, and name of the player connected, connection state, idle time, and
hostname.

The following switches are available:

-k or -o   Show only outlaws (killers and thieves).
-p         Show only sockets in the playing sockets.
-d         Show only non-playing (deadweight) sockets.
-l min-max Show only sockets whose characters are from level min to max.
-n <name>  Show the socket with <name> associated with it.
-h <host>  Show all sockets from <host>.
-c list    Show only sockets whose characters' classes are in list.

See also: DC, SLOWNS
wizonly