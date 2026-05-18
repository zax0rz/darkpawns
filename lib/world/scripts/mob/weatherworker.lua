-- weatherworker.lua - Combat AI: 20% chance SPELL_DISRUPT in combat
-- Source: scripts_full_dump.txt ./mob/archive/weatherworker.lua

function fight()
  if (number(0, 4) == 0) then			-- 20% chance of casting
    spell(ch, NIL, SPELL_DISRUPT, TRUE)
  end
end
