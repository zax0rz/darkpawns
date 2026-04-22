-- medusa.lua - Petrification gaze on look
-- Ported from original medusa.lua in Dark Pawns MUD (lib/scripts/mob/archive/medusa.lua)
-- If a player looks at the mob, they are turned to stone. Attached to mobs 14101 and 14102.
-- Source: medusa.lua lines 1-35

function oncmd()
  local command = ""
  local subcmd = ""

  if (strfind(argument, "%a%s") ~= NIL) then                     -- medusa.lua line 8
    command = strsub(argument, 1, strfind(argument, "%a%s"))     -- medusa.lua line 9
    subcmd = gsub(argument, command.." ", "")                    -- medusa.lua line 10
  else
    command = argument                                           -- medusa.lua line 12
  end

  if ((command == "look") or (command == "examine")) then        -- medusa.lua line 15
    if ((subcmd ~= "") and strfind(strlower(me.alias), strlower(subcmd))) then  -- medusa.lua line 16
      if (number(0, 100) > number(0, 100)) then                  -- medusa.lua line 17: random chance
        act("With a sound like that of a crashing wave, $N slowly turns to stone!", -- medusa.lua line 18
          TRUE, me, NIL, ch, TO_NOTVICT)
        act("With growing horror and increasing agony, your body slowly turns to stone!", -- medusa.lua line 20
          TRUE, ch, NIL, NIL, TO_CHAR)
        raw_kill(ch, me, SPELL_PETRIFY)                          -- medusa.lua line 22
        return (TRUE)
      end
    end
  end
end

function fight()
  dofile("scripts/mob/magic_user.lua")                           -- medusa.lua line 29
  call(fight, me, "x")                                           -- medusa.lua line 30
end