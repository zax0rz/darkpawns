# Verification note — archived Lua scripts report (2026-09-24)

**Artifact:** `archive/dp-archive-scripts-report-2026-09-24.md` (544 lines, 103,927 bytes).
Received as an attachment from Zach 2026-09-24 20:13 EDT; "holy shit look what I discovered.
please save this report somewhere."

## What I checked (same evening, against the sources)

| Report claim | Check | Result |
|---|---|---|
| 165 archived scripts: 115 mob, 14 obj, 36 room | `ls lib/scripts/{mob,obj,room}/archive \| wc -l` | 115 / 14 / 36 ✓ |
| plus 2 in `room/30/` | `ls lib/scripts/room/30` | `pattern_3065.lua`, `pattern_dmg.lua` ✓ |
| `create_event()` absent from this C build | `grep -n create_event src/scripts.c` | body commented at :247, cmdlib entry commented at :1616 ✓ |
| Zones 14,17,22,30,37,53,61,62,68,101,102,120 deleted | `ls lib/world/zon/<z>.zon` | all 12 MISSING (95 zon files total) ✓ |
| `cuchi.lua` hard-codes a privilege escalation | `sed -n 1,24p lib/scripts/mob/archive/cuchi.lua` | real: `if (ch.name == "Orodreth") then … ch.level = LVL_IMPL; save_char(ch)` ✓ |
| Go tree binds functions C never had (report marked this UNVERIFIED) | `git grep origin/main -- pkg/scripting/` | **verified**: `engine.go:646` create_event, `:673-675` get_group_lvl/get_group_pts/skill_group; source comments cite `scripts.c lua_create_event() lines 247-316 (commented out in original)` ✓ |

Not independently checked: the per-script table's 165 rows, the vnum-absence percentage (51%),
and the SVN revision attributions.

## Why this matters (Daeron's read)

1. It is a map of the world that was **deleted**, not merely unattached: twelve zones have no
   world files at all, and roughly half the vnums the scripts name are gone or have been
   reused (dough 8015 is now "a piece of meat").
2. The Go port implements the API these scripts call — including `create_event`, which C
   commented out — so on the Go side the archive is **latent content**, not dead code.
3. The archive documents *intent* in prose, which the C does not. Useful when judging whether
   a surviving C special still matches what its author meant.
4. One re-activated file: `mob/never_die.lua` is live and byte-identical to the archive copy
   apart from its header comment. Any other silent re-attachment would be a divergence risk.
