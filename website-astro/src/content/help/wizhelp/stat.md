---
title: "STAT"
description: "Usage: stat [player | object | mobile | file | room] <name>"
date: 2026-04-28
draft: false
section: "help"
aliases: ['/help/stat']
---

Usage: stat [player | object | mobile | file | room] &lt;name&gt;

Gives information about players, monsters, and objects in the game.  The type
argument is optional.

[STAT](/help/wizhelp/stat/) [PLAYER](/help/info/pk-player-killing-pkill-pkilling-pkiller/) will search only for players; useful for statting people with
names such as Red or Cityguard.

[STAT](/help/wizhelp/stat/) [OBJECT](/database#item-4330) will search only for objects.

[STAT](/help/wizhelp/stat/) [MOBILE](/help/info/mob-mobile-npc-mobs/) will search only for monsters.

[STAT](/help/wizhelp/stat/) FILE is used to stat players who are not logged in; the information
displayed comes from the playerfile.

[STAT](/help/wizhelp/stat/) [ROOM](/help/wizhelp/flow-room-flow-room-flow-north-flowing/) is used to stat the room.

Examples:

  > stat fido
  > stat player red
  > stat mobile red
  > stat file niandra
  > stat object thunderbolt
  > stat room

See also: [VSTAT](/help/wizhelp/vstat/)
wizonly