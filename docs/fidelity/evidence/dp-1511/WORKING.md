# DP-1511 working log (unfinished)

Base: origin/main 441cadb40, 2026-11-09. Brief:
`/home/zach/BRIEF-2026-11-09-sol-dp1511-refusal-gate.md`.
Worktree `/home/zach/wt/sol-dp1511`, branch `codex/sol-dp1511`.
No commits/PR yet. No game/source/governing-document edits.

Read AGENTS, RULEBOOK R1-R5j, DEPTH_TESTING, oracle-differential-testing;
read Linear DP-1511, DP-1456, DP-1510 through connector. No delegated agents.

First full census registered (dirty source frozen before later inventory additions):
`/home/zach/Archives/darkpawns/oracle-runs/2026-11-09/sol-dp1511-gate`.
Wait already invoked in exec session 72450; it may still be running. Do not
read running census logs or results. Read summary after completion, then only
necessary result rows. Subsequent inventory changes require another full run.

Raw inventory: inventory.py -> warmup-verbs.tsv: 10696 warmup steps,
223 distinct raw tokens, 71 matching registered C command names. Raw tokens
also include editor text, OLC menu choices and harness controls; matching a
registered command does not mean that occurrence was at a game prompt.
Need add separately counted forced verbs (rent and others) and document dispatch
call paths, no-refusal handlers and silent refusals.

collect.py has explicit source line selections and extracts adjacent C literals,
then generates refusal-sites.tsv and setuprefusal_inventory.go (preserves old
exact inventory). All source selected was read/grepped this session, but final
entries/call paths still need careful audit. Latest generation: 335 additional
exact entries, 88 patterns, 440 source sites. Includes item/equipment helpers,
handler equip rejection, cast/recite generic dispatch, movement/door/follow,
clan, boards/shop special dispatch, rent. Dynamic shop-file messages lack src
literals and need documented handling; don't invent them.

Template compiler in setuprefusal.go supports printf and act substitutions,
anchors whole lines, bounds fields to 1000 runes; numeric fields <=20 digits.
Need consider raising bound to C MAX_STRING_LENGTH (8192, src/structs.h:643)
without exceeding RE2 counted-repeat constraints, and tightening pronoun patterns.
Potential integer scanner format support needs review. Original exact table intact.

setuprefusal_pattern_test.go added; no final validation yet. Existing package tests
passed before adding those tests and later inventory. make fmt passed once, but
rerun after edits. New tests read retained-witness JSON from evidence docs.

Witness collector witnesses.py scans completed 2026-11-05..08 census-dump files,
only files beginning normalized C oracle blocks. It retains path and line for
real C outputs; no synthetic witnesses. Current session 20331 is running an
optimized collector with deduped lines (previous session 30064 stopped by its
own PID 582382). It read the inventory BEFORE latest additions (then 316 exact,
73 patterns); rerun after final inventory. Results: refusal-witnesses.json and
missing-witnesses.json. Need ensure it never consumes Go sections (dump format
usually C only). Tests must cover each new entry with actual C dump and a negative
per pattern; incomplete until missing witnesses resolved.

R5h still TODO: old scenario is `git show 36a8b9b6e^:cmd/dp-oracle-diff/scenarios/cast-create-water-depth.txt`.
Repair commit 36a8b9b6e (#2389). Recreate old MaxExisting1 scenario on throwaway
commit/worktree (run all required gates before committing), run census wrapper,
retain expected red at get chalice, show current version clean at original seeds.

Remaining: audit inventory/class completeness; finish witnesses/tests; full census
triage (intended allowance with specific reason, scenario repair with witness,
demotion, Go divergence demote/report only). Original manifest seeds before/after
repairs. More than ~40 permits batch PR with temporary reasons exactly
pending DP-1511 triage, list every pending. No new divergence pins. Finish full
census clean after triage, run each required gate separately, commit/push/open PR.
Title intermediate `(part of DP-1511)`, final `(DP-1511)`. Report inventory,
pattern count, R5h, census directory, triage table, Go divergences, Decisions for Zach.
Stop for Claude review in Zach's session and Zach merge; never invoke models.
Do not start Go1.27 brief until this lands.
