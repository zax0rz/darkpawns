-- gazer.lua - Combat AI: 20% chance SPELL_MINDBLAST in combat
-- Source: scripts_full_dump.txt ./mob/archive/gazer.lua

function fight()
  if (number(0, 4) == 0) then				-- 20% chance of casting
    spell(ch, NIL, SPELL_MINDBLAST, TRUE)
  end
end
