---
title: "SET"
description: "Usage: set [ file | player ] <character> <field> <value>"
date: 2026-04-28
draft: false
section: "help"
aliases: ['/help/set']
---

Usage: set [ file | player ] <character> <field> <value>

[SET](/help/wizhelp/set/) is an extremely powerful command, capable of setting dozens of aspects of
characters, both players and mobiles.

[SET](/help/wizhelp/set/) [PLAYER](/help/info/pk-player-killing-pkill-pkilling-pkiller/) forces set to look for a player and not a mobile; useful for
players with names such as 'guard'.

[SET](/help/wizhelp/set/) FILE lets you change players who are not logged on.  If you use [SET](/help/wizhelp/set/) FILE
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
    { "maxhit",         LVL_GRGOD,      BOTH,   [NUMBER](/help/info/number-attacks/) },
    { "maxmana",        LVL_GRGOD,      BOTH,   [NUMBER](/help/info/number-attacks/) },  
    { "maxmove",        LVL_GRGOD,      BOTH,   [NUMBER](/help/info/number-attacks/) },
    { "hit",            LVL_GRGOD,      BOTH,   [NUMBER](/help/info/number-attacks/) },
    { "mana",           LVL_GRGOD,      BOTH,   [NUMBER](/help/info/number-attacks/) },
    { "move",           LVL_GRGOD,      BOTH,   [NUMBER](/help/info/number-attacks/) },
    { "align",          LVL_GOD,        BOTH,   [NUMBER](/help/info/number-attacks/) }, 
    { "str",            LVL_GRGOD,      BOTH,   [NUMBER](/help/info/number-attacks/) },
    { "stradd",         LVL_GRGOD,      BOTH,   [NUMBER](/help/info/number-attacks/) },
    { "int",            LVL_GRGOD,      BOTH,   [NUMBER](/help/info/number-attacks/) },
    { "wis",            LVL_GRGOD,      BOTH,   [NUMBER](/help/info/number-attacks/) },
    { "dex",            LVL_GRGOD,      BOTH,   [NUMBER](/help/info/number-attacks/) },  
    { "con",            LVL_GRGOD,      BOTH,   [NUMBER](/help/info/number-attacks/) },
    { "sex",            LVL_GRGOD,      BOTH,   MISC },
    { "ac",             LVL_GRGOD,      BOTH,   [NUMBER](/help/info/number-attacks/) },
    { "gold",           LVL_GOD,        BOTH,   [NUMBER](/help/info/number-attacks/) },
    { "bank",           LVL_GOD,        PC,     [NUMBER](/help/info/number-attacks/) },  
    { "exp",            LVL_GRGOD,      BOTH,   [NUMBER](/help/info/number-attacks/) },
    { "hitroll",        LVL_GRGOD,      BOTH,   [NUMBER](/help/info/number-attacks/) },
    { "damroll",        LVL_GRGOD,      BOTH,   [NUMBER](/help/info/number-attacks/) },
    { "invis",          LVL_IMPL,       PC,     [NUMBER](/help/info/number-attacks/) },
    { "nohassle",       LVL_GRGOD,      PC,     BINARY },  
    { "frozen",         LVL_FREEZE,     PC,     BINARY },
    { "practices",      LVL_GRGOD,      PC,     [NUMBER](/help/info/number-attacks/) },
    { "lessons",        LVL_GRGOD,      PC,     [NUMBER](/help/info/number-attacks/) },
    { "drunk",          LVL_GRGOD,      BOTH,   MISC },  
    { "hunger",         LVL_GRGOD,      BOTH,   MISC },    
    { "thirst",         LVL_GRGOD,      BOTH,   MISC },  
    { "outlaw",         LVL_GOD,        PC,     BINARY },
    { "name",           LVL_GRGOD,      PC,     MISC },  
    { "level",          LVL_GRGOD,      BOTH,   [NUMBER](/help/info/number-attacks/) },
    { "room",           LVL_IMPL,       BOTH,   [NUMBER](/help/info/number-attacks/) },  
    { "roomflag",       LVL_GRGOD,      PC,     BINARY },
    { "siteok",         LVL_GRGOD,      PC,     BINARY },
    { "deleted",        LVL_GRGOD,      PC,     BINARY },
    { "class",          LVL_GRGOD,      BOTH,   MISC },  
    { "nowizlist",      LVL_GOD,        PC,     BINARY },  
    { "quest",          LVL_GOD,        PC,     BINARY },
    { "loadroom",       LVL_GRGOD,      PC,     MISC },
    { "color",          LVL_GOD,        PC,     BINARY },
    { "idnum",          LVL_IMPL-1,     PC,     [NUMBER](/help/info/number-attacks/) },
    { "passwd",         LVL_IMPL-1,     PC,     MISC },    
    { "nodelete",       LVL_GOD,        PC,     BINARY },
    { "cha",            LVL_GRGOD,      BOTH,   [NUMBER](/help/info/number-attacks/) },
    { "olc",            LVL_SET_BUILD,  PC,     [NUMBER](/help/info/number-attacks/) },
    { "race",           LVL_GOD,        PC,     MISC },  
    { "kills",          LVL_GRGOD,      BOTH,   [NUMBER](/help/info/number-attacks/) },  
    { "pks",            LVL_GRGOD,      BOTH,   [NUMBER](/help/info/number-attacks/) },
    { "deaths",         LVL_GRGOD,      BOTH,   [NUMBER](/help/info/number-attacks/) },
    { "home",           LVL_GRGOD,      PC,     [NUMBER](/help/info/number-attacks/) },
    { "tattoo",         LVL_GRGOD,      PC,     [NUMBER](/help/info/number-attacks/) },
    { "origcon",        LVL_GRGOD,      PC,     [NUMBER](/help/info/number-attacks/) }, 
    { "chosen",         LVL_GRGOD,      PC,     BINARY },
    { "clan",           LVL_GRGOD,      PC,     [NUMBER](/help/info/number-attacks/) },

See also: [STAT](/help/wizhelp/stat/)
wizonly