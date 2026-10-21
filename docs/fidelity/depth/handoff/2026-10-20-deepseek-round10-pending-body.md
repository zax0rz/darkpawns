# DeepSeek — round 10: the missing attack roll

Date: 2026-10-20 · Branch: `deepseek/round10` (off `origin/main` `50fb6bc63`) ·
Rules: R1, R1a, R3, R5c, R5e, R5g, R5h

Round 9 diagnosed `carve-depth`'s first divergent draw as "C rolls one more
attack than the port". This round finds the swing, fixes the class, and clears
the DRAW row on every seed it covers.

## Part 1 — the skipped swing

### Root cause

The port removed a dead mobile from the room at the moment of death:

```go
// pkg/game/death.go, handleMobDeath (before this round)
deadMob.Flags |= 1 << uint(MobFlagExtract)
w.pendingMobileExtractions[deadMob] = struct{}{}
delete(w.activeMobs, deadMobID)   // <- MED-011, "prevents two combat rounds…"
```

C does not. `extract_char` on a mobile **only sets `MOB_EXTRACT` and counts the
pending extraction** (`src/handler.c:1194-1209`); the body leaves
`character_list` and `world[room].people` in `extract_char_final`, which
`extract_pending_chars()` runs at the top of the next heartbeat
(`src/comm.c:803-805`, `src/handler.c:1221-1248`). Until that pass the body is
still in the room:

- room listings skip it — `MOB_EXTRACT`
  (`src/act.informative.c:257-268`, ported in `pkg/game/look.go`),
- an immortal's `stat room` lists it,
- and **`get_char_room_vis` still finds it** (`src/handler.c:1276-1300` has no
  `MOB_EXTRACT` filter), so the next command that names it still reaches it.

`carve-depth` does exactly that: `kill fish` is the implementor's
`do_kill` → `raw_kill(vict, TYPE_SLASH)` (`src/act.offensive.c:155-161`), which
flags the body; the next command, `hit fish`, still resolves it in C, so `hit()`
runs its full swing — to-hit `number(1,20)` and the weapon's damage dice
(`src/fight.c:1814-1852`) — and `damage()` then refuses the swing. The port had
already deleted the body, answered "They don't seem to be here.", and lost the
draw. That is the missing attack at stream 4098.

`do_kill`'s `raw_kill` also never sets `POS_DEAD` (only `die_with_killer` does,
`src/fight.c:610-622`), so the flagged body is still standing: `damage()`'s
corpse guard (`src/fight.c:1319-1326`) does not fire, and the swing falls through
to the **peaceful** arm (`src/fight.c:1336-1342`), which is why C prints "This
room just has such a peaceful, easy feeling..." — a player-visible byte, not
just a draw.

### The fix

1. **`pkg/game/death.go`** — stop deleting the body at death. It stays in
   `activeMobs` (so room queries and name resolution see it) and is removed by
   the existing pending-extraction pass, exactly where C removes it.
2. **`pkg/combat/engine.go`** — `PerformPendingBodySwing` runs C's `hit()`
   against a flagged body: the swing consumes its to-hit and damage draws and
   `damage()`'s protection block decides. `CombatPair.AllowRetiredDefender`
   admits the body to that single swing; nothing is enrolled, because C's
   `damage()` calls `set_fighting` only after the protection block
   (`src/fight.c:1403-1445`).
3. **`pkg/session/combat_cmds.go`** — `do_hit` uses that entry when the resolved
   victim is already flagged (C's `hit()` has no such distinction; the port needs
   it only because it fences retired bodies out of the engine).

### Evidence

`carve-depth`, `--draw-parity`, seeds 1, 2, 3, 5, 8 — clean, and the two engines
now consume the *same* number of draws:

```
carve-depth seed 1: no normalized divergence  draw-parity MATCH  entries=4112 stream=4112
carve-depth seed 2: no normalized divergence  draw-parity MATCH  entries=4115 stream=4115
carve-depth seed 3: no normalized divergence  draw-parity MATCH  entries=4112 stream=4112
carve-depth seed 5: no normalized divergence  draw-parity MATCH  entries=4113 stream=4113
carve-depth seed 8: no normalized divergence  draw-parity MATCH  entries=4111 stream=4111
```

New focused vehicle `dead-body-hit-depth` (same room, no carving at all: `get`,
`wield`, `kill fish`, `stat room`, `look`, `hit fish`, `hit fish`) on the same
five seeds — clean, `draw-parity MATCH` 4069-4073. Its C blocks pin all three
halves:

```
--- [kill fish]      You chop her to pieces!  Ah!  The blood!
--- [stat room]      Chars present: DeadBody(PC), fish(MOB)      <- still listed
--- [look]           (the fish is absent; the corpse is listed)  <- hidden
--- [hit fish]       This room just has such a peaceful, easy feeling…
```

Revert controls (run 2026-10-20, kept out of the repo):

| control | result |
|---|---|
| restore `delete(w.activeMobs, deadMobID)` in `handleMobDeath` | `carve-depth` red at stream 4098 (`c=Number(1,20)` vs `go=Number(0,3)`); `dead-body-hit-depth` red at stream 4071 (`c-step=probe step 6: hit fish` vs Go step 7) |
| drop the `PerformPendingBodySwing` branch in `cmdHit` | the swing is refused before the gate; the peaceful line disappears |

### The class (R5c)

Every death path keeps the body until the pass, not just `do_kill`: normal
combat deaths, spells (`DoSpellDamage`), ranged
(`ApplyRangedProjectileDamage`), `DoTurn`'s disintegration and the mobile-special
seams all go through `handleMobDeath`. Ten tests that asserted "gone from
activeMobs immediately" were updated to C's two-step reality (flagged, still
listed, then freed by the pass):
`TestRangedDeathHasNoKillerBookkeeping`, `TestDoSpike_MobTarget`,
`TestDoTurnDisintegratesEligibleUndeadAtFifteenLevelAdvantage`,
`TestKillPayout_Backstab_MobRemovedFromActiveMobs`,
`TestKillPayout_Backstab_CorpseCreated`, `TestKillPayout_Bash_RemovesMobFromWorld`,
`TestKillPayout_HandleDeath_MobRemovedFromActiveMobs`,
`TestKillPayout_DoSpellDamage_RemovesMobFromWorld`,
`TestSendSkillResult_KillHit_DoesNotEnrollCorpse`, `TestInstakillFiresAtImplLevel`
— plus a new `TestPendingBodyStaysAddressableUntilExtractionPass`.

### Named gap (implementor `kill` only)

When a flagged body's `damage()` gates **pass** (a non-peaceful room), C damages
the body and can reach a second `raw_kill` on it. That is **defined** behaviour,
not undefined: the second `extract_char` returns early at the already-set flag
(`src/handler.c:1197`), so nothing is freed twice and nothing is used after
free; the extraction pass still runs once at the next heartbeat. The observable
result is a second, empty corpse and a repeated set of death messages.

The port runs the swing's draws and `damage()`'s gates, then stops before
applying damage to a body awaiting extraction, so that second corpse and its
messages are not reproduced. Reachable only through the implementor `kill`
command (`do_kill`'s `raw_kill` is the only path that flags a body while leaving
it standing). Earlier drafts of this note called it R1a; it is not — it is a
missing C behaviour with a known C result, recorded here and left to Zach to
schedule.

## Part 2 — the index clusters

`checkload-depth`, `peek-gates-depth`, `sedit-entry-depth` (and
`sedit-session-depth`, `idlist-depth`) — the whole 19–24 cluster — share **one**
signature, measured this round:

```
checkload-depth  seed 3: MISMATCH stream=65031 c=Number(0,3)=3  go=Number(0,20)=18  c-step=probe step 1: checkload  go-step=setup
peek-gates-depth seed 3: MISMATCH stream=64750 c=Number(0,3)=3  go=Number(0,20)=18  c-step=probe step 1: peek       go-step=setup
sedit-entry-depth seed 3: MISMATCH stream=65031 c=Number(0,3)=3  go=Number(0,20)=18  c-step=warmup                   go-step=setup
idlist-depth     seed 3: MISMATCH stream=65031 c=Number(0,3)=3  go=Number(0,20)=18  c-step=warmup                   go-step=setup
```

Go is one `Number(0,20)` ahead before the first probe command, at the ~64–65k
draw mark (a setup-phase stream). The one `number(0,20)` in the mobile-pulse path
is `loud_mobs` (`src/spec_procs2.c:1532-1550`): C gates on **`GET_NOISE(ch)`**, a
runtime per-mob string, while the port reads `proto.Noise`
(`pkg/game/room_activity.go`, `loudMobs`). A mob whose noise is set or cleared at
runtime therefore draws in one engine and not the other. That is the lead; the
fix is to carry noise on the instance, not the prototype. Not fixed here.

`circle-aware-depth` (rows on seeds 2, 3, 5, 8) is a **different** cause and
survives this round's fix:

```
seed 2: MISMATCH stream=4103 c=Number(0,500)=285 go=Dice(0,4)=0  c-step=probe step 2: ~dpclock pulse 20  go-step=probe step 1: circle seagull
seed 3: MISMATCH stream=4111 c=(log ends)        go=Dice(0,4)=0  c-step=after the last recorded step     go-step=probe step 2: ~dpclock pulse 20
seed 5: MISMATCH stream=4102 c=Number(0,500)=281 go=Dice(0,4)=0  (same shape)
seed 8: MISMATCH stream=4101 c=Number(0,500)=46  go=Dice(0,4)=0  (same shape)
```

C is inside a violence round's attack-count roll (`number(0,500)`, the PC extra
attack at `src/fight.c:1939`) where Go rolls a **zero-dice** damage
(`Dice(0,4)`): Go reaches its swing's damage roll one step before C reaches the
same point. `do_circle` is the vehicle; the first question for the next round is
whether Go's `circle` resolves its victim and damage before the pulse's violence
round does. `checkload-depth`'s *other* row (index 40 in round 9's table)
resolves to the same `Number(0,20)` signature as its cluster-mates, so the
index-40 grouping in the round-9 handoff splits.

## Gates

Run individually, each checked for its own exit status:
`make fmt`, `go build ./...`, `go vet ./...`, `go test ./...`,
`golangci-lint run ./...`, `make deadcode-ratchet`, `make fidelity-depth`
(6,715 cases, 6,624 proven/delegated), `make fidelity-units`,
`make string-census`, `gosec -severity=high -quiet ./...`.

All exit 0 on `becab5f0c` (43 packages of `go test ./...`, 0 failures).
Targeted census — the smallest set that can fail on this change (the death and
hit surfaces, plus the row's own scenario):

```
deepseek-r10-pending-body-b @becab5f0c: scenarios=8 passed=8 expected=0
  unpinnable=0 stale=0 failed=0 infra=0 timed_out=0 unstable=0 verdict=CLEAN
```

Scenarios: `carve-depth`, `dead-body-hit-depth`, `hit-depth`, `combat-death`,
`death-followers-depth`, `{bash,backstab,circle}-skill-message-death-depth`.
Evidence: `~/Archives/darkpawns/oracle-runs/2026-10-20/deepseek-r10-pending-body-b/`.

Re-run after the R5c audit (adds the room-lane scenarios, because `cRoomPeople`
no longer skips flagged bodies):

```
deepseek-r10-pending-body-c @0f875e7cf: scenarios=8 passed=8 expected=0
  unpinnable=0 stale=0 failed=0 infra=0 timed_out=0 unstable=0 verdict=CLEAN
```

Scenarios: `carve-depth`, `dead-body-hit-depth`, `hit-depth`, `combat-death`,
`death-followers-depth`, `spec-npc-elements-guardian-depth`,
`spec-proc-elements-platforms`, `spec-proc-elements-master-column-all`.
Evidence: `~/Archives/darkpawns/oracle-runs/2026-10-20/deepseek-r10-pending-body-c/`.

Pre-existing red, not from this round:
`spec-proc-elements-galeru-alive-dead` reports one extra blank line in Go's
`say` block on both `origin/main` (checked in a scratch worktree) and this
branch, with draw parity matching. Left as a finding.


## Draw-row state after round 10

| scenario | rows | state |
|---|---|---|
| `carve-depth` | 5 | **cleared** — seeds 1, 2, 3, 5, 8 clean with equal draw counts |
| `checkload-depth`, `peek-gates-depth`, `sedit-session-depth`, `sedit-entry-depth`, `idlist-depth` | 12 | open — one cause: Go's extra `Number(0,20)` (runtime `GET_NOISE` vs `proto.Noise`), lead recorded |
| `circle-aware-depth` | 4 | open — Go rolls `Dice(0,4)` where C rolls the violence round's `Number(0,500)`; `do_circle` ordering |
| `house-depth`, `shop-*`, `will-zen-meditation-depth`, `mudlog-crash-load-entry`, `spec-proc-*` | 15 | unchanged from round 9 (setup-phase / undiagnosed) |

## Decisions for Zach

- **The pending-body swing stops before applying damage**, so the implementor
  `kill` → `hit` → second-death path (a second empty corpse plus repeated
  messages) is a named gap, not ported. It is defined C behaviour, not R1a; the
  port needs a damage path that can carry a flagged body through `die` and
  `raw_kill` a second time. Alternative: port it now.
- **`pkg/combat` gained one entry point** (`PerformPendingBodySwing`) plus a
  `CombatPair` flag. Alternative: keep the engine untouched and accept the
  divergence.

## R5c audit — every C `MOB_EXTRACT` check, mapped to Go

`extract_char` leaves the body in `character_list` and `world[room].people`
for up to one pulse, so every C loop that walks either list had to be checked
(`grep -rn MOB_EXTRACT src/*.c`):

| C site | what C does | Go | state |
|---|---|---|---|
| `src/fight.c:1321` | `damage()` refuses silently when `GET_POS(victim) <= POS_DEAD` **and** the flag is set; otherwise it logs `SYSERR: Attempt to damage a corpse.` | `pkg/game/damage_gate.go` `DamageRefused` refuses silently on `pos <= PosDead` | matches for the player-facing bytes; the `SYSERR` mudlog line is not emitted (invisible, noted) |
| `src/fight.c:2030` | `perform_violence` runs a mob's special only on `!MOB_EXTRACT` | the round's mob-special seam skips `HasMobFlag(MobFlagExtract)` (`pkg/session/manager.go`, `SetMobSpecialFunc`) | matches |
| `src/act.informative.c:508` | `list_char_to_char` skips flagged bodies | `pkg/game/look.go` skips `MobFlagExtract` in the room listing | matches (pinned by `dead-body-hit-depth`'s `look`) |
| `src/db.c:2230` | zone reset `'R'` (remove mobile) skips flagged **and fighting** bodies | `pkg/game/spawner.go` `removeMobFromRoom` skips both | matches (pre-existing) |
| `src/interpreter.c:1453` | `special()`'s room-mobile loop skips flagged bodies | `pkg/session/character_dispatch.go` (`runCharacterSpecials`) and `pkg/game/flee.go` (`movementSpecialBlocksMob`) skip `MobFlagExtract` | matches |
| `src/scripts.c:1985` | `table_to_char` writes no field on a flagged body | `pkg/game/world_bridge.go` returns before writing when `!IsAlive() \|\| MobFlagExtract` | matches (pre-existing, cited there) |
| `src/utils.c:954` | `check_dead()` is true for a flagged mob; `mobile_activity`/`mobact` helpers use it | `pkg/game/mobact.go` gates on `!IsAlive() \|\| MobFlagExtract` before acting | matches |
| `src/db.c:2108` | `'M'` reset counts `mob_index[].number`, decremented only in `extract_char_final` | `pkg/game/world_player.go` `countMobInstances` counts `activeMobs` **and** queued bodies, flagged included | matches — a pending body still occupies a slot, as in C |

Loops C does **not** guard, checked for the opposite error:

- **Room-special bodies.** `start_room` (`src/spec_procs.c:2209`) and the
  Elemental Temple loops (`src/spec_procs3.c:948, 1012, 1164, 1195, 1254`) walk
  `world[ch->in_room].people` with no flag test, so C moves a pending body like
  any other occupant. `cRoomPeople` used to skip flagged mobs — an extra guard C
  does not have — and now includes them (`pkg/game/actor_specials_rooms.go`).
- **Name resolution.** `get_char_room_vis` has no flag test
  (`src/handler.c:1276-1300`), and Go's `ResolveCharInRoom` matches (no filter).
  That is what the `hit` in `dead-body-hit-depth` rides on.
- **`stat room`** lists the body in C (`world[room].people`), and Go's
  `GetMobsInRoom` includes it — pinned by the scenario.
- **Aggro, wander and `mobile_activity`** reach a flagged body only through
  `check_dead`, which returns true, so neither engine acts on it.
