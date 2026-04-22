-- beggar.lua - Ambient begging sound strings
-- Source: scripts_full_dump.txt ./mob/archive/beggar.lua

function sound()
  if (number(0, 1) == 0) then
    say("Spare a coin, buddy?")
  else
    emote("jingles his cup.")
  end
end
