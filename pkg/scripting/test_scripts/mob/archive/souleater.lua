-- souleater.lua - Eats souls to allow passage through gate
-- Ported from original souleater.lua in Dark Pawns MUD (lib/scripts/mob/archive/souleater.lua)
-- When given a soul (obj vnum 4618), the mob eats it and teleports the player through gate.
-- Otherwise returns item and growls. Attached to mob guarding a gate.
-- Source: souleater.lua lines 1-30

function ongive()
  local alias = ""

  if (obj.vnum == 4618) then                                   -- souleater.lua line 4
    act("$n peers at the soul, then licks $s lips.",           -- souleater.lua line 5
        TRUE, me, NIL, ch, TO_ROOM)
    say("This will do nicely, you may pass...")                -- souleater.lua line 7
    act("$n pops the soul into $s mouth and swallows it, a hideous screaming ringing in your ears.", -- souleater.lua line 8
        TRUE, me, NIL, ch, TO_ROOM)
    act("$n pushes $N through the gate.",                      -- souleater.lua line 10
        TRUE, me, NIL, ch, TO_NOTVICT)
    tport(ch, 14405)                                           -- souleater.lua line 11
    act("$N stumbles through the gate, pushed from the other side.", -- souleater.lua line 12
        TRUE, me, NIL, ch, TO_NOTVICT)
  else
    act("$n peers at $p closely before handing it back.",      -- souleater.lua line 15
        TRUE, me, obj, ch, TO_ROOM)
    emote("growls, 'Are you mocking me?'")                     -- souleater.lua line 16
    if (strfind(obj.alias, "%a%s")) then                       -- souleater.lua line 17
      alias = strsub(obj.alias, 1, strfind(obj.alias, "%a%s")) -- souleater.lua line 18
    else
      alias = obj.alias                                        -- souleater.lua line 20
    end
    action(me, "give "..alias.." "..ch.name)                   -- souleater.lua line 21
  end
end