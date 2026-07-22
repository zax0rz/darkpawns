> ✅ **PREREQUISITE DP-1085 LANDED (PR #335).** Go now routes hometown-K newbies to 8162 like C, so
> both actor and observer co-locate. `scenarios/movement.txt` has been **reworked** to navigate the
> shared 8162 newbie-zone graph (8162 ⇄ 8161 ⇄ 8004, byte-identical `80.wld`) and the §1 RED below is
> the **actual, re-verified oracle output** (Claude ran it 2026-07-14) — not the earlier speculative
> table. It is tighter than expected: DP-1085 aligned the start rooms, so directional auto-look, the
> leave broadcast, and basic sit/stand/sleep/wake now **match**; the surviving divergences are listed
> below.

# BRIEF — Domain 5: Movement (move / position / follow / enter) — O25/O26/O27/O28

**For:** codex (frontier). **Owner of gate:** Claude (runs the oracle red→green, reviews vs C).
**Branch:** `refactor/domain-movement` off current `main`.
**Findings:** DP-1112/O25 (live movement), DP-1121/O26 (position commands), DP-1122/O27 (follow),
DP-1123/O28 (enter command).
**Method rules:** read `src/act.movement.c` + `src/interpreter.c` in the C oracle clone
(`~/.openclaw/workspace/darkpawns-c-oracle`) directly — do not trust Go comments. This fix is
gated by an **oracle red→green run** (see [[darkpawns-oracle-proof-gate]]): a green build is NOT
sign-off.

---

## 1. The bug (proven RED)

The RED is captured by `cmd/dp-oracle-diff/scenarios/movement.txt` (actor + co-located passive
`observer`, hometown K, both at 8162 now that DP-1085 has landed; a `north`/`south` round trip lets
the observer witness leave+arrive, then position/follow/enter run at 8162). Run it:

```
DP_ORACLE_BIN="$HOME/.openclaw/workspace/darkpawns-c-oracle/bin/circle" \
  go run ./cmd/dp-oracle-diff --scenario movement
```

### 1a. Oracle-PROVEN divergences (this scenario, verified 2026-07-14)

| # | probe | audience | C oracle | Go port | finding |
|---|---|---|---|---|---|
| 1 | `south` (returning to observer's room) | observer | `Movactor arrives from the north.` | `Movactor has arrived.` | O25 — arrival message drops direction; C reverses the travel dir |
| 2 | `follow Movobserv` | observer (leader) | `Movactor starts following you.` | `Movactor now follows you.` | O27 — wrong leader wording (`add_follower`, utils.c:496 `$n starts following you.`) |
| 3 | `follow Movobserv` (already following) | actor | `You are already following her.` | `You are already following Movobserv.` | O27 — C uses the **pronoun** `$M` (act.movement.c:905), Go interpolates the name |
| 4 | `enter` | actor | `You are already indoors.` | `Unknown command: enter` | O28 — `enter` not registered |
| 5 | `enter fountain` | actor | `There is no fountain here.` | `Unknown command: enter` | O28 — `enter` not registered |

**What now MATCHES (DP-1085 aligned the start rooms — do not "fix" these):** directional
auto-look on `north`/`south`; the **leave** broadcast (`$n leaves north.` — single-word names need
no conjugation so it already matches); and basic `sit`/`stand`/`sleep`/`wake` transitions incl. the
`$n clambers to $s feet.` room broadcast. The port already routes these through faithful code.

### 1b. Real per §5, but NOT exercisable in this scenario → cover with unit tests / richer fixtures

These are genuine C behaviors (cited in §5) that the current two-mortal scenario can't trigger
because it has no mount, no `AFF_SLEEP`, no charm, and no third follower. Implement them per §5 and
**lock them with unit tests** (the oracle can't prove them until a fixture provides a mount item /
sleep affect / charmed mob / a follower chain):

- **O25:** `AFF_SNEAK` suppresses leave+arrive; closed/secret-door failure text (`The <keyword> seems
  to be closed.` / `Alas, you cannot go that way...` vs Go's generic); exhaustion wording +
  follower variant; follower dragging with `POS_STANDING` gate + `AFF_HIDE` clear.
- **O26:** mount gating (`stand`→dismount; `sit`/`rest`/`sleep`→`You can't rest while mounted.`);
  `AFF_SLEEP` on `wake` (self → `You can't wake up!` + `$n tosses and turns uncomfortably.`; other →
  `You can't wake $M up!`); `wake <target>` success/awake/dead messages via act().
- **O27:** charm-master guard (`But you only feel like following $N!`); `circle_follow` loop
  rejection; mob resolution via `get_char_room_vis`.
- **O28:** named-door entry moves; sole-indoor-exit entry when outdoors; `leave`.

> Consider a follow-up scenario that spawns a charmed mob + a sleep-scroll fixture to bring O26/O27
> mount/sleep/charm under the oracle later; for THIS PR, unit tests are the gate for 1b.
| `leave` (indoors) | actor | exits via sole outdoor exit | (not registered either — same class) | **missing** |

---

## 2. Root cause

Three overlapping implementations, none complete:

1. **`pkg/session/cmd_movement.go` `cmdMove`** — the **live** path. Calls
   `World.MovePlayer` (which handles doors/boat/tunnel/cost/death but NOT charm/mount/follower/
   hide/room-scripts), then **independently** does its own leave/arrive broadcasts, greet scripts,
   and follower dragging. Followers dragged with **no position check** (sleeping followers moved),
   **no hide clear**, and **no act()** conjugation. Replaces all C failure messages with a generic
   `You can't go <direction>.`
2. **`pkg/game/world.go` `MovePlayer`** — the called path. Handles the mechanical checks (exit
   exists, not closed, boat, tunnel, movement cost, death trap) but misses charm/mount/follower/
   hide/room-entry-scripts.
3. **`pkg/game/act_movement.go` `doSimpleMove`/`performMove`** — a more faithful port of C that
   has charm checks + follower position/hide clearing + proper act() arrival messages, but is
   **UNUSED** by the live path. Still lacks full mount support (rider/mount pair movement,
   mount exhaustion, indoor mount rejection) and room-enter scripts.

Position commands (`movement_cmds.go`) are pure session-handlers with inline `genderHisHer` instead
of `act()`, and no mount or AFF_SLEEP checks.

Follow (`cmd_group.go`) resolves only players, has no charm guard, no `circle_follow` loop check.

Enter/leave commands simply don't exist.

**The fix:** consolidate into ONE game-owned movement transaction that covers all C invariants.
Session becomes a thin delegation. Act() (already canonical from F0a, PR #297) handles all
messaging.

## 3. Scope — this PR covers O25 + O26 + O27 + O28

**In scope:**
- **Live directional movement** — one canonical `DoMove(ch, dir string)` in `pkg/game/` covering
  all C `perform_move`/`do_simple_move` invariants (charm, mount, exhaustion, boat, tunnel, closed
  doors with keyword failure text, secret doors, AFF_SNEAK suppression, hide clearing, follower
  dragging with position check, arrival/leave messages via act(), greet scripts, death traps,
  room-enter scripts). Session `cmdMove` deleted; thin delegation replaces it.
- **Position commands** — game-owned `DoStand`/`DoSit`/`DoRest`/`DoSleep`/`DoWake(ch, arg)` with
  mount checks (stand→dismount, sit/rest/sleep refused while mounted, sleep refused while mounted)
  and AFF_SLEEP guard (wake self→toss/turn, wake other AFF_SLEEP→"can't wake"). All messages via
  act(). Session handlers deleted.
- **Follow** — game-owned `DoFollow(ch, arg)` with charm-master guard (AFF_CHARM + master present
  → "But you only feel like following $N!"), `circleFollow` loop detection, mob resolution via
  `getCharRoomVis` (not just players), shadow skill support (subcmd quiet path — see C
  `do_follow` lines 904-951), proper `stopFollower`/`addFollower` semantics. Session
  `cmdFollow`/`cmdFollowMovement` deleted.
- **Enter/Leave** — game-owned `DoEnter(ch, arg)` / `DoLeave(ch)`. Enter: resolve named door
  keyword (exact match per C `str_cmp`, NOT prefix), or sole indoor destination if outdoors.
  Leave: sole outdoor destination if indoors. Both feed `performMove`.
- **`cmdMove` → `World.MovePlayer` call** retired. `MovePlayer` in world.go should become a
  lower-level teleport primitive (for spec_procs, recall, summons); the command-layer movement
  always goes through `DoMove`.

**Explicitly OUT of scope (each its own finding — do NOT do here):**
- **Mount riding system** (dismount/mount commands, ride-as-group semantics, mount exhaustion as a
  separate stat) — beyond the movement **gating** checks. The C code checks `IS_MOUNTED(ch)` and
  calls `get_mount`/`get_rider`/`unmount`; these require a mount/rider data model Go doesn't
  fully have yet. For this PR: gate against mounts (refuse movement actions while mounted where C
  does) but don't build the full riding system. Leave a `TODO` at mount check sites.
- **`circle_follow` for mob followers** — C checks all followers recursively; this PR handles
  the player→player case. Mob follower loops are a separate edge.
- **Shadow skill** (SKILL_SHADOW / AFF_DODGE affect from quiet follow) — C applies this when
  `GET_SKILL(ch, SKILL_SHADOW) > number(0,101)`. This is a skill/spell product decision (see §7
  in refactor plan). Implement the **structural path** (quiet subcmd, `add_follower_quiet`) but
  skip the actual shadow affect application — leave a `TODO`.
- **`flee`** — `cmdFleeMovement` in movement_cmds.go is combat-phase, not pure movement. Leave it
  alone; it will be addressed in the combat domain.
- **Socials bypass** (DP-1125/O32) — dynamic socials call `game.DoAction` directly bypassing
  command-table position checks. Separate finding, separate domain.

## 4. The mount problem (critical design note)

C has a full mount system: `get_mount(ch)` / `get_rider(ch)` / `IS_MOUNTED(ch)` /
`riding_mount(rider, mount)`, plus `do_dismount`. `do_stand` calls `do_dismount` when mounted.
`do_simple_move` moves BOTH mount and rider, checks mount exhaustion, rejects indoor movement
while mounted.

Go has **no mount/rider data model**. There is no `IsMounted()`, no `GetMount()`, no `GetRider()`.
The `Equipment` system has a `SlotMount` but no mount-state tracking on the player.

**For this PR:** the position commands need mount-aware gating. The simplest approach:
1. Add `IsMounted(ch) bool` to Player — checks if anything is in `SlotMount` of the player's
   equipment AND that the player is riding (you can wear a mount item without being mounted).
2. For movement: check `IsMounted` → refuse "You can't ride in there!" (indoors) and gate
   exhaustion off mount's move points (or skip if no mount model — just refuse movement while
   mounted pending the mount system). This matches C's indoor rejection.
3. For position: `stand` → attempt dismount (remove from SlotMount, set position standing).
   `sit`/`rest`/`sleep` → refuse while mounted.
4. Leave full riding (mount commands, ride-as-group, mount exhaustion tracking) as a TODO.

If this is too complex for the scope, **gate against mounts** (refuse movement actions while
   wearing anything in SlotMount) as a safe interim, and file a tracking issue. The key is:
   don't silently succeed while mounted — C refuses.

## 5. Faithful C reference (act.movement.c / interpreter.c)

### 5a. `do_simple_move` (act.movement.c:95-350)

- **Special routine check** (`need_specials_check && special(ch, dir+1, "")`) — only when
  following, avoids double spec-proc. If special returns true → return 0 (movement blocked).
- **Charm check** (line 107): `IS_AFFECTED(ch, AFF_CHARM) && ch->master &&
  ch->in_room == ch->master->in_room` → "The thought of leaving your master makes you weep."
  Return 0. **Only when in the same room as master** — charmed chars CAN move if master is
  elsewhere (that's how they chase).
- **Boat check** (line 115): `SECT_WATER_NOSWIM` for either source or destination sector →
  `has_boat(ch)` or "You need a boat to go there." `has_boat` checks: LVL_IMMORT, AFF_WATERWALK,
  AFF_FLY, ITEM_BOAT in inventory (non-wearable), ITEM_BOAT in equipment slots.
- **Movement cost** (line 130): average of `movement_loss[src] + movement_loss[dest]`, right-shifted
  by 1.
- **Mount checks** (line 134-172): if `IS_MOUNTED(ch)`:
  - Validate mount/rider pair exists (error → unmount)
  - Mount position < STANDING → refuse ("Your mount is in no position...")
  - Mount exhaustion → refuse (both rider and mount get the message)
  - Move cost deducted from **mount**, not rider
  - Non-mounted: deduct from player, mortal only
  - Exhausted + follower: "You are too exhausted to follow." vs "You are too exhausted."
- **Tunnel check** (line 182): `ROOM_TUNNEL` + `num_pc_in_room >= 1` → "There isn't enough room there!"
- **Indoor mount check** (line 188): `IS_MOUNTED && ROOM_INDOORS` → "You can't ride in there! Dismount first!"
- **Leave message** (line 196-204): if not AFF_SNEAK → `$n leaves <dir>.`
- **Arrival message** (line 219-261): complex switch on direction:
  - Cardinal dirs (0-3): reverse direction name. AFF_FLY → "$n flies in from the <dir>."
    UNDERWATER → "$n swims in from the <dir>." riding → "$n rides in from the <dir> on $N."
    default → "$n arrives from the <dir>."
  - Up (4): fly/swim/ride/climb variants of "from below"
  - Down (5): fly/swim/ride/climb variants of "from above"
- **Auto-look** (line 263): `if (ch->desc != NULL) look_at_room(ch, 0); else entry_prog(ch, ch->in_room);`
- **Greet scripts** (line 271-281): if not AFF_SNEAK → `mp_greet` + iterate mobs with
  `MOB_SCRIPT_FLAGGED(tch, MS_GREET)` → `run_script "greet"`.
- **Death trap** (line 285): `ROOM_DEATH && level < IMMORT` → log, death_cry, extract_char.
  Mount also dies.
- **Room enter script** (line 299): `ROOM_SCRIPT_FLAGGED(ch->in_room, RS_ENTER)` → `run_script "enter"`.

### 5b. `perform_move` (act.movement.c:353-400)

- Delegates to `do_simple_move` if no followers
- After successful move: iterates followers. For each follower in the **old** room at
  `POS_STANDING` or above, and who is NOT the rider of the mover:
  - `act("You follow $N.\r\n", FALSE, follower, 0, mover, TO_CHAR)`
  - `REMOVE_BIT_AR(AFF_FLAGS(follower), AFF_HIDE)` — **clear hide before dragging**
  - Recursive `perform_move(follower, dir, 1)` — followers follow

### 5c. `do_move` (act.movement.c:402-410)

Simple remapping: `perform_move(ch, cmd - 1, 0)`. C cmd numbers are 1-indexed (North=1),
directions are 0-indexed.

### 5d. Position commands (act.movement.c:696-878)

- **`do_stand`** (line 696): switch on position.
  - STANDING + mounted → `do_dismount`. STANDING + not mounted → "You are already standing."
  - SITTING → "You stand up." + room: `$n clambers to $s feet.`
  - RESTING → "You stop resting, and stand up." + room: `$n stops resting, and clambers on $s feet.`
  - SLEEPING → "You have to wake up first!"
  - FIGHTING → "Do you not consider fighting as standing?"
  - default (flying etc.) → stop floating + feet on ground
- **`do_sit`** (line 732): STANDING + mounted → "You can't rest while mounted." (note: C says
  "rest" not "sit" — this is the C text). STANDING + not mounted → sit down. SITTING → already.
  RESTING → stop resting, sit up. SLEEPING → wake up first. FIGHTING → MAD?
- **`do_rest`** (line 764): Same mount check. Same positional cascade. Default → "stop floating around, and stop to rest your tired bones."
- **`do_sleep`** (line 797): STANDING + mounted → "You can't rest while mounted." (same message
  as sit). Falls through to SITTING/RESTING case. SLEEPING → already. FIGHTING → MAD?
- **`do_wake`** (line 827): Has target form.
  - Target provided: if self-sleeping → "wake yourself first." If not found → NOPERSON.
    If target == self → fall through to self-wake. If target awake → "$E is already awake."
    If target dead (pos < SLEEPING) → "$E's in pretty bad shape!"
    If target has AFF_SLEEP → "You can't wake $M up!" (NO removal of AFF_SLEEP — it's magical)
    Otherwise: wake target to SITTING. Messages: actor "You wake $M up.",
    room "$n wakes up $N.", victim "You are awakened by $n."
  - Self-wake: if AFF_SLEEP → "You can't wake up!" + room "$n tosses and turns uncomfortably."
    If already awake → "You are already awake..." Otherwise → "You awaken, and sit up." +
    room "$n awakens."

### 5e. `do_follow` (act.movement.c:883-951)

- One arg. Not found → NOPERSON. Empty → "Whom do you wish to follow?"
- Already following target → "You are already following $M."
- **Charm guard** (line 904): `IS_AFFECTED(ch, AFF_CHARM) && ch->master` → "But you only
  feel like following $N!" (refers to current master, not the target). Return — charm blocks
  ALL follow changes.
- Self-follow: if no master → "You are already following yourself." If master → `stop_follower(ch)`.
- Other target: `circle_follow(ch, leader)` → if loop → "Sorry, but following in loops is not
  allowed." If current master → `stop_follower(ch)`.
  - `REMOVE_BIT_AR(AFF_FLAGS(ch), AFF_GROUP)` — leaving old group.
  - **Shadow skill** (quiet subcmd): if `GET_SKILL(ch, SKILL_SHADOW) > number(0,101)` or IMMORT:
    remove existing SKILL_SHADOW affect + AFF_DODGE, apply new SKILL_SHADOW affect with
    AFF_DODGE. `add_follower_quiet`. Otherwise: `add_follower`.
  - `add_follower` emits "You now follow $N." (to follower, act.movement.c:944) and
    **"$n starts following you."** (to leader, **utils.c:496** — NOT "now follows you"; Go currently
    has the wrong leader wording). Already-following uses the pronoun **"$M"** (act.movement.c:905),
    not the target's name.

### 5f. `do_enter` (act.movement.c:642-686)

- One arg. If arg provided: iterate all exits, match by **exact** `str_cmp` on exit keyword.
  Found → `perform_move(ch, door, 1)`. Not found → "There is no <name> here."
- No arg + indoors → "You are already indoors."
- No arg + outdoors: iterate exits, find first that is not closed and leads to a ROOM_INDOORS room.
  Found → `perform_move(ch, door, 1)`. None → "You can't seem to find anything to enter."

### 5g. `do_leave` (act.movement.c:688-695)

- If outdoors → "You are outside.. where do you want to go?"
- If indoors: iterate exits, find first not closed and NOT leading to ROOM_INDOORS.
  Found → `perform_move(ch, door, 1)`. None → "I see no obvious exits to the outside."

### 5h. Registry entries (interpreter.c)

Direction commands: North=1 through Down=6, all at level 1, position POS_STANDING.
`enter` at level 1, POS_STANDING, line 432.
`leave` at level 1, POS_STANDING, line 433.
`stand` at level 1, POS_RESTING (can stand from resting+).
`sit` at level 1, POS_RESTING.
`rest` at level 1, POS_RESTING.
`sleep` at level 1, POS_RESTING.
`wake` at level 1, POS_SLEEPING.
`follow` at level 1, POS_STANDING.
`group` at level 1, POS_RESTING.

---

## 6. Implementation plan

### 6a. Game-layer canonical ops (pkg/game/act_movement.go — extend existing)

**`DoMove(w, ch, dirName string) (MoveResult, error)`**
- Resolve `dirName` → direction index via `searchBlock(dirName, dirs, false)`
- Feed into updated `performMove` (which calls `doSimpleMove`)
- Return `MoveResult{NewRoom, Messages, FollowersMoved, Failed, FailureReason}`
- `performMove` additions:
  - Charm check at top (AFF_CHARM + master in same room → block)
  - Secret door keyword handling in closed-exit path
  - Proper failure text per C: "Alas, you cannot go that way..." (no exit), "The <keyword> seems to
    be closed." (closed with keyword), "It seems to be closed." (closed without keyword), "I really
    don't see how you can do anything there." (secret door, no ROOM_SECRET_MARK)
  - Exhaustion: mortal-only move point deduction. "You are too exhausted." vs
    "You are too exhausted to follow." (follower variant)
  - Tunnel: ROOM_TUNNEL + ≥1 PC in dest → refuse
  - Mount gate: if `IsMounted` → refuse "You can't ride in there!" for ROOM_INDOORS
  - Arrival messages: use act() with proper direction reverse naming, fly/swim/mount variants
  - Follower drag: position check (POS_STANDING+), AFF_HIDE clear, rider exclusion
  - Greet scripts: mp_greet + mob greet scripts (already partially in doSimpleMove)
  - Death trap: ROOM_DEATH check after move
  - Room enter scripts: RS_ENTER trigger

**`DoStand`/`DoSit`/`DoRest`/`DoSleep`/`DoWake(w, ch, arg string) ([]ActMessage, error)`**
- All positional transitions with mount gating
- All messages via act() — $n/$s pronouns, proper ToChar/ToRoom
- `DoWake` with target resolution (get_char_room_vis), AFF_SLEEP guard, self-wake toss/turn

**`DoFollow(w, ch, arg string, quiet bool) ([]ActMessage, error)`**
- Charm guard: AFF_CHARM + master → "But you only feel like following $N!"
- circleFollow: detect A→B→A loops before allowing
- Mob resolution: `getCharRoomVis` (all visible room chars, not just players)
- stopFollower: clean up old master relationship
- REMOVE_BIT AFF_GROUP when changing leaders
- Shadow skill: structural quiet path (add_follower_quiet), leave actual affect application as TODO

**`DoEnter(w, ch, arg string) ([]ActMessage, error)` / `DoLeave(w, ch) ([]ActMessage, error)`**
- Enter: exact keyword match on exit keywords, or sole indoor destination
- Leave: sole outdoor destination from indoors

### 6b. MoveResult type

```go
type MoveResult struct {
    Success      bool
    NewRoomVNum  int
    Messages     []ActMessage  // act()-renderable messages
    Followers    []string     // names of followers that were dragged
}
```

### 6c. Session adoption

Replace all handler bodies:
- `cmdMove` → thin delegation to `s.manager.world.DoMove(s.player, direction)`, render messages
  via act(), send auto-look, mark dirty. **Delete the independent follower-dragging loop.**
- Position handlers (`cmdStand`/`cmdSit`/`cmdRest`/`cmdSleep`/`cmdWake`) → thin delegations.
- `cmdFollow` + `cmdFollowMovement` → single delegation to `DoFollow`.
- Register `enter` and `leave` commands in `commands.go` (new registrations, level 1, POS_STANDING).
- **Delete `broadcastToRoom`** and `broadcastToRoomExcept` from movement_cmds.go if no other
  callers remain (check across all session files).
- **Delete `genderHisHer`** if no other callers remain.

### 6d. Circle-follow implementation

```go
// circleFollow returns true if ch following leader would create a loop.
// Walk the leader→master chain; if we reach ch, it's a loop.
func circleFollow(ch, leader *Player) bool {
    for l := leader; l != nil; {
        if l.GetName() == ch.GetName() {
            return true
        }
        next := l.GetFollowing()
        if next == "" {
            break
        }
        l, _ = w.GetPlayer(next)
    }
    return false
}
```

---

## 7. Acceptance gate (all required)

1. **Oracle red→green:** `--scenario movement` goes from the §1 divergences to **clean**
   (only expected/masked noise). Claude will run this — but run it yourself first.
2. **Observer broadcasts** are part of the diff (peer `observer`) — they must match C, not just
   the actor lines.
3. **AFF_SNEAK suppression** — hidden character movement produces no room messages (both leave
   and arrive).
4. **Follower position gate** — sleeping/resting followers are NOT dragged.
5. **AFF_HIDE clear** — followers lose hide when following (visible to room after move).
6. **Charm guard** — charmed character cannot follow a new leader while charmed.
7. **Circle-follow** — A follows B, B tries to follow A → "loops not allowed."
8. **Position mount gating** — sit/rest/sleep refused while mounted.
9. **AFF_SLEEP wake** — magical sleep prevents self-wake (toss/turn message) and others-wake
   ("can't wake").
10. **Enter/Leave** — named door entry works, sole indoor/outdoor exit works.
11. **Unit tests** for each position transition (stand from sitting, stand while mounted→dismount,
    wake self with AFF_SLEEP→toss/turn, wake target→proper act messages).
12. **Instance-safe:** never write `obj.Prototype.*`. **No WS schema break:** the
    `DoInventory`/`DoEquipment`/`RoomView` golden (`protocol_schema_test.go`) stays green.
13. `make check-fmt vet` + `go test ./...` green.

## 8. Gotchas

- **Start rooms are aligned now (DP-1085 / PR #335).** Both hometown-K newbies begin at 8162; the
  scenario navigates the shared 8162⇄8161 graph (`80.wld`, byte-identical). Do NOT reintroduce
  `recall` or a room-identity fixture, and don't assume 8004 as the start (that was the pre-DP-1085
  Go bug). If you add rooms to the nav chain, confirm the target `.wld` exists in **both** trees
  (the Go tree lacks `150.wld`/`165.wld`).
- **`enter` is position-gated (POS_STANDING).** In the scenario the actor must `stand` before
  `enter`, else C returns its position-fail string (`Maybe you should get on your feet first?`)
  instead of the enter logic. Register `enter`/`leave` at level 1, POS_STANDING (§5h).
- **The oracle can't see mounts/sleep/charm here.** Per §1b, the mount-gating, `AFF_SLEEP`,
  charm-guard, and circle-follow behaviors have no fixture in this scenario — implement per §5 and
  gate them with **unit tests**, not the oracle run.
- **`do_wake` has a `self = 0` flag** — the target-resolution block sets `self = 1` if target ==
  ch, then falls through to self-wake if `!self`. This means `wake <ownname>` works the same as
  bare `wake`. Don't accidentally send the target-wake message AND the self-wake message.
- **C's `do_sit` says "You can't rest while mounted."** not "You can't sit while mounted." — this
  is the actual C text. Match it exactly.
- **`do_follow` resolves `get_char_room_vis`** — this finds **both players and mobs**. The current Go
  code only finds players. The game layer already has `GetCharsInRoom` or similar; use it.
- **`do_enter` uses `str_cmp` (exact match)** on exit keywords, NOT `isname` (partial). This is
  intentional — entering by keyword is exact, not prefix. `find_door` uses `isname`; `do_enter`
  does not.
- **ANSI blind spot:** the oracle normalizer strips color, so any color parity must be unit-tested
  separately (see [[darkpawns-fidelity-testing]]).
- **Mount system is incomplete.** Do NOT build full riding mechanics. Gate against mounts (refuse
  actions while `IsMounted()`) and file a tracking issue. The mount data model (rider/mount pair,
  mount commands, mount exhaustion) is a separate domain.
- **`shadow` skill / AFF_DODGE** — C applies this during quiet follow (SKILL_SHADOW check vs
  `number(0,101)`). This is a skill-system concern. Implement the structural path (quiet subcmd,
  add_follower_quiet) but skip the actual affect manipulation. Leave a `TODO`.
- **Don't break `flee`.** `cmdFleeMovement` in movement_cmds.go calls `MovePlayer` directly.
  After this PR, `MovePlayer` should still work as a lower-level teleport (it's used by spec_procs,
  recall, etc.). The **command-layer** movement goes through `DoMove`, but `MovePlayer` stays as the
  primitive. Just don't delete it.
