-- drake.lua - Combat AI for drake mobs (lesser dragons, fire breath)
-- Source: drake.lua (origin/master:lib/scripts/mob/archive/drake.lua)
-- Ported for Phase 5b world restoration

function fight()
  if (number(0, 9) == 0) then         -- 10% chance per round; drake.lua line 2
    spell(ch, NIL, SPELL_FIREBALL, TRUE) -- fireball on target; drake.lua line 3
    -- NOTE: uses stubbed function spell
  end
end
