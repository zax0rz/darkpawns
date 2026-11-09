# DP-1511 refusal gate — first review batch

Historical batch-1 report (merged PR #2421). [Batch 2](batch2/README.md) clears
17 more scenarios. [Batch 3](batch3/README.md) dispositions 16 more plus two
creation-frame siblings; the current `pending-triage.json` contains 19. The original
52-name handoff list is retained in `batch1-pending-triage.json`.

This batch implements the parameterized gate and repairs 13 of the original 65 scenarios
newly rejected by the first full census. The other 52 carry the explicitly
approved temporary reason `pending DP-1511 triage`; they remain work to do after
this batch is reviewed and merged. A clean comparison with that allowance does
not establish their intended setup (R5j). No game code or divergence pin changed.

## Inventory and matcher

`warmup-verbs.tsv` freezes origin/main at
`441cadb40381faedc5719598cfc23570180ebc6a`; `inventory.py` reproduces it from
Git, independent of the later scenario repairs. It lists all 223 distinct raw
direct tokens across 10,696 warmup inputs, plus forced commands (325 force
inputs). Five command names occur only under force, giving 228 tokens in the
union. There are 71 direct tokens matching registered C command names and five
force-only names: 76 command-name candidates. Editor menu choices, editor text,
and harness directives are identified separately; a token matching a registered
name is not proof that every occurrence dispatched that handler.

`handler-audit.tsv` maps those names to C entry points. `refusal-sites.tsv`
retains the source-site inventory read from `src/` this session, including
`perform_*` helpers, `act`, printf, NOPERSON/NOEFFECT, and shop-special dispatch.
`collect.py` contains the explicit reviewed line selections and composes the
shop tell/say wrappers. `excluded-refusals.json` records unreachable,
commented-out, preempted, or out-of-scope branches rather than silently
installing unproved entries. In particular, rent is registered to do_not_here;
the receptionist/cryogenicist implementations are not assigned by this C world.
The interpreter position refusals remain excluded as requested. The internal
look-stars branch is likewise preempted for direct look by its entry position
and has no demonstrated hollow-warmup incident.

The old exact inventory is unchanged. This batch adds **312 exact strings and
81 anchored patterns**. printf fields and act substitutions are nonempty,
bounded by C MAX_STRING_LENGTH (8192), and cannot span lines; numeric fields
and act pronouns are typed. Literal punctuation is escaped. Matching reports
the actual normalized refusal, with its C citation.

`refusal-witnesses.json` contains **393 real normalized C lines**, one for every
new entry, each naming its retained dump and line. Tests enforce completeness,
match each pattern's C witness, and reject a prose suffix for every pattern;
additional tests exercise bounds, literal escaping, empty fields, pronouns,
and a scripted get-chalice refusal. `missing-witnesses.json` is empty.
`witnesses.py` extracts lines from completed C dumps, never fabricated
substitutions. Diagnostic scenario sources are retained under
`diagnostic-scenarios/`, outside the shipping corpus. Both engines' normalized
warmup blocks are now retained before gate rejection/allowance, alongside their
existing per-attempt evidence; tests cover normalization and safe paths.

This is a cited static refusal inventory, not a proof of every conceivable
script-generated sentence, silent failure, or editor interaction. No new
position-rule expansion or governing-document amendment is proposed.

## Retained census evidence

All directories below are under
`/home/zach/Archives/darkpawns/oracle-runs/2026-11-09/`; their manifests retain
exact commits, frozen dirty inputs, oracle hashes, and attempts.

| Run directory | Result and purpose |
|---|---|
| `sol-dp1511-gate` | First full: 1,971 scenarios; 65 FAIL at newly recognized warmup refusals, one alias timeout, two known unstable outcomes. `initial-failures.json` records every refusal. |
| `sol-dp1511-gate-batch1-r2` | Full rerun CLEAN: 1,965 PASS, four EXPECTED, two known unstable, zero FAIL/INFRA/TIMEOUT. No new pins. |
| `sol-dp1511-alias-recheck` | Targeted alias recheck clean with its existing expected verdict; full r2 also resolves the timeout. |
| `sol-dp1511-r5h-old-water` | Throwaway commit `dded8d629`, exact scenario from before #2389: red at warmup step 2 `get chalice`, C `You don't see a chalice here.` |
| `sol-dp1511-r5h-old-water-retained` | Same historical scenario with final evidence callback: red at the same step, both C and Go warmup dumps retained. C witness is `initial-logs/attempt.SdpwaC/cast-create-water-depth.capture1/c-warmup.txt:13`. |
| `sol-dp1511-r5h-current-water` | Current scenario CLEAN; C `A golden chalice is filled.` then `You cannot create water in that!` proves the real fill and rejection. |
| `sol-dp1511-batch1-before-seed{2,3,5,8}` | Three affected multiseed scenarios each red at every original extra seed. Seed 1 before evidence is the initial full run. |
| `sol-dp1511-batch1-after-seed{2,3,5,8}` | All three repaired scenarios clean at all four extra seeds. |
| `sol-dp1511-batch1-after-seed1` | Twelve pass; robe case exposed a second refused setup field rather than being accepted as green. |
| `sol-dp1511-batch1-robe-repair` | Corrected robe setup CLEAN. |
| `sol-dp1511-batch1-starter-cleanup` | Final inv-equip/equipment-glance cleanup CLEAN, two scenarios. |

The clean full snapshot predates the final restoration of valid carried-starter
`junk tunic`/`junk sword` steps in two repaired scenarios. Those exact final
scenarios have the last targeted clean run above. C starts these items in
inventory (`src/class.c:524,530`), so only the refused remove steps should go.
The only subsequent matcher change removed the unreachable direct look-stars
entry. This is not an exact-final-commit release gate: Claude runs the batched
combined release census under the repository process.

## Main integration

Main advanced to `0b058ea29` during this work. Its merge is included; the two
manifest overlaps preserve the incoming evidence and this batch's distinct
blocked cases. No game-code delta remains relative to main.
`sol-dp1511-main-integration` checks all 18 incoming scenarios: 17 pass, and
`carry-limit-gates-depth` newly fails at the third give to the peer. C starts
that warrior with a sword, tunic, and backpack (`src/class.c:521-534`); two new
swords reach the five-item cap. Three excess warmup gives were refused.
The repair removes those three load/give pairs. Before/after runs at original
seeds 1,2,3,5,8 are `sol-dp1511-integration-before-seed{2,3,5,8}` (seed 1 is
main-integration) and `sol-dp1511-integration-after-seed{1,2,3,5,8}`. All before
runs are red, all after runs clean. The C probe says `Your arms are already
full!` and `Fullpeer seems to have his hands full.` This is one additional
repaired scenario beyond the original 13; no pending allowance was added.
The fresh-mortal give claim remains green; the live discrepancy after first-God
level demotion has its own blocked boundary row. The clean full snapshot above
predates this integration; incoming scenarios and the repair have targeted
coverage, not a claimed exact-final full/release result.

## Triage

`triage.tsv` supplies the required scenario → verdict → evidence table for the original
65 failures plus the integration failure. `batch1-seeds.json` records original manifest seeds;
`batch1-witnesses.tsv` gives verbatim C witness lines and dump locations for all
14 repairs (13 original plus one integration). The repairs remove impossible removal of unworn starting gear,
keep valid junk of carried gear, and remove an unsupported set-vampire field
from the robe wear/remove scenario. The latter's claims now describe the actual
robe bit cycle; real vampire transformation coverage remains in its existing
separate vehicle. The thief claim has real loaded bread and seed-2 C inventory
`Nothing.` after the theft; seed 1 correctly need not steal.

The **52 names in `batch1-pending-triage.json`** are the complete temporary allowance
list. No previous opt-out was widened. They must each receive a real intended,
repaired, demoted, or divergent verdict in later batches, and the final batch
must leave zero pending reasons.

## Go differences and validation

`divergences.md` reports every live finding and its first divergent block,
including give weight/capacity, fill-word doors, switched follow/master checks,
wield ordering, clan grammar, generic write, switched-original-body teleport,
and an unresolved alias/editor sequence at a full board. Affected unit-green
claims were demoted and new uncovered cases marked blocked. `divergent-blocks.tsv`
indexes all observed differing blocks, including downstream effects. The report
also separates the multi-index fixture-ordering artifact and out-of-range
shop input. Claude files the live defects; this batch does not fix them.

All eight required gates passed independently: `go build ./...`, `go vet ./...`,
`go test ./...`, `golangci-lint run ./...`, `make fmt`, `make deadcode-ratchet`,
`make fidelity-depth`, and `make expected-divergences-check`. The last gate
retains the existing unresolved-proof baseline; no new pin was introduced.

## Decisions for Zach

1. Have Claude review this first batch in your session, then merge if accepted.
   This is an intermediate PR `(part of DP-1511)`, not the issue-closing PR.
2. Have Claude file the retained live Go findings after assessing first-block
   evidence, and track the separate multi-index fixture insertion defect and
   its five-scenario class audit.
3. After this batch lands, continue the 52 pending scenarios in later batches.
   DP-1511 remains open; the Go 1.27 brief has not started.
