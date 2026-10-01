# DP-1371 E1: name gate implementation

Fresh main base `e0b89d143` after the approved #1745 design. One session entry
parser now follows C's leading-only whitespace, alphabetic 2–20-byte names and
exact fill/reserved words before saved lookup. Initial terminal/JSON and
get_name retries share that gate. Empty/all-space input closes without text.
Password entry keeps the already selected descriptor's canonical identity.

C sites read for this batch: src/interpreter.c:1505-1520,1721,1743-1759,
853-876,1876-1879; src/structs.h:647; src/ban.c:257-291. Named non-playing
entry descriptors hold a folded name until N, close or successful world entry;
a playing descriptor permits reconnect and wins before invalid_list as C does.
The C transition to playing is src/interpreter.c:2214. The existing invalid-name
audit event is preserved. R1/R4/R5e/R5f/R5g/R5h govern the change.

The original entry.name-validation row becomes unit-green with its explicit
post-line-decoding gate boundary. Separate rows own live creation, stored
lookup order, descriptor ownership, banned substring/playing override, world
handoff, concurrent claims, canonical password identity and real WebSocket/TCP
boundaries. The live claim uses entry-name-gates@1,2,3,5,8. None of this promotes
the independent browser-rendering, deleted, load failure, password matrix or
login-restrictions rows.

DP-1378 owns entry.security-reserved-names as divergent-approved. The exact
existing list is retained and separately tested, including redundant zax0rz
(which C's alphabetic gate also rejects). DP-1379 owns entry.guest-prefix-login
as divergent-approved. Guest prefix routing, generated/supplied names, initial
state, persistence bypass and capabilities remain unchanged; command/channel
scope belongs to Zach's separate DP-1379 review.

Proofs: 87 input/route matrix cases; real SQLite and WebSocket/TCP assertions;
20 broad unit mutations plus three focused boundary mutations all pass 0/1/0.
The live parser bypass gives a content FAIL, not a timeout/build failure, then
restores to PASS. Retained scripts, tables and final source hashes are in
../evidence/2026-10-01-entry-name/. Complete logs/manifests/captures are under
~/Archives/darkpawns/oracle-runs/2026-10-01/dp-1371-p4-entry-name-*.

Existing concurrent/e2e fixtures used digit/underscore names C rejects; those
fixtures now use alphabetic identities without weakening their assertions.
The browser empty-name close test now rejects the invented goodbye and still
requires an orderly close. No shipped save/world/oracle/tooling or governing
document changes.

Proof limit and proposed follow-up: these transport tests cover printable ASCII
names and space handling. C's upstream process_input also edits backspaces,
filters non-ASCII/nonprintable bytes and doubles dollar signs
(src/comm.c:1965-1980). Go's telnet readLine has its own upstream byte handling;
that whole input-processing class needs a separate audit/differential vehicle.
The unit non-ASCII case certifies the nanny parser on a supplied argument, not
that upstream transport pipeline. No upstream filtering repair or parity claim
is included here.

Final validation and census summaries are retained beside the evidence. This
is an entry-code PR: open it and stop for Zach. The next independent E1 batch
is deleted-name handling after this merge.

Guest handoff also releases the descriptor's prior name reservation. Its new assertion failed before the repair, passes after it, and has a dedicated revert triple; guest capabilities remain unchanged. Real transport close assertions require actual orderly closure rather than accepting a read timeout. The earlier full runs and interrupted claims run are preliminary; the handoff-full and handoff-claims runs certify the final source.
