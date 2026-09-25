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

## Batches 2 to 4: the rest of the surface

Four probes (queries, flags, objects, actions) and a last miscellaneous one
certified 54 cases against C. Eight are blocked, each with a stated reason.
The pattern from the first batch held: the probes mostly found game bugs,
not binding bugs.

- `look` never printed the aura lines C puts under a character
  ("...he glows with a bright light!"). It also dropped the space C's
  `do_description` adds after a room with extra descriptions. The port had a
  special case adding that space before autoexits only, which modelled the
  symptom where it first showed up rather than its cause.
- Every mobile spawn told the room "<mob> appears." C's `read_mobile` and
  `char_to_room` are silent, so each zone reset in an occupied room showed
  players a line C never sends. Three tests had been written to swallow it.
- The legacy `spell` binding hand-wrote spell effects: "teleport" moved the
  target to a fixed room. The legacy `gossip` printed a `[gossip]` format C
  does not have.
- PR #1637, from the day before, had stored a script's `me.timer` in a new
  field, although the port already kept `GET_MOB_WAIT` elsewhere. Combat
  read one field and scripts wrote the other. The same session that shipped
  the duplicate found it while reading `cast_spell`'s gates.

## A green that proved nothing

The first action probe passed completely, but its spell groups were silent
on both sides. The healer was sitting, and both C's and the port's
`cast_spell` refuse a seated caster without a word. A probe that asks
`action(me, "stand")` first turned the silence into output. C's healer
stood, cast sanctuary and cured; the port's could not stand at all
(`stand` was not an NPC command). The fix and the proof came from making
the effect visible, not from the first green result.

## One C rule, three hand-fitted spacers

Before `do_description` was ported, the port had already reproduced its
trailing space in three places, each fitted to the one scenario where it
first diverged:

- a space before autoexits on the command path only;
- a function that re-rendered the elements_galeru look and inserted a space
  before the first player's line;
- an arrival act hard-coded as `" $n appears in a brilliant flash of light."`.

Each comment described the space as C's observed "framing". Porting the rule
once made the first two redundant, and the census caught the second when it
doubled the space. The third still stands in for C's carry-over of an unsent
space into the looker's next output, and its comment now says so.

## The syslog nobody could turn on

The Lua `log` binding was blocked for a reason outside Lua: the port's
`mudlog` was complete except for one call. `SetImmortalSessionProvider` was
defined and never called, so every `MudLog` reached the server log and no
player. Zach had tried `syslog complete` as an immortal the day before and
seen nothing. The command set the level correctly; the broadcast had no
list of sessions to walk.

Wiring it exposed the next layer. The port had four `MudLog` calls where C
has about 140. The everyday lines (connect, quit, death, link loss) were
server-log calls in the port, and two of them used the wrong message type
or level. The level arithmetic was also wrong: with both log bits set, C
counts 3 and the port counted 2.
