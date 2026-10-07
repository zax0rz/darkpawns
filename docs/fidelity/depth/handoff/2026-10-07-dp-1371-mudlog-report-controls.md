# DP-1371: report producers and editor-control reachability

Stop tier: this train changes raw report dispatch and the room publication
boundary, beyond the mudlog-only session exception. Eight cases, one commit per
case. R1/R3/R5e/R5g/R5h govern the implementation and evidence.

## Implemented boundaries

1. `do_gen_write`, `src/act.other.c:1080-1135`: all four report commands emit
   CMP/31/file FALSE before opening their file. `src/comm.c:1975-1977` doubles
   typed dollars; the handler undoubles them. Real Go raw dispatch now retains
   whitespace and locally supplies that escaping; tokenized wrappers do the
   same for dollars. These are report-only changes, not a global input rewrite.
2. New-room insertion, `src/redit.c:238-279`: C's command-renumbering switch
   omits valid `L` commands (accepted in `src/db.c:997-999`, created through
   `src/zedit.c:715,749,789-792`). Each omission emits the literal `Unknown comand`
   BRF/31/file TRUE diagnostic after publication and before dirty marking and
   the final room edit producer. Real zedit creates the loop pair in the oracle
   scenario. Replacement and abort remain silent. Go decides insertion under
   the world lock at publication time; an intervening writer is tested.

## Six valid-play exclusions

| C producer | Named proof | Closed path |
|---|---|---|
| improved-edit.c:475 | TestCImprovedEditorDefaultUnreachable | Eight direct literal action callers, all handled; whole-src caller sweep. Unknown slash input is an outer refusal, not this default. |
| redit.c:1057 | TestCReditDefaultUnreachable | All room-editor mode writers are covered parser modes or the extra-description string mode. |
| redit.c:796 | TestCReditDescriptionUnreachable | Installed string editor owns room-description input. |
| redit.c:889 | TestCReditExitDescriptionUnreachable | Installed string editor owns exit-description input. |
| medit.c:1109 | TestCMeditDefaultUnreachable | All mob-editor mode writers are covered literal parser modes. |
| medit.c:933 | TestCMeditDescriptionUnreachable | Installed string editor owns mob-description input. |

These are bounded read-only source-shape audits, not general C parsers or
claims that fabricated descriptor states behave identically. Counterexamples
remove handled states or supply unknown writers; description audits reject
missing installation, reversed input precedence, missing abort cleanup or
missing menu restoration. C source is never edited or instrumented.

`src/olc.c:232-263` calls editor setup before entering CON_REDIT/CON_MEDIT;
setup displays the initial menu. `src/interpreter.c:1729-1737` dispatches these
parsers. `src/comm.c:615-621` gives `string_add` precedence over nanny.
`src/modify.c:162-211` performs synchronous cleanup for save **and abort**;
abort may clear `d->str` before the cleanup callback, but the single-threaded
input loop cannot service another command during that callback. Room cleanup
at `src/redit.c:1065-1078` returns to the appropriate menu; mob cleanup at
`src/medit.c:1123-1126` returns to the main menu. No invalid-state
cleanup/free/tail-access behavior is invented for the mob parser.

Retained whole-source mode-writer and dispatcher/cleanup searches supplement
these bounded tests. New source shapes require re-auditing. Defensive Go parser
logging is untouched. The aggregate D7 row stays blocked; exclusions do not
claim whole-handler parity or resolve the two checked-write sites under #1825.

## Producer locks and delivery audit

- Report producer: no world, lifecycle, manager, editor or save lock held.
- Room insertion producer: caller holds `textEditMu`, then the zone's save
  mutex. The world mutex covers publication and reset-command enumeration,
  then releases before MudLog. No lifecycle lock. Dirty marking follows the
  diagnostic while the zone save mutex still protects the commit/save pair.
- Existing final room-edit producer follows release of the zone save mutex,
  with textEditMu still held. Ordinary admin CommitEditedRoom remains silent.

MudLog's file side obtains/releases the log-writer lookup lock and writes the
existing sink. The session-provider lookup lock releases before callbacks.
Manager.EachSession snapshots under manager RLock, releases it, and takes only
short manager reads for each body and switch resolution. Player flags/level
reads use player locks. Player.SendMessage reads/releases player and world
locks before calling MessageSink. Delivery performs manager lookups, then the
output-batch mutex; unstaged output uses existing snoop and send locks. None of
these paths acquires textEditMu or a zone-save mutex, and these producers never
hold world or lifecycle locks during delivery. The delivery functions are
unchanged; this audit does not introduce a second broadcast route.

## Reproduce proofs

From the repository root:

```bash
python3 docs/fidelity/depth/handoff/2026-10-07-dp-1371-mudlog-controls.py "$HOME/Archives/darkpawns/oracle-runs/2026-10-07/dp-1371-mudlog-controls-reproduction"
python3 docs/fidelity/depth/handoff/2026-10-06-dp-1371-mudlog-sites-check.py
go test ./pkg/session -run '^Test(C.*Unreachable|ReportMudlog.*|ReditInsertionMudlogBoundary)$' -count=1
```

The controls use Go overlays, leaving the checkout unchanged. Sixteen controls
require green / compiling named assertion failure / restored green. Each case
has its own named control; shared-helper mutations may fail more than one proof.

## Retained evidence and census scope

Evidence root:
`~/Archives/darkpawns/oracle-runs/2026-10-07/dp-1371-mudlog-controls-proofs/`.
Contains paired seed-1 C dumps, reproduced missing insertion diagnostics,
per-case checked build/vet/full-test/game-test/lint logs, and controls.
The five-seed claims use standard `scenario@1,2,3,5,8` form.

At Zach's request the combined run started before the final four test-only
cases. `dp-1371-mudlog-report-controls-combined` records HEAD `a1cf72ff6`:
3414 pairs, 1023 duplicates removed, full CLEAN and claims CLEAN, 1974.064 s.
All later changes are tests, proof tooling and documentation; production code
and the scenario corpus are unchanged. Final validation records their own
HEAD separately, rather than claiming the census ran on that later SHA.

## Retained frontier

C's omitted L renumbering can also leave its room-index reference unchanged
when a newly inserted room precedes that reference. Go uses VNUM references.
The current scenario inserts after the loop room; it proves the diagnostic,
not parity of that separate automatic-reset reference behavior. No repins or
oracle instrumentation are used. The checked-write investigation and other
D7 families remain independent work.
