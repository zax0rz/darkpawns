-- Source: scripts_full_dump.txt ./mob/archive/teleport_vict.lua
-- Attached to mob 14405. Automatically teleports victim away each fight round.

function fight()
  -- Automatically teleport the victim away. Attached to mob 14405.

  action(me, "scoff "..ch.name)
  act("$n says, 'You can't harm me, mortal. Begone.'", TRUE, me, NIL, NIL, TO_ROOM)
  spell(ch, NIL, SPELL_TELEPORT, TRUE)
  me.pos = POS_STANDING
  ch.pos = POS_STANDING
end
