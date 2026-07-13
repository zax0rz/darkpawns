# BRIEF 2026-07-11 — GLM: DP-1008 Animate Dead fidelity (charm / follower / pfail)

**Executor:** GLM. **Branch:** `glm/dp1008-animate-dead` (fresh off current `main`,
own clone/worktree — never share a HEAD). **One PR.** Claude verified the C spec
and the Go interface design below against source; implement to match exactly.

## The C spec (authoritative — src/magic.c `mag_summons`, SPELL_ANIMATE_DEAD)

Claude read this directly. The order of operations matters:

```
case SPELL_ANIMATE_DEAD:
    if obj is NULL / not ITEM_CONTAINER / not a corpse (val 3): fail msg 7; return
    handle_corpse=1; mob_num=MOB_ZOMBIE(10); pfail=8

if IS_AFFECTED(ch, AFF_CHARM):                 -> "You are too giddy to have any followers!"; return
if num_followers(ch) >= GET_CHA(ch)/2:         -> "You can't have any more followers!"; return
if number(0, 101) < pfail(8):                  -> summon-fail msg; return
mob = create_mobile(ch, MOB_ZOMBIE, GET_LEVEL(ch)/2, FALSE)
SET_BIT_AR(AFF_FLAGS(mob), AFF_CHARM)          -> the zombie is CHARMED (controllable pet)
add_follower_quiet(mob, ch)
stc("The corpse starts to twitch, then stands with a life of it's own!", ch)
# handle_corpse: move corpse's contents into the mob, then extract_obj(corpse)
```

## What Go has today (pkg/spells/affect_spells.go, `case SpellAnimateDead:` ~line 552)

Currently: finds a room corpse by keyword, spawns MOB_ZOMBIE(10) at level/2,
removes the corpse, calls `AddFollowerQuiet(mob, ch)`, prints the message.

**Missing (the DP-1008 fix — add all four):**
1. caster "giddy" AFF_CHARM check
2. follower cap `num_followers(ch) >= GET_CHA(ch)/2`
3. pfail roll `number(0,101) < 8`
4. **`SET_BIT AFF_CHARM` on the spawned zombie** (Go adds a follower but never charms it)

## CRITICAL design constraint — import cycle

`pkg/spells` **cannot import `pkg/game`** (game imports spells → cycle). Everything
game-specific must go behind a **new World interface method in pkg/game**, called
from pkg/spells via a local `interface{...}` type-assert on the `world` param —
exactly like the existing `SpawnMobWithLevelI` / `AddFollowerQuiet` pattern in that
same block.

### CRITICAL bit-numbering (do NOT get this wrong)

- The charm bit index in pkg/game is the **internal** `affCharm = 21`
  (pkg/game/affects_constants.go:29). `MobInstance.SetAffected(bit)` and
  `Player.IsAffected(bit)` both take the **bit INDEX** (`1<<bit`).
- `engine.AFFCharm` is the **mask** `1<<10` = 1024 — a DIFFERENT numbering used for
  the `ActiveAffects` engine-flag list. **Never pass `int(engine.AFFCharm)` to
  `MobInstance.IsAffected/SetAffected` or `Player.IsAffected`** — 1024 >= 64 so it's
  a no-op / broken shift. (This exact bug already exists at affect_spells.go:466 and
  :530 — see the separate ticket; do NOT copy that pattern.)
- Because the correct index (`affCharm=21`) lives unexported in pkg/game, the charm
  set + giddy check MUST be done in pkg/game, not pkg/spells.

## Implementation plan

### 1. pkg/game — add two World methods (they know affCharm=21, CHA, followers)

```go
// CanRaiseUndeadI reports whether ch may animate a corpse right now, mirroring
// mag_summons' pre-checks. Returns (false, playerMessage) when blocked.
// - ch charmed (affCharm) -> "You are too giddy to have any followers!\r\n"
// - follower count >= GET_CHA(ch)/2 -> "You can't have any more followers!\r\n"
func (w *World) CanRaiseUndeadI(ch interface{}) (bool, string)

// CharmAndFollowI sets AFF_CHARM (affCharm index) on the raised mob and makes it
// a quiet follower of leader — the SET_BIT_AR + add_follower_quiet pair.
func (w *World) CharmAndFollowI(mob, leader interface{})
```

- Giddy check: type-assert ch to the concrete player/mob and call `IsAffected(affCharm)`.
- Follower count: use `w.GetFollowers(name)` (party.go). **Verify whether charmed
  MOBS are counted** — C `num_followers` counts the whole follower list incl. mobs.
  If Go only tracks player-followers, count mob followers too (check how animated/
  charmed mobs register as followers); if that's not tracked, count what you can and
  add a `// DP-1008: player-followers only; mob-follower registry TODO` note so the
  divergence is explicit rather than silent.
- CHA: `ch.GetCha()` (Player.GetCha in player_social.go:166, MobInstance.GetCha).
- Charm the mob: `mob.(interface{ SetAffected(int) }).SetAffected(affCharm)` then the
  existing quiet-follow (`AddFollowerQuietMob` / world AddFollowerQuiet).

### 2. pkg/spells — wire the checks + pfail into `case SpellAnimateDead:`

Order (match C): after the corpse is found and before spawning —
```go
// giddy + follower-cap (delegated to game; import cycle forbids doing it here)
if c, ok := world.(interface{ CanRaiseUndeadI(interface{}) (bool, string) }); ok {
    if ok2, msg := c.CanRaiseUndeadI(ch); !ok2 { sendToCaster(ch, msg); return }
}
// pfail = 8: number(0,101) < 8  ==  rand.IntN(102) < 8
if rand.IntN(102) < 8 {
    sendToCaster(ch, "You failed.\r\n") // mag_summon_fail_msgs[fmsg], fmsg=0 here
    return
}
```
Then spawn as today, and replace the bare `AddFollowerQuiet(mob, ch)` with a call to
`world.CharmAndFollowI(mob, ch)` so the zombie is charmed AND following.

Keep the existing room-corpse-by-keyword lookup (the C obj-target model is a
separate, pre-existing divergence — out of scope for DP-1008).

## Tests (pkg/game for the world methods; pkg/spells for wiring)
- Charmed caster → animate blocked with the giddy message, no mob spawned.
- Follower count at CHA/2 → blocked with the cap message.
- Successful animate → spawned mob HAS the charm bit set AND is following the caster.
- pfail: not directly unit-testable via the global RNG (math/rand/v2 can't be
  seeded); assert the success path over enough iterations that at least one success
  occurs, OR structure CanRaiseUndeadI/charm so the deterministic parts are covered
  and leave pfail as a documented ~8% branch. Do NOT add a flaky assert on the roll.

## Ground rules
Own clone/worktree; `git status` clean before every commit; verify against
`src/magic.c` not this paraphrase; `go build ./... && go test -race ./...` +
gofmt/gofumpt clean before push. One PR `glm/dp1008-animate-dead → main`, body lists
each of the 4 C mechanics + how it's covered. Claude verifies the diff + bit-numbering.

## Do NOT touch
- affect_spells.go:466 / :530 charm-check bug — filed as **DP-1015** (separate).
- The corpse targeting model (room-keyword vs C obj-target) — out of scope.
