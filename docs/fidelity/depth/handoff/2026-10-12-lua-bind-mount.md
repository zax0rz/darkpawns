# Lua `mount` binding — bounded depth handoff

This slice closes `lua.bind-mount` at D3 through the real Lua bridge. `lua_mount`
passes the actual `char_data` from its `ch` table to `do_ride`, `do_dismount`, or
`unmount` (`src/scripts.c:871-906`). The Go adapter now resolves the supplied
`CharRef` to that live player or mobile body; it does not create a player
facade for a mobile.

For a mobile `ride` caller, C's `CAN_MOUNT(ch)` gate rejects the actor
(`src/utils.h:362-366`) after normal `do_ride` target gates
(`src/act.other.c:1545-1593`). A descriptorless mobile receives no player
bytes and no mount/follower state is created. `dismount` and `unmount` operate
on the actual actor. `do_dismount` emits its private and room effects before
calling `unmount` (`src/act.other.c:1597-1616`); `unmount` clears only
`AFF_MOUNT` on the rider and the selected mounted mobile follower, retaining
the master/follower relationship (`src/utils.c:378-385`, `617-632`).

The shared C unmount behavior is also used by the legacy `Unmount` and
`Player.Unmount` paths. Those helpers now clear rider and mount `AFF_MOUNT`
without dropping the mount's master/follower relation; only the player's
`MountName` cache is cleared. Recall invokes unmount before Go's mount-aware
player transfer so the horse stays in its old room, matching C's
`spell_recall` relocation boundary (`src/spells.c:157-160`). Jail relocation
uses the same affect-only helper. Follower teardown clears the Go-only
`MountRider` cache after the actual following relation is removed. The
extraction-specific `clearMountedPair` path remains separate.

The live scenario starts with a player riding a scripted mountable crocodile.
Its `oncmd` calls `mount(me, nil, "unmount")` with the mobile itself as the
Lua rider; a second player's attempted ride then reaches C's charmed-mount
master refusal. The paired transcript matches through the actor, room peer,
and second rider: the final private line is `Its master would not like that!`.
This reaches the previously unported mobile-body binding path; it does not
claim NPCs can ride in C or establish generic NPC command parity.

Focused tests cover real player and mobile `CharRef` callbacks, mounted flags,
the retained mount master link, first-mounted-follower ordering, private bytes,
and room delivery. Regression tests also cover direct legacy unmount, recall
and jail relocation, and follower teardown. The compiled mutation controls
prove that removing mobile unmount dispatch, using the retained rider link
instead of `AFF_MOUNT`, or restoring name-only legacy unmount makes intended
assertions fail. Their diffs and logs are under
`~/Archives/darkpawns/oracle-runs/2026-10-12/dp1371-lua-mount-controls/`.

The mobile proof covers a descriptorless NPC body and peer room output; it does
not establish switched-mobile private-output routing. The original census at
`f5442066fa154597db745e2806610f2c0e6d7cf1` is retained as superseded evidence
at `~/Archives/darkpawns/oracle-runs/2026-10-12/dp1371-lua-mount-depth-combined/`.
The corrected source-tip census is retained at
`~/Archives/darkpawns/oracle-runs/2026-10-12/dp1371-lua-mount-unmount-fix-combined/`.
