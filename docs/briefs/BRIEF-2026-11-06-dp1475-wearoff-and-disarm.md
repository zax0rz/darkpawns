# Brief for DeepSeek: DP-1475, mob wear-off messages and the paladin disarm line

2026-11-06, by Claude for Zach to hand out. Two small fixes. Claude rechecked the first finding while writing this brief, and **the issue's description of it is wrong**; the ground truth below replaces it.

Read first: `AGENTS.md` and `docs/fidelity/RULEBOOK.md`, from `origin/main`. Work in `~/wt/cline-dp-1475`, on branch `cline/dp-1475-wearoff-and-disarm` cut from `origin/main`.

## Ground truth (grepped by Claude at `origin/main`, 2026-11-06)

### 1. Spell wear-off on a mob

**C** is `affect_update`, at `src/magic.c:431-455`:
- It walks `character_list`, so mobs and players alike, and each character's affects **in list order**.
- When an affect expires, it sends the wear-off message with `send_to_char(spell_wear_off_msg[type], i)` and then `"\r\n"`. That goes to the character itself, never to the room.
- It sends only when this is the last record of that spell: `!af->next || af->next->type != af->type || af->next->duration > 0`. Multi-record spells send once.
- For an ordinary mob, `send_to_char` goes nowhere, because there's no descriptor. **For a mob a wizard has `switch`ed into, it reaches the wizard.**

**Go** is `World.AffectUpdate`, the mob branch (`pkg/game/affect_update.go:85-112`):
- It sends `w.roomMessage(roomVNum, shortDesc + " " + wearOff)`, but only `if !mob.IsNPC()`. `MobInstance.IsNPC()` always returns `true` (`pkg/game/mob.go:821-823`), so that broadcast **can never fire**. Luna's PF-102 verdict called it invented output; it's actually dead code.
- **The real divergence:** a switched wizard never sees a mob's wear-off messages, where C shows them.
- Go already has the right delivery: `MobInstance.SendMessage` (`mob.go:987-998`) reaches a switched wizard and does nothing otherwise, exactly as C's `send_to_char` does.
- **A second divergence:** the mob branch ranges over the `mob.CustomData` map, so two affects expiring on the same tick come out in random order. C uses the affect list's order.

### 2. The paladin's failed disarm

**C** is `do_disarm`'s failure arm (`src/new_cmds2.c:243-247`), which the paladin special reaches through `do_disarm(ch, GET_NAME(vict), 0, 1)` (`src/spec_procs.c:562`):
- TO_CHAR: "You try to disarm $N but fail, tumbling to the ground in the process!"
- TO_VICT: "$n tries to disarm you but fails and falls flat on $s face **instead!**"
- TO_NOTVICT: "$n tries to disarm $N, but fails and falls flat on $s face!"
- Then `GET_POS(ch) = POS_SITTING`.

**Go** is `mobDisarm` (`pkg/game/spec_paladin.go:52`, failure arm at `:89-94`):
- Its victim line drops "instead!".
- It sends **no TO_CHAR line**. That's silent for an ordinary mob, but a switched wizard should see it.

## Do this

1. **Wear-off:**
   - Replace the dead room broadcast with `mob.SendMessage(wearOff + "\r\n")`, matching C's two `send_to_char` calls.
   - Keep C's last-record condition. Show how Go stores multi-record spells on a mob and that it sends once.
   - Iterate the mob's affects in C's order. If Go has no ordered affect list for mobs, report what order C's list has for the affects in your scenario, and make Go match it. Don't invent an order.
2. **Disarm:** compare the **whole** `mobDisarm` with C's `do_disarm` at subcmd 1 (`src/new_cmds2.c`), both arms. Fix the "instead!" and add the TO_CHAR lines, sent through the same `Act` path the rest of the function uses.
3. **Proof:**
   - **Wear-off:**
     - A scenario where a wizard switches into a mob that has a short spell (one the wizard can cast on it), waits for it to expire, and sees the wear-off line. It needs a C witness (R5j). The switched-actor scenarios from PR #2299 (`attitude-loot-switched-output-depth`, and those in `docs/fidelity/evidence/pf-102/`) show how to set up a switch.
     - A unit test for the ordering, with two affects expiring together.
   - **Disarm:**
     - A scenario where a paladin mob fails a disarm on a player, showing the victim line. The draw that picks case 5 and the disarm roll must both go your way, so choose the seed and stats deliberately, and say how.
     - If no seed makes it reachable within reason, a unit test citing C is the proof; say why.
   - Revert each fix, and its proof must go red (R5h).
4. **Census:** targeted, through `scripts/census.sh`: your new scenarios, every scenario that sends `switch`, and every scenario with a paladin mob or `disarm` (`grep -l -i disarm cmd/dp-oracle-diff/scenarios/*`).

## Rules

- **Gates:** run each on its own and check its exit status:
  - `go build ./...`
  - `go vet ./...`
  - `go test ./...`
  - `golangci-lint run ./...`
  - `make fmt`
  - `make deadcode-ratchet`
  - `make fidelity-depth`
  - `make expected-divergences-check`
- **C citations:** cite C from `src/`, grepped in your session (R5g).
- **Repository hygiene:** never switch branches in `/home/zach/darkpawns`, never `git stash`, and never stop processes by name.
- **Governing docs:** don't edit `RULEBOOK.md`, `north-star-1to1.md` or `AGENTS.md`.
- **No outside help:** don't invoke another agent, model or paid service.

Open a PR, ending its description with "Decisions for Zach". Then stop and report:
- both fixes;
- the scenarios and their witnesses;
- the census run directories;
- the revert proofs.

Claude reviews in Zach's session.
