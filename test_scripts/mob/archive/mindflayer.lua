-- mindflayer.lua - Combat AI for mind flayer mobs
-- Ported from original mindflayer.lua in Dark Pawns MUD (lib/scripts/mob/archive/mindflayer.lua)
-- Mind flayer has two psychic attacks selected by a 1-in-16 die roll.
--   switch == 0 or 5 (~2/16): SOUL_LEECH — tentacles drain soul energy
--   switch == 15 (~1/16): PSIBLAST — psionic mind blast

function fight()
  local switch = number(0, 15)                          -- mindflayer.lua line 2: roll 0–15

  if ((switch == 0) or (switch == 5)) then              -- mindflayer.lua line 4: ~2/16 chance, SOUL_LEECH
    act("The tentacles on $n's face surge forward, wrapping around $N's head!",
         TRUE, me, NIL, ch, TO_NOTVICT)                 -- mindflayer.lua line 5
    act("The tentacles on $n's face surge forward, wrapping around your head!",
         TRUE, me, NIL, ch, TO_VICT)                    -- mindflayer.lua line 7
    spell(ch, NIL, SPELL_SOUL_LEECH, FALSE)             -- mindflayer.lua line 8: SPELL_SOUL_LEECH (83)
  elseif (switch == 15) then                            -- mindflayer.lua line 9: ~1/16 chance, PSIBLAST
    act("Blood runs from $N's nose and ears as $n stares intently at $M.",
         TRUE, me, NIL, ch, TO_NOTVICT)                 -- mindflayer.lua line 10
    act("$n stares intently at you.. you feel $m battering your mind!",
         TRUE, me, NIL, ch, TO_VICT)                    -- mindflayer.lua line 12
    spell(ch, NIL, SPELL_PSIBLAST, FALSE)               -- mindflayer.lua line 13: SPELL_PSIBLAST (100)
  end
end
