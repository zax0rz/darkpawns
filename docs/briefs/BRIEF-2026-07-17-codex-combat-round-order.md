# BRIEF (codex) — combat-round order parity: mirror C's `combat_list` (DP-combat-death keystone)

**Owner:** codex. **Gate:** Claude runs `combat-death` red→green + full sweep under `DP_CLOCK` (workers have no `DP_ORACLE_BIN`). **Branch off `main`, one PR.**
This is the keystone for the whole combat-death campaign — the death ladder and damage tiers can't be gated until multi-round combat is faithful, because the fight desyncs long before anyone dies. Draw-sensitive engine work; match C exactly.

## The divergence (established RED)
Scenario `cmd/dp-oracle-diff/scenarios/combat-death.txt` (`hit trainee`, then N×`~dpclock pulse 20` = one `PULSE_VIOLENCE` round each). The initiating `hit` is green, but **every pumped round diverges on combatant ORDER**:
- **Round 1, C:** `[mob swings at you][you swing at mob]` — the mob acts first.
- **Round 1, Go:** `[you swing at mob][mob swings at you]` — the player acts first.

Each swing draws RNG, so the reversed order desyncs the shared stream → different miss/hit variants every round → the whole fight (and who dies, when) cascades apart. **Root cause = the order combatants act within a round.**

## C truth — `combat_list` prepend, walked head-first
`set_fighting` (`fight.c:207-217`) **prepends** the newly-fighting char to the global `combat_list`:
```c
ch->next_fighting = combat_list;   /* :214 */
combat_list = ch;                  /* :215  → ch is now the HEAD */
FIGHTING(ch) = vict;
GET_POS(ch) = POS_FIGHTING;
```
During `hit trainee`, `hit()` calls `set_fighting` for the **player** (`fight.c:1408`, "Fighting starts here") and then, in the same damage path, for the **retaliating mob** (`fight.c:1445 set_fighting(victim, ch)`). Prepend semantics ⇒ after `hit`, `combat_list = [mob, player]` (mob prepended last = head). `perform_violence` (`fight.c:1898`) walks `combat_list` **head → tail**, so **the most-recently-engaged combatant swings first** — here, the mob. Each char in the list swings at its own `FIGHTING` target. `stop_fighting` removes the char from `combat_list`.

**Net rule to reproduce: within a round, combatants act in reverse-engagement order (last to enter the fight swings first). Removal happens on stop-fighting/death/flee.**

## Go today — map iteration, initiator-first, no ordering
- `StartCombat` (`engine.go:206-250`) creates **one** `combatPairs[{attacker,target}]` entry and calls `attacker.SetFighting(...)` (`:237`) then `defender.SetFighting(...)` (`:238-240`). No reciprocal pair; **no ordered structure** anywhere.
- `PerformRound` (`engine.go:314-345`) ranges `ce.combatPairs` (**a Go map — non-deterministic order**), builds `edges` from the fixed slice `[]Combatant{pair.Attacker, pair.Defender}` (Attacker/**player-first**), dedups by name, then processes edges in that order. For the 2-combatant fight this yields `[player→mob, mob→player]` — **player swings first** (reversed vs C), and with 3+ combatants the map adds nondeterminism on top.
- `redirectAttacker` (`engine.go:~606`) is the only other place fighting is (re)established.

## The fix — an ordered engagement list mirroring `combat_list`
1. Add an engine field: an ordered list of currently-fighting combatants, e.g. `combatOrder []Combatant` (or `[]string` names resolved via a lookup). This is the `combat_list` analog — **per-combatant, not per-pair.**
2. **Prepend** a combatant to `combatOrder` at every point the engine registers it as newly fighting — mirroring each C `set_fighting` call site: the `attacker.SetFighting` and `defender.SetFighting` in `StartCombat` (`:237`/`:238-240`), and `redirectAttacker`. Prepend only if not already present (a char is in `combat_list` at most once). Order matters: whatever sets fighting **last** ends up at the head. In `StartCombat` that's attacker-then-defender ⇒ after a player-initiated `hit`, head = defender(mob) → mob acts first, matching C.
3. **Remove** a combatant from `combatOrder` on `StopCombat`/death/flee (mirror `stop_fighting`). Check every path that clears `FIGHTING` today.
4. Rewrite `PerformRound` to iterate `combatOrder` **head→tail** (drop the map-range + `edges` slice). For each combatant still fighting, resolve its target via the existing `findFightingTarget` and run its `processCombatPair` swing(s). Keep the existing `seen`/dead-target guards and `numAttacks` loop (`GetAttacksPerRound`) unchanged — only the **iteration source and order** change.
5. Preserve per-swing draw order inside `processCombatPair`/`performOneHit` (do not touch the swing internals — combat-swing is green and must stay green).

**Determinism:** `combatOrder` must be a slice/list, never map iteration. With one fight this fixes the reversed order; with multiple fights it also removes the nondeterminism.

## Landmines
- The mob's retaliation `set_fighting` in C happens **during** `hit` (so the mob is in `combat_list` before the first pumped round). Ensure Go registers the defender's engagement during `hit trainee` too (it does today via `StartCombat` `:238-240`) so round 1 already has both in `combatOrder` in the right order.
- Do NOT create a reciprocal `(mob→player)` pair to "fix" order — the fix is the ordered combatant list, not pair duplication.
- `combatOrder` mutates mid-round (deaths remove combatants); snapshot under the lock the same way `PerformRound` already snapshots `edges`, and skip combatants whose position is already `PosDead` or who stopped fighting.

## Acceptance (Claude-gated, from a PR-branch worktree)
1. `--scenario combat-death` → `no normalized divergence` (every round + the death + the aftermath look match C).
2. `--scenario combat-swing` stays green (single synchronous `hit` — unaffected, but prove it).
3. Full committed sweep under `DP_CLOCK` stays green.
4. `DP_CLOCK` unset ⇒ no behavioral change to normal play beyond deterministic round ordering.
