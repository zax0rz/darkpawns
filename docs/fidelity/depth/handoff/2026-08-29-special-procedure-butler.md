# Depth-fidelity handoff — butler

Date: 2026-08-29  
Slice: special procedure \`butler\`  
Registration: mob vnum \`8092\` at \`src/spec_assign.c:300\`  
C definition: \`src/spec_procs3.c:144-196\`  
Code commit: \`5d021f463\`  
PR: #780, merged as \`4f2c248e5\`; hosted run \`33290472759\`

## Queue position

The special-procedure inventory was refreshed through \`butler\` in
\`src/spec_procs3.c\`. \`butler\` is claimed here and must not be repicked. The
next active unclaimed special is \`mortician\`, defined at
\`src/spec_procs3.c:807-856\` and registered for mob vnum \`8095\` at
\`src/spec_assign.c:301\`. The remaining special inventory must continue in
source-and-registration order.

## C call path and branch map

The audit followed \`mobile_activity()\` at \`src/mobact.c:68-93\`, the registered
special dispatch in \`src/interpreter.c:1407-1456\`, and \`SPECIAL(butler)\` at
\`src/spec_procs3.c:144-196\`. The authored vnum-8092 servant was isolated in a
scriptless disposable room with the three required containers.

The commandless autonomous path rejects a nonempty command, a sleeping mob, or
a fighting mob. It resolves the visible \`case\`, \`cabinet\`, and \`chest\` with
\`get_obj_in_list_vis\`; any missing required container returns FALSE without
output. Each room object is eligible only when \`CAN_GET_OBJ\` passes the take
bit, visibility, carry-weight, and carry-count gates. The C loop processes at
most four objects, increments \`got\` before the put attempt, and routes armor
and worn objects to the case, weapons and fireweapons to the cabinet, and all
other types to the chest.

Each accepted object emits the C room \`get\` Act, moves through the mob
inventory, opens the selected closeable container, and invokes
\`perform_put\`. The put check uses the container's base weight plus the
object's weight against \`GET_OBJ_VAL(cont, 0)\`; a failed put leaves the object
in mob inventory but still counts toward \`got\`. Any nonzero \`got\` closes all
three closeable containers. The Go room-list snapshot reverses the append
order so the observed scan matches C's prepended \`obj_to_room\` list. The
relevant C helpers were checked in \`src/utils.h:543-549\`,
\`src/handler.c:901-915,1328-1355\`, \`src/act.item.c:53-65\`, and
\`src/act.movement.c:477-515\`.

## RED → GREEN

On main, the first vehicle exposed a final container-state divergence. After
the collection implementation was exercised with a peer present, the
confirmed RED also showed Go scanning the room in the opposite visible order,
using lower-fidelity get output, omitting the put and close room Acts, and
leaving the close-state bytes divergent.

The Go-only fix uses the canonical \`Act\` path for get, put, open, and close;
matches C's visibility and carry predicates; preserves the four-item and
capacity-failure state transitions; and reverses the copied room slice to
account for the actual Go append versus C prepend room-list ordering. No
\`src/\` or \`darkpawns-c-oracle/\` files were edited.

## Proof and verification

Scenario: \`cmd/dp-oracle-diff/scenarios/spec-proc-butler.txt\`  
Focused tests: \`pkg/game/spec_butler_test.go\`

The registered vnum-8092 vehicle is oracle-green with \`--show-oracle\` at
seeds 1, 2, 3, 5, and 8. The transcript proves the four get/put routes, exact
room audience bytes, and final close Acts; focused tests cover the entry and
container gates, all \`CAN_GET_OBJ\` predicates, four-item cap, type routing,
capacity failure, transfer state, conditional door Acts, and autonomous
registered dispatch.

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
\`33290472759\`. The optional \`build-and-push\` and \`deploy\` jobs were skipped by
workflow policy; no CI retry was needed because the checks fired normally.

The manifest now reports:

\`\`\`text
1173 total; 1128 proven/delegated, 13 blocked, 32 excluded
Actionable completion: 1128/1141 = 98.9%
\`\`\`

## Fidelity rulings

This slice follows R1 (exact room Acts and audience bytes), R2 (registered
commandless special dispatch), R3 (deterministic ordering and state), R4 (no
invented container or failure output), and R5e (verified the actual
\`mobile_activity\`, \`special\`, \`CAN_GET_OBJ\`, \`perform_put\`, door, and object
movement call paths). R5b/R5c apply to the shared movement and Act primitives
and were kept within their confirmed contracts.

## Next action

Return to \`main\`, pull, run the frontier, reread the depth-testing
instructions and this handoff, then map \`SPECIAL(mortician)\` and its vnum-8095
registration before taking the next unclaimed slice.
