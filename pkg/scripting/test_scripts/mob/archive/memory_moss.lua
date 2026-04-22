-- memory_moss.lua - Removes spell affections from players
-- Ported from original memory_moss.lua in Dark Pawns MUD (lib/scripts/mob/archive/memory_moss.lua)
-- At any time when a player is present, there will be a 20% chance that the mob
-- will remove all spell affections from a random player within the room. Attached to mobs 10107 and 10108.
-- Source: memory_moss.lua lines 1-37

function fight()
  call(onpulse_pc, me, "x")                                      -- memory_moss.lua line 2
end

function onpulse_pc()
  local vict = NIL
  local counter = 0

  if (number(0, 4) == 0) then                                   -- memory_moss.lua line 9: 20% chance
    if (room.char) then
      repeat                                                     -- memory_moss.lua line 12: Locate a random PC
        vict = room.char[number(1, getn(room.char))]
        counter = counter + 1
        if (counter > 100) then                                  -- memory_moss.lua line 16
          return
        end
      until (not isnpc(vict))                                    -- memory_moss.lua line 19
    
      if (vict.level >= LVL_IMMORT) then                         -- memory_moss.lua line 22: Don't hit the Imms
        return
      end

      if (cansee(vict)) then                                     -- memory_moss.lua line 26: Can the mob see the player?
        act("A single touch of $N disrupts your concentration.", -- memory_moss.lua line 27
          TRUE, vict, NIL, me, TO_CHAR)
        act("$n looks bewildered as $N creeps across $s boots.", -- memory_moss.lua line 29
          TRUE, vict, NIL, me, TO_ROOM)
        unaffect(vict)                                           -- memory_moss.lua line 33: Remove spell affections
      end
    end
  end
end