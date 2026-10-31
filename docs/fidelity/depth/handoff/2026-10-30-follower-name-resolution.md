# Name-based follower resolution — audit list for after the release (2026-10-30, deepseek)

Found while fixing `print_group`'s leader lookup (#2250). Deferred: this is a
correctness class across several commands, not a colour fix, and the release
branch should not take it.

## The defect

C stores the follow link as a **pointer** — `ch->master` — and every consumer
walks the leader's own list, `k->followers` (`src/utils.c:463-475`,
`src/act.other.c:649-676`, `src/fight.c:637-657`). Two NPCs that share a name are
therefore two unrelated follower lists.

The port stores the name, and `World.GetFollowerActors(name)` rebuilds the list by
matching `GetFollowing()` against that string. With duplicate NPC names the match
merges both lists, and `GetMobByName` (a substring match over a map-ordered slice)
can pick either duplicate from run to run — an R3 violation, not just an R1 one.

`#2250` fixed `print_group` only. `World.FollowingActor(body)` (the retained body
through `combatFollowingBody`) and `World.FollowersOf(body)` are the identity-based
replacements; `SetFollowingBody` is the setter that keeps duplicates apart
(`pkg/game/follow.go:395`).

## Sites to check

Every one needs its C call path read before changing it (R5e); some may be
legitimately name-keyed because C is too (a player-only lookup, a persisted field,
a display string).

| site | what it builds |
|---|---|
| `pkg/game/act_movement.go:547` | follower notices on movement |
| `pkg/game/out_of_band.go:81` | out-of-band follower broadcast |
| `pkg/game/flee.go:286` | followers fleeing with the leader |
| `pkg/game/party.go:125` | `GetFollowerActorsInRoom`, used by `do_group all` (`pkg/session/cmd_group.go:47`) |
| `pkg/session/wiz_stats.go:399` | `stat`'s `Master is: … Followers are:` list |
| `pkg/session/cmd_combat_special.go:69` | combat-special follower sweep |
| `pkg/game/party.go:26, 47, 63, 68, 162` | the party helpers themselves |
| `pkg/game/world_movement.go:290, 295` | movement follower scan |
| `pkg/game/animate_dead.go:47` | animated follower lookup |

## What a fix looks like

Take the leader's body (`World.FollowingActor`) and list through
`World.FollowersOf`, as `print_group` now does. Where the caller genuinely only
has a name — a saved field, a not-yet-resolved command argument — keep the name
form but make the duplicate case explicit rather than silently merging.

## Evidence to keep

`TestGroupListResolvesTheFollowedDuplicateNpc` (`pkg/session`) is the pattern: two
same-named NPCs with different stats, one follower on each, and assertions that
the resolved body is the followed one and the lists do not merge. Against the
pre-fix resolution it fails all three assertions at once.
