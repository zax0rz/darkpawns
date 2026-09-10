-- snake.lua - Combat AI for snake mobs
-- Ported from original snake.lua in Dark Pawns MUD (lib/scripts/mob/archive/snake.lua)
-- Snake has a level-scaled chance to poison its target on each combat round.

function fight()
  -- Poison bite: chance = 1/(103 - ch.level), so higher-level targets get poisoned more often.
  -- At level 1 the denominator is 102; at level 30 it is 73. snake.lua line 2.
  if (number(0, (102 - ch.level)) == 0) then             -- snake.lua line 2: level-scaled poison chance
    act("$n bites $N!", TRUE, me, NIL, ch, TO_NOTVICT)   -- snake.lua line 3
    act("$n bites you!", TRUE, me, NIL, ch, TO_VICT)     -- snake.lua line 4
    spell(ch, NIL, SPELL_POISON, FALSE)                   -- snake.lua line 5: apply SPELL_POISON (33)
  end
end
