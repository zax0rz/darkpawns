---
title: "ANSI COLOR"
description: "Usage: color [off | sparse | normal | complete]"
date: 2026-04-28
draft: false
section: "help"
aliases: ['/help/ansi-color']
---

Usage: color [off | sparse | normal | complete]

If you have a color-capable terminal and wish to see useful color-coding
of information, use the [COLOR](/help/commands/ansi-color/) command to set the level of coloring you see.

  > color off
  This command disables all color.

  > color sparse
  > color normal
  > color complete

These three commands turn color on to various levels.  Experiment to see
which level suits your personal taste.

'color' with no argument will display your current color level.
'color on' defaults to 'complete'.

Using color will slow down the speed at which you see messages VERY slightly.
The effect is more noticeable on slower connections.  Even if you have
color turned on, non-colorized messages will not be slowed down at all.

See also: [COLORSPRAY](/help/spells/colorspray-color-spray/), "[COLOR](/help/commands/ansi-color/) [SPRAY](/help/spells/colorspray-color-spray/)"
