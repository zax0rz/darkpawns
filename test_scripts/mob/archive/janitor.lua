-- Source: scripts_full_dump.txt ./mob/archive/janitor.lua
-- janitor.lua — bribe emote; ongive acknowledgement; onpulse_all picks up non-corpse
-- items from the floor.
-- TODO: requires iscorpse(obj) implementation — checks if object is a player/mob corpse
-- TODO: requires canget(obj) implementation — checks if mob can pick up the object

function bribe()
  emote("tips his hat and smiles a thank you.")
end

function ongive()
  if (obj.cost <= 10 or obj.type == ITEM_DRINKCON) then
    say("Thanks for helping to clean this place up...")
  else
    say("Wow, this is pretty neat, thanks!")
  end
end

function onpulse_all()
  if (room.objs) then
    for i = 1, getn(room.objs) do
      obj = room.objs[i]
      if (not iscorpse(obj) and canget(obj)) then	-- TODO: requires iscorpse/canget
        act("$n picks up some trash.", FALSE, me, NIL, NIL, TO_ROOM)
        objfrom(obj, "room")
        objto(obj, "char", me)
      end
    end
  end
end
