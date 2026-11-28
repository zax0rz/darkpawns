# Special-procedure audit — 2026-11

One row per C `SPECIAL(...)` definition in `src/spec_procs.c`,
`src/spec_procs2.c` and `src/spec_procs3.c`, audited by hand against its Go
port(s). Audit only: no code was changed. Brief:
`docs/briefs/BRIEF-2026-11-27-kimi-special-proc-audit.md`.

## Count (differs from the brief)

The brief says 121 specials. `grep -c '^SPECIAL('` does give 121 lines
(41 + 50 + 30), but 10 of those are forward declarations (lines ending in
`;`), and 2 real definitions are invisible to that pattern because they are
written `SPECIAL (name)` with a space:

- `src/spec_procs.c:1072` — `SPECIAL (mini_thief)`
- `src/spec_procs2.c:282` — `SPECIAL (couch)`

So the true number of definitions is **113** (40 + 1 in spec_procs.c,
42 + 1 in spec_procs2.c, 29 in spec_procs3.c), and the TSV has exactly one
row per definition: **113 rows**.

## Columns

- `c_name`, `c_file:line` — the C definition, cited from grepped lines (R5g).
- `assigned_in_c` — `mob:`/`obj:`/`room:` vnums from `ASSIGNMOB`/`ASSIGNOBJ`/
  `ASSIGNROOM` in `src/spec_assign.c`, or `no` when the special is only
  declared/defined but never assigned (31 specials). `dump` additionally
  covers every death room when `dts_are_dumps` is set
  (`src/spec_assign.c:638-642`).
- `go_player_lane` — the `RegisterSpec("<name>", …)` call (file:line), or
  `none` (`ninelives`, `mirror`).
- `go_actor_lane` — the `RegisterActorSpec("<name>", …)` call, or `none`.
- `strings`, `gates`, `draws`, `lists`, `effects` — the five checks from the
  brief. `ok` means byte/behavior parity; anything else names the exact
  mismatch with C citations.
- `summary` — `faithful` (85), `minor` (21), `stand-in` (7).

Helper semantics used to judge `strings`: Go `sendToChar` appends `\r\n`
(pkg/game/spec_procs.go:89); `Player.SendMessage`/`SendLiteral` are verbatim;
`Act` appends `\r\n` to each rendered line exactly like C's `act()`
(pkg/game/act.go:626+). A C literal ending `\r\n` therefore maps to
`sendToChar` without the terminator; a C literal ending `\n\r` (LF-CR) only
matches if the Go side sends it verbatim — several mismatches are exactly
this.

## Stand-ins (7), sorted by reachability

Assigned in C (reachable in principle):

1. **field_object** (`obj:50,52`) — Go rewrites the special: no `dice(VAL0,
   VAL1)` roll (fixed `level/2+1`, no draw), players only (C hits every
   person in the room), kill at PosDead (-11) instead of C's `GET_HIT <= 0`,
   hardcoded `SpellPoison` at the mob's level instead of the object's
   spell/level. Mitigating: C's body only runs at `cmd == 0`, and object
   specials are only ever invoked from `command_interpreter` with a real
   command (`src/interpreter.c:1427-1470`), so the special is inert in C;
   with the shipped world values (`0 0 0 0`, lib/world/obj/0.obj:222+) C
   would roll `dice(0,0)` anyway. Go's damage arm is likewise unreachable
   (every Go dispatch passes a real command), so nothing player-facing
   diverges today — but the port is a rewrite, not a port.

Never assigned in C (unreachable in both engines unless someone assigns
them; Go registers all of these):

2. **no_move_down** — the brief's example. C sends three `act()` lines and
   lets hunting players pass (`src/spec_procs3.c:316-335`); Go prints one
   pre-rendered `"<mob> blocks your way down."` to the room
   (`pkg/game/spec_procs3.go:303`) and drops the `HUNTING` gate.
3. **black_undead_knight** / 4. **red_undead_knight** — the brief's second
   example. Go rolls `number(0,3) == 0` where C rolls `!number(0,2)`;
   pre-renders the battle cry with runtime mob names instead of C's fixed
   `"$n sees the knight dressed in red and gives a battle cry!"`; sets
   target/fighting pointers instead of calling `hit()`; and hates vnums
   11470/11471, which do not exist — C hates 18401/18402 (the actual knight
   vnums, lib/world/mob/184.mob:15,27). Shared helper
   `specUndeadKnight` (`pkg/game/spec_procs.go:1203`).
5. **fearface** — Go body is a stub that always returns false
   (`pkg/game/spec_procs4.go:545`); C's saving-throw/`do_flee` fear loop is
   entirely missing.
6. **ninelives** — no Go port at all.
7. **mirror** — no Go port at all (object special; C teleports between
   rooms 14496 and the mirror's room).

## Cross-cutting patterns (the "anything that didn't fit" list)

- **`isname` → `isnameWithAbbrevs` drift.** C's exact-token `isname` became
  the prefix-matching `isnameWithAbbrevs` in two ports: `medusa`
  (`src/spec_procs2.c:1561`) and `mortician` (`src/spec_procs3.c:838`), each
  calling `isname` directly. Each lets Go accept abbreviations C rejects.
  Three rows previously listed here — `assassin`, `eq_thief`,
  `elements_load_cylinders` — were false positives: they go through C's
  `get_char_room`/`get_obj_in_list_vis`, which match with
  `isname_with_abbrevs` themselves (`src/handler.c:877`, `:1341`), so Go's
  prefix match is faithful there. (`quan_lo`, `eviltrade`, `con_seller`
  correctly use exact-match `isCName`/EqualFold.)
- **`strings.TrimSpace` vs C's `skip_spaces`.** C strips leading whitespace
  only, so a trailing space fails its `strcmp`/`strcasecmp` gates
  ("couch ", "horse ", "door ", "itoh "); Go's `TrimSpace` accepts them.
  Cosmetic but byte-visible in gate behavior. `con_seller` and
  `portal_to_temple` do it right (leading-only trim).
- **`extract_obj` → `MoveObjectToNowhere`.** `dump`'s room cleanup, `fido`,
  and `roach` move the consumed object to nowhere instead of freeing it
  (C frees). Not player-visible; the object leaks in nowhere.
  `couch`/`eq_thief`/`elements_minion` use the real `ExtractObject`.
- **`giveTattoo` drops C's `affect_total(ch)`** after `tattoo_af(ch, TRUE)`
  (`src/spec_procs2.c:942`) — all four tattoo specials.
- **Missing actor lane:** every movement guard has an NPC-actor lane except
  `castle_guard_east`, so an NPC mover (e.g. a charmed pet) passes east
  where C yells and hits (`src/spec_procs2.c:1945-1957`).
- **`wall_guard_ns` player lane** returns false where C delegates to
  `fighter()` on `cmd || sleeping || fighting` (`src/spec_procs2.c:2268-2270`);
  the actor lane has the delegation, the pulse path does not.
- **Approved divergences (not counted as errors):** `cuchi` level backdoor
  and `pray_for_items` immortality grants removed per DP-1373 (Zach);
  `evillead` inert per Zach 2026-11-18 (C never fetches `obj`);
  `start_room`'s figure-line ending follows the -O0 reference oracle over
  the C source's overlapping-sprintf bytes (DP-1307).
- **LF-CR (`\n\r`) endings:** `whirlpool` normalizes C's `\n\r` to `\r\n`
  with a transport-canonicalization comment; `dracula`'s mesmerize line and
  `enter_circle`'s "Enter what?" go through `sendToChar`/`SendMessage` and
  emit `\r\n` where C emits `\n\r`. `brass_dragon` and the element rooms
  preserve the C bytes verbatim.

## Corrections after review

Ember's review (#3105) checked rows against C and found five cells wrong;
fixed 2026-11-28, each re-verified against the cited C line:

- `assassin`, `eq_thief`, `elements_load_cylinders` — the claimed
  `isname`-exact-token drift does not exist: C's `get_char_room` and
  `get_obj_in_list_vis` match with `isname_with_abbrevs`
  (`src/handler.c:877`, `:1341`), so Go's prefix match is faithful. Those
  checks are now `ok`.
- `tattoo4` — Go's extra `IS_NPC` gate is real textually but inert:
  `Player.IsNPC()` always returns false (`pkg/game/player_stats.go:442`).
  Kept, marked inert.
- `suck_in` — the `strcasecmp` mismatch is inert: C lowercases the token
  with `one_argument` (called at `src/spec_procs.c:2289`; `LOWER` at
  `src/interpreter.c:1276`) before comparing, and Go's `oneArgument`
  lowercases too (`pkg/game/act_comm.go:749`).

No `summary` value changed (still 85 faithful / 21 minor / 7 stand-in).

## Bookkeeping (verified before push)

```
$ awk -F'\t' 'NR>1' special-procs-audit.tsv | wc -l
113
$ awk -F'\t' 'NR>1 {print $NF}' special-procs-audit.tsv | sort | uniq -c
     85 faithful
     21 minor
      7 stand-in
$ awk -F'\t' 'NR>1 && $3=="no"' special-procs-audit.tsv | wc -l
31
```

Every row has 11 tab-separated columns; `c_name` values are unique.
