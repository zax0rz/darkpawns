# D7: interactive OLC memory-commit producers

Stop tier: descriptor-owned editor handlers. Fresh main `07c781aee` (merge
#1814). Five cases, one commit per producer, then integration evidence.
Rules R1, R3b, R5c/e/f/g/h.

## Contracts

All five use CMP, LVL_BUILDER (= LVL_IMMORT, src/olc.h:54), file TRUE;
no invis term and no final punctuation.

| Handler | C producer | Payload | Order |
|---|---|---|---|
| redit_parse | src/redit.c:686 | OLC: %s edits room %d | memory commit, log, cleanup, acknowledgement |
| zedit_parse | src/zedit.c:878 | OLC: %s edits zone info for room %d | acknowledgement, memory commit, log, cleanup |
| oedit_parse | src/oedit.c:1034 | OLC: %s edits obj %d | same |
| medit_parse | src/medit.c:741 | OLC: %s edits mob %d | same |
| sedit_parse | src/sedit.c:838 | OLC: %s edits shop %d | same |

A log belongs to affirmative save confirmation, never discard. It occurs while
PLR_WRITING remains set: the actor must not receive its own producer. zedit
uses the edited room VNUM, not the zone number. Buffered acknowledgements in
zedit/oedit/sedit flush before the producer; medit sends directly and redit's
acknowledgement stays after cleanup. Superseded oedit/medit slog diagnostics
are removed; their error diagnostics remain.

## Fail-capable proofs

Each TestOLCMemoryMudlog{Redit,Zedit,Oedit,Medit,Sedit} uses registered entry
and real editor inputs, with a temporary parsed-world fixture. At file-log
write, a callback checks the changed memory field, PLR_WRITING and the actor's
queued acknowledgement. Qualified complete observer31, complete observer30,
normal observer40 and an invisible40 actor prove filters and actor exclusion.
Discard produces no file/observer output and no committed field. The fixture
never invokes a disk save or writes world files.

All five tests failed on missing producer assertions on the base. The focused
oracle vehicle edits each family and confirms it, using a separate complete
syslog immortal observer. The complete retained C transcript was read: every
confirmation block contains its expected actor acknowledgement, observer log
and victim room cleanup act. sedit creates a new shop in the disposable world;
the other four edit existing records. Five-seed claims are 1,2,3,5,8.

Reproduce compiling triples from the final tip:

```sh
python3 docs/fidelity/depth/handoff/2026-10-06-dp-1371-mudlog-olc-memory-controls.py /path/to/retained-controls
```

Optional third argument chooses a control. Per-case removal, wrong type,
wrong minimum premature writing cleanup, commit-order and acknowledgement-order controls must fail a named test
assertion, never compilation. Outputs retain green/revert/restore separately.

## Other readers and lock acquisitions

All five descriptor input handlers hold textEditMu. Internal commits take the
per-zone save mutex, then existing world locks; those locks release before the
new producer. The producer still holds textEditMu, matching existing room Act
and SendMessage usage in these handlers. It does not hold world, zone-save,
manager, player or lifecycle locks. No new mutexes, unlock/relock windows, registry
or ownership change is introduced.

MudLog completes Alog's file lock before taking the provider snapshot.
Manager.EachSession snapshots under manager RLock and releases before callback;
its switch lookup uses existing manager state. Player flag/level reads release
the player lock before delivery; Player.SendMessage snapshots world MessageSink
under world RLock and releases before the sink. The sink uses existing session,
switch, heartbeat staging, snoop and send routing. It never takes textEditMu;
the prompt pass that does take it runs separately. File-probe tests also read
world snapshots only after the commit locks released. Selected tests run with
-race. No lifecycle lock is added.

Per-case other readers: room/zone/object/mobile/shop runtime and dirty markers
receive exactly their prior commit operations. The inserted log reads only the
actor name and edited VNUM; zedit reads roomVNum. Logging precedes each existing
reservation release and writing cleanup. Admin editors and SaveOLCZone call
shared commit/save APIs rather than these descriptor confirmation handlers,
so gain no C player-facing producer. No durable serializer, loaded-object
ownership, frontend, browser/auth, transport implementation or admin endpoint
changes. Telnet oracle proof covers telnet inputs only; it does not certify
admin editing behavior.

## Retained frontier and evidence

D7 stays blocked. Disk-error, invalid-mode, new-zone, saveall, file-edit and
improved-edit diagnostics still need per-producer reconciliation. WHOD network
lifecycle, NPC force and A1 load anomaly retain their existing boundaries.
No oracle, harness, pin, expected-divergence or governing-doc changes.

Evidence root: ~/Archives/darkpawns/oracle-runs/2026-10-06/ under
`dp-1371-mudlog-olc-memory-proofs`, `dp-1371-mudlog-olc-memory-target` and
`dp-1371-mudlog-olc-memory-combined`. Final committed SHA, gates and census
verdict are recorded in the PR and census manifest. A running census is not
a completed validation claim.
