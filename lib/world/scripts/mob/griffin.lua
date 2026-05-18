-- griffin.lua - Combat AI: 10% chance SPELL_FLAMESTRIKE in combat
-- Source: scripts_full_dump.txt ./mob/archive/griffin.lua

function fight()
  if (number(0, 9) == 0) then				-- 10% chance of casting
    spell(ch, NIL, SPELL_FLAMESTRIKE, TRUE)
  end
end
