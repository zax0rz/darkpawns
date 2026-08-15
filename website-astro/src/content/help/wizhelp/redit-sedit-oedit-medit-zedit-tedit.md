---
title: "REDIT SEDIT OEDIT MEDIT ZEDIT TEDIT"
description: "Usage:"
date: 2026-04-28
draft: false
section: "help"
aliases: ['/help/redit-sedit-oedit-medit-zedit-tedit']
---

Usage:

redit                           - edit the room you are standing in
redit &lt;virtual room num&gt;        - edit/create room
redit save &lt;zone&gt;               - save all the rooms in zone to disk

zedit                           - edit the zone info for the room
					you are standing in
zedit &lt;virtual room num&gt;        - edit the zone info for that room
zedit save &lt;zone&gt;               - save all the zone info for that zone
					to disk
zedit new &lt;zone&gt;                - IMPLs only - create a new zone.

oedit &lt;virtual obj num&gt;         - edit/create object
oedit save &lt;zone&gt;               - save all the objects in zone to disk

medit &lt;virtual mob num&gt;         - edit/create mobile
medit save &lt;zone&gt;               - save all the mobiles in zone to disk

sedit &lt;virtual shop num&gt;        - edit/create shop
sedit save &lt;zone&gt;               - save all shops in zone to disk.

tedit 	   			- list text files
tedit &lt;file&gt;			- edit a text file

set &lt;player name&gt; olc &lt;zone&gt;    - IMPLs only - allow player to edit
olc                             - List all the things that have been edited
                                   	but not yet saved.

WARNING:  This [OLC](/help/wizhelp/olc/) will let you set values to values that
shouldn't be set.  For example, it'll let you set a mobile with a
[GROUP](/help/commands/group/) flag.  This is good in the sense that it allows you to test
anything you please, but bad in the sense that builders can crash
the mud with ease. (Hey, that rhymes!).
/****************************************************************
In short: If you don't know what it does, ask before using it!!!!
****************************************************************/

See also: [SET](/help/wizhelp/set/) [OLC](/help/wizhelp/olc/) [RLIST](/help/wizhelp/rlist-mlist-olist-zlist/)
wizonly