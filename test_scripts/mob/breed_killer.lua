-- breed_killer.lua - AI for killing vampires and werewolves
-- Based on original breed_killer.lua from Dark Pawns MUD
-- Ported for Phase 3D (simplified version)

function fight()
  dofile("scripts/mob/fighter.lua")
  call(fight, ch, "x")
end

function onpulse_pc()

  local vict = NIL

  if (room.char) then
    for i = 1, getn(room.char) do
      vict = room.char[i]
      if (not isnpc(vict) and cansee(vict)) then
        -- Check for vampire or werewolf flags
        -- Note: aff_flagged() and plr_flagged() are stubbed in current engine
        -- For now, just check if player has vampire/werewolf in name as a simple test
        if (string.find(string.lower(vict.name), "vampire") or 
            string.find(string.lower(vict.name), "werewolf")) then
          if (number(0, 5) == 0) then
            act("You hear a low growl in the back of $n's throat.", TRUE, me, NIL, NIL, TO_ROOM)
          else
            say("Die, nightbreed!!")
          end

          -- Check if we have a stake or spike
          -- obj_list() is not implemented, so assume we do
          local hasWeapon = true -- Assume we have weapon for now
          
          if (hasWeapon) then
            if ((me.level > vict.level) or ((vict.level - me.level) < number(0, LVL_IMMORT)) or
              vict.pos == POS_SLEEPING) then
              act("$n drives a weapon into the chest of $N!", TRUE, me, NIL, vict, TO_NOTVICT)
              act("$n drives a weapon into your chest with a solid blow!", TRUE, me, NIL, vict, TO_VICT)
              raw_kill(vict, me, 0) -- TYPE_UNDEFINED = 0
              return
            else
              act("$N growls in anger as $n tries to attack $M!", TRUE, me, NIL, vict, TO_NOTVICT)
              act("$n comes at you with a weapon, but you dodge the attempt!", TRUE, me, NIL, vict, TO_VICT)
              return
            end
          else
            action(me, "kill "..vict.name)
            return
          end
        end
      end
    end
  end
end