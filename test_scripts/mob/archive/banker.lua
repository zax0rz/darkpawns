-- When the player gives the mob a bond certificate (obj 8030), the player is given
-- the amount of gold specified by the certificate. If the player gives the mob a
-- gold certificate (obj 1248), the player is given 1000 gold.
-- SOURCE: original banker.lua lines 1-6

function ongive()
  local gold = 0

  -- Check if the object given is a bond certificate
  -- SOURCE: original banker.lua lines 8-9
  if (obj.vnum == 8030) then
    -- Determine the amount of gold based on the certificate's value field
    -- SOURCE: original banker.lua lines 10-15
    if (obj.value[1] == 1) then
      gold = 100
    elseif (obj.value[1] == 2) then
      gold = 500
    elseif (obj.value[1] == 3) then
      gold = 1000
    end

    -- Destroy the certificate
    -- SOURCE: original banker.lua lines 16-17
    extobj(obj, "destroy")
  end

  -- Check if the object given is a gold certificate
  -- SOURCE: original banker.lua lines 19-22
  if (obj.vnum == 1248) then
    gold = 1000
    extobj(obj, "destroy")
  end

  -- If no valid certificate was given, return
  -- SOURCE: original banker.lua lines 24-25
  if (gold == 0) then
    return
  end

  -- Give the player the gold
  -- SOURCE: original banker.lua lines 26-31
  act("$n hands you "..gold.." gold coins.", FALSE, me, NIL, ch, TO_VICT)
  act("$n hands $N "..gold.." gold coins.", FALSE, me, NIL, ch, TO_NOTVICT)
end

-- VNUM ISSUES:
-- Missing object VNum: 1248
-- Available object VNum: 8030
-- TODO: Check if missing VNum 1248 exists in other world files