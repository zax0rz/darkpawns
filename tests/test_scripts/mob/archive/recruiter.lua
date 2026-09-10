-- recruiter.lua
-- Source: scripts_full_dump.txt lines 4448-4458
-- Original: Ambient NPC with sound() trigger — random emotes when players are present.
-- Triggers: sound()

function sound ()
  -- Source: scripts_full_dump.txt lines 4449-4453
  if (number(0, 1) == 0) then
    emote("smiles at you")
  else
    emote("shuffles some papers around on his desk.")
  end
end
