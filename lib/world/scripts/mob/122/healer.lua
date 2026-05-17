-- healer.lua - Old ssauran healer (zone 122, mob 12220)
-- Heals players when given gold. Ambient chatter.
-- Trigger flags: sound(16) + ongive(8) = 24

function sound()
  local lines = {
    "The healer mutters ancient prayers under his breath.",
    "The healer examines his worn herbs with care.",
    "The healer looks at you and smiles warmly.",
    "The healer says, 'I can heal your wounds, if you have the coin.'",
  }
  act(lines[number(1, getn(lines))], TRUE, me, NIL, NIL, TO_ROOM)
end

function ongive()
  if (obj.vnum == 0) then
    return
  end

  -- Heal on gold given
  if (ch.gold >= 100) then
    ch.gold = ch.gold - 100
    act("$n closes $s eyes and places $s hands on $N.", TRUE, me, NIL, ch, TO_NOTVICT)
    act("$n places $s hands on you. You feel a warm glow.", TRUE, me, NIL, ch, TO_VICT)
    spell(me, ch, SPELL_HEAL, FALSE)
    save_char(ch)
  else
    act("$n says, 'I need at least 100 gold to heal you.'", TRUE, me, NIL, ch, TO_VICT)
  end
end
