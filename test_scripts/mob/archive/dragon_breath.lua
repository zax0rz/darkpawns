-- dragon_breath.lua - Combat AI for dragon mobs with breath weapons
-- Source: dragon_breath.lua (origin/master:lib/scripts/mob/archive/dragon_breath.lua)
-- Ported for Phase 5b world restoration

function fight()
  -- Vnum-to-spell mapping for each dragon variant; dragon_breath.lua lines 2-9
  local dragons = {
    [4209]  = SPELL_FROST_BREATH,     -- dragon_breath.lua line 3
    [4705]  = SPELL_FROST_BREATH,     -- dragon_breath.lua line 4
    [10200] = SPELL_FIRE_BREATH,      -- dragon_breath.lua line 5
    [10300] = SPELL_FIRE_BREATH,      -- dragon_breath.lua line 6
    [10301] = SPELL_ACID_BREATH,      -- dragon_breath.lua line 7
    [10302] = SPELL_LIGHTNING_BREATH, -- dragon_breath.lua line 8
    [20027] = SPELL_GAS_BREATH        -- dragon_breath.lua line 9
  }
  if (number(0, 14) == 0) then        -- ~6.7% chance per round; dragon_breath.lua line 11
    spell(ch, NIL, dragons[me.vnum], FALSE) -- breath weapon on target; dragon_breath.lua line 12
    -- NOTE: uses stubbed function spell
  end
end

function greet()
  if (me.vnum == 4209) then           -- vnum 4209 only; dragon_breath.lua line 16
    act("$n looks at you.", TRUE, me, NIL, NIL, TO_ROOM)                         -- dragon_breath.lua line 17
    act("$n growls, 'So, you have found my lair...'", TRUE, me, NIL, NIL, TO_ROOM) -- dragon_breath.lua line 18
    act("$n exclaims, 'For that you must die!'", TRUE, me, NIL, NIL, TO_ROOM)   -- dragon_breath.lua line 19
    action(me, "kill "..ch.name)      -- auto-aggro greeter; dragon_breath.lua line 20
    -- NOTE: uses stubbed function action
  end
end
