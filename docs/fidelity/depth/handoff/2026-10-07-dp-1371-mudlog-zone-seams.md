# DP-1371 D7: zone diagnostics and argument-control closure

Stop tier: post-open diagnostic rendering changes the shared atomic-save
boundary and command/save-all writer selection. Fresh main 842c5897b (#1845).
Eight site cases: seven resolved, one explicitly skipped pending design.
One commit per case; R1/R3/R5e/R5g/R5h apply.

## Resolved cases

| Producer | Result | Proof |
|---|---|---|
| zedit.c:369, failed fopen | unit-green, bounded common parent obstruction | TestZeditOpenParentMudlog and TestZeditOpenParentUsesWriterRoot plus unchanged reference-C save 10 with obstructed zon parent |
| zedit.c:436, unknown command omitted | unit-green, real command/save-all | TestZeditUnknownCommandMudlogBoundary plus unchanged reference-C imported conditional X/Z opcodes |
| zedit.c:757, ARG1 display default | excluded from valid editor input | TestCZeditArg1DisplayDefaultUnreachable |
| zedit.c:795, ARG2 display default | excluded from valid editor input | TestCZeditArg2DisplayDefaultUnreachable |
| zedit.c:850, ARG3 display default | excluded from valid editor input | TestCZeditArg3DisplayDefaultUnreachable |
| zedit.c:1071, ARG1 parser default | excluded from valid editor input | TestCZeditArg1ParseDefaultUnreachable |
| zedit.c:1136, ARG2 parser default | excluded from valid editor input | TestCZeditArg2ParseDefaultUnreachable |

### Open-parent error

`src/zedit.c:366-370`: BRF/31/file TRUE, no invisibility term, after the
existing Saving all zone information acknowledgement and CMP producer.
The existing classifier maps only ENOENT/ENOTDIR at open/mkdir with a present
missing/non-directory parent. No permissions/resource/write/close/chmod/rename
failures are relabeled as fopen. The command producer runs after the save
wrapper releases the zone mutex. Save-all already logs this error once; admin
remains silent. Success clears the real dirty marker; failure retains it.

Zone-parent classification follows WorldPath, the actual zone writer root, rather than parsed SourceDir. A separate test and revert control use different roots to prove it.

The initial C attempt used save 30, whose C room-range lookup selected zone 27;
it failed our expected-number fixture assertion and is retained as an invalid
fixture attempt. The corrected save 10 run demonstrates the actual zone-10
payload. Go's custom fixture selects zone 30 and asserts its own dynamic number.

### Unknown reset commands

`src/db.c:1575-1607` loads unknown opcodes and their if_flag. Renumbering does
not reject them. The boot reset normally disables them, but `src/db.c:2084-2091`
skips a conditional command if its predecessor failed. A copied zone 10 with
failed G and conditional X/Z therefore retains both unknown commands. The
unchanged reference proves that actual zedit save 10 emits the two exact
BRF/31/file-TRUE diagnostics in source order and omits both records, while an
obstructed open emits only the open diagnostic. No C instrumentation or source
edits; only disposable input data changes.

Go keeps zeditDiskArgs free of broadcasts. The post-open renderer logs in the
writer's scan and skips unknown commands, skips '*' silently and stops at S.
Command save and OLC save-all select this diagnostic mode; the existing Locked
entry used by SaveOLCZone/admin stays silent. The byte-buffer atomicWriteFile
wrapper retains its existing interface and behavior. The new renderer runs
only after temp open succeeds, before write/close/chmod/rename; error propagation,
atomic replacement, temp cleanup and dirty-marker policy are preserved.

Units exercise the real parser and loaded zone, command and save-all dispatch,
independent BRF observer, below-level refusal, actor invisibility independence,
file payload and count/order, acknowledgement timing, open-first silence,
pre-publication target preservation, dirty timing, disabled-command and
terminator negatives, and admin silence. Imported X/Z are the proven live
boundary; the supplied disabled marker and post-terminator tail are additional
writer contract controls, not claims about authored C syntax.

### Argument defaults

`src/zedit.c:1002-1035` is the only selected-command writer and accepts only
MOPEDGRL before either direct ARG1 display or the IF_FLAG transition. New blank
N commands remain at command-type selection until this gate succeeds.

ARG1 display handles all eight types; D/R/L assign the room and bypass ARG1
input (`src/zedit.c:729-760`). ARG2 display handles all eight (`768-799`).
M/O/G finish in ARG2; only E/P/D/R/L reach ARG3 (`806-854`, `1078-1140`).
The ARG1 parser consequently receives M/O/P/E/G (`1039-1075`); ARG2 has all
eight arms. The four ARG3 calls belong to ARG2 parsing. Imported unknown
commands may appear in the main menu, but editing them still passes through
command-type validation before argument entry.

Read-only bounded source-shape audits assert command validation, the literal
mode-writer/caller sets, covered incoming types and the whole-src ownership
sweep. Counterexamples remove a handler locally while keeping callers intact,
remove a handler globally, or remove the validation gate. Changed source
shapes require re-auditing. These are not general C parsers or fabricated
invalid-descriptor C executions. Defensive Go error branches remain unchanged.

## Skipped: corrupt object extra-description representation

`src/oedit.c:418-426` logs only null keyword/description pointers. This site
is reachable: `src/db.c:2690-2744` returns NULL for an empty authored tilde
field. The unchanged reference logs a copied object with such a node.

A distinct valid live state is produced by F -> keyword -> description ->
single line -> /d1 -> /s -> menu exit -> internal save. `src/improved-edit.c:207-209`
keeps an allocated empty string after deleting the last line; both pointer
checks pass and C writes `E\nallocated~\n~\n` with no corruption diagnostic.
The corrected reference experiment proves that state and saved bytes.

Go's real editor reaches that allocated-empty state, but publication loses its
editor-only presence metadata and writeOeditObj drops the record for empty text.
The retained Go overlay compiles and fails its named assertion demanding the
C file record. It is intentionally not installed in the test suite, and no green
or approved divergence is claimed. The first reference experiment selected E
(the applies menu) instead of F and is retained but not counted as proof.

A blanket empty-string MudLog would misclassify valid editor output. This case
remains blocked rather than silently widening or inventing C's guard. Before
repair: design presence ownership through loading, editor operations and
abort/save, prototype publication/cloning, runtime refresh and web/Lua writers;
retain unchanged serialized formats. C serializes allocated-empty as an empty
tilde field, so reload may correctly become NULL again. Presence must not be
inferred solely from text or from a transient descriptor after publication.
A separate design-first repair is proposed; this train does not choose a
representation or change the object writer.

## Locks and other readers

- Parent-error producer: no world/player/manager/lifecycle/editor/save lock held;
  the zone wrapper has returned. The helper's world/source lookup releases its
  read lock before MudLog.
- Unknown-command producer: zoneSaveLock held across snapshot/render/write/dirty
  cleanup; snapshot world lock released before rendering. No lifecycle or editor
  lock. OLC save-all enters the same per-zone lock; acknowledgement precedes it.
- MudLog file/provider lookup locks release before callbacks. EachSession releases
  manager snapshot locks before delivery; body/flags/level/color use existing
  short manager/player reads. Player.SendMessage releases player/world locks
  before MessageSink. Delivery uses existing manager lookups, heartbeat staging,
  snoop and send locks; none acquires zoneSaveLock. No new delivery route.

Other readers: SaveOLCZone keeps the non-broadcast writer; tests cover its actual
multi-kind save route. The file formatter remains shared and diagnostic-free.
Existing redit/oedit/medit/sedit/index saves use the unchanged byte-buffer
wrapper; their callbacks/state and output are unchanged. Existing atomic writer
and OLC concurrency tests plus focused race checks cover the new renderer path.
No source world, save format, identity, transport or oracle change. D7 aggregate
and the two checked-write rows remain blocked.

## Reproduce and retained evidence

From the repository root:

```bash
python3 docs/fidelity/depth/handoff/2026-10-07-dp-1371-zone-seams-controls.py --output "$HOME/Archives/darkpawns/oracle-runs/2026-10-07/zone-seams-reproduction"
python3 docs/fidelity/depth/handoff/2026-10-07-dp-1371-zedit-parent-C.py --output "$HOME/Archives/darkpawns/oracle-runs/2026-10-07/zone-parent-C-reproduction"
python3 docs/fidelity/depth/handoff/2026-10-07-dp-1371-zedit-unknown-C.py --output "$HOME/Archives/darkpawns/oracle-runs/2026-10-07/zone-unknown-C-reproduction"
python3 docs/fidelity/depth/handoff/2026-10-07-dp-1371-oedit-null-C.py --output "$HOME/Archives/darkpawns/oracle-runs/2026-10-07/object-null-C-reproduction"
python3 docs/fidelity/depth/handoff/2026-10-07-dp-1371-oedit-empty-Go.py --output "$HOME/Archives/darkpawns/oracle-runs/2026-10-07/object-empty-Go-reproduction"
```

Controls use Go overlays and require green / compiling named assertion failure /
restored green. Each resolved case has a control; unknown-command controls also
remove disabled filtering, leak admin output and move rendering before open.
Eleven controls in total, including a differing-source-root classification revert. Reference scripts copy the library, retain SHA256 and raw
transcripts, restore disposable files in finally and stop their own process.
The object Go reproduction expects an assertion red; it never modifies the checkout.

Evidence root: `~/Archives/darkpawns/oracle-runs/2026-10-07/dp-1371-mudlog-zone-seams-proofs/`.
Every case records independent build/vet/full-test/game-test/lint results before
commit. Final depth/units/string/inventory, lint, race and control records carry
the final HEAD. One combined census is required on that committed tip; its
verdict is reported separately, not claimed by this handoff before completion.

Earlier per-case evidence retains its original pre-autosquash HEAD. The writer-root repair was folded into the open-parent case to retain one commit per case; final controls, gates and census identify the integrated tip.
