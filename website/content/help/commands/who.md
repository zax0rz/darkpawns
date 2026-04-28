---
title: "WHO"
date: 2026-04-28
draft: false
section: "help"
aliases: ['/help/who']
---

Usage: who [minlev[-maxlev]] [-n sname] [-c classlist] [-s] [-o] [-q]

Lists the people currently in the game.  Some people may be invisible.
Command-line options can be used to limit the listing.  The parameters
can be specified on the command-line in any order.

minlev, maxlev : list only people whose level is at or above minlev, and
                 optionally, at or below maxlev
-n : list only people whose names or titles contain sname
-c : list only people of a certain class, where classlist is any 
     combination of the letters w, m, c, t, n, m, a, p
-s : list names in the short form (4 columns of names, without titles or
     flags)
-o : list only outlaws
-q : list only people who are on the Quest

Examples:

  > who -c wc -s -l 20
  List, in short form, warriors and clerics at or above level 20

  > who 15-25 -o 
  List all outlaws between levels 15 and 25.