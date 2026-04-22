-- paralyse.lua - Combat AI: chance-based bite that casts SPELL_PARALYSE
-- Source: scripts_full_dump.txt ./mob/archive/paralyse.lua
-- TODO: requires SPELL_PARALYSE constant (not in original globals.lua; value 105 assigned in engine.go)

function fight()
  if (number(0, (102 - ch.level)) == 0) then
    act("$n bites $N!", TRUE, me, NIL, ch, TO_NOTVICT)
    act("$n bites you!", TRUE, me, NIL, ch, TO_VICT)
    spell(ch, NIL, SPELL_PARALYSE, FALSE)
  end
end
