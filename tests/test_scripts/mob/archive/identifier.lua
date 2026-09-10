-- identifier.lua
-- Source: scripts_full_dump.txt lines 3059-3156
-- Original: Item identification NPC with dynamic pricing formula and spell context swap.
-- Pricing: <5000 cost = cost/10, >=5000 cost = cost*0.14, +cost/20 if ITEM_MAGIC.
-- On give: swaps me/ch context so spell(SPELL_IDENTIFY) runs with player as caster.
-- Triggers: oncmd, ongive()

function oncmd()
  -- Source: scripts_full_dump.txt lines 3061-3096
  local command = ""
  local subcmd = ""
  local price = 1

  if (strfind(argument, "%a%s") ~= NIL) then
    command = strsub(argument, 1, strfind(argument, "%a%s"))
    subcmd = gsub(argument, command.." ", "")
  else
    command = argument
  end

  if (command == "list") then
    tell(ch.name, "Just read the sign!")
    return (TRUE)
  elseif (command == "value") then
    if (subcmd == "") then
      tell(ch.name, "Value what?")
      return (TRUE)
    end

    if (not obj_list(subcmd, "vict")) then
      tell(ch.name, "You don't seem to have that.")
      return (TRUE)
    else
      -- Dynamic pricing formula: source lines 3085-3091
      if (obj.cost < 5000) then
        price = round(obj.cost / 10)
      else
        price = round(obj.cost * 0.14)
      end

      if (obj_flagged(obj, ITEM_MAGIC)) then
        price = price + round(obj.cost / 20)
      end
    end

    tell(ch.name, "I'll identify that fully for about "..price.." coins.")
    return (TRUE)
  end
end

function ongive()
  -- Source: scripts_full_dump.txt lines 3098-3156
  local temp_ch = NIL
  local price = 1

  -- Dynamic pricing formula (same as oncmd)
  if (obj.cost < 5000) then
    price = round(obj.cost / 10)
  else
    price = round(obj.cost * 0.14)
  end

  if (obj_flagged(obj, ITEM_MAGIC)) then
    price = price + round(obj.cost / 20)
  end

  if (ch.gold < price) then
    tell(ch.name, "That's a fine item, but I'll need "..price.." coins from you "
                  .."to id it.. and you're a little short..")
    tell(ch.name, "Keep it until you get the gold.")
    action(me, "give all "..ch.name)
  else
    ch.gold = ch.gold - price
    save_char(ch)

    act("\r\n$n studies it carefully, comparing it to ancient texts, weighing it on scales, and "
        .."chanting a number of odd spells over its surface.", TRUE, me, NIL, NIL, TO_ROOM)

    act("$N touches your forehead, and knowledge fills your mind.\r\n", TRUE, ch, obj, me, TO_CHAR)
    act("$N touches $n gently on the forehead.", TRUE, ch, obj, me, TO_NOTVICT)

    -- Context swap: swap me and ch so SPELL_IDENTIFY runs with player as caster
    -- Source: scripts_full_dump.txt line 3143-3145
    temp_ch = me
    me = ch
    spell(NIL, obj, SPELL_IDENTIFY, FALSE)
    me = temp_ch

    act("\r", FALSE, ch, NIL, NIL, TO_CHAR)
    act("\r", FALSE, ch, NIL, NIL, TO_CHAR)
    action(me, "give all "..ch.name)
  end
end
