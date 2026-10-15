# DeepSeek — round 6: the room-light counter gap, and where §7 stands

Date: 2026-10-15 · Branch: `deepseek/draw-list-order` (off `origin/main`
`037fc398b`) · Rules: R1, R3a, R5c, R5e, R5g, R5h

No production change. Zone boot order is ruled out per the brief (`lib/world/zon/index`
is numeric and sorted), and the round's scenario runs could not be taken: a census
(`batch-l`, 3,429 pairs) started mid-round and held the host with ~37 workers, so
both `dp-oracle-diff` attempts returned empty output. The findings below are
static, from `src/` and `pkg/` reads.

## The room-light counter is missing from the port (R1 gap, new)

C keeps a per-room light **counter** and mutates it in five places:

```
src/handler.c:521   char_from_room      world[ch->in_room].light--
src/handler.c:542   char_to_room        world[room].light++
src/handler.c:740   equip_char          world[ch->in_room].light++   (a WORN light)
src/handler.c:782   unequip_char        world[ch->in_room].light--
src/handler.c:1064  extract_obj/from_char  world[ch->in_room].light--
```

and `CAN_SEE` depends on it: `LIGHT_OK(sub) = !AFF_BLIND && (IS_LIGHT(sub->in_room)
|| AFF_INFRAVISION)`, `IS_LIGHT(room) = !IS_DARK(room)`, and `IS_DARK(room) =
!world[room].light && (ROOM_DARK || ((SECT != INSIDE && SECT != CITY) &&
(SUN_SET || SUN_DARK)))` (`src/utils.h:254-261`, `:515-517`).

Go's `Room.Light` is only the parsed world-file value: `Room.IsLight()` returns
`r.Light > 0` (`pkg/parser/wld.go:58-60`) and **no code in `pkg/` increments or
decrements it** — grepping `light++`/`light--`/`AddLight`/`AdjustLight`/`roomLight`
across `pkg/game` returns nothing. So a room lit only by a carried or worn light
object, or by a light-producing affect, is dark in Go and lit in C, and every
`canSee`-dependent path (the target resolvers, `act()`'s `$n`/`$N`, `PERS`,
`look`) degrades to the dark-room behaviour.

**This is not §7's cause in room 8162**, and that is worth stating precisely:
8162's flags are `80 28 0 0 0 0` (`lib/world/wld/80.wld:3055`), so `ROOM_DARK`
(bit 0) is clear and the sector is 0 (`SECT_INSIDE`), which makes `IS_DARK(8162)`
false regardless of the counter — the room is lit by construction. It is a real
gap for other rooms and other scenarios, and it wants its own fix with a unit
control for the five mutation sites; it is not on this branch because the brief
allows no speculative patch and no oracle run was available to build the control.

## §7 — the resolver: the two room sources disagree

`canSee` is *not* the explanation here (above). The remaining suspect is the
**room-index source**, and rounds 4 and 5 already produced the evidence that the
two sources disagree:

- Round 4, inside `cmdHit`'s `!found` branch, `w.GetMobsInRoom(s.player.GetRoom())`
  printed **no** mobs for 8162 at the moment `kill fish` ran.
- Round 5's wander trace printed the fish's own state in the same window:
  `room=8162 pos=8`, i.e. `mob.GetRoom()` says 8162.

`GetMobsInRoom` (`pkg/game/world.go:1147-1166`) filters `w.activeMobs` by
`GetRoom()`, while `MobileActivity`/`RoomActivity` walk `cOrderedCharacters()`
(`pkg/game/world_order.go`). If a reset- or fixture-spawned mobile is in the
character list but not in `activeMobs`, every `GetMobsInRoom` consumer — including
`ResolveCharInRoom` — misses it while the mob still pulses, wanders and renders.
That single divergence explains "present, standing, keywords `fish carve_meat`,
resolves nothing".

**Next check (one run, once the census is clear):** the `DP_ZZRES` trace is
written and ready — it prints, for every candidate whose keywords contain `fish`,
the name, keywords, `canSee`, `isnameWithAbbrevs`, both rooms and `IsRoomDark` —
plus a one-line `len(w.activeMobs)` vs `len(cOrderedCharacters())` probe at the
same instant. If the fish is absent from `activeMobs` while present in the
character list, the fix is the spawn path's registry insert (`World.spawnMob` sets
`w.activeMobs[w.nextMobID] = mob`, `pkg/game/world.go:954`, so the divergence
would be a spawn path that bypasses it, e.g. the harness fixture's or a
zone-update reset's).

## Thread (1) — list order: method, and what is already ruled out

Not advanced this round (no runs). Ruled out by the brief: zone boot order
(`lib/world/zon/index` is numeric and sorted). Already verified in earlier rounds:
`nextCharacterSequence` increments once per spawn with no ties
(`pkg/game/world.go:950-951`), so `cOrderedCharacters` is exactly the reverse
spawn order, which is C's `read_mobile` prepend.

The remaining suspects, in the order I would test them:

1. **`GetMobsInRoom` vs `activeMobs`** (above) — same root, and it would also
   shift target selection in `damage()`'s room walks, which is where §3's
   "C reaches a player victim, Go does not" would come from.
2. **The fixture's insertion point**: `applyMobFixtures` patches the zone files
   for both engines, so the fixture's `M` command sits at a fixed place in the
   file; if the two engines' reset walks treat the appended command's position
   differently (Go's `zone.Commands` order vs C's `ZCMD` scan), the fixture mob's
   list position differs.
3. **Players at login**: C inserts the descriptor's character into
   `character_list` at `char_to_room`; Go assigns `player.characterSequence` in
   `AddPlayer` (`pkg/game/world.go:551-552`). If the login insert happens at a
   different point relative to the setup pulses' resets, the player's position
   shifts and every per-mob draw after it is reassigned.

The measurement to take, once the host is free: a diagnostic scenario probe that
walks the affected rooms with `at <room> look` (or `mwhere <vnum>`) so both
engines' room contents and order are *byte-compared* in the transcript, alongside
Go's `cOrderedCharacters` dump — the first position where the two lists differ is
the bug.
