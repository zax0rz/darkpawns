# Depth-fidelity handoff — mortician

Date: 2026-08-29  
Slice: special procedure \`mortician\`  
Registration: mob vnum \`8095\` at \`src/spec_assign.c:301\`  
C definition: \`src/spec_procs3.c:807-856\`  
Code commit: \`65c318c8c\`  
PR: #781, merged as \`f06b190f0\`; hosted run \`33291082724\`

## Queue position

The special-procedure inventory was refreshed through \`mortician\` in
\`src/spec_procs3.c\`. \`mortician\` is claimed here and must not be repicked.
The next active unclaimed special is \`conjured\`, defined at
\`src/spec_procs3.c:859-892\` and registered for mob vnums \`81-86\` at
\`src/spec_assign.c:186-191\`. Continue the remaining special inventory in
source-and-registration order.

## C call path and branch map

The audit followed the player command special dispatch in
\`src/interpreter.c:1407-1456\`, \`SPECIAL(mortician)\` at
\`src/spec_procs3.c:807-856\`, the global \`object_list\` construction in
\`src/db.c:1888-1894\`, object movement in \`src/handler.c:81-104,901-915\`,
and direct tells through \`src/act.comm.c:901-930\`.

Mortician is command-only: commandless autonomous calls return FALSE, while
unknown commands also fall through. \`list\` computes
\`GET_LEVEL(ch) * 116\` and calls C \`do_tell\`, producing an actor-only
direct tell. \`retrieve\` first gates on sufficient gold with the same direct
tell. If funded, C scans the entire newest-first global \`object_list\` for
the first object whose name matches the player's name, whose val[3] is
nonzero, and whose type is \`ITEM_CONTAINER\`. It does not require an
\`IsCorpse\` marker or a corpse keyword.

On a match C moves the object from its current room and prepends it to the
player's room, emits \`The Mortician dumps your corpse on the ground.\` to the
actor, emits \`The Mortician dumps $n's corpse on the ground.\` to other room
players, deducts the fee, and returns TRUE. The object remains eligible for a
later retrieve; the procedure never clears val[3]. With no match it emits the
exact direct no-corpse tell and makes no gold mutation.

## RED → GREEN

On main, the vehicle showed three confirmed divergences: Go emitted a
lowercase \`the Mortician\` direct tell, rejected C's qualifying type/value
object because its internal \`IsCorpse\` bit was false, and consequently
omitted both retrieval room bytes and the fee deduction. The C vehicle also
confirmed that its global scan reaches an object dropped in another room.

The Go-only fix uses the canonical direct-tell and \`Act\` audience primitives,
matches C's type/value/name predicate, orders the global registry by descending
monotonic object ID to mirror C's prepended \`object_list\`, moves to the
player's room with C-style room prepend, and preserves the exact fee/state
branches. No \`src/\` or \`darkpawns-c-oracle/\` files were edited.

## Proof and verification

Scenario: \`cmd/dp-oracle-diff/scenarios/spec-proc-mortician.txt\`  
Focused tests: \`pkg/game/spec_mortician_test.go\`

The registered vnum-8095 vehicle is oracle-green with \`--show-oracle\` at
seeds 1, 2, 3, 5, and 8. The vehicle proves list cost, affordability,
type/value predicate, global cross-room retrieval, actor/peer audience bytes,
gold deduction, and the exact no-corpse branch. Focused tests add command
surface, newest-first ordering, C-style room insertion, and object state.

Required local gates passed:

\`\`\`text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .   # clean
git diff --check
\`\`\`

Hosted \`test\`, \`security\`, and \`lint\` checks all passed in run
\`33291082724\`. The optional \`build-and-push\` and \`deploy\` jobs were skipped
by workflow policy; no CI retry was needed because the checks fired normally.

The manifest now reports:

\`\`\`text
1182 total; 1137 proven/delegated, 13 blocked, 32 excluded
Actionable completion: 1137/1150 = 98.9%
\`\`\`

## Fidelity rulings

This slice follows R1 (exact direct tells and actor/peer bytes), R2 (the
command-only list/retrieve surface), R3 (newest-first object order and fee
state), R4 (no invented corpse marker requirement or output), and R5e
(verified the actual \`special\`, \`do_tell\`, global object-list, object
movement, and \`act\` call paths). R5b/R5c apply to the shared object movement
and audience primitives used here.

## Next action

Return to \`main\`, pull, run the frontier, reread the depth-testing
instructions and this handoff, then map \`SPECIAL(conjured)\) and registrations
\`81-86\` before taking the next unclaimed special slice.
