# DP-1371 D7: four OLC open-parent producers

**Stop tier.** Implements approved #1825 Train 1 only, based on fresh main
`00f32f867`. Four cases, one commit each, then proof integration and one combined
census. File-editor storage/producer work and checked stdio writes are later
work under the approved sequence.

## Boundary and C evidence

REDIT `src/redit.c:291-294`, OEDIT `src/oedit.c:347-350`, MEDIT
`src/medit.c:349-352`, SEDIT `src/sedit.c:481-483` log their exact open-failure
payloads at BRF, level 31, file=true. Actor invisibility does not raise that
minimum. Command acknowledgement and CMP save announcement precede the attempt
(`src/olc.c:186-226`); an open error adds no new acknowledgement and retains
the dirty marker.

Retained reference-C experiment boots the unchanged reference binary from a
copied, empty-player library. After boot, it replaces each world subdirectory
with a regular file, issues the actual `redit/oedit/medit/sedit save 30` command,
and restores the directory in finally. REDIT's missing-parent variant is also
captured. All five raw C responses contain the expected error following the
existing Saving all ... line. No oracle/src edit, instrumentation, shared
binary replacement or live server access. It uses one level-40 BRF actor;
independent lower-level recipient filtering is tested in Go with the existing
proved shared MudLog consumer, not claimed as a new differential scenario.

## Classifier and readers

`olcOpenParentObstruction` accepts only a wrapped PathError from open/mkdir with
ENOENT/ENOTDIR, and then requires the actual parent to be missing or not a
directory. Valid parents, permission/resource errors, generic errors, and
write/sync/close/chmod/stat/rename stages are excluded from this producer
mapping. It does not rewrite/wrap the returned save error. It does not pre-open,
truncate, chmod or change the target. It does not alter MkdirAll, atomic
replacement or dirty-marker policy. Temporary filenames do not select payloads.

Each existing command error arm calls the helper with its exact C payload and
editor kind. All four calls are after the save wrapper returns. SourceDir is
used for obj/mob/shp, WorldPath for wld; absent world/source state is not a file
open claim. Filesystem inspection can race another operator repairing/removing
the obstruction; the classifier conservatively requires both the recorded
failure and a present parent obstruction, so a repaired parent can suppress a
log rather than broadening the mapping.

Other readers: SaveOLCZone/admin still use the same Locked writers and return
errors with no new broadcast. Saveall, editor commit, disconnect cleanup,
source serialization, web routes, caches, paths and existing success producers
are untouched. The shared atomic writer has no edits. The registry and save-list
retain their owners; the tests set and inspect the real per-kind dirty marker.
The site inventory says implemented-boundary **only for parent obstruction**;
broader fopen permissions/resources and missing-directory creation differences
remain unproven under blocked mudlog.unported-sites. No complete-fopen claim.

## Locks

For all four producers, command dispatch holds no manager, world, player,
editor or lifecycle lock across the handler. The save wrapper takes zoneSaveLock,
snapshots acquire/release world read locks, filesystem I/O runs under the zone
lock, and the wrapper releases it before the command helper logs. GetParsedWorld
reads and releases its own world lock before inspection. The file-log-time
assertion takes the zone mutex with TryLock to prove it is already released.

MudLog takes/releases its file/provider locks, then EachSession snapshots under
manager read locks. Active-switch lookup is manager-locked without lifecycle
lock. Player flags/level/color and Player.SendMessage release their player/world
read locks before calling MessageSink. MessageSink resolves the descriptor,
then delivery acquires heartbeat staging or session send/output/snoop locks.
None is retained by these producers. Existing full delivery remains shared;
no output, transport or switch routing changes. Admin's whole-zone mutex still
covers its multi-file writes, but no producer is added there.

## Fail-capable proofs and reproduction

Four real-command tests fail on missing producer before the fix, compile and
pass after it. Each checks exact independent BRF observer bytes, below-level
and syslog-off exclusions, invisibility independence, file payload, error order,
no extra acknowledgement, retained dirty marker, released zone lock, admin
silence and successful-save negative case. REDIT includes missing-parent too.
Classifier tests cover the positive errno/stage/parent conjunction and negative
stages, permission, valid parent, nil/plain errors and a descendant of a file.

```sh
python3 docs/fidelity/depth/handoff/2026-10-06-dp-1371-olc-parent-C.py \
  --output /tmp/olc-reference-parent
python3 docs/fidelity/depth/handoff/2026-10-06-dp-1371-olc-open-controls.py \
  --output /tmp/olc-open-controls
```

The second command temporarily changes the isolated Go worktree and restores
source in finally. Four producer removals and three classifier-guard removals
must compile, fail their named assertion, and pass again after restoration.
Build errors or timeouts do not count. Evidence and separate per-case/final
gate logs are retained under
`~/Archives/darkpawns/oracle-runs/2026-10-06/dp-1371-olc-open-errors-proofs/`.
Final census and CI records are in the PR; no final verdict is claimed until
its committed-tip combined run completes. No repins, dropped seeds, expected
divergences or oracle changes.

## Bounded checked-write experiment: inconclusive, retained blocked

Under Zach's follow-up approval, booted the unchanged reference C binary from
another disposable empty-player library and replaced existing zone-10 `.mob`
and `.shp` targets with symlinks to `/dev/full` **after boot**. Restored each
target and killed the disposable process afterward. Neither actual save
command emitted its checked-write diagnostic; both issued only the existing
Saving all ... acknowledgement. Raw responses and full transcript are retained.

This does not prove the checks unreachable. It supplies a failing backing
store but does not identify a negative return from either checked fprintf;
stdio can buffer the header and fail an unchecked body/final flush. It therefore
cannot justify mapping Go's single buffer Write, Sync or Close to either C
producer. No oracle instrumentation was introduced or proposed as necessary.
Per the approval, leave both checked-write sites blocked with this evidence;
no further experiment train is scheduled. Other stimuli/platforms remain
unproven. Reproduce with the adjacent olc-checked-write-C.py --output <directory>.
The two explicit blocked manifest rows refine the existing blocked aggregate;
they are not new regressions or approved divergences.

An initial parent-guard mutation left a local variable unused and was rejected
as a build failure, not counted. The corrected mutation preserves variable use
and fails the valid-parent assertion; its green/revert/restore logs replace that
invalid triple. The old pre-save announcement test now expects the new error
line on this obstruction and filters its ordering probe to the announcement,
so a second error write cannot mask an ordering regression.
