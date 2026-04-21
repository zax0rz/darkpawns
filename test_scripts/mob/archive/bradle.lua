-- bradle.lua - Combat AI for Bradle (poison bite, level-scaled chance)
-- Source: bradle.lua (origin/master:lib/scripts/mob/archive/bradle.lua)
-- Ported for Phase 5b world restoration

function fight()
  -- Bite chance scales with target level: higher level = more frequent bites
  -- (102 - ch.level) means a level 1 victim has a 1-in-101 chance, level 20 a 1-in-82 chance, etc.
  -- bradle.lua line 2
  if (number(0, (102 - ch.level)) == 0) then
    act("$n bites $N!", TRUE, me, NIL, ch, TO_NOTVICT) -- bradle.lua line 3
    act("$n bites you!", TRUE, me, NIL, ch, TO_VICT)   -- bradle.lua line 4
    spell(ch, NIL, SPELL_POISON, FALSE) -- injects poison via bite; bradle.lua line 5
    -- NOTE: uses stubbed function spell
  end
end
