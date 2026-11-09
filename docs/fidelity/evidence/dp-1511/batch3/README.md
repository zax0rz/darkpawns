# DP-1511 — batch 3

This batch dispositions 16 of the 35 pending entries: **13 repaired, two
intentional refusals, and one Go disagreement with destination bytes blocked**.
The creation-input class audit additionally repairs `act-messages` and
`afk-depth`. The current pending ledger contains **19** scenarios.

The final shipped 18-scenario target is CLEAN with zero draw divergences. The
repaired three-audience door diagnostic is retained red outside the shipping
corpus, with a blocked manifest row. No game implementation, refusal pattern,
normalizer, governing document, or divergence pin changed.

## Setup repairs and class audits

C `do_load` puts every successfully loaded object directly in the actor's
inventory (`src/act.wizard.c:1368`). Seventeen immediately following room-get
commands in 12 scenarios were therefore refused despite the required item
already being carried. These are redundant operations, not evidence that the
following spell/consumption probe lacked its object. Removing them clears the
gate without inventing a game change. The adjacent direct load/get class audit
covers the whole warmup corpus, dropping from 17 candidates to zero;
`audit-load-get.py` reproduces both TSV inventories from pinned base
`ed3c89f988983ffbd6807da44ad3e6e5e4df60b6` or `--working-tree`.

Appraise's helmet was never spawned and is unrelated to its carried-bread
probe. Remove that get/wear pair; C's real estimates and argument gates remain
witnessed (`src/act.other.c:1784-1817`). The original door close was already
refused because the C zone reset closes it. The final local door vehicle
explicitly locates its actor and observer and exercises real open/close acts.

The door's purported destination peer never entered C: `CON_QSEX` accepts
only M/F, while its setup supplied N (`src/interpreter.c:2012-2026`). The
recognizable-creation-frame audit finds five such paired-engine setup blocks
across three scenarios. `act-messages` has the same non-entering observer;
`afk-depth` has an extra C token, but subsequently recovers at M. Fix all three
fixtures, preserving that distinction. `audit-creation-sex.py` reproduces the
five rejected C-style inputs before and zero after, including whether a later
input recovers the sex prompt. This is a token audit of recognizable creation
frames, not an assertion about every possible saved-player login.

`act-messages` now has three real co-located clients and an explicitly loaded
disposable inert scroll. Its formerly silent observer receives the whisper and
recite messages (`src/act.comm.c:1006-1018`, `src/spell_parser.c:684-688`). AFK's
peer receives the actual enable/return room acts (`src/act.other.c:1283-1300`).

## Intentional ban refusals

Both ban scenarios keep the existing allowance, replacing the pending reason
with the exact asymmetry: C's historical whitehouse.gov entry is removed, while
Go starts empty and intentionally answers `That site is not currently banned.`
C answers `Site unbanned.` (`src/ban.c:213-244`). The retained warmup witnesses
identify each engine separately. The probe then proves the empty ban list and
real add/list/unban behavior. No other warmup commands were added to these two
allowances; they are not evidence of a failed intended setup.

## Actual C outcomes

`triage.tsv` is the scenario → verdict → C witness table for all 18 changed
scenarios. `witnesses.json` contains 81 real C event lines, with dump locations,
line numbers and command/audience blocks, plus four paired C/Go intentional
warmup lines. `collect-witnesses.py` reproduces these from completed CLEAN runs.

The spell outcome allocations in the current five-seed runs differ from older
manifest prose; update that prose rather than attributing an absent event to a
scenario name. Each reported miss and hit is checked against its complete
actor/victim/observer triplet from `lib/misc/messages` (C dispatch:
`src/fight.c:1023-1095`):

| Scroll case | Miss seeds | Hit seeds |
|---|---|---|
| Color Spray | 1 | 2,3,5,8 |
| Dispel Evil | 3,8 | 1,2,5 |
| Dispel Good | 3,8 | 1,2,5 |
| Energy Drain | 1 | 2,3,5,8 |

All show the real recite/dissolution and confirmed Sanctuary/protection flags.
`spell-outcomes.tsv` records every pair. `sleep-outcomes.tsv` records all 20
self-quaff outcomes, verifies sand/bare component output, and distinguishes the
outlaw refusal, successful sleep, and saving throw (`src/magic.c:1199-1248`).
`vampire-outcomes.tsv` confirms real Dracula bites and consumption: day has zero
night-only satiation warnings; sunset and dark have four each at every seed
(`src/act.item.c:982-991,1121-1126`).

The cylinder scenario retains its original pickup probe. My initial edit
accidentally removed it along with the warmup get; that inadequate first green
is retained. Restoring it yields actual talisman pickup and both audiences of
cylinder creation/removal at every claimed seed (`cylinder-outcomes.tsv`,
`src/spec_procs3.c:1070-1144`).

## Go disagreement: destination door blank line

Run `sol-dp1511-b3-witness-repair-seed1` is NOT_CLEAN. Its first divergent block
is `open door south [other]`, after a valid M-sex destination peer is explicitly
teleported across the door. Normalized retained output is:

- C: `The doors are opened from the other side.\n\n\n<PROMPT>\n`
- Go: `The doors are opened from the other side.\n\n<PROMPT>\n`

Close has the same missing newline. Both captures preserve the difference;
draw parity matches (4119 entries). `door-divergence.json` retains the actual
first-capture block values and path. C formats an embedded CRLF and sends it
through act, which appends another (`src/act.movement.c:519-530`,
`src/comm.c:2473-2477`). The Go player lane sends the formatted line directly
(`pkg/game/act_movement.go:721-727`); the NPC lane has the same source pattern
at 1110-1115, a class-audit candidate without a live NPC proof in this batch.
Existing substring-only tests establish notification/recipient filtering, not
exact terminators.

The blocked `door.destination-message-terminator` row records this defect.
`diagnostic-scenarios/object-door-destination-red.txt` preserves the corrected
three-audience source. Shipping `object-door-messages` now explicitly claims
only the actor/origin-room messages, with its own green row. The destination
case is not represented as green. Claude files the Go item; no implementation,
normalization exception, or pin is added here.

## Retained runs and checks

All directories are under
`/home/zach/Archives/darkpawns/oracle-runs/2026-11-09/`; `runs.tsv` records final
summaries. Wrapper manifests retain the exact dirty frozen sources and engine
hashes, and attempts retain both engines' blocks and draw logs when enabled.

| Run | Result |
|---|---|
| `sol-dp1511-b3-before-seed{1,2,3,5,8}` | All 68 selected pairs red at setup refusal (16/13/13/13/13). |
| `sol-dp1511-b3-elements-before-seed{2,3,5,8}` | Four additional original cylinder claims red; its manifest notes claimed these seeds without putting them in the proof field. |
| `sol-dp1511-b3-after-seed{1,2,3,5,8}` | 72 comparisons CLEAN, zero draw divergences. Cylinder pickup and door destination reach were still inadequate as described above. |
| `sol-dp1511-b3-witness-repair-seed{1,2,3,5,8}` | Restored cylinder cycle clean at all five seeds. Seed-1 run also retains the destination-door FAIL; no new pin. |
| `sol-dp1511-b3-sex-class-before` | Two legacy comparisons CLEAN with zero draw divergence; silent observer/rejected input still present. |
| `sol-dp1511-b3-sex-class-after` | Both siblings CLEAN with zero draw divergence and real connected audiences. |
| `sol-dp1511-b3-door-origin-final` | Actor/origin-only door proof CLEAN with zero draw divergence. |
| `sol-dp1511-b3-final` | All 18 shipped scenarios CLEAN, zero draw divergences. Final metadata/annotation edits do not change these engine inputs. |

All original claimed seeds are retained before/after. The known door destination
disagreement is the one new Go finding; it remains outside the green proof.
All eight required gates pass independently: build, vet, all Go tests, lint,
formatting, deadcode ratchet, fidelity depth, expected-divergence checks.

The gate inventory is unchanged: 312 added exact strings and 81 added patterns,
all 393 backed by C witnesses. The original verb inventory remains 10,696 direct
inputs, 223 raw direct tokens plus five force-only names, and 76 command-name
candidates. R5h remains `sol-dp1511-r5h-old-water-retained` red before #2389 and
`sol-dp1511-r5h-current-water` clean with the real fill. The retained full census
is `sol-dp1511-gate-batch1-r2` CLEAN, predating this batch. This PR uses targeted
coverage; Claude runs the batched exact-commit combined release gate.

## Decisions for Zach

Have Claude review this intermediate batch and file the retained door-byte
finding, then merge if accepted. Nineteen scenarios remain pending; DP-1511
stays open. Continue those after this batch lands. No Go 1.27 work has started.
