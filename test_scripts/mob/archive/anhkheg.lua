-- anhkheg.lua - Combat AI for anhkheg (giant insectoid, acid spitter)
-- Source: anhkheg.lua (origin/master:lib/scripts/mob/archive/anhkheg.lua)
-- Ported for Phase 5b world restoration

function fight()
  if (number(0, 4) == 0) then         -- 20% chance per round; anhkheg.lua line 2
    spell(ch, NIL, SPELL_ACID_BLAST, TRUE) -- acid blast on target; anhkheg.lua line 3
    -- NOTE: uses stubbed function spell
  end
end
