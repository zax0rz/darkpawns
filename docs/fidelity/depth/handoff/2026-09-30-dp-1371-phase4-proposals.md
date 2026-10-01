# DP-1371 Phase 4a: proposals for the 72 blocked cases

Survey date: 2026-09-30. Base: `origin/main` at `cb16689a1d6a9f270a17d6ab33602ba27a816d19` (after #1727, #1728 and #1729). Scope: ADDENDUM D, survey and proposals only. No game, harness, scenarios, manifests, reference binary or governing documents change. All 72 rows remain blocked. No new oracle or unit-green claim is made by this survey.

## Inventory and evidence boundary

The inventory is derived with the shared loader, rather than a text match:

```python
import sys
sys.path.insert(0, "scripts")
from fidelity_manifest import load_rows
blocked = [row for row in load_rows() if row["status"] == "blocked"]
assert len(blocked) == 72
```

Every row's C site was read, then followed to the relevant caller/helper where the manifest summary was insufficient (R5e/R5g). The appendix records current source locations, correcting shifted citations in this proposal only. C citations refer to this base's read-only `src/`; `lib/misc/socials` supplies social data. Go references identify the current implementation, not proof of parity. Prior failed probes are **historical manifest observations**, not fresh executions. No census is needed for this docs-only PR. Future code and proof PRs must retain their evidence through `scripts/census.sh`, use `wait`, and obey R1–R5, including fail-capable proofs (R5h).

Sizes are relative planning estimates: S is a focused proof or local fix; M is a multi-branch feature or harness change; L crosses dispatch, persistence or runtime boundaries. They are not calendar commitments. Each subgroup below is a separate proposed PR; shared root causes take precedence over manifest ownership.

## Corrections to the provisional grouping

- Existing `keep-prompts` invokes `NormalizeKeepPrompts`, which returns captured text unchanged. ANSI, whitespace and line endings are retained; telnet IAC handling still occurs at the transport. `[creation:*]` enables setup comparison, `<RELOGIN>` captures reconnect setup, and `[peer-drop]` disconnects a passive peer after warmup. Thus raw text and descriptor-less players are not wholly absent harness capabilities (`internal/oraclediff/scenario.go`, `normalize.go`, `relogin.go`; `cmd/dp-oracle-diff/main.go`). Start with these controls before adding one narrow missing capture/fixture feature.
- The two prompt rows and sleeping gsay naming are production byte gaps, not reasons to wait for a new harness. The idle rows likewise record two concrete production divergences.
- Shoot has a partial implementation, but its simplified roll, damage, waits and audience path differ from C. It belongs with feature repairs. C's entire `do_shoot` has **no WAIT_STATE**: the wait row's note must eventually describe proving no added wait, not porting an imagined one. C directly updates HP and calls death handling; it does not call `damage`/`skill_message`. NPC relocation/retaliation must not be applied to player victims.
- Special ordering now exists in `pkg/session/commands.go:719-729,826-950` with focused tests in `special_order_test.go` and `room_obj_script_gates_test.go`. The DP-1336 pre-resolution note is stale. This is a proof-completion candidate, not a new architecture project.
- Shop parsing and ordered snapshots now exist (`pkg/parser/shop.go`, `pkg/game/world_sedit.go`); `cmdShow` still returns without the shop report. Port the formatter/pager over existing records, rather than re-porting a parser.
- Forced rent has `RentOut`, object filtering and persistence seams. C's configured `free_rent = YES` selects that path (`config.c:106`; `limits.c:443-447`). The two roundtrip rows do not demand an entire receptionist implementation. `zreset` already reaches a reset engine, and set-skill already maps a spell number to the player skill map; both need state/draw proofs first.
- Object-magic sleep is reachable. `mag_objectmagic` passes the drinker as victim (`spell_parser.c:699-718`), then `call_magic:406-505` reaches `mag_affects`. `TAR_NOT_SELF` is checked by **cast_spell** at :886, not this item path. An outlaw drinker can pass the PC outlaw gate; a non-outlaw drinker reaches its refusal. Do not exclude this row or substitute the cast-sleep proof.
- `lua.bind-extchar-deferred` was missing from the provisional C list and is included here. No shipped caller is not evidence that a public binding is unreachable: paired disposable Lua fixtures already exist.

## Root-cause accounting

| Subgroup | Root cause | Rows |
|---|---|---:|
| A1 | Unresolved oracle/runtime anomalies | 5 |
| A2 | Confirmed overlapping C report writes | 1 |
| B1 | Raw entry capture coverage | 4 |
| B2 | Disposable report-file fixture | 1 |
| C1 | Generic NPC command dispatch | 3 |
| C2 | Shared Lua state and nested writeback | 2 |
| C3 | Character lifecycle and mobile riders | 2 |
| D1 | Werewolf corpse consumption | 1 |
| D2 | Ranged target state machine | 9 |
| D3 | Missing/incomplete show report output | 6 |
| D4 | Visibility and descriptor output framing | 4 |
| D5 | Idle room translation and close ordering | 2 |
| D6 | Unwired commandless/login consumers | 2 |
| D7 | Incomplete mudlog producer coverage | 1 |
| E1 | Entry matrix and identity evidence | 14 |
| E2 | Reachable item/binding proof vehicles | 5 |
| E3 | Descriptor and rent roundtrip proof vehicles | 5 |
| E4 | Shared dispatch/reset proof completion | 5 |
| **Total** | **A=6; B=5; C=7; D=25; E=29** | **72** |

## A: reference-oracle decisions

### A1 — accuse and load (5 rows; M investigation, repair size unknown)

C's social loader interprets `#` as a NULL field (`act.social.c:201-275`); the actual accuse record is `lib/misc/socials:1-9`, not :58-66. The no-argument and target arms specify ordinary literals (`act.social.c:102-151`; `comm.c:2392-2555`). Mob load reaches `char_to_room`, a random creation act and two explicit acts (`act.wizard.c:1224-1249,1287-1321`). Historical notes describe pointer-like or empty actor output. **The exact defect causing these five anomalies is not established by the source read.** Do not describe an unspecified heap problem as a diagnosed repair.

Proposal: reproduce each against the pinned reference, trace resolved command index, loaded social fields, mob existence and descriptor queues in a throwaway diagnostic checkout, then propose the smallest diagnosed oracle repair. Keep diagnostic instrumentation separate from reference bytes; never quietly promote it. Audit sibling loader/buffer uses if the defect is shared (R5b/R5c). A failed load setup may also explain janitor disappearance, so establish spawn state before interpreting empty blocks.

Proof: pre-repair reproducer, unit/diagnostic assertion of the bad field or queue boundary, then exact actor/victim/observer transcripts at the recorded failing seeds; perturb the corrected field/write to make the proof fail. A candidate reference requires the promotion procedure and full census, with old binary and both hashes retained. Risk: copying run-varying garbage or “repairing” a setup failure. **Zach decision:** authorize diagnosis now, then approve a concrete reference patch after the cause is shown, or retain these rows blocked pending that investigation. No guessed C patch is proposed for approval here.

### A2 — wiznet self-overlap (1 row; S local repair, M promotion)

`act.wizard.c:1957-1976` repeatedly uses `sprintf(buf1, "%s...", buf1, ...)`: overlapping source and destination is a concrete UB site. The descriptor filters/order and literal content are specified in :1948-1980. `print_zone_to_buf:2225-2231` and `Crash_listrent` contain related self-append patterns; audit the class without conflating source UB with proof that every report currently fails.

Proposal: bounded append at the existing end or separate temporary formatting, preserving all intended strings, descriptor order and gates. Proof: populated online/offline/invisible/qualified roster, raw bytes and audience assertions, negative control that drops one qualifying entry, then candidate-reference full census under R1a. Risk: overflow bounds or changed roster order. **Zach decision:** approve a narrowly scoped oracle-repair investigation/patch and reference promotion, or leave blocked; this is not a Go `divergent-approved` proposal.

## B: narrow harness proposals

### B1 — four raw MOTD cells (4 rows; S existing vehicle, M only if a checkpoint is needed)

C selects returning mortal/immortal MOTD at `interpreter.c:1919-1938`, sets color in :1988-2008, and prints new-character MOTD at :2123-2132. Existing raw text capture plus setup/relogin can observe these bytes. `keep-ansi` alone still permits other normalization and is insufficient for a raw framing claim.

Proposal: first use `keep-prompts` with setup/relogin and saved mortal/immortal color-on/off identities. If whole setup introduces unrelated bytes outside this claim, add **one entry-stage capture boundary** around password-success/MOTD, retaining the unmodified captured bytes in evidence. Do not normalize away a red or compare only stripped ANSI. Telnet negotiation belongs in a separate protocol assertion if it is included in the row's intended contract.

Proof: oracle raw equality for all four cells, plus unit tests that detect removal of one escape sequence and alteration of CR/LF, reset or press-return spacing. Risk: a slice that excludes the byte under test, or first-player God bootstrapping mistaken for a mortal. **Zach decision:** approve existing-control experiments first; authorize a stage checkpoint only if their retained output demonstrates the need.

### B2 — sysfile success/pager (1 row; S fixture, M reusable fixture if necessary)

C chooses bugs/ideas/todo/typos then reads and pages the file (`act.wizard.c:3412-3443`; `modify.c:430-452`). Go reads the matching `lib/misc` path and uses `PageString` (`pkg/session/wiz_zone.go:252-283`). `do_gen_write` can create these files through commands (`act.other.c:1079-1135`), but embeds a wall-clock date.

Proposal: try a real command-generated report under a shared clock first; otherwise add a paired **temporary misc-file fixture** with identical, explicitly test-authored bytes and cleanup. It supplies test input, not invented shipped game content. Exercise multi-page next/back/quit and empty/missing cases. Proof: oracle pager bytes plus parser/copy/cleanup tests that fail when the fixture is applied on only one side or a page is dropped. Risk: wall-clock dates and line-ending conversion. **Zach decision:** choose clocked command generation or the narrower identical text fixture; no permanent misc files should be added.

## C: architecture choices

### C1 — generic NPC commands (3 rows; L)

C force and Lua action use the same `command_interpreter` for characters (`act.wizard.c:1869-1880`; `scripts.c:122-146`; `interpreter.c:883-980`). Lookup, level/position gates, command draw, native/script specials and execution are shared. Player-only Go dispatch and a small NPC action whitelist cannot prove this surface.

Options: (1) character-aware shared dispatcher and actor adapters, preserving descriptor-owned output only where C uses a descriptor — recommended fidelity route, L; (2) incrementally port reachable command families behind that **same** dispatcher, M per family with uncovered commands still blocked; (3) retain the whitelist as an explicitly approved divergence, S documentation/proof but no parity gain. A sit-only force shim is not a generic solution (R4).

Proof: both oracle and unit matrices for NPC sit/stand plus rejection, abbreviated lookup, native special consumption and a command outside the existing whitelist; player/NPC draw counts and output audiences; disabling the generic handoff must fail. Risk: fake Player bodies introduce flags, equipment, descriptor and RNG semantics. **Zach decision:** shared-dispatch design and incremental boundary, or exact whitelist behavior to approve as divergence; do not auto-exclude commands because no shipped Lua file calls them.

### C2 — Lua VM persistence/writeback (2 rows; L shared state, M nested slot)

C `open_lua_file` loads into the shared state without Go's global cleanup (`scripts.c:1666-1699`; `pkg/scripting/engine.go:294-319`). `table_to_char` uses stack slot 1 (:1975-1999), which in a nested binding can be the caller's first argument rather than the nested `ch`. That source behavior differs from writing back the intuitive current character.

Options: (1) preserve shared globals and C's nested slot semantics, with serialized reentrant VM access and scoped restoration of context globals, fidelity route; (2) isolated/cleaned globals and corrected writeback, only with Zach-approved exact divergences; (3) leave the two rows blocked while design is deferred. Namespacing is not equivalent to C's shared global collisions.

Proof: paired script A defines a non-`test_` helper, B invokes it later; a nested script changes distinct outer/inner character fields and demonstrates exactly which table writes back. Unit stack assertions and oracle state/readback, with cleanup and slot mutations each making the test fail. Risk: cross-script collisions, test-order pollution, stale table pointers and deadlock in nested calls. **Zach decision:** approve C semantics or specify isolated behavior as deliberate divergence; approve these independently, since costs and effects differ.

### C3 — deferred extchar and mobile mount (2 rows; M each)

C extraction marks flags then drains at heartbeat (`scripts.c:480-494`; `handler.c:1194-1254`; `comm.c:805`). Go's adapter immediately extracts mobs and ignores players (`pkg/game/world_bridge.go:699-706`). Deferred RawKill does not automatically fix this binding. C Lua mount delegates to ride/dismount/unmount for character riders (:871-906); Go's mount adapter handles player riders.

Proposal: route extchar through the shared deferred extraction lifecycle, respecting lookup versus display treatment before the drain; extend rider/mount links and command adapters to mobile riders without allocating synthetic players. Use existing paired Lua fixtures with real spawnable mobs. Proof: oracle before/after-pulse inworld and look, unit flags/link cleanup including player extraction; mount/dismount/unmount success/refusal and reciprocal links, each with a revert failure. Risk: stale Lua references, extraction while iterating, mount teardown/RNG effects. **Zach decision:** implement the shared lifecycle and mobile rider model, or separately approve exact unsupported behavior. No shipped callers is not an exclusion argument.

## D: concrete feature or fidelity repairs

### D1 — werewolf corpse eat (1 row; M)

`act.item.c:1050-1085` finds a floor corpse only on the werewolf arm, emits savage-eat acts, increases FULL, scatters its contents, extracts the corpse and creates prototype 19 with C's fields/timer. `pkg/game/item_consumable.go:185-191` emits part of the text and returns before those state effects.

Proposal: complete that branch with C list order and object-location operations. Proof: transformed PC, actual corpse and contents, actor/observer bytes, hunger and exact residual floor state; unit negative controls omit FULL, scattering or flesh creation, plus an oracle vehicle. Risk: copying the wrong prototype or clamping a value C doesn't clamp. **Zach decision:** authorize the C branch and fixture; no new design semantics are needed.

### D2 — shoot target state machine (9 rows; L, split into reviewable C arms)

Read the whole `act.offensive.c:746-998`, including target fallback, PC level window/fighting refusal, NPC versus PC outcome, projectile consumption/placement, HP/death, and synchronous NPC retaliation. `pkg/game/skill_c10_combat.go:189-222` uses a simplified draw/damage and `WaitCh:1`; `pkg/command/skill_commands.go:929-1040` handles generic result output and successful target transfer. C has neither that extra wait nor generic skill-message routing.

Proposal: one C-shaped execution path retaining the entry gates already proven; split review commits by target lookup/refusal, mob outcomes and player outcomes, with no claim green until all dependent branches pass. Proof: oracle all three audiences and target-room observation at seeds 1,2,3,5,8; unit exact RNG draws, projectile location/extraction, unchanged wait and direct HP/death/enrollment effects. Revert the dex term, fallback, PC window, NPC-only move or zero-wait contract and each proof must fail. Risk: shared combat red during retaliation, death order and accidental draw shifts. **Zach decision:** authorize this feature repair and its combat dependency budget; a downstream combat failure must be traced before expanding scope.

### D3 — six show reports (6 rows; S aggr, M zones/hooks/player/shops, L rent metadata if absent)

`cmdShow` has empty success arms for zones/player/rent/shops/hooks and a VNUM/name-sorted aggr list (`pkg/session/wiz_info.go:96-225`). C uses zone age/lifespan/reset/top (:2225-2231,2300-2321), durable player fields/date/played (:2323-2347), rent listing (:2349-2355; `objsave.c:275-329`), parsed shop index (`shop.c:1267-1300,1413-1449`), character-list order for aggr (:2430-2442), and cross-zone exits for hooks (:2461-2494). These are formatter/data gaps, not a single offline-player fixture problem.

Proposal batches: zones/hooks over live zone state and indexed exits; aggr over C character-list order; shops over existing parsed records/snapshots preserving C boot index, 19-row headers and LF/CR; player over login's SQLite record path using C-equivalent stored fields and deterministic timestamps; rent over existing object persistence, first inventorying whether rentcode/locate/time metadata is retained. SQLite remains the only store; do not fabricate a C filename/report from house saves or silently add an incompatible save format.

Proof: populated oracle reports and paging plus unit exact rows/order/date/precision/empty cases. Mutations reorder two entities, swap profit fields, use live rather than durable player data or omit a cross-zone exit; each must fail. Risk: C self-append UB, representation/index differences and unavailable rent metadata. **Zach decisions:** approve report batches; for rent, choose compatible metadata mapping after the audit or a specifically described divergence if C's report cannot be represented under the unchanged save-format constraint. Source UB that blocks a report belongs in A's approval process.

### D4 — visibility/output framing (4 rows; S visibility, M descriptor flush)

C TO_SLEEP allows sleeping gsay recipients and PERS uses CAN_SEE without AWAKE (`act.comm.c:846-858`; `utils.h:515-530`). Go still refuses a sleeping observer in `pkg/game/act.go:143-146`. C's descriptor pending-output pass issues listener prompts and controls blank lines (`comm.c:626-642,1620-1643`). For syslog, C emits green, message with CRLF, then reset (`utils.c:236-238`); Go assembles reset **before** CRLF (`pkg/game/logging.go:175-187`). Filter logic exists, but normalized green would not certify this framing.

Proposal: first repair CAN_SEE/PERS with a whole-class audience audit, then preserve C pending-output/flush/prompt/reset ordering through the shared session boundary. Use existing raw capture, not a gsay-only prompt patch. Proof: sleeping/awake/group actor/listener byte matrix and syslog type/level/writing/color matrix, oracle plus unit flush boundaries; reintroducing awake blindness, missing prompt, extra blank line or moved reset must fail. Risk: all asynchronous audiences and shared color formatting. **Zach decision:** approve these scoped shared fixes; any broader audience deviation gets its own trace and scope decision.

### D5 — idle close defects (2 rows; S room translation, M queued-close ordering)

C moves to RNUM 3 before close (`limits.c:438-443`), then broadcasts link loss and discards queued output (`comm.c:2092-2093,2131-2133`). Shipped RNUM 3 is VNUM 4; Go transfers to VNUM 3 (`pkg/game/limits_misc.go:79`). The historical F2 trace identifies terminal weather flushed before Go retirement, whereas C drops it (`weather.c:58-80`; `comm.c:636,2360-2363`).

Proposal: explicit C room-index translation for this lifecycle site and queued close ordering, auditing sibling RNUM/VNUM sites. Proof: a peer in each candidate destination, and an outdoor terminal-hour idler; retain the inside-room fixture only in the unrelated roundtrip proof, never in the weather proof. Oracle plus unit close/queue ordering; mutations choose VNUM3 or flush the terminal line and must fail. Risk: other pending-output consumers and session retirement. **Zach decision:** authorize these separately identified F1/F2 repairs.

### D6 — jail pulse and wizlock consumers (2 rows; M each, separate PRs)

C room_activity dispatches command 0 to jail; its timer body moves qualifying players and looks after its acts (`comm.c:691-756`; `spec_procs2.c:1470-1493`). Go `specJail` is a no-op (`pkg/game/spec_procs2.go:1628-1638`). C checks integer game_restrict at new-name acceptance and known-password login (`interpreter.c:1832-1847,1906-1914`). Go stores a threshold; a repository search finds no production caller of `WizlockLevel`/`IsWizlocked` beyond their definitions/setters.

Proposal: port the jail timer gates/body through the existing commandless dispatch; wire the integer restriction at both C login save/state boundaries. Proof: timer, mini-mud, level/hunting/invis/position matrix and exact audiences; new/returning players below/equal/above restriction, with state and no unauthorized record creation. Oracle and unit tests, with no-op jail/removed restriction mutations failing. Risk: moving during room iteration, locking and differences from approved security lockouts. **Zach decision:** authorize each C consumer and keep approved security extensions explicitly separate.

### D7 — mudlog producers (1 aggregate row; L inventory, M per producer family)

The manifest's “about 120” is an estimate, not an audited denominator. `rg -n 'mudlog\(' src --glob '*.c'` includes the definition, comments, a macro and multiline calls, so its raw count is not a count of runtime producers. Source sites span interpreter/comm, combat/magic, wizard, object saves, specials, clans/houses, OLC and Lua. The shared consumer is `utils.c:212-238`; representative remaining sites include `zedit.c:226,369,878`, `act.wizard.c:1316,1371,2051,2067`, `objsave.c:489-539,1186`, `spec_procs2.c:371,456,634,833-913`, `scripts.c:1694,1765-1800`, and `act.other.c:434,477,535,1120`.

Proposal: enumerate actual calls/macros, record literal/type/min-level/file flag, reachability and exact Go call path; separate server-only file output from descriptor-visible output without dropping file=FALSE broadcasts. Port by producer family after D4's consumer is proven. Existing logging sites do not prove remaining sites. Proof: observable triggers with qualified and filtered immortal peers, state-level units for hard-to-stage errors, and at least one removed/retyped/min-level mutation per family; a wholesale string-presence audit is not R5h proof. Risk: huge aggregate row hides partial completion, OLC error fixtures and timestamps. **Zach decision:** approve family batches and a subsequent manifest split into independently evidenced subcases; this survey does not claim a completed 120-site audit.

## E: evidence-first batches

### E1 — fourteen non-raw entry cells (14 rows; M per small state-machine batch)

The C state machine and its durable boundaries are in `interpreter.c:1492-1655,1743-2350`; first-player initialization is `db.c:3006-3036`. Existing entry tests and Phase 3b SQLite work cover parts, not the full pending matrix. Separate name/deleted/load failure, password/retry, choice/reroll, menu/world/bootstrap, duplicate identities, and browser routing into PR-sized batches. `entry.login-restrictions` depends on D6 for wizlock; identity-consumers includes Go-only agent ownership/security and must distinguish that surface from C name identity.

Proof: C differential for telnet-reachable bytes and lifecycle, real SQLite units for durable failures, browser transport contract tests against the same C-backed entry expectations. Use seeds 1,2,3,5,8 for rerolls/bootstrap RNG; two concurrent descriptors for usurp/unswitch and audiences; exact secret/prompt bytes, rejection state and no duplicate record. Each batch names a mutation that flips a gate, removes a save, changes identity casefolding or advances one draw and produces an assertion failure. Risk: random names/timestamps, existing raw setup differences, and approved security behavior incorrectly counted as C parity. **Zach decisions:** approve first batch; specify the exact security/agent subset to retain under existing approvals or newly approve as divergence. Do not approve the whole identity or restriction row by association.

### E2 — items and skill binding (5 rows; S set-skill, M transform/item vehicles)

Vampire drink/eat gates are reachable at night (`act.item.c:980-995,1121-1125`), with Go branches in `item_consumable.go`. Tattoo use calls its implemented path (`act.other.c:920-924`; `tattoo.c:31-91`; `pkg/game/other_economy.go:108-168`). Item sleep's actual call path is explained above; its outlaw/refusal/reagent/save behavior must be tested independently of cast. Lua skill maps the spell number and sets the player skill (`scripts.c:1365-1383`; `pkg/game/world_bridge.go:415-421`); C's affect_total side effects require comparing effective stats rather than asserting that a cached function exists.

Proposal: disposable real transform/time and equipped tattoo fixtures, an actual sleep potion/scroll vehicle, paired Lua readback for skill without relying on unrelated practice text. Start proof-only and trace any red before a separate fix. Proof: oracle audiences/conditions, unit cooldown/object consumption/skill/effective equipment and affect stats, exact draw counts where saving throws apply. Mutations remove night gates, cooldown, self-item dispatch or skill-number mapping and must fail. Risk: fixture transformation combat and sleep retaliation draws. **Zach decision:** authorize these proof batches; no exclusion is warranted merely by fixture difficulty.

### E3 — linkdead switch and forced-rent roundtrip (5 rows; M each family)

C switch permits descriptor-less PC possession only by Implementor and checks already-switched after command entry gates (`act.wizard.c:1175-1204`). Existing peer-drop provides a live linkdead body, not a durable offline record. Proposal: warm up a peer to the required level/visibility, drop its descriptor, pump disconnect handling, then test Implementor success, lower actor refusal and a high-level possessed body that reaches already-switched. Audit Go descriptor/original ownership and return cleanup; if PC possession is missing, promote the traced finding to a D/C repair proposal rather than asserting a fixture proves it.

Rent: C saves before extraction and restores through Crash_load (`objsave.c:745-760,912-956`; `interpreter.c:2184-2194`). Use the existing forced-idle pump plus `<RELOGIN>`, rentable nested inventory and equipped NORENT objects, verifying SQLite state as well as live re-entry. Keep the interior void fixture to isolate D5 weather while proving rent. Proof: both oracle audiences and durable/live unit state, revert descriptor reassignment or NORENT destruction to fail. Risk: peer-drop cannot currently be freely combined with every relogin topology, rent save timing, and D5 dependencies. **Zach decision:** approve existing-control vehicles first; request only a narrow staged-disconnect/reconnect extension if a retained failed setup establishes the need.

### E4 — existing dispatch and reset, with downstream dependencies (5 rows; S delegation, M special/reset matrices)

Special ordering is now implemented: prove resolution/position before specials, every native/Lua tier and first-TRUE consumption, including switched NPC script gates (`interpreter.c:883-980,1407-1481`; current Go sources above). Janitor is implemented but historical vehicles lost the native before its pulse (`mobact.c:68-93`; `spec_procs.c:750-768`): assert spawn/assignment/presence before pumping, use a stable existing reset spawn if load success is blocked, and require visible removal plus carried state.

The shared cityguard breed owner is now **oracle-green-multiseed**, not blocked (`spec-procs.tsv:71`, `spec-proc-cityguard-breed@1,2,3,5,8`). Take-to-jail calls breed_killer within its own loop before the protection scan (`spec_procs2.c:1447-1448,1679-1722`); Go deliberately does not duplicate that currently-unproven caller path. Prove caller delegation/gates/order, then reuse the owner, rather than copy its spike/stake combat.

Teleport's special already has its speech/teleport body (`spec_procs3.c:225-237`), but its row records a downstream combat/RNG red (`fight.c:1898-2032`). Recheck after recent combat fixes; if still red, isolate attacker selection/draw/message differences, retain the block and submit a separate repair scope. Reset already exists (`pkg/game/spawner.go:208-648`); prove C's M/O/P/G/E/R/D/L and loop/conditional/failure/door/age/RNG matrix (`db.c:2074-2285`; `act.wizard.c:2035-2069`).

Proof: oracle plus units for dispatch return/state, every reset command and failed conditional, multiseed teleport combat; mutations move a tier, skip janitor dispatch, disable the caller handoff, swap one reset draw or change attack ordering and must fail. Risk: misdiagnosing quiet spawn as proof, aggregate reset completeness and pre-existing combat failures. **Zach decision:** authorize evidence-first batches, the conditional delegation below, and separate follow-up scoping if combat remains red.

## Proposed reclassifications and decisions

No status is changed in this PR.

| Case/subset | Proposal | Evidence and prerequisite |
|---|---|---|
| `mob.take-to-jail-breed-killer` | Conditional `delegated` to `mob.cityguard-breed-killer` for the **shared helper only** | Owner now has five-seed proof; C caller is :1447-1448. First prove the Go caller reaches it with C gates/order. Until then remain blocked; owner proof does not repair a missing caller. |
| `objmagic.sleep-entry-gates` | **Keep blocked; reject excluded** | Potion self-target bypasses cast_spell TAR_NOT_SELF and reaches mag_affects. Replace the false reachability note only in an authorized proof PR. |
| `lua.special-order` | Candidate proof completion, **not delegated/excluded** | Current implementation/tests invalidate the old architectural note. Require tier/entry differential evidence and revert proof before selecting a proven status. |
| C2 shared globals/nested writeback, C1 whitelist, C3 unsupported mobile/player bindings | Optional, narrowly specified `divergent-approved` proposals only if Zach chooses those options | Reachable bindings and source semantics are known. No shipped caller is insufficient for excluded. Retaining unsupported behavior must be expressly approved and fail-capably tested; it remains a departure, not C parity. |
| Entry security/agent identity subset | Split C identity proof from a specifically approved extension if the audit finds different contracts | C's name policy is interpreter.c:1492-1518. Agent-key ownership has no C equivalent. Cite the actual approval and exact tested behavior, not a blanket “security” rationale. |
| A1/A2 reference repairs | Keep blocked pending approved repair/promotion | R1a reference repair does not authorize a Go divergence or earn a proof merely from source intent. |

No unconditional `excluded` proposal survived the call-path audit. No governing rule change is needed: R1a, R5e and R5h already cover these decisions.

## Recommended sequence after review

1. **4b: E4 special-order proof completion**, one row, stale diagnosis corrected and full tier negative controls. It has implemented code and existing focused tests.
2. **4c: E2 set-skill proof**, then separate tattoo and vampire vehicles; item sleep gets its own true-reachability proof PR. These avoid risky architecture work and can disclose concrete fixes.
3. **E1 entry batches**: names/load/password first; choices/rerolls and bootstrap next; menu/world/duplicate/browser/identity separately. Hold restriction completion for D6.
4. **E3 switch vehicle**, then **rent roundtrip** (isolated from D5 terminal weather). **E4 janitor, caller delegation, reset and teleport** each separately; a remaining combat red gets a new scoped proposal.
5. **B1 raw entry** using existing capture; **B2 sysfile** chooses the smallest clock/fixture extension. Any harness change requires full census, not just its own vehicle.
6. **D4 visibility**, then descriptor framing/syslog; **D5 room index**, then queued-close weather; **D6 wizlock**, then jail. Shared class audits and full censuses belong with each behavior fix.
7. **D1 corpse eat**, **D3 individual report batches**, **D2 shoot** and **D7 producer families** as larger feature work. D7 follows its proven consumer; reports reuse existing parsers/stores.
8. **A and C at Zach's discretion**. Start A diagnosis early if it unlocks janitor/report evidence, but promote only an approved demonstrated repair. Start C1 shared-dispatch design and C2/C3 choices when the owner has selected the exact semantics.

This ordering prioritizes real evidence without presenting all E rows as cheap: teleport combat, reset completeness and identity/security can still be substantial. A proof-first red is a finding to scope, not permission to fix it inside a proof-only PR. Each later addendum should name its scenario set, allowed dependencies and stop rule. Behavior/harness changes require full census; claims census requirements should be explicit for those later phases. Five-seed rows must be exercised at their claimed seeds, with baseline comparison for pre-existing failures.

## Complete row ledger

Each blocked row appears exactly once below. “Next proof” is proposed work; it is not an execution result. Group sections supply size, risks, alternatives and owner decisions. Original manifest notes remain intact, including the stale claims identified above.

| Case / manifest | Group | Current C path | Next proof / survey correction |
|---|---|---|---|
| `accuse.no-argument` ([row 4](../accuse.tsv#L4)) | A1 | `act.social.c:102-121; lib/misc/socials:1-9` | Historical actor anomaly; diagnose before proposing an oracle patch; exact actor/victim/observer control. |
| `accuse.target-success` ([row 5](../accuse.tsv#L5)) | A1 | `act.social.c:141-151; lib/misc/socials:1-9` | Historical actor anomaly; diagnose before proposing an oracle patch; exact actor/victim/observer control. |
| `accuse.target-audience` ([row 6](../accuse.tsv#L6)) | A1 | `act.social.c:141-151; comm.c:2392-2555; lib/misc/socials:1-9` | Historical actor anomaly; diagnose before proposing an oracle patch; exact actor/victim/observer control. |
| `accuse.one-argument` ([row 7](../accuse.tsv#L7)) | A1 | `interpreter.c:1265-1283; act.social.c:114-121` | Historical actor anomaly; diagnose before proposing an oracle patch; exact actor/victim/observer control. |
| `drink.vampire` ([row 14](../drink.tsv#L14)) | E2 | `act.item.c:988-995` | Reachable transform/time or tattoo vehicle; output/state/cooldown negative controls. |
| `eat.vampire` ([row 11](../eat.tsv#L11)) | E2 | `act.item.c:1121-1125` | Reachable transform/time or tattoo vehicle; output/state/cooldown negative controls. |
| `eat.werewolf-corpse` ([row 12](../eat.tsv#L12)) | D1 | `act.item.c:1058-1085` | Complete FULL/scatter/extract/prototype19 effects with actor/observer and object-state proofs. |
| `entry.name-validation` ([row 2](../entry.tsv#L2)) | E1 | `interpreter.c:1492-1518,1743-1820` | Both transport name/length/reserved/ban/retry matrix; mutate one gate. |
| `entry.browser-name-routing` ([row 5](../entry.tsv#L5)) | E1 | `interpreter.c:1743-1820` | Browser creation/retry/return transcript against same entry contract; existing smoke tests are partial. |
| `entry.deleted-name` ([row 10](../entry.tsv#L10)) | E1 | `interpreter.c:1762-1787` | Delete and reuse durable identity; exact prompt plus SQLite lifecycle, no format changes. |
| `entry.new-password` ([row 11](../entry.tsv#L11)) | E1 | `interpreter.c:1942-1987` | Length/name-equal/secret/mismatch matrix at both transports. |
| `entry.password-retries` ([row 12](../entry.tsv#L12)) | E1 | `interpreter.c:1860-1895` | Empty/wrong/correct password, durable bad-PW accounting and disconnect threshold. |
| `entry.login-restrictions` ([row 13](../entry.tsv#L13)) | E1 | `interpreter.c:1822-1847,1896-1914` | New/select bans and restriction; D6 dependency, approved lockout extensions separately. |
| `entry.duplicate-session` ([row 14](../entry.tsv#L14)) | E1 | `interpreter.c:1528-1655` | Reconnect/usurp/unswitch complete descriptor, audience and identity-case matrix. |
| `entry.creation-choices` ([row 15](../entry.tsv#L15)) | E1 | `interpreter.c:1988-2105` | Color/sex/race help/class/hometown first-character choices, including invalid inputs. |
| `entry.stats-reroll` ([row 16](../entry.tsv#L16)) | E1 | `interpreter.c:2106-2157` | Accept/reroll exact values/draws at five seeds; God versus mortal. |
| `entry.lookup-load-failure` ([row 21](../entry.tsv#L21)) | E1 | `interpreter.c:1762-1800` | Corrupt/missing durable restore and record conversion failure, not only legacy ambiguity. |
| `entry.motd-mortal-color-on-raw` ([row 26](../entry.tsv#L26)) | B1 | `interpreter.c:1978-2008;interpreter.c:2123-2132` | Raw keep-prompts/setup or relogin first; one MOTD stage checkpoint only if demonstrated necessary. |
| `entry.motd-mortal-color-off-raw` ([row 27](../entry.tsv#L27)) | B1 | `interpreter.c:1978-2008;interpreter.c:2123-2132` | Raw keep-prompts/setup or relogin first; one MOTD stage checkpoint only if demonstrated necessary. |
| `entry.motd-immortal-color-on-raw` ([row 28](../entry.tsv#L28)) | B1 | `interpreter.c:1919-1922;interpreter.c:1978-2008` | Raw keep-prompts/setup or relogin first; one MOTD stage checkpoint only if demonstrated necessary. |
| `entry.motd-immortal-color-off-raw` ([row 29](../entry.tsv#L29)) | B1 | `interpreter.c:1919-1922;interpreter.c:1978-2008` | Raw keep-prompts/setup or relogin first; one MOTD stage checkpoint only if demonstrated necessary. |
| `entry.menu-actions` ([row 30](../entry.tsv#L30)) | E1 | `interpreter.c:2165-2350` | Exit/description/background/password/delete/rejection before first entry and return. |
| `entry.world-entry` ([row 31](../entry.tsv#L31)) | E1 | `interpreter.c:2174-2243` | Saved/frozen/unhealthy room, registration once, first/return audiences. |
| `entry.god-mortal-bootstrap` ([row 33](../entry.tsv#L33)) | E1 | `interpreter.c:2184-2239; db.c:3006-3036` | Extend existing boundary test to complete seeded stats/gear oracle proof. |
| `entry.identity-consumers` ([row 35](../entry.tsv#L35)) | E1 | `interpreter.c:1492-1518` | Audit all identity consumers; separate C case policy from approved security/agent contract. |
| `force.visible-npc-target` ([row 18](../force.tsv#L18)) | C1 | `handler.c:1303-1325; act.wizard.c:1869-1880` | Shared character dispatcher, gates/specials/draws; no sit-only adapter or invented player body. |
| `force.npc-command-interpreter` ([row 19](../force.tsv#L19)) | C1 | `interpreter.c:883-980; act.wizard.c:1869-1880` | Shared character dispatcher, gates/specials/draws; no sit-only adapter or invented player body. |
| `gsay.sleeping-recipient` ([row 13](../gsay.tsv#L13)) | D4 | `act.comm.c:846-858` | Actual shared visibility/flush byte gap; raw oracle and unit mutation controls. |
| `gsay.act-listener-prompt` ([row 21](../gsay.tsv#L21)) | D4 | `comm.c:626-642,1620-1643` | Actual shared visibility/flush byte gap; raw oracle and unit mutation controls. |
| `gsay.act-echo-prompt-spacing` ([row 22](../gsay.tsv#L22)) | D4 | `comm.c:1620-1643` | Actual shared visibility/flush byte gap; raw oracle and unit mutation controls. |
| `idle.force-rent-lost-link` ([row 10](../idle.tsv#L10)) | D5 | `limits.c:438-443; comm.c:2131-2133` | F1 RNUM mapping or F2 queued-weather discard; independent peers/terminal-hour vehicles. |
| `idle.force-rent-terminal-weather-byte` ([row 15](../idle.tsv#L15)) | D5 | `weather.c:58-80; comm.c:2092,2360-2363` | F1 RNUM mapping or F2 queued-weather discard; independent peers/terminal-hour vehicles. |
| `idle.rent-roundtrip-inventory` ([row 16](../idle.tsv#L16)) | E3 | `objsave.c:912-956; interpreter.c:2184-2194` | Existing idle pump and relogin; rentable/nested/NORENT durable and restored state proof. |
| `idle.rent-roundtrip-equipped-norent` ([row 17](../idle.tsv#L17)) | E3 | `objsave.c:745-760; objsave.c:912-956` | Existing idle pump and relogin; rentable/nested/NORENT durable and restored state proof. |
| `load.mob-success` ([row 13](../load.tsv#L13)) | A1 | `act.wizard.c:1224-1249,1287-1321` | Trace actual spawn and queued acts; source read does not diagnose the historical empty output. |
| `lua.nested-writeback-slot` ([row 23](../lua.tsv#L23)) | C2 | `scripts.c:1979 (lua_pushvalue(L, 1))` | Paired synthetic scripts demonstrate exact C semantics; owner chooses parity or explicit divergence. |
| `lua.special-order` ([row 24](../lua.tsv#L24)) | E4 | `interpreter.c:883-980,1407-1481` | Already implemented; exercise complete ordered native/Lua tiers and entry gates. |
| `lua.npc-command-unported` ([row 25](../lua.tsv#L25)) | C1 | `interpreter.c:883-949` | Shared character dispatcher, gates/specials/draws; no sit-only adapter or invented player body. |
| `lua.global-function-persistence` ([row 26](../lua.tsv#L26)) | C2 | `scripts.c:1666-1699 (open_lua_file)` | Paired synthetic scripts demonstrate exact C semantics; owner chooses parity or explicit divergence. |
| `lua.bind-extchar-deferred` ([row 51](../lua.tsv#L51)) | C3 | `scripts.c:480-494; handler.c:1194-1254; comm.c:805` | Prove marked-but-findable versus skipped display, then heartbeat drain; adapter currently differs. |
| `lua.bind-mount` ([row 69](../lua.tsv#L69)) | C3 | `scripts.c:871-906` | Mobile rider reciprocal links and real ride/dismount/unmount gates; no shipped caller is not exclusion. |
| `lua.bind-set-skill` ([row 71](../lua.tsv#L71)) | E2 | `scripts.c:1365-1383; handler.c:314-377` | Paired skill-number readback plus effective stats; mapping exists, recalculation needs evidence. |
| `mudlog.unported-sites` ([row 6](../mudlog.tsv#L6)) | D7 | `utils.c:212-238; producer inventory in D7` | Audit true callsites/macros, then family proofs; raw search count includes non-producers. |
| `objmagic.sleep-entry-gates` ([row 58](../object-magic.tsv#L58)) | E2 | `spell_parser.c:406-505,699-718,886; magic.c:1199-1248` | Reachable self-potion; test outlaw refusal/success, reagent and saving throw. Reject excluded. |
| `shoot.targeted-shot` ([row 20](../shoot.tsv#L20)) | D2 | `act.offensive.c:887-980` | Repair C target branch; five-seed audiences/RNG/HP/projectile/death negative controls. |
| `shoot.targeted-shot-audience` ([row 21](../shoot.tsv#L21)) | D2 | `act.offensive.c:887-980; comm.c:2392-2555` | Repair C target branch; five-seed audiences/RNG/HP/projectile/death negative controls. |
| `shoot.target-fallback` ([row 22](../shoot.tsv#L22)) | D2 | `act.offensive.c:887-900` | Repair C target branch; five-seed audiences/RNG/HP/projectile/death negative controls. |
| `shoot.pc-level-window` ([row 23](../shoot.tsv#L23)) | D2 | `act.offensive.c:906-914` | Repair C target branch; five-seed audiences/RNG/HP/projectile/death negative controls. |
| `shoot.target-fighting` ([row 24](../shoot.tsv#L24)) | D2 | `act.offensive.c:916-921` | Repair C target branch; five-seed audiences/RNG/HP/projectile/death negative controls. |
| `shoot.mob-hit-miss` ([row 25](../shoot.tsv#L25)) | D2 | `act.offensive.c:887-980; fight.c:1314-1718` | Repair C target branch; five-seed audiences/RNG/HP/projectile/death negative controls. |
| `shoot.player-hit-miss` ([row 26](../shoot.tsv#L26)) | D2 | `act.offensive.c:887-980; fight.c:1314-1718` | Repair C target branch; five-seed audiences/RNG/HP/projectile/death negative controls. |
| `shoot.wait-and-object-consumption` ([row 27](../shoot.tsv#L27)) | D2 | `act.offensive.c:746-998` | Prove no added WAIT_STATE and exact extraction/room placement; existing note incorrectly promises a wait. |
| `shoot.no-skill-message-path` ([row 28](../shoot.tsv#L28)) | D2 | `act.offensive.c:746-998` | Direct literal bytes and HP/death path; never route to damage/skill_message. |
| `show.zones-valid` ([row 22](../show.tsv#L22)) | D3 | `act.wizard.c:2225-2231,2300-2321` | Age/lifespan/reset/top, index selection and pager; valid Go arm is empty. |
| `show.player-valid` ([row 23](../show.tsv#L23)) | D3 | `act.wizard.c:2323-2347` | SQLite durable record report, deterministic dates/played; offline fixture alone is insufficient. |
| `show.rent-valid` ([row 24](../show.tsv#L24)) | D3 | `act.wizard.c:2349-2355; objsave.c:275-329` | Audit rent metadata then report locates/code/load and pager; no invented house-save mapping. |
| `show.shops-list` ([row 25](../show.tsv#L25)) | D3 | `act.wizard.c:2413-2415; shop.c:1267-1300,1413-1449` | Parser exists; formatter must preserve boot index, profit fields, customer flags and 19-row headers. |
| `show.hooks-valid` ([row 27](../show.tsv#L27)) | D3 | `act.wizard.c:2461-2494` | Cross-zone exit/source/destination matrix; valid Go arm empty. |
| `show.aggr-populated` ([row 28](../show.tsv#L28)) | D3 | `act.wizard.c:2430-2442` | Populated MOB_AGGR24 list in character-list order; reject current VNUM/name sorting. |
| `mob.janitor-pulse-dispatch` ([row 66](../spec-procs.tsv#L66)) | E4 | `mobact.c:68-93; interpreter.c:1407-1456; spec_procs.c:750-768` | Confirm spawn/assignment/presence before pulse; do not read an empty block as dispatch proof. |
| `mob.take-to-jail-breed-killer` ([row 284](../spec-procs.tsv#L284)) | E4 | `spec_procs2.c:1447-1448,1679-1722` | Owner now five-seed green; prove caller/gates before conditional helper delegation. |
| `room.jail-commandless-body` ([row 287](../spec-procs.tsv#L287)) | D6 | `comm.c:691-756; spec_procs2.c:1470-1493` | Reachable pulse timer body; Go no-op requires production port and revert proof. |
| `mob.teleport-victim-combat-transcript` ([row 339](../spec-procs.tsv#L339)) | E4 | `fight.c:1898-2032; spec_procs3.c:225-237` | Recheck downstream combat at claimed seeds; isolate any residual attack/message/draw red. |
| `use.tattoo` ([row 7](../use.tsv#L7)) | E2 | `act.other.c:920-924; tattoo.c:31-91; pkg/game/other_economy.go:108-168` | Reachable transform/time or tattoo vehicle; output/state/cooldown negative controls. |
| `sysfile.read-page` ([row 20](../wizard.tsv#L20)) | B2 | `act.wizard.c:3412-3443; modify.c:430-452; pkg/session/pager.go:110-145` | Clocked bug/idea producer or identical disposable misc fixture; success and pager negative controls. |
| `switch.player-success` ([row 77](../wizard.tsv#L77)) | E3 | `act.wizard.c:1175-1204` | Existing peer-drop linkdead vehicle; prove descriptor ownership/gates before inventing harness. |
| `switch.mortal-player-level-gate` ([row 78](../wizard.tsv#L78)) | E3 | `act.wizard.c:1175-1204` | Existing peer-drop linkdead vehicle; prove descriptor ownership/gates before inventing harness. |
| `switch.already-switched` ([row 79](../wizard.tsv#L79)) | E3 | `act.wizard.c:1175-1204` | Existing peer-drop linkdead vehicle; prove descriptor ownership/gates before inventing harness. |
| `wizlock.login-threshold` ([row 92](../wizard.tsv#L92)) | D6 | `interpreter.c:1832-1847,1906-1910` | Wire C threshold at creation/password boundaries; assert no unauthorized durable creation. |
| `wiznet.gods-online` ([row 105](../wizard.tsv#L105)) | A2 | `act.wizard.c:1947-1978` | Confirmed self-overlapping sprintf; proposed bounded append requires approval and reference promotion proof. |
| `zreset.reset-zone-state` ([row 112](../wizard.tsv#L112)) | E4 | `act.wizard.c:2035-2069; db.c:2074-2285` | Existing reset engine; command/conditional/loop/door/removal/age state and RNG matrix. |
| `syslog.consumer-filter` ([row 125](../wizard.tsv#L125)) | D4 | `utils.c:212-238` | Type/level/writing audience matrix plus raw reset-after-CRLF; Go currently reverses that boundary. |

## Verification of this proposal

The shared loader returns 72 blocked rows; this ledger contains the same 72 unique IDs, once each. Only this proposal document changes. No reference binary was executed and no production access occurred. `go build ./...`, `go vet ./...`, `go test ./...` and `golangci-lint run ./...` each exited 0; lint reported zero issues. `make fmt` exited 0 and left all existing files unchanged; `git diff --check` is clean. `make hooks` installed the repository hook path. The inventory/link check also passed: 72 unique IDs match the shared loader exactly and every manifest link exists. These checks validate the docs-only branch, not the proposed future behavior proofs.
