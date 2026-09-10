-- When the player gives the mob a shop deed (obj 1222), the player is refunded the cost
-- of the shop and the deed is destroyed. If the player gives the mob a lifestyle bond
-- (obj 1245, 1246, 1247) the player is given a pack (obj 8038) and either a bond
-- certificate (obj 8030) or a gold certificate (obj 1248) depending on the lifestyle.
-- SOURCE: original clerk.lua lines 1-8

function ongive()
  local lifestyle = 0
  local pack = NIL
  local item = NIL
  local class = { [1] = { 5303, 5305, 5307 },
                  [2] = { 8040, 8021 },
                  [3] = { 8019, 8023 }
                }
  local race = { [1] = { 5331, 5314 },
                 [2] = { 8010, 5314 },
                 [3] = { 19104, 8063 }
               }

  -- Check if the object given is a lifestyle bond
  -- SOURCE: original clerk.lua lines 20-21
  if ((obj.vnum == 1245) or (obj.vnum == 1246) or (obj.vnum == 1247)) then
    -- Determine which lifestyle bond was given
    -- SOURCE: original clerk.lua lines 29-36
    if (obj.vnum == 1245) then
      lifestyle = 1
    elseif (obj.vnum == 1246) then
      lifestyle = 2
    elseif (obj.vnum == 1247) then
      lifestyle = 3
    end

    -- Destroy the bond
    -- SOURCE: original clerk.lua lines 37-38
    extobj(obj, "destroy")
  end

  -- If the object given is not a shop deed and no lifestyle bond was given, return
  -- SOURCE: original clerk.lua lines 39-40
  if ((obj.vnum ~= 1222) and (lifestyle == 0)) then
    return
  end

  -- If a shop deed was given, refund the player and destroy the deed
  -- SOURCE: original clerk.lua lines 41-55
  if (obj.vnum == 1222) then
    act("$n hands you 1000 gold coins.", FALSE, me, NIL, ch, TO_VICT)
    act("$n hands $N 1000 gold coins.", FALSE, me, NIL, ch, TO_NOTVICT)
    extobj(obj, "destroy")
    return
  end

  -- Give the player a pack and appropriate certificate based on lifestyle
  -- SOURCE: original clerk.lua lines 56-61
  pack = oload(me, 8038, "char")			-- Load the pack and bond certificate
  item = oload(me, 8030, "char")
  
  if (lifestyle == 3) then
    item = oload(me, 1248, "char")
  end

  -- Give the pack and certificate to the player
  -- SOURCE: original clerk.lua lines 62-67
  action(me, "give", pack, ch)
  action(me, "give", item, ch)

  -- Give the player starting equipment based on class and race
  -- SOURCE: original clerk.lua lines 68-84
  for i = 1, getn(class[lifestyle]) do
    item = oload(me, class[lifestyle][i], "char")
    action(me, "give", item, ch)
  end

  for i = 1, getn(race[lifestyle]) do
    item = oload(me, race[lifestyle][i], "char")
    action(me, "give", item, ch)
  end

  -- Inform the player
  -- SOURCE: original clerk.lua lines 85-88
  act("$n hands you a pack and a certificate.", FALSE, me, NIL, ch, TO_VICT)
  act("$n hands $N a pack and a certificate.", FALSE, me, NIL, ch, TO_NOTVICT)
end

-- VNUM ISSUES:
-- Missing object VNums: 1222, 1245, 1246, 1247, 1248, 5303, 5305, 5307, 5331, 5314
-- Available object VNums: 8038, 8030, 8040, 8021, 8019, 8023, 8010, 19104, 8063
-- TODO: Check if missing VNums exist in other world files or need to be created