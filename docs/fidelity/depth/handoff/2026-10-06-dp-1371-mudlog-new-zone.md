# D7: new-zone producers

Stop tier: descriptor-issued zedit creation/file handling. Fresh main
bf476f7b1 (merge #1817; includes #1815 and the CI timeout repair).
Six producer cases, one commit per producer, then integration evidence.
Rules R1, R3b, R5c/e/f/g/h.

## Contracts

Cite: zedit_new_zone, src/zedit.c:110-229; LVL_BUILDER is LVL_IMMORT
(src/olc.h:54). No invis term, CRLF or final punctuation in these payloads.

| Case | C producer | Payload | Type/min/file | Order |
|---|---|---|---|---|
| success | src/zedit.c:226 | OLC: %s creates new zone #%d | BRF/LVL_BUILDER/TRUE | after files, indices and memory creation; before Zone created acknowledgement |
| zone open | src/zedit.c:133 | SYSERR: OLC: Can't write new zone file | BRF/LVL_IMPL/TRUE | first file; log then return |
| world open | src/zedit.c:150 | SYSERR: OLC: Can't write new world file | same | after zon, before mob |
| mob open | src/zedit.c:169 | SYSERR: OLC: Can't write new mob file | same | after wld, before obj |
| object open | src/zedit.c:178 | SYSERR: OLC: Can't write new obj file | same | after mob, before shp |
| shop open | src/zedit.c:187 | SYSERR: OLC: Can't write new shop file | same | after obj, before indices and memory |

The failure classifier unwraps the existing WriteNewZoneFiles error, accepts
only os.PathError Op=open and maps the five file extensions. It does not parse
error prose, change the shared writer or invent a producer for write/close
errors: C checks fopen but not fprintf/fclose. Existing slog errors remain.
The superseded structured success diagnostic is replaced by C MudLog.

## Proofs and reproducibility

All six registered-command tests failed on missing producer assertions before
repair. They use makeZeditTestWorld's temporary WorldPath; each error fixture
makes just the target 31.<extension> a directory. At log time, earlier files
exist, later files do not, indices are unchanged and no zone is in memory.
Success probes check all five files/indices and the new memory zone at log
time, with no actor acknowledgement queued yet. Qualified BRF observer and
below-level complete/off observers prove type and minimum. An invisible40 actor
proves success has no invis minimum. Coverage and ceiling refusals produce no
log. TestNewZoneMudlogOpenClassifier rejects other operation/error classes.

The focused success oracle creates unoccupied zone2 in disposable world copies,
then repeats it and requests327. Its full C transcript was read: Zone created,
the independent observer's exact creation log, then both refusal strings with
empty observer blocks. Success claims use seeds1,2,3,5,8. The five physical
open-failure fixtures are unit-only; no live C error-oracle proof is claimed.

Reproduce compiling green/revert/restore controls at the tip:

```sh
python3 docs/fidelity/depth/handoff/2026-10-06-dp-1371-mudlog-new-zone-controls.py /path/to/retained-controls
```

Optional third argument selects one control. Removal/type/minimum for all
producers; success invis, acknowledgement, memory and index ordering; open-only
classifier; and shared file-order mutation. Reds require named assertions and
exclude build failures. Each temporary source mutation restores in finally.

## Other readers and lock acquisitions

Success reads actor name and new zone number only; no file format, index
algorithm, creation refusal, world insertion or reset-clock state changes.
Each open-error case reads the structured error already returned by the shared
writer; no additional filesystem probe or write is made. Partial-file retention
and immediate return stay unchanged for each of the five failure positions.
Go's web new-zone endpoint uses olc.WriteNewZoneFiles and UpdateNewZoneIndex,
not the descriptor handler, so gains no new C producer; it and its HTTP error
policy are unchanged. Logging does not move into shared file or world APIs.

The descriptor command takes no textEditMu or lifecycle lock. World coverage
snapshots and CreateZone take/release existing world locks. Creation/index I/O
returns before the producer. No zone/world/manager/player lock is held across
MudLog. Alog's file lock completes before provider snapshot; manager RLocks
release before callback, player/world reads release before delivery. Existing
heartbeat staging, switch routing, snoop and send locks remain the shared
consumer's paths. No new lock or registry is added; selected tests run -race.

## Frontier

This proves six producer boundaries, not full new-zone behavior. The index
failure diagnostics (src/zedit.c:263-270) use self-overlapping sprintf and stay
blocked pending a demonstrated oracle repair. Go also creates the memory zone
before its index loop, whereas C runs indices first; that pre-existing ordering
needs repair/evidence when the index-error producers are addressed. Both sides
have completed those operations at the success producer tested here; this
train does not change their relative order or claim it proven.

Saveall, disk-error, invalid-mode, file-edit/improved-edit, other D7 families,
WHOD network lifecycle and A1/C1 remain. Aggregate D7 stays blocked and inventory
partial. No oracle/reference, harness, pin, ledger or governing-doc edits;
no production access or deployment.

Evidence root: ~/Archives/darkpawns/oracle-runs/2026-10-06/ under
`dp-1371-mudlog-new-zone-proofs`, `dp-1371-mudlog-new-zone-target` and final
`dp-1371-mudlog-new-zone-combined`. Final SHA, gates and census verdict belong
in the PR and run manifest; a running census is not a completed verdict.
