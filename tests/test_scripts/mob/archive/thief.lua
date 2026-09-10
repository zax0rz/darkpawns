-- Source: scripts_full_dump.txt ./mob/archive/thief.lua
-- thief.lua — onpulse_pc that pickpockets gold from players at 20% chance per pulse
-- (number(0,4)==0), then level-scaled detection chance, stealing 1-10% of player gold.

function onpulse_pc()
  if (number(0, 4) == 0) then
    if (me.pos == POS_STANDING) then
      if (room.char) then
        for i = 1, getn(room.char) do
          if (not isnpc(room.char[i]) and (room.char[i].level < LVL_IMMORT)) then
            if (number(0, room.char[i].level) == 0) then
              act("You discover that $n has $s hands in your wallet.",
                FALSE, me, NIL, room.char[i], TO_VICT)
              act("$n tries to steal gold from $N.", TRUE, me, NIL, room.char[i], TO_NOTVICT)
            end

            local gold = round((room.char[i].gold * number(1, 10)) / 100)
            if (gold > 0) then
              me.gold = me.gold + gold
              room.char[i].gold = room.char[i].gold - gold
              save_char(room.char[i])
            end
          end
        end
      end
    end
  end
end
