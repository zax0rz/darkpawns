-- ettin.lua - Combat AI for ettin mobs
-- Ported from original ettin.lua in Dark Pawns MUD (lib/scripts/mob/archive/ettin.lua)
-- Ettin hurls boulders at its target for raw HP damage; 25% chance per combat round.

function fight()
  local damage = number(10, 30)                          -- ettin.lua line 2: 10–30 raw damage

  if (number(0, 3) == 0) then                            -- ettin.lua line 4: 25% chance (1-in-4)
    -- Announce the boulder hurl to bystanders and victim
    act("$n hurls a large boulder at $N, crushing $S!", TRUE, me, NIL, ch, TO_NOTVICT)  -- ettin.lua line 5
    act("$n hurls a large boulder at you, crushing your body!", TRUE, me, NIL, ch, TO_VICT)  -- ettin.lua line 6

    -- Apply raw HP damage directly to victim's table field (ch.hp)
    -- NOTE: ch.hp is written back to the engine via tableToChar() after script execution.
    -- This bypasses the normal spell/combat damage path — the original also did this directly.
    ch.hp = ch.hp - damage                               -- ettin.lua line 7: raw HP reduction

    save_char(ch)                                        -- ettin.lua line 8: persist HP change
  end
end
