-- clerk.lua - Gives starting gear to newbies
-- Based on original clerk.lua from Dark Pawns MUD
-- Ported for Phase 3D (simplified version)

function ongive()
-- When the player gives the mob a shop deed (obj 1222), the player is refunded the cost
-- of the purchase. Also handles character creation letters (1245-1247).

  local lifestyle = 0
  local alias = ""

  -- Check for character creation letters (1245, 1246, 1247)
  if ((obj.vnum == 1245) or (obj.vnum == 1246) or (obj.vnum == 1247)) then
    -- In original: check if letter belongs to player via obj.val[1]
    -- For now, assume it does
    
    if (obj.vnum == 1245) then
      lifestyle = 1
    elseif (obj.vnum == 1246) then
      lifestyle = 2
    elseif (obj.vnum == 1247) then
      lifestyle = 3
    end
  end

  -- If not a shop deed (1222) and not a creation letter
  if ((obj.vnum ~= 1222) and (lifestyle == 0)) then
    say("I'm sorry, I have no use for that...you better keep it.")
    if (strfind(obj.alias, "%a%s")) then
      alias = strsub(obj.alias, 1, strfind(obj.alias, "%a%s"))
    else
      alias = obj.alias
    end
    action(me, "give "..alias.." "..ch.name)
    return
  end

  -- Handle character creation letter
  if (lifestyle ~= 0) then
    say("Ah "..ch.name..", I've been expecting you. Welcome!")
    act("$n disappears into a side room and rummages around.", TRUE, me, NIL, NIL, TO_ROOM)

    -- Remove the letter
    extobj(obj)
    obj = NIL
    
    -- Give starting gold based on lifestyle
    local startingGold = {300, 200, 100} -- Poor, Average, Wealthy
    ch.gold = ch.gold + startingGold[lifestyle]
    
    -- Give some basic equipment
    local equipment = {
      [1] = {5303, 5305, 5307}, -- Poor equipment
      [2] = {8040, 8021},       -- Average equipment  
      [3] = {8019, 8023}        -- Wealthy equipment
    }
    
    for _, vnum in ipairs(equipment[lifestyle]) do
      local item = oload(me, vnum, "char")
      if item ~= NIL then
        -- Mark as no-sell
        obj_extra(item, "set", 1) -- ITEM_NOSELL flag
      end
    end
    
    -- Give a backpack with some items
    local pack = oload(me, 8038, "char") -- Backpack
    if pack ~= NIL then
      -- Add a bond certificate
      local bond = oload(me, 1248, "char")
      if bond ~= NIL then
        objfrom(bond, "char")
        objto(bond, "obj", pack)
        -- Set bond values: lifestyle and player ID
        bond.val = {lifestyle, ch.id or 0}
        save_obj(bond)
      end
    end
    
    -- Schedule equipment delivery
    create_event(me, ch, NIL, NIL, "equipped", 1, 1) -- LT_MOB = 1
    return
  end

  -- Handle shop deed refund (90% of cost)
  if obj.val and obj.val[4] then
    local refund = math.floor(0.9 * (obj.val[4] * 10000))
    ch.gold = ch.gold + refund
    
    tell(ch.name, "I'm sorry the store didn't work out for you.")
    act("$n returns a large portion of your gold, keeping the rest as account fees.",
      TRUE, me, NIL, ch, TO_VICT)
    act("$n gives $N a large portion of gold.", TRUE, me, NIL, ch, TO_NOTVICT)
    
    extobj(obj)
  else
    say("This deed appears to be invalid.")
  end
end

function equipped()
-- Give them all of the goodies they need
  action(me, "give all "..ch.name)
  create_event(me, NIL, NIL, NIL, "commence", 4, 1) -- LT_MOB = 1
end

function commence()
-- Now that the player is equipped with some starting equipment, provide guidance
  say("You should now work towards improving your experience in combat. You can fight creatures"..
    " beyond the city walls. When you have sufficient experience, you can seek out the city's teacher.")
  say("They can train you in a number of skills and spells that you will need to survive,"..
    " and you can usually find them in the city library.")
  say("I've given you a bond certificate that you should take to the bank after reaching"..
    " your 5th level. They will provide you with a sum of gold coins to assist you further.")
  emote("goes back to signing some paperwork.")
end