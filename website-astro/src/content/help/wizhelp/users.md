---
title: "USERS"
description: "Usage: users [switches]"
date: 2026-04-28
draft: false
section: "help"
aliases: ['/help/users']
---

Usage: users [switches]

[USERS](/help/wizhelp/users/) gives a list of all sockets (i.e., connections) currently active on the
[MUD](/database#item-5810).  The multi-column display shows the socket number (used by [DC](/help/wizhelp/dc/)), class,
level, and name of the player connected, connection state, idle time, and
hostname.

The following switches are available:

-k or -o   Show only outlaws (killers and thieves).
-p         Show only sockets in the playing sockets.
-d         Show only non-playing (deadweight) sockets.
-l min-max Show only sockets whose characters are from level min to max.
-n &lt;name&gt;  Show the socket with &lt;name&gt; associated with it.
-h &lt;host&gt;  Show all sockets from &lt;host&gt;.
-c list    Show only sockets whose characters' classes are in list.

See also: [DC](/help/wizhelp/dc/), [SLOWNS](/help/wizhelp/slowns/)
wizonly