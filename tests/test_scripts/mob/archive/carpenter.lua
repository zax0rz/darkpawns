-- carpenter.lua - Ambient toolbelt/sweat sounds
-- Source: scripts_full_dump.txt ./mob/archive/carpenter.lua

function sound()
  if number(0, 1) == 0 then
    emote("adjusts his toolbelt.")
  else
    emote("wipes the sweat of labor from his brow.")
  end
end
