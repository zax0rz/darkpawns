# DP-1371 D7: complete producer-site inventory and bounded train map

**Tier: proof-only inventory.** No game, harness, oracle, census or governing
rules change. Fresh main `3ccb7de31` (merge #1818). Implements Zach's goal
section “2026-10-06: mudlog-only session trains are local, and bigger D7 trains”.

## What is complete, and what is not

The accompanying [site table](2026-10-06-dp-1371-mudlog-sites.tsv) reconciles
**every C mudlog producer site**, including macro calls, to its enclosing
function, payload source, type, minimum level, file flag, Go owner or missing
owner, implementation state, proof boundary and proposed train. There are
**207 active sites**: 199 direct calls plus eight `whod.c` LOG invocations.
One further row retains the disabled `comm.c:1581` call (`#if 0`), for **208
inventory rows**. These are producer sites, not unique payloads or behavior
cases: Lua load errors and rent/kill payload arms can need several proofs per
site; a shared diagnostic can have several callers.

The raw 204 direct matches reconcile as follows: consumer `utils.c:212`,
commented calls `db.c:2026` and `scripts.c:283`, WHOD macro definition
`whod.c:39`, and disabled call `comm.c:1581` are not active direct producers.
204 − 5 + 8 = 207. A wider `mudlog` scan over `src/*.[ch]` finds no second
wrapper; `utils.h` holds the declaration and consumer commentary. The
`ZONE_ERROR` wrapper calls `log_zone_error`, whose two producers are both
listed, with the six trigger sites separately named in their notes.

**This completes enumeration and ownership grouping, not D7 parity.** The
Go state labels are source-audit labels, not manifest statuses:

- `implemented-boundary` (53): an existing boundary has a named prior owner;
  keep its established proof scope. This does not certify every close path,
  force NPC execution, mob-load actor bytes, or whole handlers.
- `implemented-needs-proof` (57): live script bindings have the diagnostic
  and shared consumer, but this inventory adds no fail-capable contract proof.
- `missing` (96): counterpart lacks the producer, or the C path has no Go
  counterpart. “Missing” does not assert reachable equivalence or authorize a
  log at an approximate path.
- `mismatch` (1): explicit Lua `log` success shares the file=false bridge
  used by diagnostics; C requires file=true. A distinct API needs a stop train.
- `disabled-C` (1): structurally disabled C connection log, not a runtime case.

The earlier [inventory](2026-10-05-dp-1371-mudlog-inventory.md) is the dated
partial audit/history; this table supersedes its family-only enumeration.
`mudlog.unported-sites` remains blocked. No existing depth status changes.

## Reproduce and challenge the enumeration

From the repository root:

```sh
python3 docs/fidelity/depth/handoff/2026-10-06-dp-1371-mudlog-sites-check.py
rg -n 'mudlog\s*\(|#\s*define.*mudlog|\bLOG\(' src --glob '*.[ch]'
```

The read-only handoff checker verifies the complete uncommented direct and
macro invocation set, exact calls, unique sites, required reconciliation
fields, the disabled branch, and the eight-site group bound. It is a source
reconciliation check, not an execution or C parser substitute. Payload
builders and enclosing functions were read from C separately; they are not
inferred as semantics by the checker. Numeric locations cite `src/` (R5g).
A removed table row causes an assertion failure; retained green/remove/restore
check is under the evidence root below. Source-string spaces and punctuation
remain literal in the payload column (not normalized player bytes).

## Train order and tier gates

Each group is at most eight producer sites. Split further if its behavior
proofs pass about eight cases or production diff passes about 500 lines.
One commit per resolved behavior case; per-case compiling revert triples;
per-case other-reader and held-lock audit; one combined census at the tip.
Do not equate a site count with the number of assertions or branches.

| Order / group | Sites | Planned work / prerequisite | Tier |
|---|---:|---|---|
| 1 `local-01` | 8 | file-edit delete/save success; OEDIT/SEDIT defaults; help miss; bug/idea/typo/todo shared report producer; clan withdraw/deposit | Local (mudlog-only), **only after** per-call lock audit |
| 2 `game-milestones` | 8 | level advance; stable collect failure; kender steal; remort; assassin refusal/hire; Medusa; attitude loot | Local candidates; level/persistence and combat-lock callers must be audited; any seam change makes train stop |
| 3 `steal` + `olc-zone-tail` | 5 | steal equipment/inventory/failure; zone ARG3/default diagnostics | Local candidates; C cleanup order and every live caller first |
| 4 `olc-error-seams` | 8 | room/object/mob/shop open/write diagnostics and file-editor delete/write failures | Stop if classified I/O seam or error handling is needed; never equate all AtomicWrite failures to fopen |
| 5 `olc-control-seams` | 8 | saveall; internal room-reset unknown command; room description/default; mob description/default; text-editor invalid action | Stop: missing routing, cleanup/return order and state topology need separate proofs |
| 6 `olc-zone-seams` | 8 | zone write/unknown command; corrupt object extra-description; zone ARG display/parse diagnostics | Stop if shared writer/report signatures or behavior must change |
| 7 `game-diagnostics` | 8 | death-cry NOWHERE; kill milestone; cross-room damage; death-trap caller; hunting; spike kill; protection evil/good | Audit combat callbacks, body identity and world locks; stop if any callback/order seam changes |
| 8 `rent-entry` | 7 | missing file, poverty, rented/crash/cryo/forced/undefined save kind at entry | Stop: actual durable loader and admission boundary; legacy absent metadata remains absent under prior decision |
| 9 `rent-receptionist` | 1 | rent/cryo command producer (two payload arms) | Stop/design: no C receptionist analogue; quit RentOut cannot stand in |
| 10 `admission-admin` | 7 | ban/unban, DNS/ban admission, ident, losing character/no-character close | Stop: helper signatures, acknowledgement ordering, retained-session locks or missing worker |
| 11 `engine-diagnostics` | 8 | object weight, two signal workers, Act target, autowiz, manual spell, paired zone-error diagnostics | Stop: missing state/worker/gate behavior; no indiscriminate diagnostic addition |
| 12 `lua-proof-01` through `08` | 57 | source-ordered existing bridge diagnostics (8 per group, final group 1) | Proof-only if tests expose no repair; use real WorldScriptableAdapter Log and independent observer |
| 13 `lua-seams` | 8 | canget/direction/iscorpse/isfighting/item_check/skip_spaces diagnostic gates | Stop: several legacy or absent registered paths; don't silently add command routing |
| 14 `lua-engine` | 6 | explicit log file flag; raw-kill diagnostic; load and run errors | Stop/design: bridge API, error classification/cache and engine lock audit |
| Separate `whod-lifecycle` | 4 | listener open/close/reopen/request logs | Design-first: no Go daemon; command on/off proof is not daemon proof |
| Separate `oracle-index` | 2 | overlapping scratch-buffer new-zone index errors | Diagnose/propose oracle repair, then stop; memory/index order repair separately |
| Separate `oracle-stable` | 1 | stable buy failure formats msg but logs global buf | Diagnose actual stale-buffer output; never substitute guessed Mount text |

`landed` contains the 53 prior boundaries; `not-runtime` contains only the
explicitly disabled connection log. No dropped source sites. The train map is
implementation scheduling, not an exclusion ledger.

## Lock and proof admission for local trains

The table deliberately does not grant a universal lock-safe verdict. Before
inserting **each** producer, trace its live call path and state exactly which
locks remain held, including callers and automatic paths. Compare against
MudLog's file lock, provider snapshot, manager/player/world reads and delivery,
including switches, heartbeat staging and snooping. “The function has no lock”
is insufficient if its caller holds the world, player or lifecycle lock.

Under the approved exception, production edits must be only MudLog additions
or slog replacements (and necessary imports). No Act/output changes, routing,
function signatures, mutation-order or error handling. Cleanup-dependent
logs must be inserted at the C boundary without moving cleanup. Existing
writing flags affect the actor's eligibility; use an independent observer.
Every PR starts **Local (mudlog-only)** and names why it qualifies. Anything
outside the exception is a stop train. Prepare the following train's C reads,
call paths, test design and lock audit during census; no heavy parallel runs.

## Evidence

`~/Archives/darkpawns/oracle-runs/2026-10-06/dp-1371-mudlog-inventory/`
retains direct source contexts, extraction/reconciliation field notes, the
read-only checker control and separate gates. No census needed for this
inventory: it changes docs and the aggregate's note only, not scenarios,
proof selections, production or harness. No production access or deployment.
