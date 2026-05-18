-- pet_store.lua
-- Source: scripts_full_dump.txt lines 4132-4185
-- Original: Pet store owner (mob 21245). Displays pets from adjacent room (room.vnum + 1),
-- sells them at level*100 gold, and makes the pet follow the buyer.
-- Triggers: oncmd()
-- Engine gaps: load_room(), mload(), follow()

function oncmd()
  -- Source: scripts_full_dump.txt lines 4134-4185
  local command = ""
  local subcmd = ""
  local buf = ""
  local price = 0
  local vnum = 0
  local found = NIL
  local temp = NIL

  if (strfind(argument, "%a%s") ~= NIL) then
    command = strsub(argument, 1, strfind(argument, "%a%s"))
    subcmd = gsub(argument, command.." ", "")
  else
    command = argument
  end

  if (not ch) then
    return
  end

  if (command == "list") then
    local pet_room = load_room(room.vnum + 1)   -- Look in the stored pet room
    act("Available pets are:", FALSE, ch, NIL, NIL, TO_CHAR)
    for i = 1, getn(pet_room.char) do
      local pet = pet_room.char[i]
      if (isnpc(pet)) then
        price = pet.level * 100                   -- 100 gold per pet's level
        buf = format('%5d - %s', price, pet.name)
        act(buf, FALSE, ch, NIL, NIL, TO_CHAR)
      end
    end
    return (TRUE)
  elseif (command == "buy") then
    if (subcmd == "") then
      tell(ch.name, "What would you like to buy?")
      return (TRUE)
    end

    local pet_room = load_room(room.vnum + 1)
    for i = 1, getn(pet_room.char) do
      local pet = pet_room.char[i]
      if (strfind(pet.alias, subcmd)) then
        found = TRUE
        price = pet.level * 100
        vnum = pet.vnum
      end
    end

    if (not found) then
      tell(ch.name, "What would you like to buy?")
      return (TRUE)
    end

    if (ch.gold < price) then
      tell(ch.name, "You don't have enough gold, come back later!")
      return (TRUE)
    end

    -- Load the pet into the current room and make it follow the buyer
    pet = mload(vnum, room.vnum)
    temp = me
    me = pet
    follow(ch, TRUE)
    me = temp
    ch.gold = ch.gold - price
    save_char(ch)
    tell(ch.name, "Here you are, may you enjoy your pet.")
    act("$n purchases $N as a pet.", TRUE, ch, NIL, pet, TO_ROOM)
    return (TRUE)
  end
end
