# entry.god-mortal-bootstrap

R1/R3/R5g: src/db.c:2976-2989,3006-3078; src/class.c:501-712; src/constants.c:1124-1146,1187-1209; src/handler.c:559-571,939-951.

The God block never runs do_start and leaves practices zero. The previous Go constructor leaked two mortal sessions. Starting-kit allocation now creates the pack before class objects, and inserts thief picks before bread and water. Since obj_to_obj prepends, the resulting contents are water/bread/picks. C's kit switch gives remort classes the default club, including Magus and Assassin; assassin starting skills remain separately granted by C's following switch.

TestEntryBootstrapMatrix compares exact God/mortal resources, skills, conditions, kit order, body and level draws at five seeds across all six accepted classes. TestEntryBootstrapRemortKits covers every remort class reached by wizard demotion. Six live vehicles passed in dp-1371-entry-bootstrap-final, exposing God resources and empty inventory, and every mortal class's numeric stat report, inventory and pack contents.

R5h triples [0,1,0]: bootstrap-practices, bootstrap-pack, bootstrap-kit, bootstrap-remort. Original bootstrap failures are retained in entry-train-proofs/bootstrap-before; fixed/reverted/restored logs are in the corresponding archive directories. Required gates: bootstrap-gates.

Other readers: GiveStartingItems is also called by guest entry (DP-1379 approved guest overlay) and wizard demotion. Both share the C kit order; remort kits are independently checked. BootstrapFirstPlayerGod's only production caller is accepted character creation; practice counts feed practice, learning, reports and character-data persistence. Creation already persisted the God count as zero, so this repairs the live state without a format change. Inventory order and object IDs feed look, object selection, scripting and later inventory persistence. Shared movement APIs preserve location bookkeeping; no direct location mutation. The unused DoStart return-value helper is not a production reader and is outside this path. No admin authentication, ownership, transport JSON or logging policy changes.

An exploratory God practice probe found the existing class-filtered Go skill list differs from C's all-skill God list and causes different pager state. That report surface is excluded from these vehicles; the practice count is directly state-proven. The negative attempts are retained in entry-train-proofs/bootstrap-oracle; no expected-divergence pin was added.

The God practice-list class-filter divergence is reproduced against the reference oracle on fresh origin/main 9e6a50b2a in dp-1371-entry-bootstrap-practice-main-fixture (both attempts). The baseline also has the now-fixed two-practice leak; that first line differs from the fixed experiment, while the missing all-class skill list and pager mismatch are the same report finding. experimental-god-practice.scenario preserves the negative vehicle outside the proving corpus.
