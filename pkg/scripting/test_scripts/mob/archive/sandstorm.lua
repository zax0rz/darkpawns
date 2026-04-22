-- sandstorm.lua - Teleports players randomly within zone
-- Ported from original sandstorm.lua in Dark Pawns MUD (lib/scripts/mob/archive/sandstorm.lua)
-- During a fighting round, there is a 50% chance of attempting and then a 50% chance
-- for success of teleporting a PC to a random location within the zone. Attached to mob 6102.
-- Source: sandstorm.lua lines 1-48

function fight()
  if (number(0, 1) == 0) then                                   -- sandstorm.lua line 6: 50% chance
    if (number(0, 1) == 0) then                                 -- sandstorm.lua line 7: 50% chance
      create_event(me, NIL, NIL, NIL, "port", 1, LT_MOB)        -- sandstorm.lua line 8
    end
  end
end

function port()
  local vict = NIL
  local counter = 0

  if (room.char) then
    repeat                                                     -- sandstorm.lua line 16: Locate a random PC
      vict = room.char[number(1, getn(room.char))]
      counter = counter + 1
      if (counter > 100) then                                  -- sandstorm.lua line 20
        return
      end
    until (not isnpc(vict))                                    -- sandstorm.lua line 23

    if (vict == NIL) then                                      -- sandstorm.lua line 25
      return
    end

    act("You gasp in horror as $n is drawn within the whirling funnel cloud!", -- sandstorm.lua line 28
      TRUE, vict, NIL, NIL, TO_ROOM)
    act("Suddenly, you find yourself spinning violently within the whirling cloud!", -- sandstorm.lua line 30
      FALSE, vict, NIL, NIL, TO_CHAR)
    act("You crash to the ground, badly battered and separated from your companions.\r\n", -- sandstorm.lua line 32
      FALSE, vict, NIL, NIL, TO_CHAR)
    tport(vict, number(6101, 6299))                            -- sandstorm.lua line 34: Send to random location in zone
    act("From out of nowhere, $n crashes to the ground, narrowly missing you!", -- sandstorm.lua line 35
      TRUE, vict, NIL, NIL, TO_ROOM)

    vict.pos = POS_STANDING                                    -- sandstorm.lua line 38: Required to end the fight
    vict.hp = vict.hp - 75                                     -- sandstorm.lua line 39
    save_char(vict)                                            -- sandstorm.lua line 40
  end
end