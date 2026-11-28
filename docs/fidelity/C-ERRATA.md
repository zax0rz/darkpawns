# C errata: bugs in the original, with what was meant

The Classic port reproduces what the original build showed players, bugs
included (RULEBOOK R1, R1a). This file lists the C bugs where the intent is
clear, so a later edition can offer the correction (the way Crispy Doom
builds switchable fixes on top of Chocolate Doom's exact vanilla behavior)
without anyone having to rediscover them. **Nothing here changes Classic.**

An entry needs: the C lines, what players saw, what was evidently intended,
and the evidence for the intent. A bug without clear intent doesn't belong
here; it stays a plain divergence question. Zach rules on each entry.

| # | Where (C) | What players saw | What was meant | Evidence for intent | Ruled | Found |
|---|---|---|---|---|---|---|
| E-001 | `lua_raw_kill`, `src/scripts.c:1231,1253`, calling `raw_kill`, `src/fight.c:534` | Every Lua `raw_kill(vict, killer, type)` death leaves the plain corpse, "The corpse of X is lying here." `scripts.c` declares `raw_kill` with three parameters, but the real function takes two (victim, attack type), so C reads the killer pointer as the attack type: 0 when there is no killer, an unmatched pointer value otherwise. That holds on the 32-bit original build and on the x86-64 oracle alike. | The script's `type` argument picks the corpse wording, as `make_corpse` does for every other death (`src/fight.c:280-361`), and the undead-dust branch (`src/fight.c:574-577`). | The wrapper reads and passes `type` (`src/scripts.c:1244,1253`), and every caller names a specific death: `raw_kill(..., pyros, SPELL_HELLFIRE)` and `raw_kill(pyros, NIL, SPELL_DISINTEGRATE)` (`daemonic_focus.lua:193,210`), `raw_kill(ch, me, SPELL_PETRIFY)` (`medusa.lua:22`). Only scripts under `lib/world/scripts/*/archive/` call it (medusa, breed_killer, daemonic_focus), so Classic players can't reach it. | 2026-11-28: Classic matches C (plain corpse); the correction is recorded here | Neon, darkpawns#3107 (`death-route-lua-raw-kill-depth`) |
