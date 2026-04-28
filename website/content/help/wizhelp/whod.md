---
title: "WHOD"
date: 2026-04-28
draft: false
section: "help"
aliases: ['/help/whod']
---

The who daemon is run on a separate port. The following commands exists:

name    : Toggles peoples name on/off the list (useless)
class   : Toggles peoples level on/off the list
title   : Toggles peoples title on/off the list
wizinvis: Toggles whether or not wizinvis people should be shown on the list
site    : Toggles peoples site names on/off the list
wizlevel: Toggles whether or not to show wizards class and level
on      : Turns the whod on, and thereby opens the port
off     : Turns the whod off, and thereby closes the port

NOTE:     The on/off feature is only made to use, if someone starts polling
        a few times a second or the like, and thereby abusing the net. You
        might then want to shut down the daemon for 15 minutes or so.
wizonly