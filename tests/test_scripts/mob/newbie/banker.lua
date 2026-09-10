-- banker.lua - Banking system for newbies
-- Based on original banker.lua from Dark Pawns MUD
-- Ported for Phase 3D (simplified version)

function ongive()
-- Allows players to cash in their bond certificates (obj 1248) to receive gold.
-- The amount of gold depends on the lifestyle decided during character creation.

  local alias = ""

  if (obj.vnum ~= 1248) then
    say("I'm sorry, I have no use for that...you better keep it.")
    if (strfind(obj.alias, "%a%s")) then
      alias = strsub(obj.alias, 1, strfind(obj.alias, "%a%s"))
    else
      alias = obj.alias
    end
    action(me, "give "..alias.." "..ch.name)
    return
  end

  if (ch.level < 5) then
    say("I'm sorry, you'll need to be level 5 before I can accept this.")
    if (strfind(obj.alias, "%a%s")) then
      alias = strsub(obj.alias, 1, strfind(obj.alias, "%a%s"))
    else
      alias = obj.alias
    end
    action(me, "give "..alias.." "..ch.name)
    return
  end

  -- Check if certificate belongs to player
  -- In original: obj.val[2] contains player ID
  if obj.val and obj.val[2] then
    -- For now, assume it belongs to the player
    -- In full implementation: check obj.val[2] == ch.id
    
    say("Ah, I see you're progressing well "..ch.name)
    
    -- Calculate gold: 1500 / lifestyle (1=poor, 2=average, 3=wealthy)
    local lifestyle = obj.val[1] or 2
    local goldAmount = math.floor(1500 / lifestyle)
    
    act("$n hands $N "..goldAmount.." gold coins.", TRUE, me, NIL, ch, TO_NOTVICT)
    act("$n hands you "..goldAmount.." gold coins.", TRUE, me, NIL, ch, TO_VICT)

    ch.gold = ch.gold + goldAmount
    extobj(obj)
    obj = NIL
  else
    say("Hmmm...this certificate appears to be invalid!")
    act("$n files $p into a desk drawer.", TRUE, me, obj, NIL, TO_ROOM)
    extobj(obj)
    obj = NIL
  end
end

function greet()
-- Banker greeting
  say("Welcome to the bank, " .. ch.name .. "!")
  say("I can help you with bond certificates when you reach level 5.")
  say("Just give me your bond certificate when you're ready.")
end