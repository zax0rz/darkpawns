-- caerroil.lua - Combat AI for Caerroil (self-heals during combat)
-- Source: caerroil.lua (origin/master:lib/scripts/mob/archive/caerroil.lua)
-- Ported for Phase 5b world restoration

function fight()
  -- Original comment says "10% chance" but number(0,15) gives a 1-in-16 (~6.25%) chance;
  -- preserved verbatim from source; caerroil.lua line 2
  if (number(0, 15) == 0) then
    spell(me, NIL, SPELL_HEAL, TRUE) -- heals self; caerroil.lua line 3
    -- NOTE: uses stubbed function spell
  end
end
