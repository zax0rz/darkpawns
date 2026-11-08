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

## Why this is reported, not shipped

The brief says: *"If the fix would change how every NPC talks, stop after step 1 and report."* Adding the gate does that: it silences **every** mobile whose effective INT or WIS is 0, whatever the caller. Concretely, the gate is faithful but has a wider blast radius than the one scenario:

- It breaks `TestHandleDeathDyingSpecArmPuffSaysLine` and `TestLuaCharTablePlayerFields`, whose mobs are stat-less struct literals (`parser.Mob{...}` bypasses the parser's `Int = 11`/`Wis = 11` defaults, `pkg/parser/mob.go:398-404`).
- The precise reason a real prototype that defaults to INT/WIS 11 ends up with 0 effective stats at conjure time was not pinned down this session (the oracle scenarios that would `stat` a summoned angel repeatedly failed to capture the stat: the actor enters combat with the angels, and the angel was not visible to a separated statter).

So the fix needs scoping — either gate only the paths whose C site is `do_say`, plus make the synthetic test mobs carry real stats, or address the stat default — and that scoping is Zach's/Claude's call. The five `docs/fidelity/depth/cast.tsv` rows still cite DP-1493; they cannot be upgraded to the real cause until the fix lands.

## Citations

`src/act.comm.c:759-821` (do_say), `src/act.comm.c:767-771` (the INT/WIS gate), `src/utils.h:234` (PLR_FLAGGED), `src/spec_procs3.c:877-883` (conjured default branch), `src/spells.c:1170-1215` (spell_divine_int), `pkg/game/npc_command.go:253-268` (npcSay), `pkg/game/spec_procs3.go:1263-1267` (specConjured), `pkg/parser/mob.go:398-404` (mob stat defaults).
