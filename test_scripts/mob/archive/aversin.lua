-- Source: scripts_full_dump.txt ./mob/archive/aversin.lua
-- aversin.lua — sound emote + fight trigger that delegates to take_jail.lua + onpulse_pc that
-- delegates to guard_captain.lua. Attached to the Aversin guard mob.

function sound()
  emote("looks at you.")
  say("Carry on, citizen")
end

function fight()
  dofile("scripts/mob/take_jail.lua")
  call(fight, me, "x")
end

function onpulse_pc()
  dofile("scripts/mob/guard_captain.lua")
  call(onpulse_pc, me, "x")
end
