---
title: "PURGE"
date: 2026-04-28
draft: false
section: "help"
aliases: ['/help/purge']
---

Usage: purge [name]

Purge destroys things.  If used with no arguments, it will clear a room of all
mobiles and objects, but will not harm players.  If the argument is an object,
that object (and all of its contents, if any) will be destroyed.  If the
argument is a mobile, the mobile will be destroyed, leaving its inventory and
equipment lying on the ground.

If the argument is the name of a player, the player's inventory and equipment
will drop on the ground and the player will lose his/her connection.  For this
reason, PURGE should only be used on players in disciplinary situations.

Examples:

  > purge
  > purge Pom
  > purge sword
  > purge dragon

See also: DC
wizonly