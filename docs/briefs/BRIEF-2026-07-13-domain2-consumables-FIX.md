# BRIEF 2026-07-13 — Consumables (Domain 2) FIX pass — make the oracle go green

**Executor:** Kimi (continues your own Domain-2 work). **Branch:** continue on the existing
**`fix/consumables-domain`** (already pushed; your WIP is commit `019755c`). `git fetch && git checkout
fix/consumables-domain && git pull`, then add fix commits on top. Do NOT start a new branch.
Claude reviewed the first pass and ran the oracle you couldn't — this brief is the gap list.

## Why this is a FIX pass, not "done"

Your implementation is genuinely good — faithful C wording, clean `act()`, and the
prototype-mutation invariant holds (zero `Prototype.Values` writes, instance `GetValue`/`SetValue`
throughout — keep it that way). **But unit tests alone are not sign-off.** Claude ran the oracle and
it does NOT pass. The unit tests construct items in memory, so they miss the integration failures
below. Per the oracle-proof gate, this domain isn't done until the oracle is green.

## You CAN run the oracle — here's exactly how (this was the missing piece)

The C oracle binary is on this machine. Set the env var and run:
```
DP_ORACLE_BIN=$HOME/.openclaw/workspace/darkpawns-c-oracle/bin/circle \
  go run ./cmd/dp-oracle-diff -scenario consumables
```
It boots both the Go port and the C oracle, drives your scenario, and prints only the **diverging**
blocks. "Done" = the consumable probes show **no normalized divergence** (world/mob/score noise from
un-migrated commands is fine to leave — see below). Paste the before/after in your PR.

## Blocker #1 — eat/pour/fill can't resolve their target item (the real bug)

The oracle showed the consumable commands failing to find items that C finds:
```
[eat bread]         C: "You eat a loaf of bread." + "You are full."   Go: "You don't seem to have a bread."
[pour skin out]     C: "You empty a water skin."                      Go: "You can't find it!"
[fill skin fountain]C: "You gently fill a water skin from ..."        Go: "You can't find it!"
```
Notably `drink skin` did NOT diverge. The strong hypothesis: **your scenario depends on `get <item>
pack` (get-from-container), which is an un-migrated / known-buggy domain (DP-1091/1092)** — so the
items never actually land in the Go inventory, and only a *pre-held* item (the skin, briefly) resolves.

**Do BOTH:**
1. **Root-cause it.** Add a temporary trace (or a Go test that mirrors the scenario) to confirm whether,
   after `get skin pack` / `get bread pack`, the item is actually in `ch.Inventory`. If get-from-container
   is the culprit, the consumable code may be fine — but the domain still can't be proven until the
   scenario stops depending on it (next point). If the item IS in inventory and eat/pour still can't find
   it, that's a real bug in the consumable resolution (`ch.Inventory.FindItem` / `findObjectInRoomByName`)
   — fix it.
2. **Rewrite the scenario to ISOLATE consumables** (Blocker #2).

## Blocker #2 — the scenario is not a valid proof (isolate it)

`cmd/dp-oracle-diff/scenarios/consumables.txt` currently walks through wandering-mob rooms and diffs
`score`/`quit` — all un-migrated, all noise, plus it leans on get-from-container. Rebuild it to test
ONLY consumables, deterministically:
- **Use a consumable the newbie already HOLDS** (no `get` from a pack), or spawn one via an immortal
  setup char (`load obj <vnum>`), so item acquisition isn't a confound.
- **Prefer deterministic paths:** `eat <food>` (amount = val0), an **alcoholic** drink (deterministic
  amount), `pour ... out` (puddle), `fill`. AVOID relying on **water** drink numeric output — the amount
  is `number(3,8)` RNG (Tier-2); keep the water/RNG amount in the seeded Go unit test only.
- **Do not** diff `score`, `quit`, or walk through mob-populated rooms. Keep the probe local to one room.
- Green = the consumable probes show no normalized divergence.

## Blocker #3 — condition changes don't persist across save

`GainCondition` updates only `p.Conditions[]`, but the save path serializes the stale
`p.Hunger`/`p.Thirst`/`p.Drunk` fields (`pkg/game/save.go:207-209`). So eating/drinking changes the
in-session condition, then **loses it on save→reload** — half-defeating DP-1100. Fix by syncing on save:
either serialize from `Conditions[]` (`Hunger: p.Conditions[CondFull]`, etc.) or copy `Conditions[] →
Hunger/Thirst/Drunk` immediately before serialization. Add a save/reload round-trip test asserting a
post-eat fullness survives.

## Minor findings (fix while you're here)
- **Fountain depletion:** you guard depletion behind `ITEM_DRINKCON` ("fountains infinite"), but C
  `do_drink` decrements `GET_OBJ_VAL(temp,1)` unconditionally (act.item.c:1023). Match C — remove the
  guard (fountains just have large val1).
- **`SetWeight(negative)` footgun:** a negative arg silently clears the instance override and restores the
  **prototype** weight. In `DoPour`, `SetWeight(GetWeight()-amount)` can go negative if weight < amount →
  wrong. Clamp weight at 0 without resetting to prototype (mirror C `weight_change_object`, which floors at 0).
- **Drink amount clamp:** you clamp `amount` to ≥0; C allows it to go negative (thirst 26–40 + alcoholic).
  Match C (drop the clamp) unless you can show C never reaches it.
- **Werewolf TODO:** the corpse-rip branch references `DP-1102`, which is not the werewolf follow-up. Either
  file a real `[Fidelity]` follow-up issue for the werewolf savage-eat (mangled-flesh proto 19) and cite it,
  or leave a plain `// TODO` — don't cite an unrelated ticket.

## Keep intact (don't regress)
- The prototype-mutation invariant: NO `Prototype.Values` writes; instance `GetValue`/`SetValue` only.
  Keep the two-instance isolation test.
- Faithful C wording via `Act()`.
- The `TestPlayer.json` fixture was reverted (it was a test artifact) — do NOT re-commit condition changes
  to it; if a test mutates a fixture, fix the test to use a temp copy.

## Success criteria (this pass is done when ALL hold)
1. `DP_ORACLE_BIN=... go run ./cmd/dp-oracle-diff -scenario consumables` → **no normalized divergence** on
   the consumable probes (paste it). eat/pour/fill resolve their items and match C.
2. Isolated, deterministic scenario (no get-from-container dependency, no score/quit/mob-room noise).
3. Condition persistence fixed + a save/reload round-trip test.
4. Minor findings addressed; werewolf TODO cite fixed.
5. `go build ./... && go vet ./... && go test ./...` + gofumpt + lint green; instance-isolation test intact.

## Wrap-up
Commit onto `fix/consumables-domain`; push; open (or update) the PR with the **green** oracle report +
the save/reload test inline; STOP — Claude QAs against `origin/main` + `src/act.item.c` and merges.
Closes DP-1100 + DP-1101.
