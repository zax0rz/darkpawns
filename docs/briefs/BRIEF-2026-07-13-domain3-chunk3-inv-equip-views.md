# BRIEF 2026-07-13 — Domain 3 Chunk 3: `inventory` / `equipment` view fidelity (Kimi)

**Executor:** Kimi. **Branch:** base off current `main` (chunks 1+2 merged) — `git fetch && git checkout main
&& git pull && git checkout -b fix/inv-equip-views`. **Reviewer/merger:** Claude, vs `origin/main` +
`src/act.informative.c`, gated by the oracle. Closes **DP-1102** (O12). This is the LAST chunk of Domain 3;
it's a display-fidelity fix — mechanical, but the exact bytes matter.

**Oracle-proof gate:** green build/test is NOT sign-off. Done = the `object-inventory`-style oracle probes for
`inventory` and `equipment` show **no normalized divergence** vs the C oracle. You CAN and MUST run the oracle
(command in §5). Player-facing wording/format is first-class fidelity — spacing and labels count.

---

## 0. What's wrong today (both in pkg/session/cmd_inventory.go)

- **`cmdEquipment`** ranges a Go **map** (`for slot, item := range equipped`) → **nondeterministic slot order**;
  wrong header ("You are wearing:" — C says **"You are using:"**); wrong labels (`"%-10s: %s"` with the Go slot
  name) and wrong empty-state.
- **`cmdInventory`** prints one short-desc per item with **no grouping** — C collapses identical items into one
  line with a count. Header/empty-state also diverge.

Neither goes through a game-layer view; both just `s.sendText(...)`. **The WS client is decoupled** (it gets
inventory via the separate `VarInventory` subscription), so this chunk touches ONLY the command text output —
do NOT redesign any WS/MsgState schema or rendering boundary. Just fix the two commands' text to match C.

---

## 1. `equipment` — port C `do_equipment` (src/act.informative.c) verbatim

```c
send_to_char("You are using:\r\n", ch);
for (i = 0; i < NUM_WEARS; i++)          // FIXED C order, i = 0..21
  if (GET_EQ(ch, i)) {
    if (CAN_SEE_OBJ(ch, GET_EQ(ch,i))) { send_to_char(where[i], ch); show_obj_to_char(eq, 1 /*shortdesc*/); }
    else                               { send_to_char(where[i], ch); send_to_char("Something.\r\n", ch); }
    found = TRUE;
  }
if (!found) send_to_char(" Nothing.\r\n", ch);
```

Rules:
- Header **exactly** `"You are using:\r\n"`.
- Iterate in **C WEAR order**, printing only occupied slots. Each line = the `where[i]` label (below, fixed
  width, trailing spaces included) immediately followed by the item short-desc + `\r\n`
  (or `"Something.\r\n"` if the viewer can't see it — `CAN_SEE_OBJ`).
- If nothing is worn: **` Nothing.\r\n`** (note the LEADING SPACE — different from inventory's `Nothing.`).

**`where[]` — the 22 labels, C `WEAR_` index order (src/constants.c), copy trailing spaces EXACTLY:**

| idx | C WEAR_ | where[] label (with trailing pad) |
|-----|---------|-----------------------------------|
| 0 | LIGHT | `<used as light>      ` |
| 1 | FINGER_R | `<worn on finger>     ` |
| 2 | FINGER_L | `<worn on finger>     ` |
| 3 | NECK_1 | `<worn around neck>   ` |
| 4 | NECK_2 | `<worn around neck>   ` |
| 5 | BODY | `<worn on body>       ` |
| 6 | HEAD | `<worn on head>       ` |
| 7 | LEGS | `<worn on legs>       ` |
| 8 | FEET | `<worn on feet>       ` |
| 9 | HANDS | `<worn on hands>      ` |
| 10 | ARMS | `<worn on arms>       ` |
| 11 | SHIELD | `<worn as shield>     ` |
| 12 | ABOUT | `<worn about body>    ` |
| 13 | WAIST | `<worn about waist>   ` |
| 14 | WRIST_R | `<worn around wrist>  ` |
| 15 | WRIST_L | `<worn around wrist>  ` |
| 16 | WIELD | `<wielded>            ` |
| 17 | HOLD | `<held>               ` |
| 18 | THROW | `<held>               ` |
| 19 | ABLEGS | `<worn about legs>    ` |
| 20 | FACE | `<worn on face>       ` |
| 21 | HOVER | `<hovering near head> ` |

**The Go↔C slot-mapping table (IMPLEMENT THIS EXACTLY — do not invent your own order):** Go's `EquipmentSlot`
enum (pkg/game/equipment.go) is a different order/set. Build a fixed slice that yields the Go slot for each C
WEAR index in order, so you iterate C order over Go's storage:

| C idx | Go slot |
|-------|---------|
| 0 LIGHT | `SlotLight` |
| 1 FINGER_R | `SlotFingerR` |
| 2 FINGER_L | `SlotFingerL` |
| 3 NECK_1 | `SlotNeck1` |
| 4 NECK_2 | `SlotNeck2` |
| 5 BODY | `SlotBody` |
| 6 HEAD | `SlotHead` |
| 7 LEGS | `SlotLegs` |
| 8 FEET | `SlotFeet` |
| 9 HANDS | `SlotHands` |
| 10 ARMS | `SlotArms` |
| 11 SHIELD | `SlotShield` |
| 12 ABOUT | `SlotAbout` |
| 13 WAIST | `SlotWaist` |
| 14 WRIST_R | `SlotWristR` |
| 15 WRIST_L | `SlotWristL` |
| 16 WIELD | `SlotWield` |
| 17 HOLD | `SlotHold` |
| 18 THROW | (no Go slot — skip) |
| 19 ABLEGS | (no Go slot — skip) |
| 20 FACE | (no Go slot — skip) |
| 21 HOVER | (no Go slot — skip) |

- The **singular legacy** Go slots `SlotFinger`, `SlotNeck`, `SlotWrist` and the **invented** `SlotEar`,
  `SlotShoulder`, `SlotBack` have **no C WEAR equivalent**. First check whether the equip system actually
  populates them for real world items (grep `wearBitForPosition` / `EquipForPlayer` and how `.obj` wear-flags
  resolve to slots). **If real items land in `SlotFinger`/`SlotNeck`/`SlotWrist` (not the R/L pair), STOP and
  ask Claude** before guessing — the singular↔dual reconciliation is a real ambiguity, not yours to invent. If
  those singular/invented slots are vestigial (never populated), the table above is complete; note that in the PR.

---

## 2. `inventory` — port C `do_inventory` + `list_obj_to_char` (src/act.informative.c)

```c
send_to_char("You are carrying:\r\n", ch);
list_obj_to_char(ch->carrying, ch, 15, TRUE);   // groups identical items; empty → "Nothing.\r\n"
```

- Header **exactly** `"You are carrying:\r\n"`.
- **Grouping:** walk the carried list; for each **visible** item, group by the tuple **(vnum, extra_flags,
  weight, short_description)** — C `oc_onlist` matches on all four. Identical items collapse into one node with
  a `count`.
- **Render (C `oc_show_list`):** a single item (count 1) prints its short-desc; a stacked group (count > 1)
  prints a count prefix then the short-desc. **Port `oc_onlist` / `oc_get_node` / `oc_add_front` /
  `oc_show_list` faithfully — do NOT paraphrase the format.** `do_inventory` passes `mode = 15`; that selects
  the flag combination `oc_show_list(head, ch, mode&8, mode&4, mode&2, mode&1)`. The exact prefix/padding/weight
  columns are fiddly — replicate the C `sprintf` lines exactly and let the **oracle be the byte-level arbiter**
  (iterate until the stacked and single-item lines match C). `mode & 16 == 0` ⇒ short-description (not the room
  long-desc).
- **Empty:** `"Nothing.\r\n"` (NO leading space — contrast §1's ` Nothing.\r\n`).

Reuse, don't reinvent: `pkg/game/look.go:658 listObjToChar` already renders room contents — model the grouping on
it (or generalize it) rather than writing a second stacking impl if it already groups. Confirm its grouping key
matches C's (vnum+extras+weight+text); if the room one diverges too, fix both to the C key.

---

## 3. Move the logic to the game layer (light touch)

Follow the Domain pattern: put the faithful builders in `pkg/game` (e.g. `DoInventory(ch)` / `DoEquipment(ch)`
emitting via `SendMessage`/`Act`, alongside the other `Do*` in item_transfer.go), and have session
`cmdInventory`/`cmdEquipment` delegate (like `cmdGet`/`cmdDrop` already do). Keep `markDirty(VarInventory)` etc.
intact. Do NOT build a new Result-DTO or WS schema — text output only.

---

## 4. Invariants

- No `Prototype.Values` reads/writes — use instance `GetValue` if you need obj values (weight via `GetWeight`).
- Faithful strings via C source, not memory. Trailing spaces in `where[]` are load-bearing.
- Don't regress the WS `VarInventory` subscription path — leave it alone.

---

## 5. Oracle proof — how

```
DP_ORACLE_BIN=$HOME/.openclaw/workspace/darkpawns-c-oracle/bin/circle \
  go run ./cmd/dp-oracle-diff -scenario inv-equip
```
Author `cmd/dp-oracle-diff/scenarios/inv-equip.txt` (model on `object-inventory.txt`; `[fixture]` `quiet-mobs` +
`spawn-obj` to seed items in room 8004). Probe must exercise:
- `inventory` with **duplicate items** (spawn ≥2 of one vnum, `get` them) to prove **grouping/count**, plus a
  distinct single item; and an **empty** inventory case (`drop all` then `inventory` → "Nothing.").
- `equipment` after **wearing several items across different slots** whose Go-enum order differs from C WEAR
  order (e.g. wield a weapon + wear body + hold light) to prove the **fixed C order + labels**; and an empty
  case → " Nothing.".
Cite the vnums you use in a scenario comment. Green = no normalized divergence on the inventory/equipment lines.
If the normalizer masks trailing whitespace, great; if not, match C's padding exactly. Paste before→after.

---

## 6. Success criteria (done when ALL hold)

1. `equipment`: header "You are using:\r\n", fixed C WEAR-order slots with exact `where[]` labels, "Something."
   for unseen, " Nothing." when empty — oracle-green.
2. `inventory`: header "You are carrying:\r\n", identical items grouped with C's count format, "Nothing." when
   empty — oracle-green.
3. Slot mapping implemented from §1's table; singular/invented-slot ambiguity resolved (or escalated to Claude,
   not guessed).
4. Logic delegated to game layer; session commands thin; WS `VarInventory` path untouched.
5. No `Prototype.Values`. `go build ./... && go vet ./... && go test ./...` + gofumpt + golangci-lint green.

---

## 7. Wrap-up — COMMIT AND PUSH (don't leave work uncommitted in the shared clone)

```
git add -A && git commit -m "fix(views): C-faithful inventory + equipment display (DP-1102)"
git push origin fix/inv-equip-views
```
Open a PR with the **green** oracle report inline. STOP for Claude — Claude QAs vs `src/act.informative.c`, runs
the oracle independently, and merges. Do NOT close Linear (Claude does it). Closes **DP-1102** and completes
Domain 3.
