-- Source: scripts_full_dump.txt ./mob/archive/mymic.lua
-- mymic.lua — onpulse_pc that steals gold and food from players in the room.
-- Steals 1 out of 10 pulse ticks (number(0,9)==0); gold amount is 1-10% of player gold.
-- TODO: requires steal(ch, obj) implementation — steal an item from a character

function onpulse_pc()
-- Mob will steal gold and/or food from a player in the same room.

  if (room.char) then					-- Are there people here?
    local gold = 0					-- Initialise local "gold"
    if (number(0, 9) == 0) then			-- Steal 1 out of 10 times
      for i = 1, getn(room.char) do
        if (not isnpc(room.char[i])) then
          ch = room.char[i]
          gold = round((ch.gold * number(1, 10)) / 100)
          if (ch.objs) then
            for j = 1, getn(ch.objs) do
              if (ch.objs[j].type == ITEM_FOOD) then
                if (number(0, 3) == 0) then
                  steal(ch, ch.objs[j])		-- TODO: requires steal() implementation
                end
              end
            end
          end
          me.gold = me.gold + gold		-- Increase mymic's gold
          ch.gold = ch.gold - gold		-- Decrease player's gold
          save_char(ch)
        end
      end
    end
  end
end
