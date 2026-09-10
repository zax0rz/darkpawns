-- pyros.lua — Pyros combat + death handler (mob 1410)
-- Source: lib/scripts/mob/archive/pyros.lua
-- Bug fix: fight() was calling call(fight, room.char, "x") — room.char is the entire
-- room character table, not the combat target.  Must pass ch (the combat target).

function fight()
-- Allow Pyros to cast sorcery spells. Attached to mob 1410.
  dofile("scripts/mob/sorcery.lua")
  call(fight, ch, "x")  -- fixed: was room.char; ch is the combat target
end

function death()
-- Reset global variable to allow another ceremony.
  keep_pyros = FALSE
end
