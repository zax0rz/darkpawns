---
title: "STAT"
description: "Usage: stat [player | object | mobile | file | room] <name>"
date: 2026-04-28
draft: false
section: "help"
aliases: ['/help/stat']
---

Usage: stat [player | object | mobile | file | room] <name>

Gives information about players, monsters, and objects in the game.  The type
argument is optional.

STAT PLAYER will search only for players; useful for statting people with
names such as Red or Cityguard.

STAT OBJECT will search only for objects.

STAT MOBILE will search only for monsters.

STAT FILE is used to stat players who are not logged in; the information
displayed comes from the playerfile.

STAT ROOM is used to stat the room.

Examples:

  > stat fido
  > stat player red
  > stat mobile red
  > stat file niandra
  > stat object thunderbolt
  > stat room

See also: VSTAT
wizonly