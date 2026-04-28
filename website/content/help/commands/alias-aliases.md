---
title: "ALIAS ALIASES"
description: "Usage: alias [alias name] [command]"
date: 2026-04-28
draft: false
section: "help"
aliases: ['/help/alias-aliases']
---

Usage: alias [alias name] [command]

An alias is a single command used to represent one or more other commands.
The ALIAS command can be used to manipulate your character's aliases.  Type
ALIAS alone to see a list of your currently defined aliases.

  > alias

Type ALIAS <alias name> <command> to create a new alias.  The alias name must
be one word.  The command can be any string.  Aliases may not be nested or
used recursively.  Any command may be overridden by an alias (other than the
ALIAS command itself, of course).

  > alias gbb get bread bag
  > alias gac get all corpse

Type ALIAS <alias-name> to delete an alias that has already been defined.

  > alias gbb

A semicolon (";") can be used to define multiple actions to be taken with
a single alias.

  > alias eat get bread bag; eat bread

Note that the 'eat' contained within the eat alias is interpreted as the MUD's
internal eat command; aliases may not be nested or used recursively.  The
variables $1, $2, ... $9 may be used to represent arguments given with
an alias.

Examples:

  > alias killem cast 'fireball' $1; cast 'harm' $2
  > killem fido mage

Would cast fireball on fido, followed by casting harm on mage.  Variables
may be used repeatedly and in any order:

  > alias helpem cast 'heal' $1; cast 'bless' $1; cast 'armor' $1

The special variable $* represents the entire input line after an alias:

  > alias tr tell rasmussen $*
  > tr hi there, how are you today?
  You tell Rasmussen, 'hi there, how are you today?'

See also: ! ^