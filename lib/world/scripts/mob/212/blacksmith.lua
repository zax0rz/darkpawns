-- blacksmith.lua - Hashkar the blacksmith (zone 212, mob 21210)
-- Forges weapons and armor when given materials. Ambient chatter.
-- Trigger flags: sound(16) + ongive(8) = 24

function sound()
  local lines = {
    "Hashkar hammers a glowing piece of iron on the anvil.",
    "Hashkar quenches a red-hot blade in the water trough.",
    "Hashkar says, 'Bring me the right materials and I can forge anything.'",
    "Hashkar wipes the sweat from his brow and eyes your equipment.",
  }
  act(lines[number(1, getn(lines))], TRUE, me, NIL, NIL, TO_ROOM)
end

function ongive()
  if (obj.vnum == 0) then
    return
  end

  -- Simple forge: give iron ore + 500 gold = steel sword (vnum 21200 placeholder)
  if (obj.vnum == 21250) then
    if (ch.gold >= 500) then
      act("$n takes $p and begins working the forge.", TRUE, me, obj, NIL, TO_NOTVICT)
      extobj(obj)
      ch.gold = ch.gold - 500
      local item = oload(me, 21201, "char")
      act("$n gives $N $p.", TRUE, me, item, ch, TO_NOTVICT)
      act("$n gives you $p.", TRUE, me, item, ch, TO_VICT)
      save_char(ch)
    else
      act("$n says, 'I need 500 gold for my work.'", TRUE, me, NIL, ch, TO_VICT)
      return_obj(obj)
    end
  else
    act("$n says, 'I don't know what to do with this.'", TRUE, me, NIL, ch, TO_VICT)
    return_obj(obj)
  end
end
