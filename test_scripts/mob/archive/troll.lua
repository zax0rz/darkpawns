-- troll.lua - Combat AI for troll mobs
-- Ported from original troll.lua in Dark Pawns MUD (lib/scripts/mob/archive/troll.lua)
-- Trolls regenerate HP out of combat (onpulse_all) and also during combat (fight trigger).
-- Regeneration formula: me.level * 2, capped at me.maxhp.  troll.lua lines 3, 13.

-- NOTE: uses stubbed function onpulse_all — this trigger fires every world pulse for all mobs.
-- me.hp and me.maxhp write-back: changes to the me table are applied via tableToMob()
-- after script execution in the engine (engine.go tableToMob).

function onpulse_all()
  -- Regenerate only when below max HP  -- troll.lua line 2
  if (me.hp < me.maxhp) then                                         -- troll.lua line 2
    if (number(0, 20) == 0) then                                      -- troll.lua line 3: ~5% chance per pulse
      act("$n's wounds glow brightly for a moment, then disappear!", TRUE, me, NIL, NIL, TO_ROOM)  -- troll.lua line 4
      me.hp = me.level * 2                                            -- troll.lua line 5: regen to level*2
      if (me.hp > me.maxhp) then                                      -- troll.lua line 6: cap at maxhp
        me.hp = me.maxhp                                              -- troll.lua line 7
      end
    end
  end
end

function fight()
  -- Same regeneration flash can also fire during combat  -- troll.lua line 12
  if (number(0, 10) == 0) then                                        -- troll.lua line 12: ~10% chance per round
    act("$n's wounds glow brightly for a moment, then disappear!", TRUE, me, NIL, NIL, TO_ROOM)  -- troll.lua line 13
    me.hp = me.level * 2                                              -- troll.lua line 14: regen to level*2
    if (me.hp > me.maxhp) then                                        -- troll.lua line 15: cap at maxhp
      me.hp = me.maxhp                                                -- troll.lua line 16
    end
  end
end
