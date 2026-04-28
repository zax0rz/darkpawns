---
title: "SHUTDOWN"
date: 2026-04-28
draft: false
section: "help"
aliases: ['/help/shutdown']
---

Usage: shutdown [reboot | die | pause]

SHUTDOWN shuts the MUD down.  The SHUTDOWN command works in conjunction with
CircleMUD's 'autorun' script.  If you are not using autorun, the arguments are
meaningless.  If you are using autorun, the following arguments are available:

REBOOT     Pause only 5 seconds instead of the normal 40 before trying to
           restart the MUD.

DIE        Kill the autorun script; the MUD will not reboot until autorun is
           explicitly run again.

PAUSE      Create a file called 'paused' in Circle's root directory; do not
           try to restart the MUD until 'paused' is removed.
wizonly