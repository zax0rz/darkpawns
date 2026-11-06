# Port-completion survey, 2026-11-05

Claude, opening the port-completion phase (kickoff
`~/KICKOFF-CLAUDE-2026-11-05-port-completion.md`). Measured at `origin/main`
`51e98b4f7`, latest tag `v0.1.8`, no open PRs. Every number below came from a
command run or a file read today. Anything marked *static* is a code reading,
not oracle-proven.

Three parts: an inventory of everything known to be open, classified; a draft
definition of "done", with where each line stands today; and a short list of
proposed work streams. Zach picks the targets and the order.

Classes used throughout:

1. **known divergence**: C cite in hand, Go differs;
2. **suspected divergence**: needs an oracle scenario first;
3. **proof gap**: probably faithful, but nothing can fail on it;
4. **tooling or oracle gap**: the harness can't see this surface;
5. **not port work**: features, ops, website. Parked.

## 1. Headline numbers

| Measure | Today | Source |
|---|---:|---|
| C command table entries | 508 (323 commands, 183 socials, 2 abbreviation stubs) | `scripts/gen_reachability.py` |
| ...with at least one scenario that *mentions* them | 506 | `scripts/gen_scenario_coverage.py` (static: mentioned, not passing, not able to fail) |
| ...registered but not wired up | 4 | same |
| Depth-manifest cases (command manifests) | 7,021; 6,928 proven or delegated, 4 blocked, 77 excluded, 12 approved divergences | `scripts/gen_fidelity_depth.py` |
| "Actionable completion" headline | 99.9% | same |
| ...of which hollow (setup refused, so the case never ran) | at least 269 spell-item rows plus 3 transfer rows (DP-1456) | witness audit, 2026-11-01 |
| Surface inventories left *out* of that headline | 19 blocked families in `surface-inventory.tsv`, 18 blocked shop branches, 1 blocked mob-command call | read today |
| Random output mutations the full census catches | 23 of 40 (57.5%) | PF-102 (pkg/game, pkg/combat, pkg/command only) |
| Packages whose tests have been audited | 1 of 33 (`pkg/combat`) | PF-103 to PF-106 |
| Ledger rows `needs-verification` | 13, 12 of them older than v0.1.8 | `EVIDENCE_LEDGER.tsv` |
| Open Linear issues, DP team | about 150; roughly 55 are port work | Linear, today |

The 99.9% and the 57.5% are both true at once. The manifests count cases
someone wrote down and proved. The mutation sample asks what happens to a
message nobody chose. The gap between them is the work of this phase.

**Tooling note.** The depth report's headline leaves out
`surface-inventory.tsv` and `surface/*.tsv`. Those files hold the lifecycle
surfaces (death and corpse teardown, rent, save and load, regeneration, weather,
pulse order, connect and disconnect), and that is where returning players have
found most bugs (field note 2026-10-25). A "done" number should include them.

## 2. Known leads carried in, verified today

| Lead | Class | What I checked today |
|---|---|---|
| **PF-105, merged mob wait counters** | 2 | Still present. C's `WAIT_STATE` writes `ch->wait` only (`src/utils.h:462-464`). `GET_MOB_WAIT` is written only by the Lua bridge (`src/scripts.c:2017`) and the combat decrement (`src/fight.c:1979`). C's flee and cast gates read `GET_MOB_WAIT` (`src/act.offensive.c:377`, `src/spell_parser.c:845`). In Go, player skills write the mob's one field (`pkg/command/skill_commands.go:1792-1794`), and flee and cast read it (`pkg/game/flee.go:62`, `pkg/game/world_bridge.go:1193`). Needs an oracle scenario: bash a wimpy mob, then watch it try to flee. |
| `TestCombatTranscript_Golden` checks Go against Go | 3 | Still at `pkg/combat/combat_transcript_golden_test.go:57`. |
| `TestProcessCombatPair_MobWithWaitStillAttacks` | 2/3 | Still at `pkg/combat/engine_test.go:1115`. Its correct value depends on PF-105. |
| `combat.AttitudeLoot` duplicate | 2 | Still present: `pkg/combat/fight_core.go:1087`, called at `:632`. The game layer has `pkg/game/attitude_loot.go:12`, citing `src/fight.c:1108`. Same shape as the `counter_procs` duplicate fixed in #2277. |
| **Strength-zero override** | 2 | Go sets damage to 1 whenever strength is 0, after any attack type (`pkg/combat/formulas.go:552-554`). C does it only in the normal weapon arm, not the disembowel arm beside it (`src/fight.c:1876-1885`). **Strength 0 is reachable in C**: `affect_total` clamps with `MAX(0, GET_STR(ch))` (`src/handler.c:357`). Still to check: which Go attack types reach this function. |
| **`give bread me`** | 1 | **Confirmed, static, and part of a class.** C's `give_find_vict` looks the target up with `get_char_room_vis` (`src/act.item.c`), which returns the giver for "me" or "self" (`src/handler.c:1285`), so C says "What's the point of that?". Go's `doGive` uses two hand-written helpers, `FindMobInRoom` then `FindPlayerInRoom` (`pkg/game/item_transfer.go:603`, `:620`, `pkg/game/item_helpers.go:344-360`). Neither handles "me" or "self", neither checks visibility, neither handles `2.guard`, and they search mobs before players instead of in room order. So Go prints `NOPERSON`. The faithful resolver already exists: `ResolveCharInRoom` (`pkg/game/target.go:49`, citing `src/handler.c:1276-1300`). This is the stand-in shape again: a faithful path, and beside it a paraphrase on a hot path. **Not yet in Linear.** |
| Petitioner coin draw | 4 | Still `blocked` (`mob.mp-sound-petitioner-coin`, `docs/fidelity/depth/mobprog.tsv`). |
| Five named scenarios that miss their command | 3 | `door-basic`, `serpent-depth`, `weather-sky-transitions`, `skills-practice` and `shoot-mob-hit-depth` (PF-102). Not yet repaired. |
| 15 merge and 4 delete rows, combat test audit | 3 | On hold in `docs/research/test-audit/pilot-combat/settled.tsv`. |
| Proposed AGENTS.md gate line (`make fidelity-depth`) | — | Waiting for Zach's approval. |

~~Something else turned up while I read: the kickoff cites "R1–R5i" and "R5i",
but `RULEBOOK.md` ends at R5h. Either a rule is still in draft, or the kickoff
has a typo. Zach to say which.~~

**Correction (same day):** R5i exists. It was added on 2026-11-05 in PR #2271
("a fixture that silences a surface needs a green scenario that unsilences
it"). The kickoff was right. I had read the rulebook from the main checkout's
local `main`, which was behind `origin/main`. Every other number in this
survey was measured at `origin/main`.

## 3. Inventory of open items

Linear is the canonical tracker. GitHub issues mirror it (for example, #2269 is
DP-1456), so items are listed once, by DP number.

### 3.1 Known divergences (class 1)

These have a C cite and are waiting for a fix.

- **Combat and death:**
  - DP-1372, autogold loot names the killer;
  - DP-1369, milestone reward draw (check whether #2277 closed it);
  - DP-1296, combat position messages bypass `act()`;
  - DP-1331, `mag_damage` floor, zero damage, nil world;
  - DP-1319, mob scripts get no target against a mob;
  - DP-1390, mob hitroll and damroll leave out equipment.
- **Objects and spells:**
  - DP-1447, worn and carried object timers never tick;
  - DP-1341, enchant weapon;
  - DP-1377, vampire daylight water;
  - DP-1446, `sendAffectRoom` uses hand-pasted names;
  - DP-1131, stacked room objects.
- **Prompts and output framing:**
  - DP-1308, a missing blank line before the prompt;
  - DP-1306, the normal prompt after the pager prompt;
  - DP-1227, the long listings need C's pager;
  - DP-1229, `prompt` is not an alias of `display`.
- **Connect and session lifecycle** (stop tier):
  - DP-1407, `dc` skips the link-lost message;
  - DP-1408, the connection-limit refusal is silent;
  - DP-1352, the idle timer;
  - DP-1425, DP-1424 and DP-1423, crash-save, `PLR_IT` and clan `is_playing`;
  - DP-1443, `save_char` re-equips (R3).
- **Commands:**
  - DP-1361 and DP-1348, `track` and `olc` are unported;
  - DP-1393, `restore` numbered targets;
  - DP-1171, the `hide` daytime guard;
  - DP-1437, `peek` draw for immortals;
  - DP-1076, `do_start` ordering;
  - DP-1292, extra-description edits don't persist.
- **Specs:** DP-1455 and DP-1413, Mickey and Mallory. These are post-launch content, recorded divergences.
- **New:** `give … me/self` and its lookup class (§2).
- **Logging:** DP-1340, the remaining mudlog sites.

### 3.2 Suspected divergences (class 2)

Each needs an oracle scenario before a fix brief.

- PF-105, the wait counters.
- The `AttitudeLoot` duplicate.
- The strength-zero override.
- DP-1321 (in progress): after a flee, commands still answer as if fighting.
- DP-1354, combat teardown with several attackers.
- DP-1345, `god-skill-keys` at seed 2.
- DP-1177, newbie stat-roll draws.
- DP-1444, the room light counter.
- DP-1322, a skull rider in the wrong zone.
- DP-1313, bulletin boards piling up on the live server.

### 3.3 Proof gaps (class 3)

Probably faithful, but nothing can fail on these.

- **DP-1456:** 269 spell-item rows and 3 transfer rows run with refused setup.
- **DP-1398:** `steal-depth` runs in a peaceful room.
- **The 17 green mutations of PF-102,** including the five named scenarios.
- **`TestCombatTranscript_Golden`.**
- **The 19 merge and delete rows** from the combat test audit.
- **DP-1231:** `compare`'s output path.
- **DP-1326:** prompts are uncertified.
- **The surface inventory's 19 blocked families.** The lifecycle ones matter most: death cycle, save and load, regeneration, weather, pulse order, descriptor lifecycle.
- **18 blocked shop branches.** Several are C undefined behaviour (signed overflow) and need an R1a decision from Zach, not a test.
- **`entry.identity-consumers`.**

### 3.4 Tooling and oracle gaps (class 4)

- **No check that setup succeeded.** This is the root of DP-1456, and it has now happened twice: peer-login in September, the spell matrix in October.
- **DP-1439:** long `DP_CLOCK` pulses lose Go output.
- **DP-1202:** wait-decrement isn't frozen, so mid-combat timing can't be reproduced.
- **DP-1367:** a failed port bind lets the harness drive another worker's server.
- **DP-1357:** `c-missing` uses an inverted predicate, so it hides port gaps.
- **The petitioner coin:** the draw-parity mode reports agreement while the arms differ.
- **The normalizer folds line terminators and strips color** (R1b). Only `keep-ansi` fixtures and C byte-checks cover them.
- **The depth report leaves out the surface inventories** (§1).
- **The mutation population script missed `Dream` and `DreamTravelFn`** (PF-102).

### 3.5 Not port work (class 5), parked

- **Committed to v0.1.9:**
  - finish the docs project;
  - remove Grapevine.
- **Not committed to a release:**
  - client and transport: the Option B project, Mudlet DP-1301 and others;
  - the MCP and agent layer;
  - the DP-Goat research items;
  - the website database UI (F-* and P-* items);
  - the archive (DP-1351);
  - ops: DP-1020, DP-1328, DP-1421, DP-1285, DP-1388;
  - flaky tests: DP-1454, DP-1441, DP-1445, DP-1363;
  - the luaedit regressions DP-1394 and DP-1376, which are urgent but are editor tooling, not game bytes.

Approved divergences (DP-1399, DP-1379 and the others) are settled and need no
work.

### 3.6 Ledger rows still `needs-verification`

- **Port-fidelity:**
  - PF-001, PF-002, PF-003 (July);
  - PF-024 (September);
  - PF-105 (today).
- **Other tracks:** MA-002, MA-003, MA-004, GM-001, RO-001, RO-002, RO-004, RO-005, all from the summer.

All but PF-105 are older than v0.1.8.

## 4. A draft definition of "done"

Each line is measurable, and each has a reading for today. The targets are
Zach's to set. The middle column only says what can be measured.

| # | The port is done when… | Today |
|---|---|---|
| D-1 | every C command and spec proc has a Go path reached by at least one green scenario that **fails when the path is broken** (R5h) | 506 of 508 commands are *mentioned* in a scenario. "Fails when broken" has never been measured per command. PF-102 shows that a name isn't proof. |
| D-2 | the depth headline counts **every** manifest, including the surface inventories, and has no hollow rows | 99.9% on command manifests. The surface inventories sit outside it, with 38 blocked items, and at least 272 rows are hollow (DP-1456). |
| D-3 | a pre-registered random one-letter mutation sample of player-output functions is caught at or above a target rate, **in every package that prints** | 23 of 40 (57.5%) in pkg/game, pkg/combat and pkg/command. pkg/session and pkg/spells haven't been measured. |
| D-4 | every package with game tests has passed a blind test audit, and its fix rows are mutation-proved | 1 of 33 packages (`pkg/combat`). Its F rows are fixed; 19 merge and delete rows are on hold. |
| D-5 | no known divergence (class 1) is open, except approved or recorded ones | About 30 open, listed in §3.1. |
| D-6 | no `needs-verification` port-fidelity ledger row is older than one release | 4 of 5 are. |
| D-7 | no setup step in any scenario is refused unless the scenario says it expects that | Not measured. No check exists (the root of DP-1456). |

D-7 is new to this draft. Without it, D-1 to D-3 can be satisfied by scenarios
that never ran. Zach may prefer to fold it into D-2.

## 5. Proposed work streams, in order

These are suggestions for Zach. No briefs are written yet.

1. **Make setup failures visible, then re-prove the hollow rows** (DP-1456,
   D-7, D-2).
   - **Agent:** Sol. It ran the witness audit and knows the cluster.
   - **First brief:** a harness check that fails a scenario when a setup line
     gets `Huh?!?` or a refusal, unless the scenario declares it. Then repair
     the 52 spell-item scenarios and `transfer-depth`.
   - **Smallest census that can fail:** the 52 `spellmatrix-*` scenarios plus
     `transfer-depth`, and one deliberately broken control that must go red.
   - **Tier:** stop tier (harness), so Zach merges.
   - **Why first:** every later measurement trusts greens. This one stops
     greens from coming from scenarios that never ran.

2. **Verify, then fix, the known leads.**
   - **Agents:**
     - Luna, for PF-105, because the field split crosses packages (R5c);
     - DeepSeek through Cline, for the test rebuilds (the golden transcript and
       the wait test). It did the combat fix phase cleanly;
     - MiMo, for the `give` lookup class audit: every in-room character lookup
       that doesn't route through `ResolveCharInRoom`.
   - **First brief:** PF-105's oracle scenario alone (bash a wimpy mob, then
     flee; a mob caster casts after a bash). The fix brief follows only if
     it goes red.
   - **Smallest census:** the new scenario, plus the existing `flee-*`,
     `bash-*` and mob-cast scenarios.

3. **Expand the oracle where PF-102 found it blind.**
   - **Agent:** Luna. It ran the probes and the run directories are its own.
   - **First brief:** make each of the 17 green probes go red, starting with
     the five named scenarios, by fixing the scenario, not the game. Re-run
     the same mutation to prove it.
   - **Smallest census:** each repaired scenario, under its probe mutation.
     Each must turn red.

4. **Extend the random mutation sample to pkg/session and pkg/spells** (D-3).
   - **Agent:** Luna, with the same pre-registered method and a population
     script fixed for selector calls.
   - **First brief:** freeze and hash the populations, commit the predictions,
     and draw 40 per package.
   - **Census cost:** one full census per probe, so about 80 full censuses.
     Zach may prefer 20 per package.
   - **Order:** after stream 3, so the baseline isn't measured against
     scenarios we already know are blind.

5. **Test audit of pkg/spells** (D-4).
   - **Agents:** MiMo and DeepSeek blind, then a Sol pass, as for combat.
   - **First brief:** the combat brief, retargeted.
   - **Census:** none. It's read-only, and its mutations run in scratch
     worktrees.
   - **Order:** it can run alongside any of the above. It touches no shared
     files.

The v0.1.9 commitments (finish the docs, remove Grapevine) run beside these
streams and gate the release, not the port.

## Decisions for Zach

- The targets for D-1 to D-7, and whether D-7 belongs in the definition.
- Whether the depth headline should include the surface inventories (a tooling change).
- ~~R5i: a draft rule, or a typo in the kickoff?~~ (Settled: R5i exists; see §2.)
- An R1a ruling on the shop branches that rest on C's signed overflow.
- Whether to approve the proposed `make fidelity-depth` gate line for AGENTS.md.
- The order of the work streams, and which to start.
