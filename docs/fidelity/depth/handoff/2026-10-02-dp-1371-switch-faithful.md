# E3 faithful player switch train

Zach approved the attached-body design in #1757, including player switch and
return. No NPC-only restriction or new divergence is introduced. This train
starts at #1757's tip `7da317cc1`; it is stacked while that PR awaits merge.

## Invariant and representation

With **no active switch**, readers preserve main's behavior. New selection,
permission, reconnect, retirement and restoration decisions are gated on an
active PC switch or on a character participating in one. GetSession remains the
identity registry API. The attached-body lookup is distinct from that API.
Existing main output framing, ordinary linkdead queues, foreign-candidate save
refusal, entry and transport security remain intact; the normal-path test is
`TestSwitchOrdinaryReadersMatchMain` and the full corpus checks the same invariant.
Lifecycle and manager locks serialize new pointer transitions. SaveToStore now
uses its ownership-checked concrete argument rather than rereading a mutable
session pointer; its ordinary callers already pass that identical pointer.

A switched descriptor retains its original registry key and original Player,
but `s.player` points to the acting PC. The borrowed body's retained linkdead
holder remains in the registry for lifecycle/store metadata, without gaining a
transport. Body output, room look, wizard body lookups, combat/GMCP routing and
wait draining select the attached descriptor; original-body output has no
recipient. Store saves select concrete ownership and keep each body's OLC zone.
OLC authorization and set/stat read that same body-owned zone, while the
original descriptor holder retains the original's metadata.
Explicit return reattaches the original and detaches the borrowed body. Closing
the descriptor leaves **both** bodies descriptor-less, without a return message.

Original login is UNSWITCH. Borrowed login is USURP; it steals the descriptor,
retains a fresh linkdead lifecycle holder for the original, and adopts the live
borrowed body. Both paths discard only the newly restored candidate's objects.
Stale transport teardown cannot unregister either retained/reconnected body.
Extraction returns the descriptor before choosing its menu/retirement path;
forced idle closes an attached descriptor before extraction, whereas an original
that had no descriptor must not close the switched descriptor merely because it
has an idle marker. Return also closes a descriptor occupying the original.

The existing NPC adapter and generic NPC dispatcher remain C1's frontier, as in
the approved design; this train does not synthesize PC objects for NPCs or claim
that dispatcher complete.

## C read and proof boundaries

- `src/act.wizard.c:1175-1204`: parser, visible-world resolver, self/live descriptor,
  Implementor PC gate, attachment; `:1207-1223`: return, including an occupied original.
- `src/interpreter.c:909-914`: acting body's command level; `:921-922`: separate NPC gate.
- `src/act.wizard.c:1154-1159`: snoop explicitly protects the original's level.
- `src/olc.c:178-179`, `src/act.wizard.c:775,3003`: OLC grants/reports/edits
  use the character's zone, even while its descriptor acts through another PC.
- `src/comm.c:2128-2135,2144-2146`: close saves the acting character and leaves
  the original descriptor-less; it does not invoke do_return.
- `src/interpreter.c:1543-1571,1624-1653`: original-first UNSWITCH, attached USURP,
  concrete candidate free, reconnect flags and exact mode output.
- `src/handler.c:1107-1110,1158-1175`: extraction's two return points/menu ownership.
- `src/limits.c:442-444`: idle close is gated on an attached descriptor.

The three live vehicles use the existing peer-drop seam. The dropped peer is
placed in real shipped room 3001, remote from the actor at 8162, before its
socket closes. Thus its lost-link act cannot race the actor's first probe.
The earlier candidate's nonexistent room 1001 did not move it, and the failed
run is retained. No harness timing change or output suppression was added.
The remote vehicles match all three C branches at seed 1; the completed train
claims seeds 1,2,3,5,8, verified by the final combined gate.

## Reader proofs and controls (R5h)

Evidence root:
`~/Archives/darkpawns/oracle-runs/2026-10-02/dp-1371-switch-faithful-proofs/`.
Each named `*-before.log`, `*-reverted.log`, `*-restored.log` triple is **0 → 1 → 0**,
with the red caused by an assertion. `local-gates-before.log` and
`readers-before.log` retain failures before implementation; `lifecycle-before.log`
retains each lifecycle reader's failure on the attachment commit.

| Reader / state | Test | Triple prefix |
|---|---|---|
| attach, level and already-switched priority | TestSwitchAttachAndReturn | attach |
| output routing | TestSwitchOutputRouting | output |
| MovementLook | TestSwitchMovementLook | look |
| PlayerSaver | TestSwitchPlayerSaver | saver |
| DP-1381 both identity/name holds | TestSwitchDP1381Holds | holds |
| explicit original-level snoop | TestSwitchSnoopProtectsOriginalLevel | snoop |
| OLC body grant and set/stat readers | TestSwitchOLCBodyMetadata | olc-zone / olc-set / olc-stat |
| one descriptor wait drain | TestSwitchWaitDrainsOnce | drain |
| HandleTransportDisconnect | TestSwitchHandleTransportDisconnect | disconnect |
| performDupeCheck original | TestSwitchPerformDupeCheckOriginal | reconnect-original |
| performDupeCheck borrowed | TestSwitchPerformDupeCheckBorrowed | reconnect-borrowed |
| ExtractPendingChars both identities/idle | TestSwitchExtractPendingChars | extraction |
| occupied original return | TestSwitchReturnDisconnectsOccupiedOriginal | return-occupied |
| loaded candidate objects | TestSwitchReconnectCandidateObjects | switch-candidate |
| retained original after borrowed usurp | TestSwitchPerformDupeCheckBorrowed | original-holder |

DP-1381's existing guard already checks switched originals and concrete bodies;
its isolated descriptor-reader regression control removes the original arm
while disabling the redundant world fallback in the fixture. No new security
policy is needed. TestSwitchDescriptorReaders covers named/concrete helpers and
descriptor enumeration. TestSwitchReadersConcurrentLifecycle overlaps switch /
return with output, lookup, MovementLook and name holds.
TestSwitchDisconnectConcurrentReconnect races disconnect with both login identities.
The new paths run under `go test -race` (ten repetitions), with retained results.

## Lock acquisitions on new paths

No new path holds `m.mu` while calling world methods, closing a descriptor,
releasing an entry name, or taking `playerLifecycleMu`. Helpers distinguish
public lifecycle acquisition from internal calls, avoiding recursive lifecycle locks.

| Path | Acquisitions, in order; release boundaries |
|---|---|
| attached/body/name lookup | `m.mu.RLock`; descriptor checks use `sendMu.RLock`; body name getters use player read locks; manager released before delivery |
| switch attach | lifecycle → world resolver's world/player read locks (released) → `m.mu.Lock` for pointers (released) → player linkless locks; Send uses snoop read/send read locks |
| return | lifecycle → attached lookup (released) → optional internal disconnect below → `m.mu.Lock` for detach (released) → player linkless locks; no second lifecycle acquisition |
| MovementLook | body lookup (manager released) → existing world/player render locks → existing observation/snoop/send locks |
| output routing | body lookup (manager released) → snoop read lock (released) → send read lock; no lifecycle/world lock in the sink |
| PlayerSaver | identity lookup, then manager read lock for concrete ownership (released) → existing PlayerToRecord player/inventory/equipment snapshots and SQLite save; **no lifecycle acquisition**, including callers under a world lock |
| deferred extraction | world pending-work read check (released), active-switch idle check (manager released); return immediately if no work; otherwise lifecycle → world extraction lock (released) → manager read snapshot (released) → internal return/retirement; editor, snoop, player, inventory/equipment, world removal and send locks remain in their existing cleanup order, never nested under manager |
| disconnect | lifecycle → existing editor cleanup locks/world transitions (released) → player linkless lock → world act/output → saved-record snapshot/SQLite → transport detach → manager detach (released) → player locks; prompt flush after detach |
| both reconnect identities | lifecycle → manager decision (released) → manager detach (released) → player linkless locks → optional NewSession connection-number manager lock (released) → manager registry/attachment publication (released) → entry-name/send close locks and socket close → world concrete-candidate discard → manager-guarded finish pointer assignment (released) → player setters/entry release/world audience/send |
| OLC grant/set/stat | manager read snapshot and concrete body metadata selection; setter uses manager write lock for the identity owner; no lifecycle/world lock in the selection |
| DP-1381 holds | entry-name lock → manager read/player-name locks (released) → world fallback read locks → release entry-name lock; takes no lifecycle lock |

`Register`'s pre-existing manager-held CloseSend path is not expanded here;
new reconnect close calls occur after releasing the manager lock. Both lifecycle
and name-hold order are explicit so review can check their interaction.

## Other readers and retained frontier

Aliases belong to Player and follow the acting body automatically. Prompt,
position and command handlers use the actual `s.player`; snoop is the explicit
original-level exception. Identity keys/authentication remain original-owned.
Admin login, OLC and file-edit authorization read the durable identity record;
no new admin policy is introduced. Transport security and JWT ownership remain
at the existing entry boundary. Go-only structured room/combat output follows
the attached body. Descriptor enumeration suppresses the dormant holder only
while its body has an active switched descriptor.

Still pending: `entry.browser-name-routing`, `entry.identity-consumers`, the
wizard `storedPlayer`/RecordToPlayer temporary loaded-inventory ownership and
its `World.PlayerStoreEdit` sibling, legal-quit rented-object cleanup, and
`entry.login-restrictions` until D6. These are tracked, not silently certified
by the telnet switch vehicles.

## Validation record

Before this train, the three requested #1757 pairs were rerun **at 7da317cc1**:
mount-ridden-depth@5 PASS (19.345 s), informative-residual-depth@2 PASS
(11.584 s), accuse-depth@3 EXPECTED (54.190 s); all three targeted verdicts CLEAN.
Retained run names: `dp-1371-1757-mount5-recheck`,
`dp-1371-1757-informative2-recheck`, `dp-1371-1757-accuse3-recheck`.
They do not rewrite the prior NOT_CLEAN combined verdict.

Normal gates run separately before each commit. One initial helper lint run
raced a moved **test** file; it failed typecheck and was fully rerun clean before
the helper commit. No production mutation or malformed red is a proof.
Final combined results are retained at the train tip and reported in the PR.
Zach will playtest switch live before deploy; this task accesses no production.

## #1758 review correction

MovementLook takes only the manager-locked body lookup before rendering, never
playerLifecycleMu. A concurrent detach may deliver one final look to the old
descriptor. Empty extraction ticks likewise avoid the lifecycle lock; pending
player or mob extraction (including legacy flags), or an idle-disconnected acting
body in an active switch, enters the serialized pass. The test
TestOrdinaryMovementAndEmptyExtractionAvoidLifecycleLock holds the lifecycle
lock and requires both ordinary paths to finish, preserving the main invariant.
