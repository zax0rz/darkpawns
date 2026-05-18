-- shopkeeper.lua
-- Source: scripts_full_dump.txt lines 4693-4703
-- Original: Attached to shopkeeper mobs. Attacks specific animal mobs on sight.
-- Triggers: greet()

function greet()
  -- Original: Attacks mobs with vnums 8063, 12115, 18203 on sight
  -- Source: scripts_full_dump.txt line 4695
  if (ch.vnum == 8063) or (ch.vnum == 12115) or (ch.vnum == 18203) then
    emote("mutters something about filthy animals in $s shop.")
    action(me, "kill "..ch.name)
  end
end
