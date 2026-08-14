---
title: "SHUTDOWN"
description: "Usage: shutdown [reboot | die | pause]"
date: 2026-04-28
draft: false
section: "help"
aliases: ['/help/shutdown']
---

Usage: shutdown [reboot | die | pause]

[SHUTDOWN](/help/wizhelp/shutdown/) shuts the [MUD](/database#item-5810) down.  The [SHUTDOWN](/help/wizhelp/shutdown/) command works in conjunction with
CircleMUD's 'autorun' script.  If you are not using autorun, the arguments are
meaningless.  If you are using autorun, the following arguments are available:

REBOOT     Pause only 5 seconds instead of the normal 40 before trying to
           restart the [MUD](/database#item-5810).

[DIE](/help/info/die-death-condie/)        Kill the autorun script; the [MUD](/database#item-5810) will not reboot until autorun is
           explicitly run again.

PAUSE      Create a file called 'paused' in Circle's root directory; do not
           try to restart the [MUD](/database#item-5810) until 'paused' is removed.
wizonly