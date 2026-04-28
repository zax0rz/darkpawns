---
title: "BUILDING BUILD"
description: "Some not-so-obvious things about OLC:"
date: 2026-04-28
draft: false
section: "help"
aliases: ['/help/building-build']
---

Some not-so-obvious things about OLC:

REDIT: Room flags: REGEN_ROOM .. PCs (and mobs) rege mana/HPs faster in this
			room. 
		   BFR  .. Bad for recall, recall doesn't work here.
                   NEUTRAL .. can't die here. PCs get sent back to the temple
	                  instead. As such, entry to these rooms (or the rooms
                          themselves), should be !MOB
		   PEACEFUL .. violence and violent magic isn't allowed here.
		   ATRIUM, *, and HCRSH ..  do not use
		   PRIVATE .. Only two people allowed here. If more try to 
			enter, they get the following message:
			" A private conversation is going on in that room"
		   TUNNEL ..  Only one person allowed here. If more try to 
			enter, they get the following message:
		        "There's no room for you there!"
 		   BFS, *, ATRIUM, HCRSH, HOUSE, and **: do not use
	See Also: HELP FLOW
			
As of 2.2, Sector types only affect movement rate.

The sectors: Earth, Air, Water, Fire are for use in the elemental planes.

OEDIT:  When making a liquid container, the first alias-name should be the
name of the liquid initially in the container (unless it's empty, of course).
The second name should be the item.. only 2 names should be used.
Example: water skin   OR     beer bottle 
Cost, vs. cost-per-day: The cost is the value of the item. This should not 
exceed 10000. Cost-per-day is the number used to calculate wether or not
the item will load. In most cases, assign cost-per-day = cost. If cost per
day is > 10000, the difference is the % chance the item will have to load.
Example: 
  item: a loaf o bread    cost:5 cost-per-day:10100 (loads 100% of the time)
  item: a dagger of blood cost:8500 cost-per-day:8500 (approx 10% of the time)
  item: a helm of greed   cost:300  cost-per-day:10050 (loads 50% of the time)

To be able to backstab with an item, it must be type "pierce". Clerics can not
use slashing weapons, even if they don't have a !CLERIC flag on them.

HELP NAMED
HELP ACTION

MEDIT:  Setting the LEVEL (choice 6) puts defaults into the other blanks, so
set the mobs level first, then the other slots, else they will get over-
written.
When setting a mob with flag 21 (RNDLD-ZONE), use zedit to load him anywhere
in your zone. The RNDLD-ZONE flag will cause his location to actually be a
random room in the zone. It won't load in GODROOM, PRIVATE, !MOB, or DEATH 
rooms.
Noise is the message sent to rooms surrounding the mob every once in a
	while... An example for a dragon might be:
	"You hear the sound of great wings rustling."
	or perhaps:
	"A cloud of noxious gas wafts into the room."
See HELP NOISE

  AWARE (flag 5) mobs can't be backstabbed. AGGR24 (flag 20) makes the mob 
aggressive to PCs level 24 and above. AGGR24 mobs speak before attacking, and 
sometimes gossip when they have killed. A mob flagged AGGR24 and AGGR(flag 6) 
will attack other mobs that are 3+ levels below him, but only when fully 
healed.  HUNTER(flag 19) mobs will hunt through !mob rooms, and a hunting mob 
ignores their own SENTINEL(flag 2) assignment.
  
  If you add "carve_meat", "carve_rabbit", "carve_fish", or "carve_bird" to a
mob's namelist, they will become carveable.

SEDIT: Sell rate should be lower than buy rate. Using the Rooms menu(choice R)
then adding a new sell room (choice A)  with the vnum of 1 will make a mob 
sell anywhere.. good for making wandering shopkeepers.

ZEDIT: If you use an item/mob created in another zone, post it on the board so
Serapis or Oddity can up the max_exist for the entire world. 
About max-exist: since %load controls objects loading, set the max_exist for
eq loading on a mob to 999, unless it an extremely powerful item or an artifact.
Set the max_exist for eq loading directly into a room (like ATMs) to 1.

CAUTION: Items given to shopkeeper's are affected by %load. If you want a 
shopkeeper to ALWAYS have a certain item (like, for example, the baker to have 
bread), make the item's cost-per-day 10100.

wizonly