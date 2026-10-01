# E1 name gate evidence

Fresh main base `e0b89d143` (after #1745). The four pre-fix real-login assertion
failures are retained in the merged design evidence
`../2026-10-01-entry-name-design/`. This batch adds one C name parser at the
entry boundary, then the explicitly approved DP-1378 security-name overlay;
DP-1379 guest-prefix routing remains ahead of the C gate with its capabilities
unchanged. Governing documents, reference oracle and shipped world are untouched.

C authority: src/interpreter.c:1505-1520,1721,1743-1759,853-876;
src/structs.h:647; src/ban.c:257-291. The password state uses the descriptor's
named character (src/interpreter.c:1876-1879), and world entry makes the
descriptor playing (src/interpreter.c:2214), releasing its entry claim.

`TestEntryNameGateMatrix` crosses 29 raw input cases at each of initial JSON,
initial terminal and get_name retry (87 cases): exact close/rejection/confirmation
state and bytes, leading-only whitespace, ASCII letters, 2/20 boundaries,
fill/reserved words, invalid punctuation and repeated prompts. Separate proofs
cover real SQLite lookup order and stored case, invalid substrings and C's
playing-descriptor override, creation/menu ownership and N/close/world release,
concurrent claims, canonical password identity, and real WebSocket/TCP boundaries.
Browser-authored rendering stays owned by entry.browser-name-routing.

`revert_proofs.py` copies the candidate Go sources/tests into isolated
`~/dp-p4-entry-name-proof`, then applies/restores 19 production mutations.
`revert-triples.tsv` records assertion-only 0/1/0 results. The focused boundary
runner records three more 0/1/0 triples: concurrent ownership, the exact reserved
list (including zax0rz, independently of C's syntax rejection), and TCP initial
raw whitespace. Full stage output is retained under
`~/Archives/darkpawns/oracle-runs/2026-10-01/dp-1371-p4-entry-name-complete-proofs/`.
No compile failure or timeout counts as evidence.

The live `entry-name-gates` vehicle sends identical paired creation input,
initially rejecting Fighter123, a_b, x, a 21-letter name, THE and SELF, then
rejecting from/someone after N before accepting Biko. The retained seed-1 C
creation block visibly contains eight rejection prompts, the two confirmations,
N's retry and actual creation/menu/world entry. The scenario parser trims spaces;
raw whitespace claims rely on the unit and real transport proofs above.

Live revert: targeted candidate PASS, bypass parseEntryName in the isolated
`~/dp-p4-entry-name-oracle-proof` checkout, require transcript FAIL with zero
infrastructure failures/timeouts, restore parser, require PASS. The census
wrapper retains all stages. An earlier misconfigured attempt lacked the scenario
file and is retained but rejected as proof. The first full run is also retained
as preliminary, preceding the audit/descriptor-password finalization; only the
final full and claims runs certify the final source.

All census runs use scripts/census.sh. Names and exact summaries are recorded
in census-summaries.txt after completion. All five claimed live seeds are
1,2,3,5,8 in standard vehicle@1,2,3,5,8 form. source-sha256.tsv pins the final
Go files and live scenario. No pins, new divergence ledger entries, dropped
seeds or oracle edits are used.

The existing concurrent/e2e tests used digit/underscore fixture names C rejects;
those are replaced with alphabetic names while retaining their journeys. The
old browser-close test expected an invented goodbye; it now checks C's text-free
orderly empty-name close. Guest commands/channels and unrelated entry matrices
are not claimed complete here.

The eight-worker full-final attempt was explicitly stopped through the runner's
TERM cleanup trap after discovering that the wrapper default is 36 workers.
It remains retained as KILLED and is not evidence of a clean census.
