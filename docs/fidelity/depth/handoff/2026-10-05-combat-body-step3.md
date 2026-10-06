# Combat body identity — stack 3/4: stop, death and extraction

**Stop tier:** combat identity and lifecycle cleanup. Base: reviewed stack 2,
`e125deede` (`codex/combat-body-step2`, #1800). Part of approved #1796; merge
all four stacks together only after stack 4's combined census and Zach's live
playtest. Shoot Train 2 remains parked. R1, R3, R4, R5b/e/g/h govern this change.

## Behavior and source

Ordinary `StopCombat` now stops only its actual body. An incoming one-way fight
survives. If its old opponent is dead or has fled, it retains the first eligible
attacker in combat-list order, including a same-description NPC, without
reordering the body. C: `src/fight.c:230-254`; posture is standing followed by
`update_pos`, including a healthy sleeping body. Pair retargeting preserves only
immutable start metadata, never copying flags concurrently used by an old hit.

Full retirement is separate: remove that body, clear fighters pointing at it
through the ordinary retarget boundary, and remove only its pairs/parry/order.
A transient mutex-protected retirement bit prevents an old round snapshot from
re-enrolling an extracted body. New/restored players reset it in AddPlayer;
there is no serialized state or second registry. C:
`src/handler.c:1144-1153`, `src/fight.c:534-583,1898-2032`.

Leaving a room stops old-room fighters pointing at the departing body **only
when it was itself fighting**, then stops the departing body, before changing
rooms. The unusual one-way exception is C's explicit FIGHTING guard:
`src/handler.c:504-529`. PlayerTransfer/MobTransfer, the spell teleport bridge,
SetRoom and normal movement retain actual objects. Name-taking CharTransfer
still selects once; standalone spell adapters retain their old fallback.
Same-room transfers still relink through char_from_room. Idle explicitly stops
the opponent first, then the idle body (`src/limits.c:422-433`). Retreat also
uses this boundary through do_simple_move (`src/act.offensive.c:1001-1093`);
its old unit incorrectly assumed the lack of an explicit stop preserved combat.

Death, ExtractMob, pending player/mobile extraction and exact player removal
invoke engine cleanup after releasing World.mu. Pending possessions and object
ownership retain their established timing: mobile handles remain available in
the existing pending set; player NOWHERE follows cleanup while its old room is
still available for C's stop/retarget test. No extraction-timing certification
of C3, event ownership redesign or mobile command/rider expansion is claimed.

The stack 2 question is resolved by retaining the actual NPC leader in the
existing relation, with duplicate-leader proof. C add_follower accepts a char
pointer without excluding NPC leaders (`src/utils.c:463-498`), and charm can
attach this edge (`src/spells.c:406-458`). Retirement clears incoming retained
NPC-leader edges by reference, preserving outgoing-follower and PC-leader
teardown behavior; the broader follower lifetime/output work remains on C3.
The existing DieFollower readers now compare held leaders, not descriptions.

## Proofs and reviewed boundaries

- Engine tests: ordinary one-way stop; dead and fled target retargeting to the
  newest eligible duplicate without changing order; retirement isolation.
- Real World duplicates fighting separate PCs: death/raw kill, ExtractMob,
  flagged pending mob, pending PC, exact player removal, player/mobile transfer,
  same-room transfer and SetRoom. The survivor retains its opponent, next
  ordered turn and own weapon; the dead armed body puts its actual weapon in
  its corpse. A real teleport spell selects the second duplicate and leaves
  the first in place.
- A callback extracts a body during a round; subsequent snapshot entries skip
  it and enrollment rejects it. Concurrent round/extraction runs under -race.
- Session tests: disconnect retains the live world fight; peace stops all four
  bodies; removal stops only the departing body's fight. Switched disconnect
  keeps both original/borrowed PCs' exact fighting references.
- Retiring one of two equal-description NPC leaders detaches only its follower.
- Existing sleeping-stop, ordinary-stop, different-room and retreat assertions
  were corrected to the cited C contract, not weakened to accept either result.

Reproduce compiled green/revert/restore controls from the repository root in a
**disposable worktree**, never concurrently with tests or another editor:

```sh
python3 docs/fidelity/depth/handoff/2026-10-05-combat-body-step3-controls.py --output /tmp/combat-step3-triples
```

Controls separately restore broad ordinary stop, disable retarget, introduce
name-wide retirement, omit extraction/pending/death/session cleanup, omit room
stop, reselect a duplicate in transfer, permit stale enrollment, conflate
retained leaders, and omit engine retirement during a concurrent round.
Every revert must compile and fail an assertion; the script restores in finally.

## Actual lock acquisitions

| Path | Acquisitions and release boundary |
|---|---|
| Stop/retire/start | Engine.mu -> one Player/Mob mutex at a time. No nested body locks and no World/manager/output callback. Enrollment rechecks retirement after taking Engine.mu. |
| Round/hit | Engine RLock snapshots order/pair, then releases; body getters each take their own mutex. Enrollment/attack-type writes use Engine.mu. Scripts, damage, messages, movement and death run outside it. Retirement is checked before hit, locked enrollment and after damage/message callbacks. |
| World retirement/death | World.mu -> existing body/equipment/object operations, release World.mu, snapshot remaining world bodies with short RLocks, then engine cleanup. Follower messages run with neither world nor engine lock held. |
| Pending extraction | Existing World.mu owns drops/removal; unlock before retirement; set player NOWHERE afterwards through its body setter. |
| Room transfer / SetRoom | Snapshot room/occupants under short world/body locks; release before engine stops; reacquire World.mu -> one body mutex for location/light/order mutation. SetRoom does its stop before taking its body/world mutation lock. |
| MovePlayer / flee / retreat | Existing world validation/cost lock, release for stopRoomFights, reacquire world -> player for move/light. No playerLifecycleMu added. |
| Session cleanup / takeover | Existing lifecycle ownership where required; manager snapshot and release, then exact World removal and unlocked engine retirement. Takeover captures the departing original when switched, never the incoming candidate. Lost transport retains world bodies. |

No engine-to-world/session/lifecycle edge is introduced. Body locks are released
before entering the engine. Ordinary movement acquires no lifecycle lock.

## Leftover-name sweep

```sh
rg -n 'GetFighting\(\).*==|GetFighting\(\).*!|SetFighting\(' pkg --glob '!**/*test.go'
rg -n 'StopCombat|RetireCombatant|RemovePlayerBody|stopRoomFights|retireCombatBody|TransferCombatant' pkg --glob '!**/*test.go'
rg -n 'GetPlayer|GetMobByName|CharTransfer' pkg/game/combat_cleanup.go pkg/game/world_movement.go pkg/combat/engine.go pkg/game/follow.go
```

First sweep has no production hits. Remaining name selectors are command-entry
CharTransfer/player compatibility, unique-player legacy follower fallback and
existing mount/navigation helpers. The live spell/body transfer and all cleanup
already holding a combat body do not reselect by name. Display projections,
command text, memory/hunting/event identity and C1/C3 frontiers remain distinct.

## Evidence and validation

Retained under `~/Archives/darkpawns/oracle-runs/2026-10-05/`
`dp-1371-combat-body-identity-proofs/`: `step3-stop-before.txt`,
`step3-triples/`, `step3-gate-*.txt`, `step3-name-sweep-{1,2,3}.txt`, final patch
and commit record. Run normal build/vet/full tests/game tests/lint and
fidelity-depth/fidelity-units/string-census plus focused body -race before
committing. No census at this intermediate stack: Zach explicitly requires
one combined run only at stack 4's tip. C/oracle/harness/governing documents and
the parked shoot branch are unchanged.

Final result: all ten listed gates passed on the final source, including focused
-race. All 13 compiled control triples passed green/red/restored-green. The
pending-extraction control additionally proves a removed body cannot enter a
stale direct hit even when moving it to NOWHERE already clears its fight.
