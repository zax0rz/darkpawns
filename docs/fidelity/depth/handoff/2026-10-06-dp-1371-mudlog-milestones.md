# DP-1371 D7: game milestone producers and switched help

Local (mudlog-only): seven producer calls or replacements and the help producer's acting-body payload. No output routing, Act path, handler signature, game mutation order or error handling changes. Base `072326bb5`, after #1819/#1820. One commit per case.

## Contracts and held-lock audit

| Case | C site | Contract / boundary | Locks at producer |
|---|---|---|---|
| help body | src/act.informative.c:1608-1610 | NRM / IMMORT / file; GET_NAME of acting PC or NPC | Manager read lock snapshots acting name (body getter takes its read lock), then releases before MudLog. No lifecycle lock added. |
| advance | src/class.c:712-715 | BRF / max(IMMORT,invis) / file; after gains and save | Player mutation lock, capacity helper lock and save callback all finish before producer. Invis getter read lock finishes before call. |
| stable collect failure | src/spec_procs2.c:456 | BRF / GRGOD / file; after failure tell | Spawn and tell helpers return before producer; no world/mobile lock retained. |
| remort | src/spec_procs2.c:833-834 | BRF / IMMORT / file; after AffectTotal, before AdvanceLevel | AffectTotal's locks finish before call. No mutation order change. |
| assassin PC roster | src/spec_procs2.c:880 | BRF / IMMORT / file; target warning, log, actor refusal | Roster snapshot and name getter release locks before call. |
| assassin hire | src/spec_procs2.c:913 | BRF / IMMORT / file; after hire Act; embedded CRLF retained | Spawn/hunting/charge/Act helpers finish before call. |
| Medusa | src/spec_procs2.c:1573 | BRF / IMMORT / file; after petrification Acts, before deaths/XP/raw kill | Room lookup read lock finishes before producer; no world lock held. |
| attitude loot | src/fight.c:1156-1158 | CMP / IMMORT / file; PC victim only, after get/junk/wear | Combat engine releases membership and body locks before DeathFunc. HandleDeath's PC branch returns from death cleanup before loot. Object/equipment helpers release their locks before producer. |

Registered command specials run after room/mobile collection snapshots release their locks. Autonomous mobile activity also releases snapshots before special invocation; cmdless branches do not reach the added PC command producers. MudLog writes its file side and then visits a session snapshot: manager lock is released before delivery, player flag/name reads finish before sending, and world/player lookups finish before the output sink's switch/snoop/heartbeat handling. None of these producers retains a world, body, lifecycle, editor or manager lock while calling that delivery path.

## Fail-capable proofs

`TestHelpMudlogActingBody` uses the real manager/provider and ordinary, switched-PC and switched-NPC commands. The game tests `TestMilestoneMudlog{Advance,Stable,Remort,AssassinPC,AssassinHire,Medusa,Loot}` invoke the registered special or actual game boundary. Their provider adapter isolates producer bytes/type/threshold and file-time ordering; it is not a claim of live session routing. Advance covers invis 0/34/40; loot also proves NPC silence. Baseline assertions fail with missing producers or original-owner help name and compile successfully.

Reproduce compiling green/revert/restore controls from the repository root:

```sh
python3 docs/fidelity/depth/handoff/2026-10-06-dp-1371-mudlog-milestones-controls.py --output /tmp/mudlog-milestones-controls
```

Eight producer controls plus the loot NPC classifier require named assertion failures and reject build failures. Retained evidence root: `~/Archives/darkpawns/oracle-runs/2026-10-06/dp-1371-mudlog-milestones-proofs/`.

## Oracle scope and retained frontier

Independent immortal observers cover switched help (before/during/after switch), advance, assassin refusal/hire, Medusa and live death-to-corpse attitude loot. The assassin observer deliberately uses BRF: the initial CMP transcript exposed the separately inventoried missing `set_hunting` CMP/file-false producer (src/utils.c:724). That anomaly is retained rather than repinned or normalized; this vehicle proves only the three BRF producers. Stable's unavailable-prototype error is unit-only.

The first remort comparison matched a refusal, not success. A follow-up load-in-room experiment closed the C connection during warmup (`load mob 4`, EOF). Both experiments are retained outside the scenario corpus; neither is a remort oracle proof. Remort remains unit-green through the registered successful handler, with no live oracle receipt claim. Diagnose the fixture/oracle anomaly separately; do not bless the false green or change the oracle in this train. No whole-handler or aggregate D7 completion claim. The PC-kender victim route remains a separate design/routing train. File-editor logical paths, report dollar processing and hunting remain separate frontiers.

Final proof-integration gates pass: fmt, build, vet, all tests, game tests, lint (0 issues), selected game/session race tests, fidelity-depth, fidelity-units (PASS=1370), string-census and diff check. All nine compiling revert triples pass. Each of the eight case commits also ran its own required gates. Combined census will run at the committed integration tip; its separate full and claims verdicts are required before merge. Preliminary uncommitted C captures are field evidence, not final-tip validation.
