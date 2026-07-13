# BRIEF 2026-07-13 — Domain 2: Consumables (eat / drink / pour / fill) — instance-safe consumable transactions

**Executor:** Kimi (mechanical, well-scoped — the primitives already exist; this is assembly + a C-faithful
port). **Context:** this is Domain 2 of the session-vs-game refactor
(`docs/research/drafts/2026-07-13-session-game-refactor-plan.md`, §3 table row "Consumables"). Foundations
F0a (act(), #297), F0b (command gates, #298) and Domain 1 (Observation, #299) are merged. Claude reviews
against `src/act.item.c` before merge. **Oracle-proof gate applies — a green build is NOT sign-off; each
behavior/wording change is proven by an oracle red→green run OR (for cross-instance state) a targeted test
that asserts the C invariant.** Player-facing output text is first-class fidelity (`[[darkpawns-oracle-proof-gate]]`).
**Branch:** `fix/consumables-domain` off current `main`. **One PR** (or a tight series).
**Fixes:** DP-1100 (O21 eat/drink), DP-1101 (O22 pour/fill). Read both Linear issues.

---

## The headline (read first): NEVER write `Prototype.Values`

The whole point of this domain is to kill the **prototype-mutation corruption class** for consumables (same
bug family as the merged `zap` fix). Today the session eat/drink/pour handlers write `item.Prototype.Values[...]`,
which mutates **every object instance sharing that vnum** server-wide. **Every** food/drink value read/write
in the canonical ops MUST go through the instance-safe boundary:
- `item.GetValue(idx)` / `item.SetValue(idx, v)` (`pkg/game/object.go:432-455`) — copy-on-write per instance.
- NEVER `item.Prototype.Values[idx]` for a mutation. This is the acceptance-critical invariant.

## The good news: the primitives already exist — this is assembly, not greenfield

Reuse, do not reinvent:
- **Conditions:** `GainCondition(p *Player, cond, delta int)` (`pkg/game/limits_condition.go:10`) with
  `CondFull` / `CondThirst` / `CondDrunk`; read via `p.Conditions[CondX]`. This is C's `gain_condition` /
  `GET_COND`. (It already handles the >20/>40 clamps and drunk messaging plumbing — verify against C, use it.)
- **Liquid name table:** `drinks[]` (`pkg/game/item_helpers.go:249`) = C's `drinks[]`.
- **Liquid affect table:** `pkg/game/liquids.go` — `drink_aff[LIQ_x][DRUNK|FULL|THIRST]` equivalent, LIQ_*
  indices matching `src/structs.h`.
- **Puddle:** `w.CreateObject(20, room)` + `puddle.SetTimer(2)` (pattern already at
  `pkg/game/mobprogs.go:349-351`) = C's `read_object(20, VIRTUAL)` puddle on pour-out.
- **act():** the merged `Act()` for every player-facing message (F0a). Do not hand-substitute `$`-codes.

## What to build — canonical ops in `pkg/game`, faithful to `src/act.item.c`

Replace the stubs (`pkg/game/act_item_stubs.go:3-15` — `EatFood`/`DrinkLiquid` literally `return 1`) and the
prototype-mutating session/game copies with real transactions. Route the session commands to them and
**delete** the session reimplementations. Match C exactly (line refs are the spec):

### `DoDrink` (C `do_drink`, act.item.c:895-1032) — handles `drink` and `sip` (SCMD_DRINK/SCMD_SIP)
Port the full gate ladder and effects, verbatim wording:
- resolve obj in inventory then room (`on_ground`); must be DRINKCON or FOUNTAIN else "You can't drink from
  that!"; on-ground DRINKCON → "You have to be holding that to drink from it.".
- drunk+thirst → "You can't seem to get close enough to your mouth." + room "$n tries to drink but misses $s
  mouth!"; full+thirst → "Your stomach can't contain anymore!"; thirst>40 → "If you drink any more, you'll
  explode!".
- `GetValue(1)==0` → "It's empty.".
- DRINK: "You drink the %s." (drinks[val2]); amount = `(25-thirst)/drink_aff[liq][DRUNK]` when DRUNK-affect>0,
  else `number(3,8)` (**RNG — see proof note**). SIP: room "$n sips from $p.", char "It tastes like %s.",
  amount=0.
- weight change; `GainCondition` DRUNK/FULL/THIRST by `drink_aff[liq][X]*amount/4`; vampire branch (msg
  "The vampirism in your body is not satiated by mere %s..."); "You feel drunk." (drunk>10), "You don't feel
  thirsty any more." (thirst>20), "You are full." (full>20).
- poison (`GetValue(3)`) → "Oops, it tasted rather strange!" + room "$n chokes and utters some strange
  sounds." + AFF_POISON affect (duration amount*3).
- decrement: `SetValue(1, val1-amount)`; if now empty → `SetValue(2,0)`,`SetValue(3,0)`, clear drinkcon name,
  and if vnum==20 (puddle) extract the object.

### `DoEat` (C `do_eat`, act.item.c:1035-1156) — handles `eat` and `taste` (SCMD_EAT/SCMD_TASTE)
- resolve food in inventory (or room only if AFF_WEREWOLF); TASTE on a DRINKCON/FOUNTAIN → delegate to
  `DoDrink` SIP; non-FOOD & level<GOD → "You can't eat THAT!"; FULL>40 → "You are too full to eat more!".
- EAT: "You eat $p." / room "$n eats $p."; TASTE: "You nibble a little bit of $p." / room "$n tastes a little
  bit of $p.".
- amount = `GetValue(0)` if EAT else 0; vampire branch else `GainCondition(FULL, amount)`; "You are full."
  (full>20); poison (`GetValue(3)` && level<IMMORT) → "Oops, that tasted rather strange!" + room "$n coughs
  and utters some strange sounds." + AFF_POISON (duration amount*2).
- EAT → extract food; TASTE → `SetValue(0, val0-1)`, and if 0 → "There's nothing left now." + extract.
- **Werewolf corpse-rip branch (act.item.c:1058-1085)** — the container-with-val3 savage-eat that spawns
  mangled-flesh proto 19: port it if straightforward; if the proto-19 spawn is fiddly, keep the branch
  structure, TODO it, and file a linked follow-up rather than silently dropping it. (Low frequency; don't let
  it block the core.)

### `DoPour` (C `do_pour`, act.item.c:1159-1335) — handles `pour` and `fill` (SCMD_POUR/SCMD_FILL)
- POUR: arg1 = from (inventory, must be DRINKCON); FILL: arg1 = to (inventory DRINKCON), arg2 = from (room,
  must be FOUNTAIN) — port both grammars and their exact error strings (see C).
- from empty → "The $p is empty.".
- POUR "out" → room "$n empties $p." / char "You empty $p.", empty the from-con, **create a puddle**
  (`CreateObject(20, room)`, copy liquid val2/val3, `SetTimer(2)`), clear from val2/val3. (This is the missing
  puddle in today's Go.)
- to==from → "A most unproductive effort."; different liquid in target → "There is already another liquid in
  it!"; target full → "There is no room for more.".
- POUR msg "You pour the %s into the %s."; FILL msg "You gently fill $p from $P." / room "$n gently fills $p
  from $P.".
- transfer: name_to_drinkcon, `SetValue(2,...)` liquid type, amount = to.val0-to.val1, decrement from,
  clamp when from runs low, poison OR (`to.v3 = to.v3 || from.v3`), weight changes. **All via GetValue/SetValue.**
- **Register `fill`** — the gate row already exists in F0b's `command_gates.tsv` (`fill POS_STANDING`); you
  only add the handler wiring + `registerCommand("fill", …)`. (C: `fill` shares `do_pour` with SCMD_FILL,
  interpreter.c:443.)

## Wiring & deletions
- Route session `cmdEat`/`cmdTaste`/`cmdDrink`/`cmdSip`/`cmdPour`(+new `cmdFill`) to the canonical ops
  (session handlers in `pkg/session/eat_cmds.go` + `pkg/session/cmd_misc.go`); **delete** the session copies
  that write `Prototype.Values` (`eat_cmds.go:62-109`, `:211-255`) and the prototype-mutating
  `doPour` (`pkg/game/item_consumable.go:8-110`).
- Fix the bridge that hard-codes `"pour"` with no fill branch (`pkg/game/act_other_bridge.go:83-84`).
- These ops mutate state + emit act() messages; **no structured RoomView / WS-schema work** (that was
  Observation-specific). The existing vitals/state push handles client updates — do not add or change any WS
  schema.

## Oracle proof (required — the gate)
Add `cmd/dp-oracle-diff/scenarios/consumables.txt` driving a mortal through deterministic probes and paste the
red→green reports:
```
eat bread          # amount = val0 (deterministic); "You eat $p." + "You are full." when applicable
drink <alcoholic>  # deterministic amount ((25-thirst)/aff); "You drink the <liquid>."
drink <empty-con>  # "It's empty."
pour <con> out     # "You empty <p>." + puddle created
fill <con> <fountain>
```
- **RNG caveat:** `drink` of a **non-alcoholic** liquid (water) uses `number(3,8)` for the amount → the exact
  resulting FULL/THIRST delta is **Tier-2 (RNG)**, not deterministically oracle-provable until C `random.c`
  is ported. So drive the oracle with **eat** (amount=val0) and an **alcoholic drink** (deterministic amount)
  for the numeric paths; cover the water/RNG amount path with a **Go unit test** using injected/seeded RNG.
  Do NOT rely on water-drink numeric output in the oracle diff.
- **Instance-safety (the corruption) — required Go test:** spawn **two instances of the same food/drink
  vnum**, consume/pour ONE, assert (a) the other instance's values are unchanged AND (b) `Prototype.Values`
  is unchanged. This is the regression guard for the headline invariant (cross-instance state isn't visible in
  one command's output, so it must be a unit test — same pattern as the zap fix's `TestZap_...`).
- **Wording:** every message above must match C verbatim (first-class fidelity). The oracle diff checks this;
  the normalizer only masks prompts/ANSI/RNG/timestamps.

## ⛔ Guardrails
- **NEVER** write `Prototype.Values` — instance `GetValue`/`SetValue` only.
- Match C exactly — read `src/act.item.c:895-1335`; reuse the existing Go primitives (GainCondition, drinks[],
  liquids.go, CreateObject(20)); don't invent wording or effects.
- Use `Act()` for messages; don't hand-substitute `$`-codes or use the dumb `broadcastToRoom`.
- Don't touch command gates (F0b owns them; `eat`/`drink`/`pour`/`fill` rows already exist), the WS schema,
  or other domains.
- Don't modify the C oracle clone / `DP_SEED`.
- Build gate green: `go build ./... && go vet ./... && go test ./...` + lint + gofumpt.

## Success criteria (PR shows ALL)
1. Canonical `DoEat`/`DoDrink`/`DoPour`(+fill) in `pkg/game`, faithful to `act.item.c`; stubs replaced;
   session prototype-mutating copies deleted; ops route through them.
2. **No `Prototype.Values` writes anywhere in the consumable path** — verified by the two-instance regression
   test (instance + prototype isolation).
3. `fill` command wired + registered (gate already in the F0b table); pour-out creates a puddle.
4. FULL/THIRST/DRUNK conditions actually change (via `GainCondition`); poison branches port faithfully.
5. **Oracle red→green** for the deterministic probes pasted inline; water/RNG amount covered by a seeded unit
   test; wording matches C.
6. Build gate green.

## Wrap-up
Commit; push; open PR with the red→green oracle report + the two-instance state test inline; STOP — Claude
reviews against `origin/main` + `src/act.item.c` (esp. that no `Prototype.Values` writes remain, the puddle/
poison/condition effects match, and wording is verbatim) and merges. Closes DP-1100 + DP-1101. **Next domain
after this:** Object/inventory (get/drop/put), then Equipment.
