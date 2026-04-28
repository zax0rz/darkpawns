---
title: "SET"
description: "Usage: set [ file | player ] <character> <field> <value>"
date: 2026-04-28
draft: false
section: "help"
aliases: ['/help/set']
---

Usage: set [ file | player ] <character> <field> <value>

SET is an extremely powerful command, capable of setting dozens of aspects of
characters, both players and mobiles.

SET PLAYER forces set to look for a player and not a mobile; useful for
players with names such as 'guard'.

SET FILE lets you change players who are not logged on.  If you use SET FILE
on a player who IS logged on, your change will be lost.  If you wish to set
a player who is in the game but is linkless, use set twice -- once with the
FILE argument, and once without -- to make sure that the change takes.

For toggled fields (BINARY), the value must be ON, OFF, YES, or NO.

The following are valid fields:

Field                 Level Required    Who     Value
-----------------------------------------------------------------------------
    { "brief",          LVL_GOD,        PC,     BINARY },  
    { "invstart",       LVL_GOD,        PC,     BINARY }, 
    { "title",          LVL_GOD,        PC,     MISC },
    { "nosummon",       LVL_GRGOD,      PC,     BINARY },
    { "maxhit",         LVL_GRGOD,      BOTH,   NUMBER },
    { "maxmana",        LVL_GRGOD,      BOTH,   NUMBER },  
    { "maxmove",        LVL_GRGOD,      BOTH,   NUMBER },
    { "hit",            LVL_GRGOD,      BOTH,   NUMBER },
    { "mana",           LVL_GRGOD,      BOTH,   NUMBER },
    { "move",           LVL_GRGOD,      BOTH,   NUMBER },
    { "align",          LVL_GOD,        BOTH,   NUMBER }, 
    { "str",            LVL_GRGOD,      BOTH,   NUMBER },
    { "stradd",         LVL_GRGOD,      BOTH,   NUMBER },
    { "int",            LVL_GRGOD,      BOTH,   NUMBER },
    { "wis",            LVL_GRGOD,      BOTH,   NUMBER },
    { "dex",            LVL_GRGOD,      BOTH,   NUMBER },  
    { "con",            LVL_GRGOD,      BOTH,   NUMBER },
    { "sex",            LVL_GRGOD,      BOTH,   MISC },
    { "ac",             LVL_GRGOD,      BOTH,   NUMBER },
    { "gold",           LVL_GOD,        BOTH,   NUMBER },
    { "bank",           LVL_GOD,        PC,     NUMBER },  
    { "exp",            LVL_GRGOD,      BOTH,   NUMBER },
    { "hitroll",        LVL_GRGOD,      BOTH,   NUMBER },
    { "damroll",        LVL_GRGOD,      BOTH,   NUMBER },
    { "invis",          LVL_IMPL,       PC,     NUMBER },
    { "nohassle",       LVL_GRGOD,      PC,     BINARY },  
    { "frozen",         LVL_FREEZE,     PC,     BINARY },
    { "practices",      LVL_GRGOD,      PC,     NUMBER },
    { "lessons",        LVL_GRGOD,      PC,     NUMBER },
    { "drunk",          LVL_GRGOD,      BOTH,   MISC },  
    { "hunger",         LVL_GRGOD,      BOTH,   MISC },    
    { "thirst",         LVL_GRGOD,      BOTH,   MISC },  
    { "outlaw",         LVL_GOD,        PC,     BINARY },
    { "name",           LVL_GRGOD,      PC,     MISC },  
    { "level",          LVL_GRGOD,      BOTH,   NUMBER },
    { "room",           LVL_IMPL,       BOTH,   NUMBER },  
    { "roomflag",       LVL_GRGOD,      PC,     BINARY },
    { "siteok",         LVL_GRGOD,      PC,     BINARY },
    { "deleted",        LVL_GRGOD,      PC,     BINARY },
    { "class",          LVL_GRGOD,      BOTH,   MISC },  
    { "nowizlist",      LVL_GOD,        PC,     BINARY },  
    { "quest",          LVL_GOD,        PC,     BINARY },
    { "loadroom",       LVL_GRGOD,      PC,     MISC },
    { "color",          LVL_GOD,        PC,     BINARY },
    { "idnum",          LVL_IMPL-1,     PC,     NUMBER },
    { "passwd",         LVL_IMPL-1,     PC,     MISC },    
    { "nodelete",       LVL_GOD,        PC,     BINARY },
    { "cha",            LVL_GRGOD,      BOTH,   NUMBER },
    { "olc",            LVL_SET_BUILD,  PC,     NUMBER },
    { "race",           LVL_GOD,        PC,     MISC },  
    { "kills",          LVL_GRGOD,      BOTH,   NUMBER },  
    { "pks",            LVL_GRGOD,      BOTH,   NUMBER },
    { "deaths",         LVL_GRGOD,      BOTH,   NUMBER },
    { "home",           LVL_GRGOD,      PC,     NUMBER },
    { "tattoo",         LVL_GRGOD,      PC,     NUMBER },
    { "origcon",        LVL_GRGOD,      PC,     NUMBER }, 
    { "chosen",         LVL_GRGOD,      PC,     BINARY },
    { "clan",           LVL_GRGOD,      PC,     NUMBER },

See also: STAT
wizonly