# DP-1371 D7: OLC error producers and atomic-save boundaries

**Stop tier — design only.** Approval requested before changing session saves,
file-editor state or I/O classification. No game, harness, oracle, governing
file or manifest status changes. Based on fresh main `9d69441d0` and DeepSeek's
read-only eight-site audit, checked against the source again.

## Problem and recommendation

C emits eight missing error producers in this inventory group. Go uses atomic
replacement instead of C's in-place stdio writes. Preserve atomic replacement,
but do not turn its every error into a C diagnostic (R1/R4/R5e).

Recommend two implementation trains and a separate checked-write investigation:

1. Four zone-open producers, initially restricted to demonstrably common
   missing/non-directory parent failures. Emit in the telnet handler after the
   save helper returns; keep admin saves free of new player broadcasts.
2. File-editor logical storage identity, successful save/delete producers
   parked from `local-01`, and verified delete/open-error boundaries together.
   Shared pathname state makes this a stop train, not mudlog-only local work.
3. Diagnose MEDIT/SEDIT's checked stdio writes independently. Do not map an
   atomic whole-buffer write, sync or close failure to a checked header call.
   Those two sites stay blocked until an actual matching boundary is proven.

Each train is stop tier, one commit per case, compiling revert controls,
other-readers and lock audits, combined census at its tip, PR and stop.
Approval of this note does not approve a new divergence, diagnostic string,
or alteration of dirty-marker, cache, acknowledgement or atomic-save policy.

## Verified contracts

| C site | Checked boundary and exact payload | Type / minimum / file |
|---|---|---|
| `src/redit.c:291-294` | fopen(w+) failure: `SYSERR: OLC: Cannot open room file!` | BRF / 31 / true |
| `src/oedit.c:347-350` | fopen(w+) failure: `SYSERR: OLC: Cannot open objects file!` | BRF / 31 / true |
| `src/medit.c:349-352` | fopen(w) failure: `SYSERR: OLC: Cannot open mob file!` | BRF / 31 / true |
| `src/medit.c:362-367` | Each mob's #vnum fprintf failure: `SYSERR: OLC: Cannot write mob file!\r\n` | BRF / 31 / true |
| `src/sedit.c:481-483` | fopen(w) failure: `SYSERR: OLC: Cannot open shop file!` | BRF / 31 / true |
| `src/sedit.c:485-487` | One fixed-header fprintf failure: `SYSERR: OLC: Cannot write to shop file!` | BRF / 31 / true |
| `src/file-edit.c:38-45` | kill_file failure: `SYSERR: Can't delete file '<storage>'.` | CMP / 40 / true |
| `src/file-edit.c:47-58` | fopen(w) failure: `SYSERR: Can't write file '<storage>'.` | CMP / 40 / true |

Mob-header checking is **per mob**, not only the file's first write. Its embedded
CRLF is literal and must be checked in raw delivered bytes. SEDIT returns from
its failed-header branch without closing FILE; reproducing that leak is not
part of the producer contract. Classifier-only tests cannot certify either
stdio call's live reachability (R5f/R5h).

C's command-level save announcement precedes disk work (`src/olc.c:186-226`);
Go already does this. Preserve it even if the disk operation later fails.
File-editor errors produce no Saved./Deleted. acknowledgement, then perform
common cleanup (`src/file-edit.c:68-72`). Do not reorder that envelope.

## Train 1: zone open failures

The four telnet handlers call saveRedit/Oedit/Medit/SeditZone. Their wrappers
acquire the per-zone save mutex and release it before returning. Shared admin
SaveOLCZone calls their Locked siblings under the same mutex. Producer insertion
belongs in the four telnet error arms, not those shared writers: the web route
has no C descriptor-command counterpart.

Use the editor kind plus a narrow inspected error boundary; never infer editor
kind from a temporary filename extension. If a typed physical-stage error is
needed, it wraps the original cause and preserves errors.Is/errors.As behavior
and all existing caller error text. It changes no return/cleanup policy.

Initially permit an open diagnostic only when a missing/non-directory parent
has been demonstrated to prevent both the atomic open and C's target fopen.
Exercise each actual handler with a parent component replaced by a regular
file; REDIT also has a missing-parent vehicle. OEDIT/MEDIT/SEDIT's MkdirAll
failure is not intrinsically a C fopen result: prove the same obstruction at
the target. Keep any C-only missing-directory versus Go-created-directory
case as an explicit unproven frontier, not silently covered.

Do **not** accept open/openat/mkdir operation names alone as universal parity:
opening a new temporary file requires directory write permission, whereas C
may rewrite an existing writable file in a non-writable directory. Conversely,
atomic rename can replace a read-only target C cannot fopen for writing.
Go target/stat/root/temporary-name failures must be distinguished. Other
permissions and resource-exhaustion cases need paired evidence before adding
their mapping. No preflight that truncates or otherwise changes the real target.

Proofs per producer: independent level-31 BRF observer, level-30 exclusion,
file payload, command announcement before error, no new acknowledgement,
dirty marker retained, successful-save negative case; delete the producer and
observe its assertion fail, then restore. Negative classification controls for
write/close/chmod/rename must not emit an open diagnostic. Admin failing saves
must preserve their HTTP/error behavior and produce no new descriptor log.
Manifest names describe the proven parent obstruction; broader fopen coverage
and the aggregate remain blocked rather than claiming completeness.

## Train 2: one file-editor storage identity and producer family

Carry an explicit C-facing logical storage string separately from the physical
path. Initialize it from the selected C file constant for tedit, including
`text/help/screen`, not blindly `text/<command>` (`src/tedit.c:45-59,97`;
`src/db.h:49-63`). Lua uses `scripts/<validated-subdir>/<filename>` or the root
form, following the actual argument assembly (`src/luaedit.c:29-54`;
`src/db.h:79`). Never derive it by stripping an arbitrary host-path prefix.
Audit every textEditState initializer and callback so room/live-string editors
cannot acquire file producers accidentally. Physical paths, rooted web writes,
symlink handling, caches and scripting failure invalidation keep their owners.

Port the save/delete success producers (`src/file-edit.c:44,57`) in this same
train because they consume the identical storage identity. Use the acting
body's name, including a switched body; preserve log-before-ack ordering.

Delete needs the whole helper contract: `src/file-edit.c:19-25` treats **any**
failed access(R_OK) as success without attempting remove, not just ENOENT.
Go's remove/non-ENOENT test is therefore not universally equivalent. First
prove readable-target remove failure, absent target, and unreadable target
against C; port a faithful readability/delete boundary if feasible, or stop
with the platform/permission evidence. Do not invent the failure message for
an access failure C skips. Ordinary tedit does not enable killOnEmpty; use the
real luaedit route for deletion controls.

File-save open producers need the same common-failure discipline as train 1.
AtomicWrite's root/stat/temp-open/write/sync/close/chmod/rename stages are not
interchangeable. No generic all-errors broadcast. Success/error/absent-delete
controls must check exact logical filename, CMP/40 for errors, CMP/34 for
success, no error acknowledgement, cleanup and writing-flag order. Include
symlink, web-cache and callback negative cases, and an independent observer
while the actor is PLR_WRITING. Retain compiling per-case revert triples.

## Checked-write investigation

MEDIT checks each header fprintf; its later body writes are unchecked. SEDIT
checks the initial header only. C stdio buffering can defer an OS failure to a
later call or fclose, which C does not check. Thus a synthetic Go PathError
with Op=write and an expected payload is only a classifier test, not a port.

Propose a disposable experiment distinguishing failure during a checked header,
an unchecked body, and final flush. Keep the shared reference binary and src
read-only. If reliable reproduction requires oracle instrumentation, propose
that exact seam separately for approval under the reference-oracle procedure.
Only then choose a C-compatible checked-write boundary; do not guess buffering
semantics or mark the two sites excluded for convenience.

The related dirty-marker differences on unchecked writes are outside these
producer-only changes. Inventory them with paired evidence before proposing a
state repair; do not clear Go markers to hide a logging gap.

## Lock and other-reader audit to carry into implementation

Zone handler producers run after the zone mutex and world snapshot locks have
been released. Admin still serializes snapshot/write/dirty-marker cleanup
under the zone mutex, with no new producer. Audit all early returns and callers.

Ordinary file completion holds textEditMu then liveTextEditMu. AtomicWrite's
writeMu has been released when it returns, so do not emit from inside that
filesystem lock. MudLog reads its provider, snapshots the manager, reads player
flags/level/color, then sends through Player.SendMessage. That path **does take
world.mu.RLock** to obtain MessageSink; absence of a world write lock is not a
complete safety argument because recursive reads can block behind a queued
writer. Require no world or player lock held at a producer. Delivery also
traverses active-switch lookup, heartbeat staging and snoop sinks: check those
concrete paths for acquisition of either editor mutex before declaring safe.

Implementation must record actual locks at every producer and audit admin,
web file saving, saveall, disconnect cleanup, switched acting-body names, live
text readers and script caches. Tests race save versus editor cleanup and
shared admin saves as appropriate. Current note makes no universal lock-safety
claim and adds no new test-only evidence to the manifests.

## Validation and approval requested

Docs only: no census. Run the required repository and fidelity gates before
opening. Approval requested for the bounded train-1 placement/classification,
the combined file-editor family in train 2, and leaving checked-write sites
blocked for the separate demonstrated investigation. All existing approvals
and stop conditions still apply. No repins, removed seeds or new divergence.
