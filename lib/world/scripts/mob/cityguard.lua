-- The cityguard script is triggered when a player enters the guard's room.
-- If the player is an outlaw (PLR_OUTLAW flag set), the guard will attack.
-- If the player is a werewolf (PLR_WEREWOLF flag set) or vampire (PLR_VAMPIRE flag set),
-- the guard will also attack.
-- SOURCE: original cityguard.lua lines 1-8

function onpulse_pc()
  -- Check if the player is an outlaw
  -- SOURCE: original cityguard.lua lines 10-11
  if (plr_flagged(ch, PLR_OUTLAW) == TRUE) then
    act("$n shouts 'Halt, outlaw!'", FALSE, me, NIL, ch, TO_ROOM)
    action(me, "kill", NIL, ch)
    return
  end

  -- Check if the player is a werewolf
  -- SOURCE: original cityguard.lua lines 15-16
  if (plr_flagged(ch, PLR_WEREWOLF) == TRUE) then
    act("$n shouts 'Die, beast!'", FALSE, me, NIL, ch, TO_ROOM)
    action(me, "kill", NIL, ch)
    return
  end

  -- Check if the player is a vampire
  -- SOURCE: original cityguard.lua lines 20-21
  if (plr_flagged(ch, PLR_VAMPIRE) == TRUE) then
    act("$n shouts 'Die, bloodsucker!'", FALSE, me, NIL, ch, TO_ROOM)
    action(me, "kill", NIL, ch)
    return
  end
end

-- Note: This script relies on plr_flagged() function which is currently stubbed
-- TODO: Implement proper plr_flagged() function in engine.go