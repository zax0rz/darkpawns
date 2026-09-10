-- Source: scripts_full_dump.txt ./mob/archive/never_die.lua
-- Attached to mob 19113. Restores mob HP to max every pulse (unkillable mob mechanic).

function onpulse_all()
  -- The mob's hit points will be restored to maximum every pulse to prevent the
  -- mob from dying. Attached to mob 19113.

  if (me.hp < me.maxhp) then
    me.hp = me.maxhp
  end
end
