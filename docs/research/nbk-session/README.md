# Mickey and Mallory session capture (2026-10-24)

Evidence for the blog post `website-astro/src/content/blog/mickey-and-mallory.md`.
The post's game logs are rendered from the raw captures in `captures/`.

## What ran

The original C server from this repository's `src/` and `lib/` (HEAD
`29994e5`), copied to a scratch directory and built with gcc 13.3 on x86-64:

```bash
gcc -g -O0 -fcommon -w -std=gnu89 -Isrc/lua/include -Isrc -c <each libdp source + circle.c>
gcc -g -O0 -fcommon -o bin/circle obj/*.o src/lua/lib/liblualib.a src/lua/lib/liblua.a -lz -lcrypt -lm
```

`src/lua/include/` is not in the tree, so `lua-include/` holds minimal Lua 4.0
declarations written to link against the prebuilt `src/lua/lib/*.a`.

This is **not** the reference oracle binary (`DP_ORACLE_BIN`, built from
`dp-oracle-seam` with the determinism seam). It is unseeded, so a rerun draws
different dice. The captures are illustrations, not oracle evidence.

## What was changed (`stand-in.patch`)

`SPECIAL(mickey)` and `SPECIAL(mallory)` (`src/spec_procs.c:1269-1422`) are
declared in `src/spec_assign.c:102-103` but never assigned to a mob, and no
world file has a Mickey or Mallory entry. The scratch copy adds:

- Mickey Knox at vnum 7969 (the `MICKEY` define), replacing the black-cloaked
  wanderer. Flags SPEC, SENTINEL, ISNPC, STAY_ZONE, HUNTER, AGGR24. HUNTER is
  required for `set_hunting()` from Mallory's proc to move him; SENTINEL kept
  him in place until the scene began.
- Mallory Knox at vnum 7972 (unused), flags SPEC, SENTINEL, ISNPC, AGGR24.
- AGGR24 (the "attitude" flag) is an inference from Zach's memory of being
  looted by them, and of their saying something after the kill. It makes
  `attitude_loot()` (`src/fight.c:1108`) loot the corpse, junk anything worth
  150 or less, wear the rest and gossip a brag. `sceneA`, `sceneB` and
  `outtake-bulette` were captured before AGGR24 was added.
- `ASSIGNMOB(7969, mickey)` and `ASSIGNMOB(7972, mallory)`, and a zone 79
  load line for 7972. Zone 79 scatters 7900-7998 to random rooms on reset
  (`src/db.c:2113-2129`), so an implementor `transfer`red both onto the canyon.

Names, levels, hit dice, damage dice and flags are invented stand-ins. Every
line of output the mobs and the fight produce comes from the unmodified C.

The scratch `lib/etc/players` baseline was removed so the first character
(Mcclusky) became the implementor that staged each scene; Wayne is a mortal
advanced to level 20 and given steel plate and a long sword.

## Capturing

`tools/mud.py` drives a telnet session and records raw server bytes, ANSI
escapes intact, as JSON events (`{"t", "out"}` and `{"t", "in"}` for typed
lines). `tools/multi.py` runs the implementor and Wayne concurrently;
`tools/stage.sh` is the implementor's setup script. Passwords in `captures/` are
replaced with `<password>`.

Each scene ran on a fresh boot:

| Log | Scene | Rendered from |
|---|---|---|
| `captures/sceneA.json` | Wayne walks west into Mickey | input 13 (`west`) to "You are dead" |
| `captures/sceneB.json` | Wayne attacks Mallory; Mickey hunts him | input 12 (`look`) to "You are dead" |
| `captures/outtake-bulette.json` | a bulette kills Wayne mid-speech | input 14 (`west`) to "You are dead" |
| `captures/hunt-wayne.json`, `hunt-witness.json` | Wayne's wimpy makes him flee three times; Mickey follows each time and kills him off-screen from the witness | |
| `captures/loot-wayne.json`, `loot-witness.json` | wimpy 0: Mickey kills Wayne, loots, wears his plate, gossips | |
| `captures/peaceful-wayne.json` | Mickey in a peaceful room (4874, The Dragon's Claw): "I'll tell you later" and vanishes | |
| `captures/mallory-wayne.json`, `mallory-witness.json` | Wayne attacks Mallory; Mickey hunts in; Mallory gets the kill and loots | |
| `captures/dt-wayne.json`, `dt-witness.json`, `dt-imp.json` | Wayne walks south off 11369 into the death trap 11376; the witness on the rim hears nothing; the implementor's first command in 11376 fires the `dump` proc (`dts_are_dumps`, `src/config.c:89`) and destroys Wayne's gear (`dt-imp` is from a second boot of the same take) | |

The witness (Scagnetti, a level 10 mortal) was made invisible by the
implementor so Mickey's `CAN_SEE` would pass him over. `tools/take.sh` boots a
fresh server per take; `tools/stage2.sh` stages Wayne, both mobs and the
witness.

```bash
python3 tools/ansi2html.py captures/sceneB.json 12 "You are dead"
```

The post's section 4 logs, as Zach trimmed them (whole rounds only):

```bash
python3 tools/ansi2html.py captures/hunt-wayne.json 13 "You are dead" > out/hunt.html
python3 tools/trim_rounds.py out hunt "165H " "164H " "141H " "119H " "53H "
python3 tools/ansi2html.py captures/peaceful-wayne.json 14 "vanishes." > out/peaceful.html
python3 tools/trim_rounds.py out peaceful
python3 tools/ansi2html.py captures/mallory-wayne.json 13 "You are dead" > out/mallory.html
python3 tools/trim_rounds.py out mallory "248H "
```

Earlier takes were discarded: one had Wayne log in beside Mickey, one walked
past Wayne's own corpse from the previous take, and two were lost to night
(the canyon is dark, and Mickey's proc needs `CAN_SEE`).

## Related finding (not fixed here)

The Go port's `specMallory` (`pkg/game/spec_procs.go:1256`) diverges from the
C: it drops the default "You stupid biiitch!" branch (rolls 6-10), replaces
"giggles madly" with a line that is not in the C source, capitalises "i", and
omits the `set_hunting(mickey, ...)` call. `specMickey` omits the
non-combat approach entirely. Both procs are unassigned in Go as in C, so no
player can reach either divergence today.
