# Reachable duplicate topology: explicit stop

Zach's 2026-10-02 decision says: no registry redesign; show reachability in real Go play; fix reachable cases narrowly; if a reachable topology cannot be fixed narrowly, stop.

## Same name, different ID is reachable

The retained diagnostic drives real SQLite, normal login/menu entry, and actual cmdSet. No map-key or ID mutation creates the topology:

1. Store Returner (ID 1) and Other (ID 2); log Returner into the game.
2. An implementor runs `set file Returner name Archived`.
3. Run `set file Other name Returner`.
4. SQLite now uniquely holds Archived/ID1 and Returner/ID2. The playing body is still Returner/ID1.

The existing file editor accepts this while the target is online (wiz_set.go cmdSetText). C also permits the offline rename (src/act.wizard.c:2889-2897), and Valid_Name permits an existing CON_PLAYING name (src/ban.c:267-270). C's perform_dupe_check compares IDNUM (src/interpreter.c:1534,1548,1560,1593): ID2 must not usurp ID1.

The diagnostic models the proposed narrow ID guard by declining takeover and proceeding through the actual menu/world-entry path. Manager.Register replaces and removes the existing Returner body by its name (manager.go:1114-1150); World.AddPlayer and the world name map cannot retain two registered bodies under Returner (world.go:489-497,525). The new ID2 enters, and the live ID1 disappears. Therefore merely fixing the comparison is insufficient to preserve C behavior. Supporting the simultaneous bodies and their object/session readers needs identity-aware registration, or an explicitly approved behavior restriction; neither is authorized by the no-registry-redesign decision.

## Same ID, folded registry key is also reachable

A case-only `set file Returner name returner` retains the same SQLite ID but changes the saved spelling. The live session key remains Returner. Normal login canonicalizes from the stored spelling and misses the exact map lookup; it goes to MOTD instead of adoption. This is produced by real command/entry paths, not a manually renamed map key. A local ID lookup can address this individual case, but does not resolve the first blocker.

## World body with no retained session

Not classified. The first reachable case meets the explicit stop condition; the remaining invariant/lifecycle proof and browser/identity cases are not claimed complete or excluded.

## Evidence and saved work

Diagnostic source: topology-reachability.go.fixture. It asserts observed topology and the insufficiency of the narrow guard; it is deliberately outside the green C-proof corpus. Run by copying it to pkg/session/entry_topology_diagnostic_test.go and `go test ./pkg/session -run '^TestEntryDuplicateTopologyReachabilityDiagnostic$' -count=1 -v`, then removing the temporary test. Retained tip log: ~/Archives/darkpawns/oracle-runs/2026-10-02/dp-1371-entry-finish-proofs/topology-reachability/tip.log. Both subcases passed their characterization assertions; this is Go runtime/state evidence, not oracle-green evidence.

Fresh-main train branch sol/p4-entry-finish has three completed commits: DP-1380 comparison accounting, CON_EXDESC editor, and concrete-owner duplicate candidate cleanup. Per-case local gates pass; nine assertion revert triples pass (one password, six editor, two candidate). Editor paired targeted census CLEAN (2/2). Candidate fidelity-units initially hit the known telnet name-hold teardown timing race; three focused repeats and the unit gate repeat passed, with the failure retained. No combined tip census or PR has been run/opened: stopped under the explicit topology decision. No registry redesign or name restriction was implemented.
