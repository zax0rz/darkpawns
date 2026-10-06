# Shoot Train 2: direct outcomes

Resumed after the combat body-identity prerequisite (#1798/#1800/#1801/#1803)
merged and passed Zach's live duplicate-mob playtest. The historical retaliation
stop is retained in `2026-10-05-shoot-retaliation-identity-stop.md`.

Stop tier, under #1792's approved two-train design. Base: origin/main
`dcea4e13b`, including Train 1 (#1794) and the complete body-identity stack. Six remaining shoot outcome cases.

## Boundaries and other readers

C `src/act.offensive.c:904-987` owns the projectile preamble, strict hit roll,
dex terms, DAMROLL/projectile/bow dice, extraction before improvement, literal
actor/victim/room output, direct HP/update_pos, and zero wait. The command now
calls a C-shaped executor instead of `DoShoot`/`SkillResult`; the unused old
implementation is removed. Generic skills, damage, targeting and wait consumers
retain their implementations. The shared dex_app table is read through an
accessor; no table or probability constants are copied into shoot.

Player hits stay in the destination room and do not enroll combat. NPC hits
relocate only survivors, preserve their pre-relocation posture, then execute one
synchronous hit. Misses preserve HP, position, skill and combat state and prepend
the projectile to the destination. Both paths detach the actual inventory object.
Existing equipment, inventory, crash-save flag and room-list readers use their
canonical movement implementations; no save-format change.

`World.ApplyRangedProjectileDamage` follows `update_pos` without damage's wound
messages. Death composes `gain_exp`'s negative branch (mortal gate, EXP/3, cap and
floor; NPCs are uncapped) with existing `RawKillCombatant`. C: `src/fight.c:186-203,
534-583,625-631`; `src/limits.c:287-325`. No killer reaches the death boundary:
no kill XP/gold/alignment, kill/PK/death counters, PK flag, CON loss, killer-aware
Lua death trigger or damage/skill/death-position messages. Corpse, affects,
mount/memory/hunting teardown and pending player extraction remain delegated to
the existing proven raw-kill boundary. Existing combat and noncombat death
callers do not change.

## Retaliation seam

`CombatEngine.PerformRangedRetaliation` uses `performOneHit`, preserving hit/damage
draws before protections and the shooter's pre-fight posture through damage
calculation. Its private pair flag gates all new branches; ordinary engine
callers retain their previous behavior. C protections and jail subdue precede
attacker enrollment; charm/switcheroo redirects follow attacker enrollment;
victim enrollment follows redirects. C: `src/act.offensive.c:955`;
`src/fight.c:1314-1458,1770-1888`. Both ensuing fighters are retained in the
engine, including the awake victim when the investigating mob is wounded.
The ranged-only hunter callback performs C's attacker hunting assignment after
the enrollment/redirect gates. `StartCombatFromMob` is unsuitable because it
enrolls before protections; `PerformUnenrolledInitialAttack` intentionally lacks
the later fighter contract. Neither existing seam changes its contract.

## Lock acquisitions

The executor holds no lock over output, movement, combat or death. Body getters
acquire their own RLock; object getters use the object's existing instance/proto
accessors. Act snapshots occupants through World RLock and delivers after the
snapshot. Player output snapshots worldRef under Player RLock, then MessageSink
under World RLock, releases both before delivery.

Projectile detach/drop/extract acquire World Lock, then the existing Inventory
lock and crash flag's Player lock as required by the canonical object movement.
Ranged HP subtraction is atomic under one body Lock; existing damage-taken
metrics are preserved. EXP mutation acquires one body Lock at a time, release
before RawKillCombatant. Existing raw-kill cleanup obtains body and world locks
at its proven boundaries; NPC corpse/extraction holds World then Mob, releases
before subsequent output and object moves. Player extraction is marked for the
existing lifecycle drain; no new lifecycle acquisition.

The bare mobile transfer uses World Lock, with individual Mob locks for room,
light and room-entry sequence. It preserves posture and applies the destination
circle timer under World Lock. No equipment or mount is moved by the bare
transfer.

The new combat entry starts without the engine lock. After protections/jail,
startCombat uses Engine Lock and individual body getters/setters (the existing
engine-to-body order), releases it before redirects. Registered-pair lookup uses
Engine RLock; victim enrollment uses Engine Lock plus existing body readers.
No engine lock is held over output, world callbacks, death or the synchronous
swing. The hunter callback takes only Mob locks through flag/getter/setter calls.
No new session-manager, transport, lifecycle, persistence or admin lock.

## Proofs and evidence

Real command fixtures cover PC/NPC hit/miss, exact actor/victim/origin/destination
bytes, damage and next RNG draw, both object dice, improvement, direct lethal
PC bookkeeping, projectile ownership and unchanged existing wait. Separate engine
proofs pin protections, jail ordering, sleeping victims, exactly one synchronous
swing, both later fighter memberships and wounded-attacker enrollment. Raw-kill
fixtures pin lethal EXP/corpse/pending extraction and unchanged killer/counter/
PK/CON state, plus HP thresholds without wound messages.

The initial command tests fail on the pre-prerequisite main's actual output/state assertions.
The current-tip controls below reproduce every green/red/restored-green triple. Mutation
controls independently remove each dex term, change strict comparison, substitute
either object dice, omit improvement/retaliation/extraction/raw-kill, move PCs,
drop into the wrong room, add wait/death counters, and change EXP/3 to EXP/37.
Every accepted red is a compiled assertion failure with restored green.

Oracle fixtures exercise forced PC/NPC hit, miss and lethal arms with real origin
and destination peers. The pre-existing shoot-target-depth becomes clean, so its
old blocked ledger entries and pin are removed rather than replaced. Fixture
setup rejection is recorded separately and is not behavioral evidence.

Retained evidence: `~/Archives/darkpawns/oracle-runs/2026-10-05/`
`dp-1371-shoot-outcomes-proofs/` and the train's named census directories.
Final verdict and per-case claims are recorded in the PR and manifests after
validation. No reference-oracle source or binary changes.

Retained class frontier: the general CharTransfer has broader fight teardown
than bare C char_from_room and does not run the destination circle_check timer.
The shoot-only transfer is therefore pointer-based and restricted to its
unfighting victim, preserving posture and room light/entry sequence and applying
that circle side effect. General transfer callers need a separate scoped audit;
this train changes none of them. C references: `src/handler.c:504-553` and
`src/utils.c:877-893`. The existing shared mobile raw-kill removes its active
registry entry immediately; C's deferred mobile extraction remains owned by the
retained C3 lifecycle frontier, rather than being silently certified here.

Structured-health audit: agent variable getters read current GetHP; engine
DamageFunc retains its existing notification behavior for retaliation. The old
skill/DoSpellDamage path did not call that engine callback either. This train
adds no output-routing, dirty-variable, session or transport branch. Existing
raw-kill teardown owns corpse persistence and pending player extraction.

## Reproducible controls and identity audit

Run in a disposable checkout, with no concurrent edits or tests:

```sh
python3 docs/fidelity/depth/handoff/2026-10-05-shoot-outcomes-controls.py /tmp/shoot-triples
```

The control runner requires an initial green, a compiled assertion failure, and
restored green for each control. The concurrent HP proof is additionally run
under `-race`; its earlier non-atomic control is retained as diagnostic history,
not counted as a deterministic triple by this runner.

`TestShootRealDuplicateMobRetaliation` spawns two real equal-description mobiles,
puts the first in an unrelated fight, and shoots through the second's retaliation
boundary. It checks exactly one immediate hit, each body's actual reciprocal
target, all four fighters' ensuing turns, and retirement isolation.

Reproduce the scoped identity sweep:

```sh
rg -n 'GetFighting\(\)|GetMobByName|GetPlayer|SetFighting\(' pkg/command/shoot.go pkg/game/shoot.go pkg/combat/engine.go
rg -n 'RangedRetaliation|RangedHunt|ApplyRangedProjectileDamage|TransferRangedVictim|DoShoot' pkg --glob '!**/*test.go'
```

The shoot and retaliation paths retain actual bodies throughout pair keys,
fighting fields, callbacks and cleanup. The only new name projection is the
hunter's existing PC prey field; this callback accepts a concrete PC and does
not reselect a body. No second registry. New combat branches are gated by the
ranged pair marker; ordinary direct and round paths retain main's behavior.
The final combined census certifies all existing claimed pairs, including the
combat identity scenario, on this train's committed tip.

## Validation before the combined run

At the resumed final source, formatting, build, vet, full tests, game tests,
lint (zero issues), diff check, depth, unit claims (1,335 passing) and string
census passed. Focused `-race` over Shoot/Ranged paths passed. All 27 independent
mutation controls completed initial green / compiling assertion failure /
restored green; restoring main's actual CmdShoot/DoShoot route additionally
failed all seven outcome test groups and restored green.

The seed-1 targeted run `dp-1371-shoot-resumed-targeted` passed all seven
scenarios (six new outcomes plus the existing shoot-target-depth), with no
expected rows or infrastructure rechecks. C transcripts were inspected for
actor, origin, destination and PC victim outputs, including lethal death cry.
The final combined run supplies five-seed proof and full/claims verdicts; its
exact committed tip and result are reported in the PR, not inferred here.
