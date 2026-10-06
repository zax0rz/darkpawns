# Combat body identity: step 1 checkpoint

**Stop tier; stack 1/4; do not merge independently.** Implements membership and
runtime opponent references from #1796. All four checkpoints must be reviewed
and merge together after the final tip's combined census and Zach's live
playtest. Shoot Train 2 remains parked. This checkpoint does not certify
callbacks, default command targets or complete death/extraction behavior.

C contract: `src/fight.c:207-254,1405-1458,1898-2032`. Keys, fighter order,
round deduplication and parried state now carry concrete bodies. The existing
engine remains the only combat registry; player and mobile numeric IDs may
collide without becoming the same body. Nil, typed-nil, value-backed and
non-comparable adapters are refused before engine map access.

Player/MobInstance store one canonical opponent reference under their existing
body mutex. String setters and exported name-backed fighting fields are removed.
GetFighting/GetFightingTarget render the opponent's name only after releasing
the reader's lock. Setter migrations carry already-selected bodies from melee,
spell/skill damage, AI and specs; local engine interfaces and query/stop/parry
callers now pass those bodies. Reference-presence and relational comparisons
are migrated where their operands were already available. Existing tests keep
their assertions, but fixtures now attach actual intended opponents; name-only
binary-gate fixtures use concrete test bodies. Production has no synthetic
opponent or name-to-body setter adapter.

`TestCombatBodyIdentityDuplicateEnrollment` spawns two actual same-description
mobs and two real registered players through World. It covers both same-room
and different-room play, separate immediate swings, all four subsequent turns
in head-prepend order, exact targets on every swing, reciprocal pair keys with
equal player/mob numeric IDs, and a replacement body not inheriting membership.
The baseline test fails on main's second enrollment, on an assertion.
`TestCombatBodyReferencesConcurrentAndReciprocal` exercises reciprocal display
reads and concurrent reference writes/clears with no nested body locks.
`TestCombatBodyParriedIsolation` proves defense state belongs to one body.
`TestCombatBodyRejectsInvalidReferences` proves invalid references cannot reach
map access or mutate membership. None of these uses a name-resolving callback
to establish the duplicate identity assertions.

## Reproducible revert triples

From a disposable checkout of this checkpoint, run:

```sh
python3 docs/fidelity/depth/handoff/2026-10-05-combat-body-step1-controls.py \
  --output "$HOME/Archives/darkpawns/oracle-runs/step1-independent-triples"
```

The script preserves/restores engine.go in finally, and independently reverts
attacker uniqueness, fighter membership, round deduplication, round target,
query target, cross-kind numeric identity and parried lookup to names/bare IDs.
It records exact commands, HEAD and each green/red/restore output; a red counts
only if compilation succeeds and an assertion fails. It is a proof replay
helper, not part of the census or server. Never run it on a shared worktree or
while a census is running.

Retained run: `~/Archives/darkpawns/oracle-runs/2026-10-05/`
`dp-1371-combat-body-identity-proofs/step1-triples/`, plus the original baseline
in `step1-before.txt`. Race and normal gate outputs live in the same proof root.

## Leftover-name sweep, with owners

Repeat from this checkpoint, including all production source (R5b/R5e):

```sh
rg -n 'SetFighting\(|FightingTarget|\.Fighting\b' pkg --glob '*.go' --glob '!**/*_test.go'
rg -n 'GetFighting\(\)|GetFightingTarget\(\)' pkg --glob '*.go' --glob '!**/*_test.go'
rg -n 'GetMobByName\(|GetPlayer\(|findRoomCombatantByName|NewNamedCombatant|func\([^)]*(name|Name)' \
  pkg/combat pkg/game/combat_wire.go pkg/game/world_scriptable.go pkg/session/manager.go \
  --glob '*.go' --glob '!**/*_test.go'
rg -n 'StopCombat\(|IsFighting\(|GetCombatTarget\(|MarkParried\(' \
  pkg --glob '*.go' --glob '!**/*_test.go'
```

These are inventory commands, not an assertion that every name is an identity
bug. Retain the raw matches; classify each family against the design audit:

- **Step 2:** all name-bearing GameCallbacks and engine hooks; embedded/fallback
  messages and exclusions, following/master redirects, weapons/affects/flags,
  group recipient enumeration and synthetic group payout wrappers; scripts;
  session health/target updates; default command targets in skills/target/specs
  and look/PERS rendering. Engine's default protection fallback and its mobile
  recovery interface still read display names; those migrate with callbacks.
- **Step 3:** world_movement and death's name-matching cleanup loops, complete
  C StopCombat retarget/remove semantics, pending extraction and disconnect
  ownership. This checkpoint changes StopCombat's parameter/key equality,
  while preserving its previous removal contract until that step.
- **Display/input or separately owned:** GetFighting method declarations and
  display projections, command/lookup matcher text, persistence/login admin
  lookups and noncombat AI target/hunting/memory relationships. Step 2 must
  still eliminate their use to rediscover an already-selected combat body.
  C1/C3 unsupported dispatch/script actor/mobile-rider behavior remains tracked.

## Lock acquisitions and other readers

Engine membership mutations use its existing mu followed by individual body
getter/setter locks; query target reads only the body reference. Round snapshots
release engine mu before executing damage, messages, scripts/specs and movement.
Display projections snapshot one body then query the other with no lock held.
New reference storage introduces no world/session/transport/lifecycle lock.
The existing removal and world cleanup lock ordering is audited/repaired in
step 3; do not infer full cleanup safety from this checkpoint's reference race
proof. No save field/schema, transport payload or object ownership change.

Passed: make fmt, go build ./..., go vet ./..., go test ./...,
go test ./pkg/game/..., golangci-lint run ./..., git diff --check,
make fidelity-depth, make fidelity-units and make string-census.
Retained gate commands and outputs: step1-gates.py and step1-gates/. Per Zach's stacked approval, the shared-path combined census runs
once on step 4's final committed tip. Intermediate green unit gates do not
permit this partial migration onto main, deploy, or resuming shoot.
