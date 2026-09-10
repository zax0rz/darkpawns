-- paladin.lua - Combat AI for paladin mobs
-- Ported from original paladin.lua in Dark Pawns MUD (lib/scripts/mob/archive/paladin.lua)
-- Paladin performs class skills during combat via action() (parry/bash/charge/disarm)
-- and uses alignment-sensitive DISPEL spells.
--
-- case distribution (number(0,20) → 0–20, 21 outcomes):
--   case 0        (~1/21): parry
--   case 1        (~1/21): bash <target>
--   case 2        (~1/21): charge <target>
--   case 3        (~1/21): dispel evil or dispel good (alignment check)
--   case 4 or >5  (~15/21): return (no action)
--   case 5        (~1/21): disarm <target>
--
-- NOTE: uses stubbed function action() — engine.go luaAction() dispatches mob commands.
-- parry/bash/charge/disarm are game skill commands; they may be stubs in the engine.

function fight()
-- Allows the mob to perform paladin skills during combat  -- paladin.lua line 2 (comment)

  local case = number(0, 20)                             -- paladin.lua line 4: roll 0–20

  if ((case == 4) or (case > 5)) then                    -- paladin.lua line 6: no-op on most rolls
    return
  end

  if (case == 0) then                                     -- paladin.lua line 10: parry (no target needed)
    action(me, "parry")                                   -- paladin.lua line 11
  elseif (case == 1) then                                 -- paladin.lua line 12: bash target
    action(me, "bash "..ch.alias)                         -- paladin.lua line 13
  elseif (case == 2) then                                 -- paladin.lua line 14: charge target
    action(me, "charge "..ch.alias)                       -- paladin.lua line 15
  elseif (case == 3) then                                 -- paladin.lua line 16: alignment dispel
    if ((me.evil == FALSE) and (ch.evil == TRUE)) then    -- paladin.lua line 17: good paladin vs evil target
      spell(ch, NIL, SPELL_DISPEL_EVIL, TRUE)             -- paladin.lua line 18: SPELL_DISPEL_EVIL (22)
    elseif ((me.evil == TRUE) and (ch.evil == FALSE)) then  -- paladin.lua line 19: evil paladin vs good target
      spell(ch, NIL, SPELL_DISPEL_GOOD, TRUE)             -- paladin.lua line 20: SPELL_DISPEL_GOOD (46)
    end
  elseif (case == 5) then                                 -- paladin.lua line 22: disarm target
    action(me, "disarm "..ch.alias)                       -- paladin.lua line 23
  end
end
