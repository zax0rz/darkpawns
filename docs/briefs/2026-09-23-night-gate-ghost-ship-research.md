# Research: The Night Gate and the Ghost Ship

Source: original C (`src/gate.c`, `src/new_cmds.c`, `src/weather.c`) and world
files (`lib/world/`). All vnums and strings verified 2026-09-23. This is the
raw material for the blog post of the same title (PR #1610, DP-1316 / issue
#1582).

## The clock that drives both

`src/weather.c`: at **hour 21** (sunset) the game calls `ghost_ship_appear()`
then `load_night_gate()`. At **hour 5** (sunrise) it calls
`ghost_ship_disappear()` then `remove_night_gate()`. Both are guarded by
`mini_mud` (skipped in mini-mud mode). Flavor worth stealing: Dark Pawns has
**two suns** — "The suns rise in the east and north."

## The night gate

- Built by **dlkarnes** (Derek Karnes, aka Serapis) on **970114** — January 14,
  1997. The design doc is embedded as a comment block in `src/gate.c`,
  including: *"the 8 gates in rooms 4011-4018 are in 'the void' or basically
  in the middle of nowhere. :P"*
- 16 entries in `gate_phase[]`. Columns: load room, moon phase to load on,
  exit room A, exit room B.
- Entries 0–7 are moon-phase-gated (one per moon phase, 0–7), loading into
  rooms 4004–4008 — all named **"A Secluded Grove"** (zone 40). The comment
  says these 8 *"will be spread around the world in various places"* —
  aspirational, never happened; they all sit in the grove.
- Entries 8–15 have phase **-1**: they never auto-load at sunset. The void
  gates.
- What appears: object **4001**, *"a shimmering blue portal"* /
  *"A shimmering portal of blue light hovers in the middle of the room."*
  Room-local message: *"A shimmering portal of blue light suddenly appears
  in the darkness!"*
- At dawn the portal is extracted with: *"The shimmering blue portal of
  light fades out of existence."*
- `enter portal` (spec proc `moon_gate`): teleports you to exit room A or B
  (coin flip, `number(0,3)<2`). Send: *"You enter the portal and are
  transferred..."* Arrival: *"With a sound like a raging river, a hole in
  space opens up and $n steps out."* Mounts travel with you.
- Because the 8 moon phases each match exactly one gate, **one portal
  appears in the grove each night**, and entering it shuffles you to one of
  two grove rooms at random. It is a moon-phase-gated teleport lottery, not
  a shortcut to anywhere.
- Related, unported-adjacent mechanic: the `gate` spell creates **red**
  portals (obj 4002) with a ticking timer, and casting one in the grove
  while a portal exists consumes the grove: *"As you watch the red portal
  slowly fade into existence, the existing portal pulses once, then begins
  to expand, consuming the new portal, and then the entire grove. The
  fabric of time and space warps and stretches"* — a whole other post.

## The ghost ship

- Zone **191: "Phantom Pirate Ship (Grimmy)"** — builder credit Grimmy.
  Rooms 19100–19199.
- At sunset, `ghost_ship_appear()`: a coin flip (`number(0,1)`, a real RNG
  draw) picks dock **19173** or **19174**. A north exit is created from the
  dock to room **19100**, and a south exit from 19100 back to the dock.
  - Dock 19173: *"You stand on small floating dock, a narrow extension of
    the main pier to the west."* (exits east to 12711, west to 4800)
  - Dock 19174: *"You stand on a small wooden dock; to the north lies only
    water. To the south is the city gate."* (south through an iron door to
    21209)
  - Room messages are room-local: *"Suddenly a ghostly ship appears to the
    north!"* on the dock, *"Suddenly a dock appears to the south!"* aboard.
- At dawn, `ghost_ship_disappear()`: the exits are freed. Aboard the ship
  you see *"Suddenly the dock to the south disappears!"* and *"The ghost
  ship has set sail!"*
- **The ship is the "Ocean Fate"**, captained by **Skullgrin** (mob 19101:
  *"a floating decomposing humanoid"* in tattered, blood-splattered
  clothing). Crew: ~18 **Phantom Crewmembers** (19100), a **Ghastly Sea
  Serpent** (19102), the rotting **Lookout** (19103), a **Prison Guard**
  (19116, guarding a brig), and a **Bitter Wind** (19119 — a wind you can
  fight).
- Entry room 19100, **"Ghost Ship Entry Ramp"**: *"You see before you an
  ominous looking rickety old plank that seems to dare you to enter the
  dark and forboding ship looming before you. You can hear what sounds
  like an unnatural wailing nearby..."* (sic — "forboding" is the file's
  spelling)
- **Quirk worth a sidebar**: room 19100 ships with a *static* south exit to
  19174, and 19174 with a static north exit to 19100 — but `appear()`
  overwrites them and `disappear()` frees them unconditionally. After the
  first dawn, the static exits are gone for good (until reboot). The
  dynamic sunset exits are the only way on and off from then on.
- **Stranding is real**: at dawn the exits are destroyed. If you are aboard
  when the ship sets sail, your way back is gone until the next sunset
  (room 19100's static down exit to 19101 still works — you can explore the
  ship, you just cannot leave). No exits lead from zone 191 to the outside
  world except the dynamic dock exits.

## What the Go port did wrong (the blog's act one)

`pkg/game/weather.go` replaced all four mechanics with world-wide
broadcasts: every player in the game saw `[ GHOST SHIP ]` / `[ NIGHT GATE
]` announcements at dawn. The strings exist nowhere in C or the world
files. The portal never materialized, the exits were never created, the
coin flip never drawn — but the announcement went out on schedule. Fixed
in PR #1610: real objects, real exits, the RNG draw where C draws it,
room-local messages, awake-recipient behavior matching C's `send_to_room`.

## Bonus weird discovery (series fodder)

`lunar_hunter()` in `src/new_cmds.c`: on full-moon days (days 22–25 of the
month), at sunset, every player above level 12 has a 1-in-6 chance of a
mob spawning in room 8067 that tells them *"<name> By the light of the
full moon.. DIE!"* A full-moon assassin with a catchphrase.

## The lunar system: vampirism and lycanthropy

Player flags `PLR_VAMPIRE` / `PLR_WEREWOLF` (`src/structs.h`), transform
via `do_transform`, forced re-transformation by moonlight. It is all one
designer's work — **dlkarnes, January 1997**:

- `full_moon()` (970128): on full-moon days (22–25) at sunset, every
  vampire/werewolf player not currently transformed is *forced* to
  transform: *"The lunar light infuses your body, forcing you to
  transform!"* / *"Racked with the pain of the transformation, your head
  is thrown back and an unearthly moan escapes your lips."*
- `lunar_hunter()` (same sunset hook): a hunter mob spawns in room 8067
  and tells a random level 13+ player *"By the light of the full moon..
  DIE!"*
- Vampires can only drink blood (`LIQ_BLOOD`, `src/act.item.c`); the
  `spike` command kills werewolves, `stake` kills vampires
  (`src/new_cmds.c`).

The moon is a game mechanic: it opens the night gates (phase-matched
portals), forces your transformation at full moon, and sends hunters
after high-level players. The night gate (970114), the full-moon
transformation and the lunar hunter (970128) are two weeks apart from
the same builder — a lunar cycle game loop, designed as a unit.
