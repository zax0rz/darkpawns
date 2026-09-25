# 2026-09-25: Certifying the Lua bindings

Field notes from DP-1333 PR 3. It certifies the Lua bindings that no script
C ships calls. Claims are backed by PF-033 to PF-035.

## The vehicle

No attached C script calls 38 of C's 54 cmdlib functions, so the census
could not see them. Two harness fixtures make them visible. A
`[lua-script <path>]` section writes one probe script into both servers'
script trees, and `mob-script <vnum> <path> <flags>` attaches it to a
mobile in both worlds. The probe is an `oncmd` script: `say <group>` runs
one group of bindings and says each result aloud, so the transcript
carries the return values. Where a binding changes the world, a later
command shows the change the way a player would: `inventory` for an
object flag, walking into a closed door for an exit flag.

## What the first probe found

The bindings are small, but the probes kept landing in game code:

- `room_to_table` sets `struct` (scripts.c:1971). The PR before this one
  had declared that it does not and excluded `save_room` as C undefined
  behaviour. It was wrong, and so was the independent review that passed
  it. `save_room` works in C, and now in the port.
- Mob flag checks by name read the prototype, but C's `MOB_FLAGGED` reads
  the mobile's own bits. A flag set at runtime was invisible to every
  name-based check. That includes the HUNTER flag a special procedure
  already sets on hired mobiles.
- The first change to an object's extra flags started from zero, not from
  the prototype's flags. `obj_extra(o, "set", HUM)` on a glowing tooth
  left it humming but no longer glowing, visible in `inventory`. The same
  path serves `enchant weapon`, `enchant armor` and the object form of
  bless, curse and invisibility. The last group used a getter whose type
  never matched, so it always started from zero too.

None of these three is a scripting bug. They surfaced because the probe
asked the game a question no game path had asked before.
