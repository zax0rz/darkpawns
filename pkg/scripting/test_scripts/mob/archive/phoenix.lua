-- phoenix.lua - Rider assists in battle, immune until phoenix dies
-- Ported from original phoenix.lua in Dark Pawns MUD (lib/scripts/mob/archive/phoenix.lua)
-- A rider (mob 1402) will assist the mob in battle but will be immune from damage
-- until the mob "crashes" to the ground in death. Attached to mob 1401.
-- Source: phoenix.lua lines 1-48

function fight()
  local trident = NIL
  local rider = NIL

  if (me.gold == 0) then                                       -- phoenix.lua line 6: first time flag
    me.gold = 1                                                -- phoenix.lua line 7
    save_char(me)                                              -- phoenix.lua line 8
    rider = mload(1402, room.vnum)                             -- phoenix.lua line 9
    act("$n, riding atop the phoenix, joins the fight and strikes at $N!", -- phoenix.lua line 10
      TRUE, rider, NIL, ch, TO_NOTVICT)
    act("$n, riding atop the phoenix, joins the fight and strikes at you!", -- phoenix.lua line 12
      TRUE, rider, NIL, ch, TO_VICT)

    trident = oload(rider, 1420, "char")                       -- phoenix.lua line 15
    local percent = number(0, 100)                             -- phoenix.lua line 16
    if (trident.perc_load < percent) then                      -- phoenix.lua line 17
      me.gold = 2                                              -- phoenix.lua line 18
      save_char(me)                                            -- phoenix.lua line 19
    end
    extobj(trident)                                            -- phoenix.lua line 21
    extchar(rider)                                             -- phoenix.lua line 22
  else
    rider = mload(1402, room.vnum)                             -- phoenix.lua line 25
    trident = oload(rider, 1420, "char")                       -- phoenix.lua line 26

    if (me.gold == TRUE) then                                  -- phoenix.lua line 28: check if trident should be equipped
      equip_char(rider, trident)                               -- phoenix.lua line 29
    else
      extobj(trident)                                          -- phoenix.lua line 31
    end

    action(rider, "kill "..ch.name)                            -- phoenix.lua line 33
    extchar(rider)                                             -- phoenix.lua line 34: Remove rider until phoenix dead
  end
end

function death()
  local rider = NIL
  local trident = NIL

  rider = mload(1402, room.vnum)                               -- phoenix.lua line 41
  action(rider, "kill "..ch.name)                              -- phoenix.lua line 42

  trident = oload(rider, 1420, "char")                         -- phoenix.lua line 44
  act("Before $n crashes to the ground, $N leaps off.",        -- phoenix.lua line 45
    TRUE, me, NIL, rider, TO_ROOM)

  if (me.gold == TRUE) then                                    -- phoenix.lua line 47
    equip_char(rider, trident)                                 -- phoenix.lua line 48
  else
    extobj(trident)                                            -- phoenix.lua line 50
  end
end