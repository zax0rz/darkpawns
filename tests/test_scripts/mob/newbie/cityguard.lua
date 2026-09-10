-- cityguard.lua - Law enforcement AI
-- Based on original cityguard.lua from Dark Pawns MUD
-- Ported for Phase 3D (simplified version)

function bribe()
  local amount = tonumber(argument)

  if ((number(0, 2) ~= 0) and (amount < 1000)) then
    say("Are you trying to bribe me? That's against the law!")
    action(me, "kill "..ch.name)
  else
    emote("glances around warily and says, 'I am off duty now...'")
    act("$n lays down and falls asleep on the job!", TRUE, me, NIL, NIL, TO_ROOM)
    me.pos = POS_SLEEPING
    save_char(me) -- Needed to get him to sleep
  end
end

function fight()
  -- Delegate to fighter AI
  dofile("scripts/mob/fighter.lua")
  call(fight, ch, "x")
end

function onpulse_pc()
-- The mob will attack any OUTLAWS, Vampires or Werewolfs as well as attacking a player
-- of opposite alignment who is already fighting.

  local vict = NIL

  -- Delegate to breed_killer AI for vampire/werewolf detection
  dofile("scripts/mob/breed_killer.lua")
  call(onpulse_pc, me, "x")

  if (not isfighting(me)) then
    if (room.char) then
      for i = 1, getn(room.char) do
        vict = room.char[i]
        
        -- Check if player is an OUTLAW
        if (not isnpc(vict) and cansee(vict) and plr_flagged(vict, PLR_OUTLAW)) then
          say("We don't like OUTLAWS in this city!")
          action(me, "kill "..vict.name)
          return
        end

        -- Check if player is fighting and has opposite alignment
        if (isfighting(vict) and cansee(vict)) then
          if (((me.evil == FALSE) and (vict.evil == TRUE)) or
             ((me.evil == TRUE) and (vict.evil == FALSE))) then
            act("$n says, 'You just pissed me off, $N!", TRUE, me, NIL, vict, TO_ROOM)
            action(me, "kill "..vict.name)
            return
          end
        end
      end
    end
  end
end

function greet()
-- City guard greeting
  if (ch.evil == TRUE) then
    say("Watch your step, " .. ch.name .. ". We're watching you.")
  else
    say("Move along, citizen. Nothing to see here.")
  end
end