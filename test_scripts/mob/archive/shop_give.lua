-- shop_give.lua
-- Source: scripts_full_dump.txt lines 4670-4691
-- Original: Validates that items given to shopkeepers are production items.
-- Prevents players from giving non-production items to player-owned store shopkeepers.
-- Triggers: ongive()

function ongive()
  -- Source: scripts_full_dump.txt line 4676
  -- item_check() validates whether the object is a production item for this shop
  -- Engine gap: item_check() is not yet stubbed — always returns false for now
  local alias = ""

  if (not item_check(obj)) then    -- Item is not a production one!
    say("Sorry "..ch.name..", but I don't deal in such items.")
    if (strfind(obj.alias, "%a%s")) then
      alias = strsub(obj.alias, 1, strfind(obj.alias, "%a%s"))
    else
      alias = obj.alias
    end
    action(me, "give "..alias.." "..ch.name)
    return
  end
end
