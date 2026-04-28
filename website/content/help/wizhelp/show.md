---
title: "SHOW"
date: 2026-04-28
draft: false
section: "help"
aliases: ['/help/show']
---

Usage: show <mode> [argument]

Valid Modes:

death          errors         godrooms       houses
player         rent           shops          stats
zones          tattoos	      aggr

The SHOW command displays information.  Some modes of show require additional
information, such as a player name.

   death: Shows all death traps in the game.
  errors: Shows errant rooms.
godrooms: Shows the rooms in the 'god zone'.
  houses: Shows the houses that are currently defined.
  player: Shows player summary information, simply provide a player name.
    rent: Shows the filename and path to a players rent file.
   shops: Shows all the shops in the game and their buy/sell parameters.
   stats: Shows game status information including players in game, mobs etc.
   zones: Shows all the zones in the game and their current reset status.
          An age of -1 means it is in the 'to be reset next' queue.
 tattoos: Shows a list of available tattoo index numbers
    aggr: Shows a list of attitude mobs

See also: STAT, ZRESET
wizonly