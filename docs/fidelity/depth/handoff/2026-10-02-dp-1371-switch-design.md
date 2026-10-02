# E3 player switch: design proposal

**For review, not implemented.** `switch.player-success`,
`switch.mortal-player-level-gate`, and `switch.already-switched` remain blocked.
The train rules allow a case needing a design decision to be skipped while
independent cases continue. The forced-rent cases finish this E3 train; this
proposal identifies the next switch repair before changing shared ownership.

## Demonstrated boundary

C `do_switch` is `src/act.wizard.c:1175-1204`: it rejects an original already
attached to the descriptor, resolves a visible world character, refuses a live
descriptor on the target, requires Implementor for a PC, then attaches the
same descriptor to the target and keeps the original. `do_return` restores the
original and disconnects any descriptor now occupying it (:1207-1223).
`close_socket` leaves a playing character descriptor-less
(`src/comm.c:2128-2135`). Command lookup uses the acting body's level
(`src/interpreter.c:909-914`); the NPC immortal restriction is separate (:921-922).
The switch table gate is `LVL_IMMORT+1` (:751), so a level-39 PC can reach the
already-switched handler. `set ... level 40` cannot raise a peer to the actor's
own level (`src/act.wizard.c:2898-2903`); the corrected vehicle uses 39.

Go's `cmdSwitch` records `switchedPlayer`, but does not attach `s.player` to it.
It also treats every retained session returned by `findSessionByName` as a live
descriptor, including Go's linkdead lifecycle holder. These are reachable
production paths, not an absent harness capability. The same-room and
short-description substring lookup also bypasses the canonical world/visibility
resolver. Those gates can be repaired locally, but that alone would leave a
false switch: score, movement, commands and output still belong to the original.

The baseline runs `dp-1371-switch-baseline` and
`dp-1371-switch-rent-baseline` retain C's switch success, target score and return
blocks on merged main's Go source. They are diagnostic NOT_CLEAN runs, not proof
claims. The peer-drop link-loss act can straddle the first probe and produces
unstable fingerprints in some attempts; that timing must also be settled before
a live scenario is claimed. Candidate vehicles are retained under
`docs/fidelity/evidence/switch-2026-10-02/`, outside the executed corpus, so they
cannot silently become new expected divergences or poison the final gate.

## Recommended implementation scope

Reuse the existing session registry and switched-original/player fields; do not
redesign the identity registry or synthesize a Player for an NPC. Add an explicit
lookup for the **descriptor attached to a concrete body**, distinct from the
registry lookup used for saved identity and entry name ownership. Scan existing
sessions under the lifecycle lock and use concrete player pointers, including
the original retained by a switched descriptor. A linkdead holder is a lifecycle
owner, not an attached descriptor.

Attach `s.player` to the possessed PC and retain the original in
`switchedOriginal`. Transfer the linkdead target's lifecycle ownership without
creating a second descriptor or abandoning a world body. On return/disconnect,
restore the original, detach the borrowed PC, and restore its descriptor-less
lifecycle ownership. Define the transfer and restoration in one helper with the
existing lifecycle lock; every failure leaves the previous owner intact.

Player command level and position gates follow the acting PC as C does; uses
where C explicitly reads `d->original` (such as snoop protection) keep that
separate policy. This is C fidelity, not a blanket replacement of
`getEffectiveLevel` in every consumer. The existing NPC adapter/dispatcher gap
remains C1's scope.

Before implementation, review this ownership split and its consumer budget:
- named world output, room look/prompt delivery, movement, heartbeat extraction
  and idle retirement route to the attached body;
- reconnect to the borrowed body usurps its descriptor, while reconnect to the
  original performs UNSWITCH; both release the other body correctly;
- return handles the C case where someone else occupies the original;
- session/name holds and DP-1381 file-rename guards retain both identities;
- store saves, aliases, inventory ownership, disconnect cleanup and Go-only
  admin/transport identity keep an explicit original-versus-actor contract.

This list is necessary: simply swapping `s.player` would make the keyed registry
send output to the wrong descriptor and make duplicate login adopt the wrong
body. `world.MessageSink`, `MovementLook`, `PlayerSaver`,
`ExtractPendingChars`, `performDupeCheck`, `HandleTransportDisconnect`,
`getEffectiveLevel` and wizard name-based helpers are concrete other readers.

## Required proof before any row is green

Use the existing peer-drop vehicle first. Establish target descriptor loss
before comparing switch output, retaining the setup/first-probe bytes. Add a
narrow staged-drop capture only if the retained timing failure demonstrates it
is necessary, per the goal's E3 approval. Prove Implementor PC success with
commands/score through the target, lower actor refusal with no ownership change,
and already-switched priority over empty/missing arguments through a high-level
PC body. Include remote visible versus invisible targets and live descriptors.

Unit proofs must cover actor/original pointers, both session owners, output
routing, return, disconnect, both reconnect identities and DP-1381 held names.
Every changed ownership or gate arm needs an assertion-failing revert triple.
One combined census at the completed implementation train tip is required.
No new security divergence is proposed by this note.

## Frontier retained from the entry work

`entry.browser-name-routing` and `entry.identity-consumers` remain blocked;
#1756 completed its entry remainders without claiming these rows. The wizard
file-editor helper (`cmdSet` → `storedPlayer` → `RecordToPlayer`) still loads
inventory into the world for an offline candidate. Its disposal needs a focused
concrete-owner test, including an online same-name body's objects and failure
returns. `World.PlayerStoreEdit` independently calls `RecordToPlayer(r, w)`;
include that sibling in the audit. The forced-rent cleanup now demonstrates
reuse of the existing concrete-owner discard helper, but does not fix or claim
those editor paths. Keep all three frontiers in the next train's handoff.
