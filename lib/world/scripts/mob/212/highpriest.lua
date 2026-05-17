-- highpriest.lua - High Priest of Kir Drax'in (zone 212, mob 21221)
-- Blesses equipment, removes curses, heals. Ambient chants.
-- Trigger flags: sound(16) + ongive(8) = 24

function sound()
  local lines = {
    "The high priest chants softly in an ancient tongue.",
    "The high priest raises his hands in quiet prayer.",
    "The high priest says, 'Place your trust in the Light.'",
    "The high priest murmurs a blessing over the altar.",
  }
  act(lines[number(1, getn(lines))], TRUE, me, NIL, NIL, TO_ROOM)
end

function ongive()
  if (obj.vnum == 0) then
    return
  end

  -- Bless equipment: give item + 200 gold
  if (ch.gold >= 200) then
    ch.gold = ch.gold - 200
    act("$n holds $p aloft and chants words of power.", TRUE, me, obj, NIL, TO_NOTVICT)
    act("$n holds your $p and blesses it with holy light.", TRUE, me, obj, ch, TO_VICT)
    spell(me, ch, SPELL_SANCTUARY, FALSE)
    save_char(ch)
  else
    act("$n says, 'A blessing costs 200 gold, child.'", TRUE, me, NIL, ch, TO_VICT)
    return_obj(obj)
  end
end
