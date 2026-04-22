-- golem_to_crate.lua
-- Source: scripts_full_dump.txt ./mob/archive/golem_to_crate.lua
-- Transport golem (mob 11700). Collects crystalline chunks (obj 11701) from mining
-- golems and deposits them into wooden crates (obj 11702) in the room.

function onpulse_all()
-- The mob will retrieve crystalline chunks (obj 11701) from any mining golem
-- and deposit them in any wooden crate (obj 11702) it comes across. Attached
-- to mob 11700.

  local alias = ""

  if (me.objs) then
    if (room.objs) then
      for i = 1, getn(room.objs) do
        if (room.objs[i].vnum == 11702) then			-- Found a wooden crate
          if (room.objs[i].weight < 20) then			-- Is the crate full?
            for j = 1, getn(me.objs) do
              if (me.objs[j].vnum == 11701) then
                alias = strsub(me.objs[j].alias, 1, strfind(me.objs[j].alias, "%a%s"))
                action(me, "put all."..alias.." in crate")	-- Put the chunks in the crate
                return
              end
            end
            break
          end
        end
      end
    end
  end
end
