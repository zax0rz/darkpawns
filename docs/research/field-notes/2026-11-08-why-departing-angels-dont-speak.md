# Why the departing conjured angels don't speak

2026-11-08, DeepSeek, for Zach. DP-1493. A field note: the cause found, the fix reported rather than shipped.

## The divergence

`spellmatrix-staff-olc-13-depth` diverges on every seed. When a Divine Intervention angel leaves mid-fight, Go prints one line C never prints:

```
 A Dominion Angel OBLITERATES you with his deadly hit!!
+A Dominion Angel states, 'My work here is done.'      <- Go only
 A Dominion Angel disappears in a flash of white light!
```

Both engines run the same code: `SPECIAL(conjured)` calls `do_say(mob, "My work here is done.", 0, 0)` then the vanish `act` (`src/spec_procs3.c:877-883`); Go's `specConjured` calls `w.NPCCommandDirect(me, "say", "My work here is done.")` then the same vanish `Act` (`pkg/game/spec_procs3.go:1263-1267`). So the vanish matches; only the say differs.

## The cause

C's `do_say` (`src/act.comm.c:759-821`) has exactly one silent early return for an NPC:

```c
if (GET_WIS(ch) == 0 || GET_INT(ch) == 0) {
    stc("You are too stupid to communicate with language!\r\n", ch);
    return;
}
```

(`src/act.comm.c:767-771`). The refusal is TO_CHAR only, so a descriptor-less mobile is simply silent. The only other early return is `PLR_FLAGGED(ch, PLR_NOSHOUT)` (`:773-777`), and `PLR_FLAGGED` is `!IS_NPC`-guarded (`src/utils.h:234`), so it cannot fire for a mobile.

Go's NPC say, `npcSay` (`pkg/game/npc_command.go:253-268`), has **no such gate**: it always emits the `$n states, '...'` line. That is the whole divergence.

## Evidence

- A plainly-loaded mob 85 (`load mob 85`, uncharmed) **does** print the say in C, in both engines. So the mob's prototype stats are not the issue in the ordinary case.
- Adding the gate to Go's `npcSay` turns `spellmatrix-staff-olc-13-depth` green (`result: no normalized divergence`); reverting it turns the same scenario red again (`result: normalized divergence detected`, fingerprint `~dpclock pulse 20 [actor]`). The gate is the difference.
- Conclusion: the summoned angel's effective `GET_INT`/`GET_WIS` is 0 at departure in **both** engines, so C silences it and Go (lacking the gate) does not.

## Why the angel's INT is 0

It is not a conjuring quirk. The staff matrix includes **mind bar** (`spell.mind-bar.matrix-staff-olc`), and a staff casts on everyone in the room, so the angels take it too:

- `SPELL_MIND_BAR` applies `APPLY_INT` with modifier `-18` (`src/magic.c:1369-1371`).
- `affect_total` clamps the result at 0: `GET_INT(ch) = MAX(0, MIN(GET_INT(ch), i))` (`src/handler.c:354`).
- A mob's INT starts at 11 plus at most 7 (`src/db.c:1034-1035`, `:1057`), so −18 always reaches 0.
- `create_mobile` touches no abilities (`src/new_cmds2.c:588-620`), which is why a plainly loaded mob 85 speaks.

So it is an ordinary rule — a mobile that a mind bar has numbed cannot speak — that Go never ported.

## The fix

The gate is **global**, exactly as in C: every C path a mobile speaks through reaches `do_say` (specials `src/spec_procs.c:621-712`, `:1826`; mobprogs `src/mobprog.c:159`, `:608-624`; hunting `src/graph.c:282`, `:293-295`; shopkeepers `src/shop.c:80`, `:128`; Lua `src/scripts.c:1285`). Go's `npcSay` now refuses before any output when `GetWis() == 0 || GetInt() == 0` (`pkg/game/npc_command.go`).

Two consequences are handled in the same PR:

- The stat-less test mobs (literals that bypass the parser's `Int = 11`/`Wis = 11` defaults, `pkg/parser/mob.go:398-404`) are given real stats, so they keep speaking. No special-case code.
- Lua (`pkg/game/world_bridge.go:335`) and the shopkeepers (`pkg/game/shop_npc.go:26`) already reach `npcSay`, so they inherit the gate.

A check of `lib/world/mob/` found no real prototype whose INT or WIS is 0, so no shipped mob is silenced by this.


## Citations

`src/act.comm.c:759-821` (do_say), `src/act.comm.c:767-771` (the INT/WIS gate), `src/utils.h:234` (PLR_FLAGGED), `src/spec_procs3.c:877-883` (conjured default branch), `src/spells.c:1170-1215` (spell_divine_int), `pkg/game/npc_command.go:253-268` (npcSay), `pkg/game/spec_procs3.go:1263-1267` (specConjured), `pkg/parser/mob.go:398-404` (mob stat defaults).
