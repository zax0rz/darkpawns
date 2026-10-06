# D7: remaining wizard mudlog command producers

Stop tier: changes session command handlers and raw force argument routing.
Base: origin/main `5b6daa738` (merge of #1807). Nine producer sites, one
commit per site, one combined census at the final tip. Governing rules:
R1, R3b, R4, R5e, R5g and R5h. This does not complete the aggregate D7 audit.

## Contracts and other readers

| Case | C source | Type / minimum | Order and other readers |
|---|---|---|---|
| load mobile | `src/act.wizard.c:1314-1316` | BRF / actor level + 1 | After world placement, before random narration; final period. Reads concrete mobile short description and current room name. Spawn, inventory, RNG and combat consumers unchanged. |
| load object | `src/act.wizard.c:1369-1371` | BRF / actor level + 1 | After inventory placement, before random narration; no final period. Object location and persistence readers unchanged. |
| purge player | `src/act.wizard.c:1437-1441` | BRF / LVL_GOD | Room disintegration act, producer, then close and extraction. Add the missing PC room act at the same boundary; NPC/object/whole-room purges acquire no producer. Identity/store cleanup remains the existing lifecycle path. |
| force single | `src/act.wizard.c:1875-1880` | NRM / max(actor level + 1, invis) | Actor acknowledgement, optional victim notification, log, then interpretation. Preserve the raw half_chop remainder at the transport boundary; the parsed wrapper remains for internal callers. |
| force room | `src/act.wizard.c:1884-1885` | NRM / max(actor level + 1, invis) | Acknowledgement and log before the target loop. Existing NPC-dispatch gap is retained, not claimed repaired. |
| force all | `src/act.wizard.c:1896-1897` | NRM / max(actor level + 1, invis) | Acknowledgement and log before the descriptor loop. Existing target selection and alias policy unchanged. |
| reset zone | `src/act.wizard.c:2063-2067` | NRM / max(LVL_GRGOD, invis) | Reset and acknowledgement precede log; log uses table index, not zone number. Reset driver, queue, occupancy and admin endpoints unchanged. |
| reset world | `src/act.wizard.c:2046-2051` | NRM / max(LVL_GRGOD, invis) | All resets and acknowledgement precede the log. This does not certify the separate zone-error producer family. |
| newbie | `src/act.wizard.c:3531-3540` | BRF / actor level + 1 | All gifts, actor acknowledgement and victim act precede the log. Concrete PC/NPC target name is used, not the typed abbreviation. Locations, equipment and saves unchanged. |

Every producer passes file=TRUE; load/purge/newbie do not gain an invisibility
term. MudLog is the proven shared consumer. No new registry, lifecycle lock,
transport output queue or persistent state is introduced. Nested `forceSessionCommand` passes the existing raw-argument API its
unchanged half_chop remainder, so another force producer does not lose spaces.
Other supported raw consumers (send, set, string, help, gecho and wiznet) use
the same existing path; plain tokenized commands and the NPC gap remain as
before. The log lines use
the existing heartbeat transaction, snoop, switch and transport delivery path.

## Proof boundary

`TestWizardMudlogProducers` reaches all nine through real registered handlers,
real bodies and the real immortal provider. Exact file/observer payloads, minimum and below-minimum levels, and
brief-versus-normal eligibility are asserted for each producer. The fixture raises actor invis above the observer for the three
non-MAX families to catch an incorrectly shared threshold helper.

`TestWizardMudlogThresholdAndOrder` proves load level+1 and pre-narration
ordering, force's invis/type filtering, zone acknowledgement-before-log,
newbie gift/message-before-log state, and purge room-act/log-before-teardown.
`TestWizardMudlogForceBeforeInterpretation` observes standing at log time and
sitting after interpretation for each branch; single-target notification must
already exist, whereas loop notifications must not yet exist.
`TestWizardMudlogForceRawRemainder` and `TestWizardMudlogNestedForceRemainder`
exercise actual raw dispatch, including forced-to-force entry, and preserve
internal/trailing spaces in the log. `TestWizardMudlogRefusals` covers the seven
staged missing/invalid/equal-or-higher-level branches, not every handler gate.
`TestWizardMudlogNewbieMobileName` proves the concrete short description and
four gifts at log time on the NPC branch. The existing D4 consumer owns
writing/color/syslog-off and its full matrix.

Fourteen compiled controls (nine producer removals, three raw/act boundaries,
and wrong-type / wrong-threshold classifiers) are reproducible at the final tip:

```sh
python3 docs/fidelity/depth/handoff/2026-10-06-dp-1371-mudlog-wizard-controls.py \
  /path/to/retained-controls
```

An optional third argument selects one case, enabling reproduction at that
case's own commit. A red requires its named assertion, not a build failure.
Baseline all-nine assertion failures and controls are retained under
`~/Archives/darkpawns/oracle-runs/2026-10-06/dp-1371-mudlog-wizard-proofs/`.
Each `case-<name>/` also contains separately exited normal gates.

`mudlog-wizard-producers` has a first-created level-40 observer, a separately
promoted level-38 actor, and a mortal victim. It covers eight producers with
five-seed claims. The mob producer has a unit proof only: `load.mob-success`
retains its A1 actor-transcript anomaly pending diagnosis. No claim about its
actor bytes is inferred from the unit proof.

The first draft's self-demotion fixture was vacuous: C demotes the actor to
level 1 and all probes refuse. Its retained green is discarded as proof. The
corrected transcript was read and contains each intended bracket line.

World reset with quiet mobiles exposes C `log_zone_error`
(`src/db.c:2048-2057`), whose two NRM/GOD producer calls are separately unported.
Quiet mobiles remove the carriers of P-command containers 9102 and 19735.
The focused vehicle supplies those objects in both disposable worlds using
existing spawn/force-load fixtures, so it exercises the wizard producer in a
valid container topology. No error bytes are normalized, no expected row is
added and no seeds are dropped. The error family remains a future D7 case.

A separate `load mob 4` candidate ends the C actor connection with EOF at
probe step 2, reproduced on fresh main `5b6daa738` with the same reference.
Logs are retained in `mob-diagnostic/` and `main-mob-repro/` under the proof
directory. This is a connection failure observation, not a guessed UB or heap
diagnosis. The candidate is retained outside the claimed corpus; the mob log
row is unit-green only and the existing A1 row stays blocked.

## Lock acquisitions

No producer is called with a body, world, manager or lifecycle lock held.
Placement/reset/getter/act calls return before producer invocation:

- Load: world spawn/placement locks are released, room getter briefly takes
  world RLock, player level getter briefly takes player RLock, then MudLog.
- Purge: world resolver and level getters release their locks; room Act
  completes; MudLog runs before UnregisterAndClose enters lifecycle cleanup.
- Force: level/room/invis getters release body locks; MudLog precedes
  forceTargets' manager RLock and all target command execution.
- Reset: ResetZone takes zoneResetMu, snapshots the zone under world RLock,
  executes the existing serialized reset, and releases zoneResetMu on return.
  The producer runs after return and acknowledgement. Admin ResetZone endpoints
  and their lock order are unchanged.
- Newbie: each canonical spawn/move returns with its locks released; victim Act
  completes; then level/name access and MudLog.

Shared consumer: Alog completes its logging lock before provider access;
provider-global RLock is released before EachSession. EachSession snapshots
manager sessions under manager RLock, releases it, then briefly reads bodies
and active switch ownership under manager RLock before callbacks. Recipient
flag/level/color getters take and release player locks. Player.SendMessage
reads player worldRef under player RLock, releases it, then snapshots the sink
under world RLock and releases it. The manager sink resolves its session under
manager RLock, releases it, and uses existing heartbeat staging/snoop/send
locks. These locks are sequential, not a new manager→world held-lock chain.

## Remaining frontier

D7 remains blocked: OLC, object saves, Lua diagnostics, combat/spec-proc,
clan/house, whod and other producer families still need per-site reconciliation
and proofs. The original inventory remains explicitly partial. NPC-force
C1, mob-load A1, browser/identity readers and the reset-state remainder are
not closed by this train. No oracle source or shared reference binary changes.

Final gates, census SHA/verdict and CI links are recorded in the PR and retained
run manifests; a running or failed census is not a completed validation claim.
