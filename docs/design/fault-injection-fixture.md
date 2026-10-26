# Fault-Injection Fixture — Design (harness, stop tier)

**Status:** design only — nothing in this document is built. **Not for implementation without Zach's approval.**
**Provenance:** Luna's triage in PR #2167 found 9 player-facing output units reachable only when a file is unreadable or corrupt. This design specifies opt-in `[fixture]` verbs that make both engines see the same fault, per unit, with the R5h control and the cost of each.
**Scope rule (R5g):** every C citation below was read in `src/` while writing this document. Every Go line number was read in this repository.

---

## 1. What the harness already gives us (the seam)

The oracle harness already builds **disposable, per-engine trees** under a scratch root (`cmd/dp-oracle-diff/main.go:305-325`):

- **C oracle**: `oracleData = <tmp>/oracle-lib`, a copy of the reference oracle's `lib/`, passed as `-d` (`main.go:473-476`). The C server `chdir`s into its `-d` directory, so every C-relative path — `etc/plrmail` (`src/db.h:74`), `etc/board.*` (`src/boards.c:92-102`), `object_idlist` (`src/act.wizard.c:3582`) — resolves inside the disposable copy.
- **Go port**: `goWorld = <tmp>/go-world`, a copy of `lib/world`; the server anchors its CWD to the world dir's parent (`cmd/server/main.go:375-395`, DP-1193), so Go-relative paths resolve under `<tmp>/` — `data/mail` (`pkg/game/mail.go:26`), `object_idlist` — while board files live inside the world copy itself (`cmd/server/main.go:663` passes `*worldDir` to `InitBoards`, whose `BasePath` prefixes `etc/board.*`, `pkg/boards/boards.go:129-137,155`).
- **Symmetric planting precedent**: `applyMiscFileFixtures` already writes the same planted file into *both* trees (`main.go:316-321`). A fault verb is the same loop with hostile content or metadata instead of a text file.
- **Mid-scenario precedent**: probe steps beginning with `~` are control directives the driver interprets rather than player input (`internal/oraclediff/scenario.go:19`, `PumpPulses`). A `~fault` step slots into the same interception point.

**Consequence: no C oracle change is required for any unit covered by this design.** The only world the oracle reads is its own disposable copy; the harness corrupts that copy the same way it corrupts Go's. The one unit that cannot be made symmetric today is a Go-side gap, not an oracle gap (§5, objsave).

## 2. Verb surface (all opt-in)

All verbs live in the `[fixture]` section (applied after tree copy, before boot) or as `~fault` steps (applied between probe steps, to both trees at the same step boundary, never sent to either engine). A scenario with no fault verb changes nothing.

| Verb | Effect |
|---|---|
| `fault-dir <logical-path>` | Replace the path with a directory in **both** trees. `fopen(..., "r"/"r+b"/"w")` and `os.Open`/`os.Create` all fail (`EISDIR`), deterministically, regardless of the runner's uid (chmod-000 depends on not being root). |
| `fault-corrupt-mail unaligned` | Plant a 1-byte mail file in both trees: the block scan ends off `BLOCK_SIZE`, failing the alignment check at boot. |
| `fault-corrupt-mail bad-header` | Plant a full-size block whose `block_type` is neither HEADER nor DELETED: the *next* `receive` reads it, fails the header check, and disables mail mid-session. |
| `fault-board <vnum> <malformation>` | Write a structurally valid board file (C layout: `int` count, then per message `board_msginfo{slot_num, level, heading_len, message_len}` + heading bytes + message bytes — `src/boards.c:495-530`, `pkg/boards/boards.go:190-221`) with one malformation: `empty-message-valid-slot` (message_len 0, slot in range), `empty-message-huge-slot` (message_len 0, slot `99999`), or `nul-heading` (heading_len 1, single NUL). |
| `~fault <same verbs>` | Re-run a verb's filesystem mutation between probe steps. |

A small **path map** in the harness resolves one logical name to both physical paths (mail: `oracle-lib/etc/plrmail` ↔ `go-../data/mail`; boards: `oracle-lib/etc/board.*` ↔ `go-world/etc/board.*`; idlist: `oracle-lib/object_idlist` ↔ `go-../object_idlist`; crash file: `oracle-lib/plrobjs/<name>.D` ↔ — see §5). The map is the only maintenance surface: a unit test walks it and fails when one side stops resolving (symmetry checker).

## 3. Unit-by-unit

### 3.1 mail — the postmaster diagnostic (1 unit)

**C path (R5g).** The postmaster prints `"Sorry, the mail system is having technical difficulties.\r\n"` whenever `no_mail` is set (`src/mail.c:484-486`). `no_mail` is set from three failure sources: a block write at an invalid position or an `fopen("r+b")` failure (`src/mail.c:136-144`), the same pair on read (`src/mail.c:169-177`), and a non-HEADER block read where a header was expected (`src/mail.c:423-427`). At boot, an unaligned or unreadable mail store fails `scan_file` and `db.c:365-370` sets `no_mail = 1` before any session opens.

**Fixture.** Two scenarios:
- *Boot fault:* `fault-corrupt-mail unaligned` + enter the game + walk to the postmaster + `check` → the diagnostic is the first mail byte the player sees.
- *Mid-session fault:* `~fault fault-dir mail` after a successful `check`, then `receive` (drives `read_mail` → `fopen` failure → `no_mail`), then `check` again → the diagnostic.

**Go branch.** `pkg/game/postmaster.go:19-21` gates the same three commands on `mailDisabled`; boot failure of the `scanFile` port sets it (`pkg/game/mail.go:116-124`), and the runtime open-failure twin is `openMailFile` (`pkg/game/mail.go:175`). Go's mail store is a real block file at `data/mail`, so both fixtures are symmetric.

**R5h control.** Revert the `mailDisabled` gate in `postmaster.go` (or make `InitMailSystem` ignore `scanFile`'s verdict) → the diagnostic disappears from the transcript → scenario red.

**Cost.** Two census rows; negligible runtime; the mail file is engine-private scratch.

### 3.2 `idlist` — wizard report open failure (1 unit)

**C path (R5g).** `do_idlist` opens `object_idlist` in the process CWD with mode `"w"` (`src/act.wizard.c:3575-3589`); `fopen` returning NULL prints `"Could not open id list file, cannot complete operation!\r\n"` (`:3585-3588`). Registered `"idlist"`, `LVL_GRGOD` (`src/interpreter.c:512`).

**Fixture.** `fault-dir object_idlist` + a GRGOD character runs `idlist`.

**Go branch.** `os.Create("object_idlist")` with the identical CWD-relative semantics and the identical refusal line (`pkg/session/wiz_info.go:458-463`); command registered at `pkg/session/commands.go:319`.

**R5h control.** Point the Go `os.Create` at a temp name (simulating a lost branch) → line missing → red.

**Cost.** One census row; needs a GRGOD character in the fixture (the standard god-creation flow already used by god scenarios).

### 3.3 `Crash_load` — non-ENOENT open failure (1 unit) — **the asymmetric one**

**C path (R5g).** `Crash_load` opens the player's crash file `"r+b"`; failure with `errno != ENOENT` prints the player-facing NOTICE block ("There was a problem loading your objects from disk. Contact a God for assistance.") and the "entering game with no equipment" mudlog (`src/objsave.c:477-491`).

**Go branch — gap.** Go stores object saves in SQLite (`pkg/session/crash_load.go:57-72`) and has **no port of the NOTICE block at all**: `snapshot == nil` maps only to C's ordinary ENOENT arm (`crash_load.go:74-90`), and the file's own comment records the corrupt-cost arm as unreachable. There is no file whose unreadability can drive a Go twin, because the twin does not exist; and a SQLite-file fault fails Go's boot outright, which is a different transcript, not this branch.

**Recommendation.** Sequence, don't fake: (1) port the NOTICE onto a distinct Go error arm of the object-save reader (code change, its own PR, R1-verbatim bytes); (2) give that arm a harness seam — the cleanest is a `~fault fault-dir object-save` verb that makes the per-player save row unreadable through an existing admin/test seam, or an `EXPECTED` divergence row in the interim. Until (1) lands, a C-side-only scenario (`fault-dir` on the C crash file) would go green-by-mismatch and must not be claimed. **Decision required — §7.2.**

**R5h control (once ported).** Break the Go error-arm detection → NOTICE missing → red.

**Cost.** Highest of the nine: one port PR plus one fixture; census row only after both.

### 3.4 boards — six guards on malformed persisted state

**C paths (R5g), with a reachability correction to the triage.** The loader (`src/boards.c:495-530`) validates count and heading length, resetting the board on `heading_len == 0` (`:503-507`) — but it loads each row's `slot_num` raw (`:502`) and only overwrites it with a fresh `find_slot()` when `message_len > 0` (`:516-519`). Therefore:

| # | Guard (C cite) | Reachable from a crafted file? |
|---|---|---|
| 1 | `read`: slot out of range → "Sorry, the board is not working." (`src/boards.c:347-351`) | **Yes** — `empty-message-huge-slot` keeps slot 99999 (`:502` never overwritten) |
| 2 | `remove`: slot out of range → "That message is majorly screwed up." (`src/boards.c:412-415`) | **Yes** — same file, `remove 1` |
| 3 | `read`: null storage → "That message seems to be empty." (`src/boards.c:357-359`) | **Yes** — `empty-message-valid-slot`: slot in range, storage never assigned |
| 4 | `show`: null heading → "Sorry, the board isn't working." (`src/boards.c:307-311`) | **No** — the loader resets any file that would null a heading (`:503-507`); this guards in-memory decay only |
| 5 | `read`: null heading → "That message appears to be screwed up." (`src/boards.c:353-356`) | **No** — same reason |
| 6 | `remove`: null heading → same string (`src/boards.c:393-396`) | **No** — same reason |

**Correction (R5e):** units 4-6 cannot be driven in C by *any* loadable board file — the C loader's own validation stands ahead of them. Driving them would require an oracle-side seam (e.g. a debug env var nulling a heading after load), which this design rules out. They stay covered by Go-side unit tests on the twin branches (`pkg/boards/boards.go:475,531-532,585`) plus this documented reachability verdict.

**Go branches.** `pkg/boards/boards.go:524-528` (slot range), `:531-532` (heading), `:535-536` (empty storage), `:585`, `:602-603` (remove pair), `:475` (show).

**Predicted divergences the fixture will pin (not invent — `EXPECTED` rows until fixed):**
- `empty-message-valid-slot`: C keeps the file's slot → unit 3 fires ("seems to be empty"); Go's loader forces `SlotNum = -1` for `message_len == 0` rows (`pkg/boards/boards.go:197,222` — the DP-1242 hardening) → Go fires unit 1's line instead. Two different diagnostics for one file.
- `nul-heading`: C stores a non-null pointer to `"\0"` and happily lists an empty heading (`src/boards.c:305-306` — the null check is on the pointer, not the content); Go strips the NUL into `""` and treats it as corruption (`pkg/boards/boards.go:194,531-532`).

**R5h control.** Delete Go's slot-range check → `empty-message-huge-slot` scenario loses its line → red (same shape for the storage check with `empty-message-valid-slot` *after* the divergence is resolved).

**Cost.** Three crafted files shared by read/remove probes; the `board_msginfo` layout is fixed by both loaders, so the fixture writer is one struct definition. Two census rows (plus the EXPECTED rows' separate ledger handling).

## 4. Verification plan (once built — nothing here is run today)

- Each fault scenario is added under `cmd/dp-oracle-diff/scenarios/fault-*` and run via `scripts/census.sh start --scenarios …`; deterministic bytes, no RNG sensitivity beyond the ordinary.
- Fault scenarios stay **opt-in and out of the default corpus** until every unit they exercise is either green or an explicitly-ledgered `EXPECTED` row; joining the claimed corpus is Zach's call (§7.4).
- The R5h controls in §3 are executed as revert checks at build time, same as any fidelity test.

## 5. Oracle impact

**None.** No unit above needs the reference binary touched: every C fault is a property of files inside the harness's disposable `oracle-lib` copy, applied before boot or between steps by the driver. The only casualty of not touching the oracle is C-side coverage of the three in-memory board guards (§3.4 units 4-6), which is declined deliberately.

## 6. Cost summary

| Unit | Fixture | New scenarios | Extra risk |
|---|---|---|---|
| mail diagnostic | corrupt/dir verbs | 2 | none |
| idlist refusal | `fault-dir` | 1 | GRGOD character needed |
| Crash_load NOTICE | **port first**, then seam | 1 (deferred) | Go storage is SQLite — no file twin |
| boards #1, #2 | `empty-message-huge-slot` | 1 | none |
| boards #3 | `empty-message-valid-slot` | shared | **predicted divergence** (loader slot policy) |
| boards #4-6 | none possible | 0 | corrected unreachable; unit tests only |

## 7. Decisions for Zach

1. **Reachability corrections to Luna's triage:** the three null-heading board guards (`src/boards.c:309,354,394` region) are unreachable from any loadable board file in C — the loader resets first (`boards.c:503-507`). I propose recording them as unit-test-only twins rather than fixture targets. Approve or overrule.
2. **`Crash_load` NOTICE:** the branch is not ported to Go at all (no twin of `objsave.c:483-487` exists; storage is SQLite). Options: (a) port the NOTICE + add a storage-error seam, then pin with a fixture; (b) ship the C-side scenario as a permanent `EXPECTED` divergence; (c) drop the unit. I recommend (a) as its own PR.
3. **Predicted board divergences:** the slot-policy and NUL-heading divergences in §3.4 are real fidelity findings this fixture surfaces. Pin them as `EXPECTED` rows with ledger entries now, fix in follow-up PRs, then flip to green claims?
4. **Corpus membership:** do `fault-*` scenarios join the default census corpus once green, or stay a permanently separate targeted set? Separate keeps the full census runtime flat; joining prevents bit-rot.
5. **Verb naming** (`fault-dir`, `fault-corrupt-mail …`, `fault-board …`, `~fault`) and the directory-over-chmod-000 default are my calls to overrule — chmod reads clearer in scenario files but is uid-sensitive on hosts that run as root.
