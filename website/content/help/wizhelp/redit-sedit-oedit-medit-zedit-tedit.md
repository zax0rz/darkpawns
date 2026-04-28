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
redit <virtual room num>        - edit/create room
redit save <zone>               - save all the rooms in zone to disk 

zedit                           - edit the zone info for the room 
					you are standing in
zedit <virtual room num>        - edit the zone info for that room 
zedit save <zone>               - save all the zone info for that zone
					to disk 
zedit new <zone>                - IMPLs only - create a new zone. 

oedit <virtual obj num>         - edit/create object
oedit save <zone>               - save all the objects in zone to disk 

medit <virtual mob num>         - edit/create mobile
medit save <zone>               - save all the mobiles in zone to disk 

sedit <virtual shop num>        - edit/create shop
sedit save <zone>               - save all shops in zone to disk.

tedit 	   			- list text files
tedit <file>			- edit a text file

set <player name> olc <zone>    - IMPLs only - allow player to edit
olc                             - List all the things that have been edited
                                   	but not yet saved.

WARNING:  This OLC will let you set values to values that
shouldn't be set.  For example, it'll let you set a mobile with a
GROUP flag.  This is good in the sense that it allows you to test
anything you please, but bad in the sense that builders can crash
the mud with ease. (Hey, that rhymes!). 
/****************************************************************
In short: If you don't know what it does, ask before using it!!!!
****************************************************************/

See also: SET OLC RLIST
wizonly