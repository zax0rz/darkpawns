# The Archived Lua Scripts of Dark Pawns (C 2.2 oracle)

Read-only research report on `lib/scripts/{mob,obj,room}/archive/*` plus `lib/scripts/room/30/*`
in `/home/zach/darkpawns-c-oracle`. No files were modified in any repository; nothing was
committed, pushed, or fetched from the network.

## Executive summary

1. 165 archived scripts survive: 115 mob, 14 obj, 36 room, plus 2 in `room/30/`. All are
   frozen at SVN rev 1424 (the bulk archive commit), 2008-04-27, author `jravn`
   (`.svn/entries`); `room/30/*` is one commit later, rev 1523, 2008-06-20.
2. They are not merely "unattached" — the world they were written for is **partly deleted**.
   Zones 14 (the old Grey Keep), 17 (fire ants), 22 (ziggurat), 30 (patterns), 37 (Wight
   Island), 53 (Mist Keep), 61, 62, 68, 101, 102 (Serapis/citadel) and 120 have no `.zon`,
   `.wld`, `.mob` or `.obj` file at all, and the object ranges 12xx/14xx were replaced.
3. So most scripts are **dead letters twice over**: no `Script:` line attaches them, and many of
   the vnums they reference no longer exist. Of the 155 vnums named in the scripts' own comments,
   **79 (51%) are absent** from `lib/world`; the ones named only in code are worse.
4. `create_event()` is the backbone of every timed script (conversations, ceremonies, quest
   chains, delayed tells) and it **does not exist in this build**: it is commented out of
   `cmdlib` at `src/scripts.c:1616`, with its body commented out at `src/scripts.c:247-300`.
   Every script that calls it stops with a Lua "call a nil value" error.
5. Four functions the scripts rely on were never written in C at all: `mxp`, `msp`,
   `get_group_lvl`/`get_group_pts`/`skill_group` (training), `get_spell`/`get_dye`, plus the
   global `vrix_teleport`. Only `room/pattern_tport.lua` still resolves its `dofile`.
6. Where the target *does* exist, the C special procedure is usually already there: the
   archive duplicates `backstabber`, `brain_eater`, `no_get`, `medusa`, `teleporter`,
   `teleport_victim`, `rescuer`, `werewolf`, `conjured`, `breed_killer`, `zen_master`,
   `dragon_breath`, `cityguard`, `cleric`, `citizen`, `janitor`, `thief`, `snake`,
   `eq_thief`, `troll`, `quan_lo`, `clerk`, `recruiter`, `elemental_*`, `cuchi`, `stableboy`,
   `identifier`, `tattoo1-4`, `horn`, `carrion`, `pet_shops`.
7. The most valuable material is the **Grey Keep daemon-summoning ritual** (bane/valoran/pyros/
   ceremonial_scroll/daemonic_focus/horn/keep_*/barrier) — a multi-room, multi-player
   ceremony with a reagent recipe, a spell-assignment mini-game and a real chance of death —
   and the **golem crystal-mining economy** (11700-11708), the earliest and only scripted
   production line in the archive.
8. The second most valuable is the **character-creation questionnaire** (`creation.lua`) and
   the **starter-kit/bond-certificate economy** (`clerk.lua`, `banker.lua`), which together
   describe an on-boarding system that no longer exists in any form in the C tree.
9. There is one genuine **privilege escalation** artefact: `cuchi.lua` grants `LVL_IMPL` (level
   40) to a character literally named `Orodreth` when they pat a pet cat — a builder's own
   character hard-coded into a script.
10. The scripts are also a jokebook (puff, bhang, seiji, mime, singingdrunk, tyr) and a
    snapshot of an earlier city roster: **Mist Keep** and **Kir Morthis** are named throughout
    and appear nowhere in the current world files — Kir Morthis is now Alaozar (zone 212).

## How to read the tables

Paths are relative to `/home/zach/darkpawns-c-oracle/lib/scripts/`. Line cites are to the
oracle tree. Trigger names are the exact `fname` strings passed to `run_script()`
(`src/scripts.c:1718`); each is gated by a bitmask on the mob/room/obj:

* mob: `MS_BRIBE`(2) `MS_GREET`(4) `MS_ONGIVE`(8) `MS_SOUND`(16) `MS_DEATH`(32)
  `MS_ONPULSE_ALL`(64) `MS_ONPULSE_PC`(128) `MS_FIGHTING`(256) `MS_ONCMD`(512)
  (`src/structs.h:663-672`)
* room: `RS_ENTER`,`RS_ONPULSE`,`RS_ONDROP`,`RS_ONGET`,`RS_ONCMD` (`src/structs.h:675-680`)
* obj: `OS_ONCMD`,`OS_ONPULSE` (`src/structs.h:683-685`) — objects can never receive
  `ondrop`/`ongive`.

What the globals mean inside a script (`src/scripts.c:1718-1770`, call sites in
`src/*.c`): `me` is always the script-owning mob/room/obj; `ch` is the *other* character —
the giver for `ongive`/`bribe`, the arriving player for `greet`/`enter`, the player for
`oncmd`, the mob's opponent for `fight` (`src/fight.c:1891` passes `run_script(victim, ch)`)
and the killer for `death` (`src/fight.c:597`). **Exception:** in the pulse and sound
triggers `ch` is the mob itself, because `src/mobact.c:157,173,192` pass `run_script(ch, ch)`.
Several archive scripts assume `ch` is a player in a pulse trigger and are therefore wrong.
`argument` is the player's typed command for `oncmd`, and the **amount only** for `bribe`.
Changes are written back to the live game only for the globals `ch` and `me` (and, for
rooms, only `sect`) — `src/scripts.c:1806-1816`, `table_to_char/obj/room` at
`src/scripts.c:1959-2100`.

"Missing" lists every unresolved call: names that are not in `cmdlib` (`src/scripts.c:1609`),
not Lua 4.0.1 base/math/str globals (`src/lua/src/lib/{lbaselib,lmathlib,lstrlib}.c`, all
registered globally via `luaL_openl`), and not defined in `globals.lua` or the file itself.
Note `sort`, `foreachi`, `getn`, `tremove`, `mod`, `exp`, `format`, `strfind` etc. **are**
Lua 4.0 and are fine; `mxp`, `msp`, `create_event`, `get_*` are not.

## Table 1 — `mob/archive/` (115 files)

| path | what it does in play | triggers | target vnum → name (zone) | C special on target | missing | notes |
|---|---|---|---|---|---|---|
| mob/archive/aki_kuroda.lua | A painter that idly splashes paint on an empty canvas; if a player types `open painting` it exclaims "Don't touch that!" and attacks. | sound, oncmd | comment `mob 12915`; code tests `obj.vnum == 12915`. 12915 exists only as **room** "A Dark Passageway" (z129, Kir-Oshi extension); no obj 12915. | none | – | In-joke: "Aki Kuroda" is the name of a real Japanese painter. |
| mob/archive/anhkheg.lua | In combat, 20% of rounds triggers acid blast at its opponent. | fight | none named; the forest anhkheg is mob 9146 (z91) *(inferred from `mob/91.mob:647`)* | none — `forest_anhkheg` is declared in `spec_assign.c` but never `ASSIGNMOB`'d | – | 5 lines; the purest monster-ability script. |
| mob/archive/aurumvorax.lua | Eats any "gold" object lying in the room or inside a corpse and charges anyone carrying or wearing gold. | onpulse_all | 9147 "the aurumvorax", golden gorger (z91) **exists** | none (`forest_aurumvorax` declared only) | – | l.5 `if (room.obj)` tests a field `room_to_table` never creates (it sets `room.objs`), so the room half silently never fires; a Lua-3-era leftover. |
| mob/archive/autodraw.lua | A conjured "draw master": players say `begin` and once 2+ are present he takes the item and randomly awards it; aborts and returns the item if the initiator leaves. | code, onpulse_all, oncmd | none | none | create_event, mxp | `code` is the intended entry point and **no C code calls it** (no `run_script(…,"code",…)` in `src/`). State lives in the mob's own `gold` field (l.133) and a `drawers` global. |
| mob/archive/aversin.lua | A guard: greets passers-by, and delegates combat and pulses to two other scripts by `dofile` + `call`. | sound, fight, onpulse_pc | none | none | – | Clever composition, now broken: the `dofile` targets `mob/take_jail.lua` (l.7) and `guard_captain.lua` (l.12) exist only under `archive/`. |
| mob/archive/backstabber.lua | Every pulse, backstabs each visible non-immortal player in the room. | onpulse_pc | 9151 "Daral" (z91, exists); 12912 "An Intersection" (z129 — a **room**, not a mob) | **ASSIGNMOB(9151, backstabber)** | – | No `break` after success (l.10), so it attempts a backstab on every PC present. |
| mob/archive/baker_dough.lua | Give the baker a roll of dough (obj 8015): he pays 20 gold and hands you a fresh loaf (obj 8010); anything else is handed back. | ongive | obj 8015 — **now "a piece of meat"**; obj 8010 "a loaf of bread" exists | none | – | The dough vnum was reused for meat, so the trigger now pays 20 gold for a pie. Part of the farmer→miller→baker chain. |
| mob/archive/baker_flour.lua | Give a sack of flour (obj 15100): pays 50 gold and produces dough (8015); give dough + 1 gold and he bakes bread (8010) cheaply. | ongive | obj **15100 absent** (15100 is now the mob "a quagga" and a room); obj 8015 reused | none | – | l.16 names the other end of the chain: "the baker in Kir Morthis", a city that no longer exists under that name. |
| mob/archive/bane.lua | Half of the Grey Keep "star" ritual: greets visitors, then after 10 s begins a staged speech, counts the players, looks for the daemonic focus (obj 1406) in someone's inventory, and dumps everyone out of the room if the group is not ready. | fight, death, greet, onpulse_pc (+bane_one…bane_six via create_event) | 1408 "Bane", 1407 "Valoran", focus 1406, exit room 1412, ritual room 1439 — **all absent** (zone 14 is gone; the Grey Keep is now z144) | none | create_event | Most ambitious conversation in the archive: a 6-beat state machine driven cooperatively with `valoran.lua` via globals `keep_conv_state`, `keep_people`, `keep_focus_owner`. Needs ≥4 people on the star points, each with a bloodwax candle (l.142-166). |
| mob/archive/banker.lua | Cashes bond certificates (obj 1248) at level 5+: pays `1500/obj.val[1]` gold, refuses certificates not stamped with your player id, and shreds them. | ongive | obj **1248 absent**; "each city banker" (unnamed) | none on mobs; the C ATM is `ASSIGNOBJ(8034/18224, bank)` | – | `obj.val[2]` holds the owner's player id (l.30) and `val[1]` the lifestyle chosen at creation — an early bound-document system. |

| mob/archive/bearcub.lua | A bear cub with no leader adopts the nearest grizzly (mob 9111) as its master by charming itself to it. | onpulse_all | 9111 "the grizzly bear" (z91, exists) | none (`forest_bear_cubs` declared only) | – | `follow(x, TRUE)` — the TRUE sets charm (`lua_follow`, `src/scripts.c:542`). |
| mob/archive/beggar.lua | Idle city ambience: "Spare a coin, buddy?" or jingles his cup. | sound | none | none | – | 6 lines. |
| mob/archive/beholder.lua | Blocks all `cast`/`recite` in its room with an eyestalk ray, randomly casts sleep/charm/curse at a player each pulse, and uses disrupt/disintegrate in melee. | oncmd, onpulse_pc, fight | 12000 — **absent** (zone 120 gone); C's beholder is 11011 (z110, exists) | **ASSIGNMOB(11011, beholder)** | – | The anti-magic aura is a genuinely novel mechanic: `oncmd` returns TRUE so the typed command never executes. |
| mob/archive/bhang.lua | "tokes up on some kind bud." | sound | none | none | – | Period in-joke; 3 lines. |
| mob/archive/blacksmith.lua | The "humming black armour" quest: accepts 22 named pieces (14201-14222), hands back spares, then when it holds all 22 forges the complete suit (14223) in a flash of magic; also converts the Stone of Serapis (10306) into 10307 for 2000 gold. | ongive, sound | objs 14201-14223 **exist** (z142); objs 10306/10307 **absent**; mob unnamed | none | – | A live descendant survives: `mob/212/blacksmith.lua` is attached to mob 21210 with bitmask 24 (`MS_ONGIVE|MS_SOUND`, `lib/world/mob/212.mob:166`). Uses `foreachi` and a mutable local `pieces` table to count what the smith holds. |
| mob/archive/bradle.lua | A biter: chance of a poisonous bite rises as the victim's level falls (`number(0, 102-ch.level)`, l.2). | fight | none named | none | – | Name unexplained (possibly a builder's mob nickname). |
| mob/archive/brain_eater.lua | Beheads any non-headless corpse in the room, eats the brain with a slurp, and gains +1 level (up to 100) or +100 hp per corpse. | onpulse_all | 14420 "an intellect devourer", 14432 "the neh-thalggu" — **both exist** (z144) | **ASSIGNMOB(14420/14432, brain_eater)** | – | l.18 records a balance change in a comment: "Used to be damroll, change if required". Ends by setting `ch = me` (l.20), apparently to force write-back. |
| mob/archive/breed_killer.lua | Hunts nightbreeds: attacks any vampire or werewolf on sight, and with a "stake"/"spike" executes them outright when the level roll or sleeping-position check succeeds; strips PLR_VAMPIRE/PLR_WEREWOLF then `raw_kill`s. | fight, onpulse_pc | none named; C assigns `breed_killer` to 7900 and 7910 | **ASSIGNMOB(7900/7910, breed_killer)** | – | `obj_list("stake","char")` searches only the *mob's* own inventory (`lua_obj_list` uses the `me` global), so the "success" branch is effectively unreachable. |
| mob/archive/cabinguard.lua | Pirate-ship cabin guard: if you `unlock` the cabin door while carrying key 19118 he curses ("Ya di'n't say mudder may I!") and attacks; without the key he tells you the cabin is off limits. | oncmd | 19114 "a Cabin Guard" (z191, exists); key obj 19118 exists | none | – | Colourful dialect; killing you *despite* the correct key is deliberate (l.27-29). |
| mob/archive/caerroil.lua | 1-in-16 chance per round to heal itself. | fight | none | none | – | 5 lines. |
| mob/archive/carpenter.lua | Idle ambience (adjusts his toolbelt / wipes sweat from his brow). | sound | none | none | – | |
| mob/archive/citizen.lua | Eight random street emotes ("Repent! The end is near!", "looks around for the guards before giving you the bird") plus a polite response to bribes. | sound, bribe | none; C's `citizen` sits on 8062 and 18202 | **ASSIGNMOB(8062/18202, citizen)** | – | Shows `sound` used as a cheap atmosphere generator: a `number(0,7)` case switch. |
| mob/archive/cityguard.lua | City guard: accepts a 1000+ gold bribe and falls asleep on duty (or kills you if the roll goes badly), fights with fighter skills, and each pulse attacks OUTLAWs, nightbreeds (via `breed_killer.lua`) and anyone fighting someone of the opposite alignment. | bribe, fight, onpulse_pc | none named; C: `cityguard` on 2747, 18215, 21200-21203, 21227, 21228 | **`cityguard` SPC on those vnums** | – | `bribe` is the only hook that passes the bribe amount (`run_script(ch, vict, NULL, …, buf2, "bribe", LT_MOB)`, `src/act.item.c:760`). l.10-11 set `me.pos` then `save_char(me)` — comment: "Needed to get him to sleep". |

| mob/archive/cleric.lua | Cleric AI: a 32-slot spell table indexed by `level/3` picks heals or harm; below 1/4 hp it teleports itself or the victim away. | fight | none (generic; C uses it for 1305, 4102, 7970, 11023, 14309…) | `cleric` SPC on many vnums | – | The "should I heal or hit?" weighting is inverted relative to its own comment (l.39-45: the `healperc` tests can never all fire). Sets both `me.pos` and `ch.pos` after teleport so the fight can end. |
| mob/archive/clerk.lua | Lifestyle starter-kit clerk: give a lifestyle letter (1245/1246/1247) and he issues a backpack (8038/8030), a bond certificate (1248) stamped with your player id and a class/race starter kit; give a shop deed (1222) and he refunds 90%. | ongive (+`equipped`/`commence` via create_event) | 4812 "a sage" (z48, exists), 5315 **absent** (Mist Keep), 18228 "a clerk" (z182, exists) | **ASSIGNMOB(18228, clerk)** | create_event | The `equip_invent`/`equip_pack` tables (l.9-18) are effectively a class/race starting-gear matrix. `commence()` states the intended newbie flow in prose: fight outside the city walls, find the teacher in the city library, cash the bond at level 5. |
| mob/archive/conjured.lua | Deletes a conjured creature the moment its charm breaks, with a special "You lose control and $N fizzles away!" message for the conjured-elemental vnums 81-84. | onpulse_pc | 81-86 | **ASSIGNMOB(81…86, conjured)** | – | Correct *because* `src/mobact.c:192` passes `run_script(ch, ch)` — `ch` and `me` are the same mob in this trigger. |
| mob/archive/creation.lua | The character-creation questionnaire: three questions (schooling/profession, background, lifestyle) with 3 answers each, whose answers set attributes, alignment, starting gold and equipment. | intended direct call from `CON_CHARCREATE` (l.1-6); helpers `question`, `question1`-`question3` | none named | none | – (mechanically; but the caller does not exist — no `CON_CHARCREATE` symbol anywhere in `src/`) | Purest lore artefact: the answer texts name Tholdur (a knight), Eldrich (the sorcerer), Le Tal (the alchemist), the monks of Mist Keep and the priests of Kir Drax'in — the intended newbie geography. |
| mob/archive/crystal_forger.lua | A forger: lists what it can make from your crystalline chunks (11701), then converts 1-5 chunks + 200-1000 gold into four crystal items (11706-11709). | oncmd | 7923 per comment — **absent as a mob**; obj 7923 is now "a beryl statue". Obj 11701 now reads "a crystal scroll"; products 11706-11709 exist but are now "a crystal necklace", "a suit of crystal armor", "a crystal dagger", "a demonic shield" | none | – | Structurally identical to `dragon_forger.lua` (same `list` / `buy <thing>` protocol, same "used" chunk extraction with `tremove`). The flavour text ("a pair of crystalline gloves") and the items actually handed over have drifted apart — the four product vnums were re-populated with a different gear set. |
| mob/archive/cuchi.lua | A pet: `pat` it and it purrs; ordinary players get 10 gold — but a character named exactly `Orodreth` is set to **LVL_IMPL (40)** and saved. | oncmd | C's special is **ASSIGNMOB(18306, cuchi)**; 18306 is Oro's little pet in the Checker Board (z183, author "Oro") | **`cuchi` SPC on 18306** | – | The most revealing artefact in the set: a hard-coded character-name privilege check (`ch.level = LVL_IMPL`, l.16) that promotes a player to implementor. |
| mob/archive/donation.lua | Donation-room sorter: picks up everything gettable, junks trash/keys/notes/pens/food/broken/sub-10-gold items, and files armour into a case, weapons into a cabinet and everything else into a chest. | onpulse_all | none named; C's equivalent is the `dump` **room** special on 8085 and 21223 | **ASSIGNROOM(8085/21223, dump)** | – | l.3 warns "relies upon the containers being present!". Returns after the first junked item, so clearing a room takes many pulses. |
| mob/archive/dracula.lua | A vampire: if a player `look`s at him they are mesmerised, bitten and flagged PLR_VAMPIRE ("the blood is the life!"); combat is delegated to `magic_user.lua`. | fight, oncmd | none named; C assigns `dracula` to 7903 and 14110 (Lothar) | **ASSIGNMOB(7903/14110, dracula)** | – | Filters immortals (l.10). Same "look at me to be transformed" mechanic as `medusa`/`warg`/`tattoo`. |
| mob/archive/dragon_breath.lua | 1-in-15 chance per round to breathe the element matching its vnum; dragon 4209 also growls "So, you have found my lair…" on greet and attacks. | fight, greet | 4209, 4705, 10200, 10300, 10301, 10302, 20027; of these **4209, 4705, 20027 exist**; 10200/10300-10302 absent (z102 gone) | **`dragon_breath` on 4209, 4705, 11000-11002, 20027** | – | A vnum-keyed spell table rather than per-mob scripts — the most maintainable design in the archive. |
| mob/archive/dragon_forger.lua | A forger: turns dragon scales (10204) + 2000-9000 gold into dragonscale gloves/leggings/sleeves/breastplate (7906-7909), and hands `look`ers a small card (obj 7903). | oncmd | 7917 per comment — now **"Adonis, Paladin of the Moonshear"**; obj **10204 absent** (z102 gone); 7906-7909 are now **mobs**, not objects; obj 7903 exists | none | – | Every output item vnum has been taken over by a mob — zone 79 ("BFM/Random Load") was repurposed wholesale. |
| mob/archive/drake.lua | 10% chance per round to cast fireball. | fight | none named | none | – | 5 lines. |

| mob/archive/elven_prostitute.lua | Ambience ("jiggles in your direction", "For a good time, give me 5 gold coins"); accepts a >10 gold bribe and takes the player into the shadows ("you decide not to watch"). | sound, bribe | none; C's `prostitute` special is on 8023 | **ASSIGNMOB(8023, prostitute)** | – | The bribe hook doubles as a paid-services hook across several scripts (also `mercenary`, `jailguard`, `warg`). |
| mob/archive/enchanter.lua | Chatty enchanter in the catacombs; give him any non-magic armour/weapon plus elemental dust (obj 1314) and he casts enchant weapon/armour at "god" level and returns the item; gives it back otherwise. | sound, ongive | 1400 per comment — **absent** (z14 gone); dust obj 1314 **exists** (z13) | none | – | His dialogue is the map of the Keep arc: "Seek out Bane and Valoran", "study the art of sorcery before climbing the tower" (l.13-15). The focus/dust economy is the entry point to the summoning ritual. |
| mob/archive/eq_thief.lua | Steals one item at a time from any non-immortal player's inventory, silently or with "hands in your wallet", with automatic success if the victim is below sleeping. | onpulse_pc | none named; C assigns `eq_thief` to 7979 and 12118 (the kender) | **ASSIGNMOB(7979/12118, eq_thief)** | – | Condition on l.6 `if ((room.char) or (me.pos ~= POS_STANDING))` is almost always true, so the standing check is ineffective. |
| mob/archive/ettin.lua | 25% chance per round to hurl a boulder for 10-30 damage. | fight | none named | none | – | Directly writes `ch.hp` and calls `save_char(ch)` — the archive's standard "script damage" idiom. |
| mob/archive/farmer_wheat.lua | A wheat farmer in Mist Keep who chats about the weather, the flour mill on the plains of Dor-Sefrith and the harvest, and periodically leaves bundles of wheat (obj 5300) on the ground for players to collect. | sound, onpulse_pc | 5305 per comment — **absent** (zone 53 gone); obj 5300 **absent** | none | – | l.12 "I hear there is a flour mill on the plains of Dor-Sefrith" makes this the head of the bread chain; `miller.lua` is the next link. |
| mob/archive/fighter.lua | Library-ish combat AI: on 30% of rounds chooses a warrior skill — headbutt, parry, bash, berserk, kick, trip — and issues it at `FIGHTING(me)`. | fight | none (loaded by `dofile` from `cityguard.lua` and `breed_killer.lua`) | none itself; C's `fighter` SPC does the same for 4914, 5200, 7901, 7902, 12111, 12850, 20002… | – | Returns TRUE so the mob's pulse is skipped after acting. | 
| mob/archive/fire_ant.lua | 10% chance per round of a poisonous bite, but only against non-NPC attackers. | fight | 1700 per comment — **absent** (zone 17 gone) | none | – | |
| mob/archive/fire_ant_larva.lua | A larva that, after a random 20-30 tick timer, forms a protective cocoon (obj 1701), setting the cocoon's own 150-300 tick timer and extracting itself. | onpulse_all | 1702 per comment — **absent**; obj 1701 **absent** | none | – | Uses the mob's `timer` field (the `wait`/pose counter) as a lifecycle clock, saved back through `save_char(ch)` — exploiting `ch == me` in pulse triggers. |
| mob/archive/forester.lua | A woodsman who chats, asks for wood, pays 50 gold for a bundle of wood (obj 1221) and destroys it. | sound, ongive | 9160 per comment — **9160 is a room** ("The Forest", z91); obj 1221 **absent** | none | – | Pairs with `obj/archive/wood_axe.lua` (chop wood → sell wood); the "wood economy" of the Great Forest. |
| mob/archive/gazer.lua | 20% chance per round to cast mindblast. | fight | none named; C's `forest_gazer` declared but unassigned | none | – | 5 lines. |
| mob/archive/golem_from_crate.lua | The *transport* golem: at the mine entrance (room 11708) it tips its crystalline chunks (obj 11701) into a bucket that disappears and empties them out of the game; elsewhere it takes chunks out of wooden crates (obj 11702). | onpulse_all | 11706 per comment — mob 11706 is now "the demon L'cath"; **obj 11702 now reads "a crystal wand"**, not a wooden crate, and no crate object survives anywhere in zone 117 | none | – | The comment's mob vnum has drifted: golem mobs 11700/11704/11705 exist. The "deposit in a bucket" half still works; the "empty a crate" half cannot, because the crate object no longer exists. |
| mob/archive/golem_miner.lua | A mining golem: 1-in-50 pulses it breaks a crystalline chunk (obj 11701, now "a crystal scroll") off the tunnel wall (max 3 carried) and hands what it has to a passing transport golem (mob 11700, "a crystal golem"). | onpulse_all | 11702 per comment — mob 11702 is now "the keeper of the flame"; mob 11700 "a crystal golem" and obj 11701 exist | none | – | The heartbeat of the crystal economy: miner → transporter → crate → forger. |
| mob/archive/golem_to_crate.lua | The transport golem: takes chunks from the miner and puts everything into any wooden crate (obj 11702) weighing under 20. | onpulse_all | 11700 per comment — mob 11700 "a crystal golem" **exists**; obj 11702 is now "a crystal wand" | none | – | Uses object *weight* as a container-fullness test (l.12), a nice period hack. Completely dead in this world: no object named "crate" remains in zone 117. |
| mob/archive/griffin.lua | 10% chance per round to cast flamestrike. | fight | none named | none | – | 5 lines. |

| mob/archive/guard_captain.lua | A guard captain who spots sleeping/loitering guards, barks "On your feet, slacker!" and orders them to his office, and frowns at citizens who greet him. | onpulse_pc | none named; hard-codes guard vnums 4807, 4815, 5302, 8060, 18215, 21200, 21201 and citizens 4801, 5319, 8062, 21243 | 8060 → `wall_guard_ns`; 18215/21200/21201 → `cityguard`; 8062 → `citizen` | – | Broken as written: l.16 assigns `room.char[i] = POS_STANDING`, which *replaces the character sub-table with a number*, so the following `save_char(room.char[i])` (l.17) fails its `lua_istable` check and logs "Invalid argument passed to lua_save_char" (`src/scripts.c:1297-1306`). Nothing actually wakes up. |
| mob/archive/guardian.lua | An elemental guardian whose song charms a player into striking another player in the room, or into hitting themselves if alone. | onpulse_pc | 1314 "a guardian spirit" (z13, exists) | **ASSIGNMOB(1314, elements_guardian)** | – | Very ambitious: it *reassigns the global `ch`* to a random PC (l.33) and issues `kill` for them, then hand-builds the room message because `act()` cannot address a third party. Has a 1,000,000-iteration escape hatch (l.29). |
| mob/archive/head_shrinker.lua | Turns the heads of other players into a wearable necklace and later stitches more heads onto it; 200 gold per head, 10-max capacity, and the necklace's extra description is a padded 4-column list of the victims' names. | oncmd (`list`, `buy`; helper `make_necklace`) | 7920 per comment — mob 7920 is now "**Pyros, the Daemon Lord**"; obj 7925 is now "a granite statue", not the necklace | none | – | The most elaborate item-crafting script in the archive: it reads `ch.objs[i].owner` (the corpse-head ownership field), pads names with a `while ((29 - length) ~= 0)` loop (l.109) and stores capacity in the necklace's own weight. |
| mob/archive/hermit.lua | A hermit who invites leaderless characters to sit by the fire. | greet | none named | none | – | 7 lines. |
| mob/archive/identifier.lua | An item appraiser: quotes a price (`cost/10` under 5000, else `cost*0.14`, +5% for magic) and, once paid, identifies the item with `SPELL_IDENTIFY`, passing it back afterwards. | oncmd, ongive | none named; C's `identifier` special is **ASSIGNMOB(8087, identifier)** | **`identifier` SPC on 8087** | – | Tells the player to "read the sign" for the price list — the list lives in a room object. Swaps `me`/`ch` temporarily (l.71-74) so the identify message is credited to the right actor. |
| mob/archive/jailguard.lua | Prison guard: takes a bribe of `level²` gold or more and throws you out of the cell (clearing the jail timer); otherwise grins. Each pulse he announces remaining hours to prisoners and ejects those whose time is up. | bribe, sound, onpulse_pc | none named; C's `jailguard` is 8088, `outofjailguard` 8089 | **ASSIGNMOB(8088, jailguard) / (8089, outofjailguard)** | – | Uses `room.exit[]` to find the one way out and `ch.jail` (the real jail timer, exposed by `char_to_table`). Pairs with `take_jail.lua`. |
| mob/archive/janitor.lua | City janitor: accepts gifts of junk, thanks donors, and each pulse picks up any non-corpse object it can get. | bribe, ongive, onpulse_all | none named; C's `janitor` is on 8061 and 21229 | **`janitor` SPC on 8061/21229** | – | v1 of `donation.lua`: this one just hoards, it does not sort. |
| mob/archive/keep_sorcerer.lua | The Keep sorcerer blocks players from going west by conjuring a magical barrier object into the room as they try to leave. | oncmd | 1404 per comment — **absent** (z14 gone); barrier obj 1421 **absent** | none | – | Companion to `obj/archive/barrier.lua`, which destroys itself when mob 1404 is no longer in the room — the barrier's own pulse *is* the "sorcerer is dead" check. |
| mob/archive/kelpie.lua | A water horse that drowns its opponent for 0-25 damage with a probability that grows with the victim's level. | fight | none named | none | – | `number(0, round(3 + ch.level/33))` — and `round()` in this build is a **truncating** cast to int (`src/scripts.c:1260`), not a real rounding function. |
| mob/archive/magic_user.lua | Mage AI: picks a spell from a 34-slot table indexed by `level/3`, aligns dispel good/evil, has a 20% chance to teleport away instead, and occasionally uses a worn staff. | fight | none (loaded by `dracula.lua` and `medusa.lua`; C's `magic_user` covers ~30 vnums) | `magic_user` SPC on many vnums | – | The most reused script in the archive — the "mage behaviour" library. |

| mob/archive/medusa.lua | Looking at her turns you to stone (`raw_kill` with `SPELL_PETRIFY`) if a coin-flip of two random rolls succeeds; combat is `magic_user.lua`. | oncmd, fight | 14101 "Echidna, the greater medusa", 14102 "Glyphar the maedar" — **both exist** (z141) | **ASSIGNMOB(14101/14102, medusa)** | – | `number(0,100) > number(0,100)` (l.17) is ~50% petrify, applied on `look` *and* `examine`. |
| mob/archive/memory_moss.lua | A moss that, 20% of pulses, touches a random PC and strips **all** spell affects from them (`unaffect`). | fight, onpulse_pc | 10107 and 10108 per comment — **absent** (zone 101 gone) | none | – | The combat version simply calls its own `onpulse_pc`, so the same effect can fire every combat round. |
| mob/archive/mercenary.lua | A hireling: a bribe of 100× your level gold makes him swear allegiance and follow you (charmed); he laughs at smaller offers, refunds if he already serves someone, and extracts himself ("back to the city") if he loses his leader outside zone 80. | bribe, onpulse_all | none named | none | – | The `onpulse_all` guard `room.vnum < 8000 or > 8099` (l.27) is a hard-coded "hirelings only exist in Kir Drax'in" rule — the only mob in the archive that polices its own zone. |
| mob/archive/merchant_inn.lua | Stage 1 of an escort quest: the merchant buttons a level-25+ player with under 50k gold at an inn table, asks for an escort to Xixieqi, negotiates yes/no, then leaves and (via `merchant_walk.lua`) spawns the travelling merchant 8 minutes later. | onpulse_pc, oncmd (+death, yes, no, yesno) | 5332 per comment — **absent** (Mist Keep); 6805 (the walking merchant) **absent**; room 5476 **absent** | none | create_event, mxp | Ambitious two-script quest with a 5000-gold reward; it refuses to start if a second merchant already exists (`inworld("mob", 6805)`, l.9). |
| mob/archive/merchant_walk.lua | Stage 2: the merchant walks itself from the gates to Xixieqi using `direction()` pathfinding one step every third pulse, forbids the escort from resting/sitting/sleeping, spawns three bandits (mob 6506) at room 7093 which hunt the pair, and pays out at room 4860 (Xixieqi). | onpulse_all, oncmd, fight, death (+init_quest, tele_mobs, escort, end_quest, attack_time, extraction) | 6805 per comment — **absent**; rooms 4860 (exists), 7093/5476/5315 **absent**; mob 6506 **absent** | none | create_event | The most advanced AI in the archive: self-navigating NPC, follower checks, ambush spawn, escort-abandonment failure and an "outlaw the player who betrays me" branch (l.113). Would need `create_event` for its 480-tick start delay. |
| mob/archive/miller.lua | A miller on the plains of Dor-Sefrith: pays 30 gold for a bundle of wheat (obj 5300), destroys it, and produces a sack of flour (obj 15100) on the ground. | sound, ongive | obj **5300 absent**, obj **15100 absent** (15100 is a mob/room) | none | – | Middle link of the bread chain: `farmer_wheat` → **miller** → `baker_flour`/`baker_dough`. Dialogue names both cities of the chain. |
| mob/archive/mime.lua | "tries to escape from an invisible box only he can see." | sound | none | none | – | 3 lines; pure joke. |
| mob/archive/mindflayer.lua | 1-in-8 chance of tentacles wrapping the victim's head (`SPELL_SOUL_LEECH`), and on a 15 rolls a psiblast with blood-from-the-nose flavour. | fight | none named; C's `mindflayer` is ASSIGNMOB(14414, mindflayer) | **`mindflayer` SPC on 14414** | – | Uses `switch == 0 or switch == 5` (2/16) for leech and `== 15` for psiblast — a slightly lopsided distribution. |
| mob/archive/minion.lua | An elemental minion that disintegrates any talisman it picks up ("eradico paratus"), and that removes the corresponding cylinder of light from its pillar if the talisman is missing from one of the four Elemental Chambers. | onpulse_all | 1313 "an elemental minion" (z13, exists); rooms/talismans 1360/1364/1380/1384 and 1300-1303 **all exist** | **ASSIGNMOB(1313, elements_minion)** and the four rooms carry `elements_load_cylinders` | – | One of the few archive scripts where **every** referenced vnum still exists — the Elemental Temple puzzle survived intact as C code. |
| mob/archive/minstrel.lua | Idle ambience: sings about your mother's beauty, or your battle conquests. | sound | none | none | – | |
| mob/archive/mount.lua | Sends an ownerless mount home (extract) when its timer expires, and if it is being ridden throws the rider off and clears the AFF_MOUNT/AFF_CHARM flags. | onpulse_all | none; C's `stableboy` handles purchase (8022) and mounts come from `MS_MOUNTABLE` | none directly | – | Relies on the mob `timer` field both as "time left" and "am I about to vanish" — the intended policy was mounts that expire and must be rebought. |
| mob/archive/mymic.lua | A mimic that 1-in-10 pulses steals 1-10% of every PC's gold and, 1-in-4 of those times, eats a piece of their food. | onpulse_pc | none named; C's `forest_mymic` is declared but unassigned; the world's mimic is mob 5700 (z57) | none | – | Gold theft is done by direct assignment (`me.gold`/`ch.gold`, l.20-21) while food is stolen via `steal()`, so the two halves fail differently. |

| mob/archive/neckbreak.lua | A specialist assassin that neckbreaks any visible player it can see, 1-in-6 pulses. | onpulse_pc | none named | none | – | Bug: it uses `ch.alias` while `ch` is the mob in this trigger, so it issues `neckbreak <its own keywords>`. |
| mob/archive/never_die.lua | Restores the mob to full hit points every pulse. | onpulse_all | 19113 per comment — mob 19113 is "the ship's Navigator" (z191, exists) | none on 19113; C's `never_die` is assigned to **14401 and 19119** | – | A duplicate copy without the header comment exists and is *live* as `mob/never_die.lua`, and it is byte-identical to the archive one apart from that comment — the one archive file that was re-activated. |
| mob/archive/no_get.lua | Guards room items: if a player `get`s or `palm`s anything from the room or a container, the mob strikes their hand and attacks. | oncmd | 14416 "a taipan man" and 14430 "a brain spider" — **both exist** | **ASSIGNMOB(14416/14430, no_get)** | – | One of the better-designed guards — it whitelists the *act* rather than the item. |
| mob/archive/paladin.lua | Paladin AI: parry, bash, charge, disarm, and an alignment-correct `dispel evil`/`dispel good`. | fight | none named; C's `paladin` is on 71 and 7915 | **`paladin` SPC on 71/7915** | – | |
| mob/archive/paralyse.lua | A bite that paralyzes, with the chance rising as the victim's level falls. | fight | none named | none | – | Uses `SPELL_PARALYSE`, which **is not defined in `globals.lua`** — the script would compare against `nil` and so pass `nil` to `spell()`. |
| mob/archive/petitioner.lua | Idle ambience ("Sign this, please! There's too much violence!"). | sound | none | none | – | |
| mob/archive/pet_store.lua | A pet-shop attendant that lists every NPC in the *next room* with a price of `level × 100`, and on `buy` loads a fresh instance of that pet for the buyer and makes it follow (charm). | oncmd | 21245 "the pet shop attendant" (z212 Alaozar) — **exists**; it reads the stock room as `room.vnum + 1` = 21246, which is "Guild Street" | none on 21245; the C pet-shop **room** special is `ASSIGNROOM(21235, pet_shops)` | – | Clever and cheap: the shop's inventory *is* a room full of mobs. The Alaozar pet shop is room 21235, so the vnum+1 arithmetic only works if the attendant stands next door to it. |
| mob/archive/phoenix.lua | A phoenix whose rider (mob 1402) joins the fight with a trident (1420), implemented by *loading and extracting* the rider each round so it cannot be killed; on death the rider leaps off first. | fight, death | 1401 (phoenix), 1402 (rider), 1420 (trident) — **all absent** (z14 gone) | none | – | Uses the mob's own `gold` field as a 3-state state machine (l.8, 20, 29). One of the most inventive (and most fragile) scripts. |
| mob/archive/porcupine.lua | 1-in-6 chance per round to fire a volley of quills for 1-10 damage. | fight | none named | none | – | |
| mob/archive/prisoner.lua | A poisoned prisoner quest: he loses 5 hp per pulse and cries for the antidote (obj 18281); give it to him and he hands over a key (18280) and a diary (12900); unlocking his shackles with the key (12901) earns 20,000 gold and he runs off south. | onpulse_pc, sound, ongive | 18245 per comment — **18245 is a room** ("The Temple Garden", z182); objs 18281/18280/12900/12901 **all absent**; mob 12900 is "a hidden guard" | none | – | Contains a typo bug: `spell(me, NILL, SPELL_REMOVE_POISON, FALSE)` (l.42) uses the global `NILL` (nil), and `SCAVENGER`-style item checks go through `me.objs` rather than `ch.objs`. |
| mob/archive/puff.lua | 21 pieces of dragon one-liners: "My god! It's full of stars!", "I'm a very female dragon", "Rule number 6…there is NO rule number 6", "NIH!". | sound | mob 1 — **exists**: `ASSIGNMOB(1, puff)` | **`puff` SPC on mob 1** | – | The richest in-joke in the set. Case 39 is unreachable (`number(0,20)`), so "I'm gonna kick your ASS!" never fires. |
| mob/archive/pyros.lua | The summoned daemon's own script: sorcery combat plus a death hook that clears `keep_pyros` so another ceremony can be held. | fight, death | 1410 per comment — **absent** (the Keep's Pyros; zone 79 still has a mob 7920 named "Pyros, the Daemon Lord") | none | – | Deliberately calls `call(fight, room.char, "x")` (l.5) rather than passing `ch`, because the daemon fights whoever is in the room. |
| mob/archive/quanlo.lua | Master Quan Lo mocks players who try to `flee`/`retreat`/`escape`: "This is not a shawade. Try it again. This time with feewing." | oncmd | 19405 (C: `ASSIGNMOB(19405, quan_lo)`, z194 Shaolin Temple / Fire Pagoda) | **`quan_lo` SPC on 19405** | – | Uses `gossip()` (a global channel) for a joke about roleplaying — and writes the accent phonetically, a period stereotype joke. |
| mob/archive/recruiter.lua | Idle ambience (smiles / shuffles papers). | sound | C: `ASSIGNMOB(16300, recruiter)` (Kir Drax'in Guard Training Centre) | **`recruiter` SPC on 16300** | – | |
| mob/archive/remove_curse.lua | A wizard who removes curses: on greeting he scans inventory and worn items for cursed (ITEM_NODROP) gear, announces his price (`level × 50`), then `remove <item>` strips the curse and charges. | sound, greet, oncmd (+curse via create_event) | none named | none | create_event | A clean, complete service script: it is one of the few that checks *worn* as well as carried items (l.37-43) and that handles "you don't have enough gold" paths explicitly. |

| mob/archive/rescuer.lua | Rescues the first NPC it sees that is fighting a player, shouting a `rescue` for that mob. | onpulse_pc | 7909 "the elven avenger" (z79, exists) | **ASSIGNMOB(7909, rescuer)** | – | Its own comment says "the first fighting mob it comes across", but l.10-11 actually scan for `isnpc(room.char[i])` fighting a non-NPC — correct as intended. |
| mob/archive/sandstorm.lua | A sandstorm mob: 25% of rounds it drags a random PC out of the fight and dumps them in a random room in the zone (6101-6299) for 75 damage, standing them up to end combat. | fight (+port via create_event) | 6102 per comment — **absent** (zone 61 gone); destination rooms 6101-6299 **absent** | none | create_event | The random destination `tport(vict, number(6101,6299))` assumes a fully-populated zone — a "hard-coded random teleport" idiom also used by `falycion_trap.lua`. |
| mob/archive/seiji.lua | In-joke ambience: "takes a deep breath from his bamboo pipe and exhales a cloud of thick smoke". | sound | none | none | – | |
| mob/archive/shop_give.lua | Prevents players from handing a player-owned shopkeeper anything the shop is not licensed to buy, using the engine's own shop table (`item_check`). | ongive | none named; applies to player-owned store keepers | the shop system itself (`src/shop.c`) | – | The only script in the archive that uses `item_check` — a hook into the C shop system rather than a substitute for it. |
| mob/archive/shopkeeper.lua | A shopkeeper who attacks pets: if the customer is mob 8063 ("Lawana Fel'Enwar"), 12115 (the seagull) or 18203 (a wharf rat) — i.e. if the *customer is an animal* — he mutters about filthy animals and attacks. | greet | 8063, 12115, 18203 — all **exist** (they are the `fido`-flagged animals) | 8063 → `fido`, 12115 → `fido`, 18203 → `fido` | – | The joke reveals the intent: the `fido` special sends animals into shops, and shopkeepers were supposed to object. |
| mob/archive/singingdrunk.lua | "sings an old war ditty... badly off-key." | sound | none | none | – | |
| mob/archive/snake.lua | A snakelike bite that does poison damage with a level-scaled chance. | fight | none named; C's `snake` is on 14103, 14127, 14415 | **`snake` SPC on 14103/14127/14415** | – | One of four near-identical "venom bite" scripts (`snake`, `bradle`, `paralyse`, `fire_ant`) that differ only in the spell cast and the level formula. |
| mob/archive/sorcery.lua | The "sorcery" spell library: picks a random victim in the room and casts one of burning hands / fireball / mind bar / dispel good / dispel evil / harm. | fight | none — it is a library, `dofile`d by `bane`, `pyros` and `valoran` | none | – | The companion to `magic_user.lua`: `magic_user` is the level-scaled mage *class*, `sorcery` is the Keep's scripted cultist spell list. |
| mob/archive/souleater.lua | A gate guardian: accepts a "soul" (obj 4618) as payment, swallows it with a horrible scream, and teleports the payer through to room 14405; anything else earns a growl. | ongive | obj **4618 absent** (4618 exists only as a room); destination room 14405 **exists** ("Inside the Gate", z144) | 14405 → mob `teleport_victim` (same zone, different mob) | – | One of the few archive scripts whose teleport destination still exists — the Grey Keep's outer gate survived the renumbering even though its inner rooms did not. |
| mob/archive/stable.lua | A stablehand: sells a mount for 300 gold (vnum chosen per stable), stables and retrieves mounts at 5 coins/day, parsing the recall-time argument to work out the bill, and returns a mount vnum for the caller to load. | code (never dispatched), plus helpers `find_mount` | 8022, 18210, 4821, 21217 → mount 8021/4823; 8022 exists ("a stableboy") | **ASSIGNMOB(8022, stableboy)** | – | `code` never runs in this build. The `collect` branch parses a machine-generated argument string with `strsub`/`strfind` (l.82-84) — the closest thing in the archive to an ABI contract with C code. |
| mob/archive/strike.lua | A mob that strikes any visible player, 1-in-6 pulses. | onpulse_pc | none named | none | – | Same `ch`-is-the-mob bug as `neckbreak.lua`: it strikes itself. |
| mob/archive/sungod.lua | The sun god extracts itself on every pulse, first destroying all its worn items, with "White-hot flames burst forth from the altar, and the fire god is gone!". | onpulse_all | 10205 per comment — **absent** (zone 102 gone); the sacrificial altar room 10221 also absent | none | – | The extraction half of `room/archive/sacrifice.lua` (which loads mob 10205 and gives it a trident 10210). Because it runs on *every* pulse with no gate, the god can never stay in the world. |

| mob/archive/take_jail.lua | The arresting half of the jail system: on being attacked, the guard stops hunting, dismounts the victim, ignores nightbreeds, and after one tick carts the offender to the right cell for their city, setting hp to 1 and `jail = 1 + level/20`. | fight, onpulse_pc (+jail via create_event) | hard-codes guard→cell maps: 5302→5365, 8001/8002/8020/8027/8059→8062, 18215/18223/18224→18290, 21227→21226; **only the z182 and z212 cells exist** (5365/8062 outside z80 are gone) | `take_to_jail` SPC on 8001, 8002, 8020, 8027, 8059; `cityguard` on 18215, 21227 | create_event | `onpulse_pc` delegates back to `cityguard.lua`, so the two files are mutually recursive libraries; the guard's *attack* behaviour and its *arrest* behaviour live in different files. |
| mob/archive/tattoo.lua | The tattoo parlour: a two-part script. `greet` offers a tattoo; `oncmd` runs the whole conversation (materials/design/dyes/ingredients), then `code` implements the `list` and `buy <n> <colours>` flow — matching designs (obj 1209) and dyes against a player's supplies, charging 1000× the design's value, setting `ch.tattoo`, and playing out the agony ("A ghastly scream is ripped from $N's lips…"). | greet, oncmd, reply, code | designs map to `TATTOO_TRIBAL…ROSE`; C's tattoos are `tattoo1-4` on 8086, 2766, **21244 "Polywig"** and 18213; obj 1209 and 1080 (design/dye) are **absent** from the current object files | **`tattoo1/2/3/4` SPC on 8086, 2766, 21244, 18213** | create_event, mxp, get_dye | The most ambitious *social* script in the archive: a 7-design catalogue (Tribal, Scorpion, Jehduti the Moon God, Aethen the Sun God, Dragon, Skull, Wild Rose), a colour-mixing rule (max 3 dyes), per-design prices, and a persistent `GET_TATTOO` bit on the character. Design text and `room/archive/tattoo_rune.lua` must agree on the numbering. |
| mob/archive/teacher.lua | The training system: `code` takes a skill/spell group name, checks `group_lvl` and `group_pts`, charges `1000 × (level+1)` gold and `1000 × exp(level+1)` unassigned experience, then raises the group and sets the skill to `level×25+5` (20% for a first purchase) and leaves the player exhausted (hp/mana/move = 1). `oncmd` implements `remove <group>` to unlearn and refund a group point. | code (never dispatched), oncmd | teacher table: 8024, **21220 (now "the tailor")**, 21214, **5307 (absent)**, 4825, 18219; advanced teachers **7971 ("Eric Draven Crow"), 7972/7973 (absent)** | 8024/21214/4825 → `guild`; 18219 → `zen_master`; 21220 → none | get_group_lvl, get_group_pts, skill_group, mxp | The clearest evidence of an entire subsystem that was planned but never finished on the C side: the whole group-points economy (`group_lvl`, `group_pts`, `exp_q`) exists in the C structs but has **no Lua API**. The `code` entry point is absent from the C dispatcher, so the teacher was never reachable even when the vnums existed. |
| mob/archive/teleporter.lua | A master who teleports himself away once he drops below half hit points ("My work here is done"), standing both parties up so the fight can end. | fight | 14411 "Master Vrixrig-Vimcuj" (z144, exists) | **ASSIGNMOB(14411, teleporter)** | – | Counterpart to `teleport_vict.lua` (which removes the *player*). |
| mob/archive/teleport_vict.lua | Automatically teleports its attacker away with a contemptuous "You can't harm me, mortal. Begone." | fight | 14405 "Gil-Glash" (z144, exists) | **ASSIGNMOB(14405, teleport_victim)** | – | Fires on *every* combat round with no probability gate, so it is a hard "you cannot fight this mob" rule. |
| mob/archive/thief.lua | A pickpocket: 20% of pulses, tries to lift 1-10% of each visible player's gold, sometimes with the "hands in your wallet" message. | onpulse_pc | none named; C's `thief` is on 3119, 12127, 18218, 20004, 21242 | **`thief` SPC on those vnums** | – | Steals from *every* PC in the room in one pass (no break), unlike `eq_thief`. |
| mob/archive/thornslinger.lua | 1-in-6 chance per round to spray thorns for 10-20 damage. | fight | none named | none | – | Note it does not call `save_char`, unlike `porcupine.lua` — an inconsistency in the archive itself. |
| mob/archive/town_teleport.lua | A paid city teleporter: `list` shows the five cities as MXP links, and saying a city name teleports you there for a distance-based fee (0 to 6000 gold) from a 5×5 price matrix keyed to your current city. | oncmd | Kir Drax'in 8013, Kir Morthis 21223, Kir Oshi 18232, Xixieqi 4804, Mist Keep 5317; **Mist Keep (z53) is gone**, the rest exist | none on those rooms | mxp | Contains a genuine fare table — one of the best artefacts for reconstructing the *relative geography* of the realm (e.g. Xixieqi→Mist Keep is only 1500, Kir Drax'in→Mist Keep 6000). The name "Kir Morthis" no longer appears anywhere in the world files; 21223 is now "The Town Dump" in Alaozar. |
| mob/archive/towncrier.lua | Ambience: "Arch Bishop Dinive to arrive on the Day of Winter Dawning!" and "By mandate of the church, no violence in town! The penalty is jail time!". | sound | none | none | – | Names a festival ("Day of Winter Dawning") and a church figure ("Arch Bishop Dinive") — lore not present in any world file. |

| mob/archive/triflower.lua | A carnivorous plant: 25% of pulses it sleeps a random PC with orange pollen; in melee it showers a random PC with sticky yellow enzyme for 10-30 damage. | onpulse_pc, fight | 20310 per comment — **20310 is a room** ("Noirwood Forest", z203); there is no mob 20310 | none | – | Note `local vict = room.char[number(1, getn(room.char))]` on l.5 is evaluated *before* the probability check, so it also reads the "last" PC in the room rather than a random one — the comment says as much. |
| mob/archive/troll.lua | A troll that regenerates to `level × 2` hp (capped at max) both out of combat (~5%/pulse) and in combat (~10%/round), with a glowing-wounds message. | onpulse_all, fight | none named; C's `troll` is on 10029, 14100, 14311, 14312, 19900, 19901 | **`troll` SPC on those vnums** | – | |
| mob/archive/tyr.lua | A drunk bartender: "Hey, you whant ta buy a drink? *Hic*", "Tha bar isth closed, so get the heel out!". | sound | none named (Tyr is also the author of zone 190, "Elemantal Plane (Tyr)") | none | – | Note all four branches roll `number(0,1)`, so the later lines are rarely reached. |
| mob/archive/valoran.lua | The other half of the Keep star ceremony: tracks state set by `bane.lua`, comments on whether the visitor is ready, leaves to the east to take its place at a point of the star (room 1443), and if the party still lacks the focus, explains where to get one and dumps everyone out. | fight, death, greet, onpulse_pc (+val_one, val_two, val_not_ready) | 1407 "Valoran", 1408 paired, rooms 1443/1412 — **all absent** | none | create_event | Cooperates with `bane.lua` through the shared globals; state 3 ("not ready") is set by `bane_one` when the party sends the wrong answer (l.84). |
| mob/archive/warg.lua | A warg: wags its tail at good-aligned arrivals and growls at evil ones; eats coins offered as a bribe; eats food given to it and plays with (then drops) everything else. | greet, bribe, ongive | none named | none | – | Reads `ch.align` (the raw alignment number) rather than the derived `evil` flag — a nice example of the Lua table exposing both. |
| mob/archive/weatherworker.lua | 20% chance per round to cast disrupt. | fight | none named | none | – | 5 lines. |
| mob/archive/werewolf.lua | A werewolf: howls (with a zone-wide echo) or tears into the victim's leg for `level × d2` damage and drains movement. | fight | 5510 "the werewolf" (z55, exists) | **ASSIGNMOB(5510, werewolf)** | – | Uses `echo(me, "zone", …)` to broadcast the howl to the whole zone — a sound effect no C special could easily do. Also writes `ch.move` (movement points), which is one of the few character fields the write-back actually supports (l.25, `src/scripts.c:2029`). |
| mob/archive/zealot.lua | "Repent sinners! The end time is near!" | sound | none | none | – | |
| mob/archive/zen_master.lua | A zen master who, 1-in-20 rounds, touches the player and then recalls them to their temple, or (on the 20 roll) advises "You have violence, but not thought" and teleports them away. | fight (+word/teleport via create_event) | 7919 per comment — **absent**; C's `zen_master` is ASSIGNMOB(18219, zen_master) in Kir-Oshi | **`zen_master` SPC on 18219** | create_event | Because the actual effects are deferred through `create_event`, in this build the master's touch resolves to nothing but a message. |

## Table 2 — `obj/archive/` (14 files) and `room/30/` (2 files)

Object scripts can only be reached through `OS_ONCMD` and `OS_ONPULSE`; `run_script` is called for
objects only from `src/interpreter.c` (equipment, inventory and room contents) and
`src/comm.c:789-795` (pulse).

| path | what it does in play | triggers | target vnum → name (zone) | C special on target | missing | notes |
|---|---|---|---|---|---|---|
| obj/archive/barrier.lua | An invisible magical barrier in the Keep: typing `west` returns "The magical barrier prevents passage to the west." and swallows the command; whenever the sorcerer (mob 1404) is no longer in the room it announces "The magical barrier disappears in a blaze of light." and destroys itself. | oncmd, onpulse | obj 1421 "a magical barrier" — **absent** (z14 gone); mob 1404 — **absent** | none | – | The mirror image of `mob/archive/keep_sorcerer.lua`: the mob loads the object, the object watches the mob. |
| obj/archive/carrion.lua | A corpse-like object; `look` at it and a carrion stalker (mob 14308) skitters out, scaled to the viewer's level and given at least half their hit points, and attacks. | oncmd | obj **14313 per comment — now a mob** ("a giant rat", z143); the spawned mob 14308 "a carrion stalker" **exists** | the C version is a **room** special: `ASSIGNROOM(14305, carrion)` | – | A "spawn scaled to you" trap, unusually modern for 2008: `carrion.level = me.level`, `carrion.hp = me.hp/2` (l.22-25). |
| obj/archive/ceremonial_scroll.lua | The Keep ceremony's reagent mini-game: `recite scroll` in the ritual chamber (room 1410) consumes the scroll and 4 seconds later checks for 5 bloodwax candles and a focus; the combination of reagents present (purified ash + pumice + daemon bone + obsidian) encodes exactly one of four spell-words onto one of `focus.val[1..4]`, glowing a different colour for each; when all four are set the daemonic focus (obj 1406) is created, and the candles extinguish 8 s later. | oncmd (+light_candles, extinguish_candles) | obj 1400 scroll, room 1410, candles 1402, bone 1401, pumice 1403, ash 1404, obsidian 1405, focus 1407 → result 1406 — **all absent** (z14 gone) | none | create_event | A deeply nested `if/else` cascade (l.68-106) that is really a reagent *encoding table*: `utbuho qibmrog` (ignite candles, yellow), `unsmecondus poih` (endure heat, green), `gewbar miobar` (summon, orange), `vibugp miobar` (banish, red) — the same strings the daemon script uses. |
| obj/archive/daemonic_focus.lua | The payoff of the ritual: `use focus` in the centre room (1445) with five lit candles and one person in each of the five surrounding rooms starts a four-phase ceremony echoed across all five rooms — candles ignite, a beam "endures heat" for everyone, a daemon is summoned, then the caster attempts the banishment with a `number(0, 34-level)` roll. On success Pyros is disintegrated; on failure he drops his sentinel flag, gains MOB_HUNTER and hunts the caster. | oncmd (+phase_two, phase_three, phase_four) | obj 1406, centre room 1445, side rooms 1439/1441/1443/1444/1446, death room 1442, mob 1410 "Pyros", items 1415-1417 — **all absent** | none | create_event | The most ambitious artefact in the archive: a 5-player, 5-room cooperative ritual with resource counting, mass `echo` into other rooms, an area kill (`raw_kill` on everyone in room 1442), an item load for the boss, and a *failure mode that turns the boss loose in the world*. |
| obj/archive/falycion_skeleton.lua | A skeleton (obj 1800) holding a scroll of recall: as soon as it is emptied it disintegrates into a pile of dust (obj 18), which is loaded into the room. | onpulse | obj 1800 — **absent** (zone 18 gone); scroll 8052 **exists**; dust 18 **exists** | none | – | The safety valve for `room/archive/falycion_trap.lua`, which always plants a skeleton with a recall scroll inside a random room so a trapped player can escape. |
| obj/archive/fire_ant_cocoon.lua | A cocoon that decrements its own timer each pulse; at zero it opens ("The strange cocoon suddenly breaks open and a mature giant fire ant emerges") and spawns a mature fire ant (mob 1700). | onpulse | obj 1701, mob 1700 — **both absent** (zone 17 gone) | none | – | The object *is* the clock: it writes `obj.timer` back with `save_obj` every pulse. |
| obj/archive/fire_ant_egg.lua | An egg that rolls 1-3 larvae; when its timer expires it announces the hatch ("A giant fire ant egg slowly opens and 2 larvae crawl out.") and loads that many larva mobs (1702). | onpulse | obj 1700, mob 1702 — **both absent** | none | – | Third file of the ant lifecycle (egg → larva → cocoon → adult): with `fire_ant.lua` and `fire_ant_larva.lua` it is a complete scripted life cycle spread across four files. |
| obj/archive/forest_tree.lua | A tree object that quietly removes itself as soon as nothing is inside it, so the zone's random-load equipment can drop it somewhere else. | onpulse | obj 9116 — **absent as an object**; 9116 is the room "The Forest" (z91) | none | – | The "delete yourself and let zone respawn relocate you" trick, also used by `spiritwood_tree.lua` — the archive's idiom for randomly-placed resources and treasure. |

| obj/archive/horn.lua | The silver horn: `use horn` blares "You hear the blaring of a loud horn" across the zone plus a note in the room; inside the Grey Keep (vnum/100 == 144) it also releases "the nightmare beast" by clearing the global `vrix_teleport` and consuming one charge from `obj.val[1]`. | oncmd | obj 14415 "a silver horn" — **exists** (z144) | **a C special already exists: `ASSIGNOBJ(14415, horn)`** | – | The clearest evidence of two systems mid-migration: one object wired to *both* a C special and a Lua script. It sets a global (`vrix_teleport`) that nothing in this C tree reads — a hook into a later, unlanded event system (Master Vrixrig-Vimcuj, mob 14411, is the `teleporter`). |
| obj/archive/mix_lookup.lua | A pure lookup table for alchemy: parse four reagent vnums out of the `mix` argument, `sort` them, and return the vnum of the resulting potion/dye/preservative — 7 four-reagent potions (1260-1265, 1279), 9 three-reagent potions (1266-1274), 4 two-reagent potions (1275-1278), 8 dyes including mix rules (green = yellow + blue, orange = yellow + red, purple = blue + red) and 2 preservatives. | code | result objs 1260-1291 — **all absent**; reagent 1225 "a bit of ash" **exists**; 1250-1256 absent; dye sources 5334 (z53 gone), 9112 "a kiwi" and 9124 (z91) exist but are unrelated items; 20307-20309 exist only as rooms (z203) | none — and no `do_mix` exists in `src/` either | – | The only *data* file in the archive, and the best single artefact for reconstructing the lost potion and dye economies. It also documents an intended API: `do_mix` was to call the script's `code` and use the returned vnum as the object to load. |
| obj/archive/spell_scroll.lua | A spell scroll: `use` it and, if you know the spell's group, your group level is high enough and you have `10000 × group_lvl` unassigned experience, the spell is learned at 100% and you are left exhausted; otherwise the parchment explains exactly what is missing. | oncmd | obj 1280 "a tattered parchment" — **absent** | none | get_spell, get_group_lvl | The learner's half of the teacher economy (`teacher.lua`). Both script and C structs assume a group / experience-point subsystem that has **no Lua API** in this build. |
| obj/archive/spiritwood_tree.lua | A treasure hunt: dig at the lone spiritwood tree (obj 20300) with a shovel worn/wielded and the treasure map (obj 9139) in inventory, and you uncover a chest (obj 20306) holding up to 4 randomly-chosen treasures from objs 422-435 (each gated by its own percentage load); the map crumbles to dust, and once the room is empty the tree removes itself so the zone can place it elsewhere. | oncmd, onpulse | obj 20300/20306 **absent as objects** (20300 and 20306 are rooms in z203 "Noirwood Forest"); map 9139 **absent as an object**; treasures 422-435 **absent** (zone 4 gone) | none | – | The only script that combines a skill-like check (`dig`), an equipment requirement (a shovel in `ch.wear`), item consumption and loot generation. Its "one chest only" global (l.32) is a hand-rolled mutex. |
| obj/archive/tinderbox.lua | The phoenix-nest escape: `use tinderbox` in the nest (room 1401) and the player *and everyone else in the room* are carried off by a giant phoenix — an eight-line descriptive passage — landing in room 1402, plus an outdoor echo so the world sees the bird pass overhead. | oncmd | obj 1412, room 1401, destination 1402 — **all absent** (z14 gone) | none | – | The best prose in the archive and the only *group* transport: l.37-40 teleport `me` and then everything else in `room.char`. |
| obj/archive/wood_axe.lua | A wood axe: `use axe` in any forest-sector room while level ≤ 10 gives a 20% chance of a saleable bundle of wood (obj 1221) and costs 20 movement points either way; it refuses non-forest rooms and mocks high-level choppers ("Chopping wood at your level? Go and kill something!"). | oncmd | obj 8029 — **absent as an object** (8029 is the room "Western Wall Road", z80); wood 1221 **absent** | none | – | The head of the wood economy, and a rare example of a script reading the *room sector* (`room.sect == SECT_FOREST`, l.21). |
| room/30/pattern_3065.lua | A four-line wrapper that loads `scripts/room/pattern_tport.lua` and calls `pattern_tport()`. | oncmd | the room itself (zone 30); zone 30 has **no `.zon`/`.wld`/`.mob`/`.obj` file at all** | none | – | `pattern_tport` **does** still exist (`lib/scripts/room/pattern_tport.lua`), making this one of the very few archived scripts whose dependency still resolves. Its teleport table names the seven cities that mattered: Kir Drax'in (8008), Alaozar (21258), Haven (12119), Kir Oshi (18203), Xixieqi (4886), Amber (2818) and Rakshasa (11295) — all still present. |
| room/30/pattern_dmg.lua | An energy pattern in the floor of zone 30: it damages anyone standing on it for 3-5 hp with three flavours of "your strength withers" message, with a 1-in-6 chance of just sparkling instead; immortals see "The pattern has no effect on you." | onpulse | zone 30 rooms (absent) | none | – | Correct in a way most pulse scripts are not: room `onpulse` passes the *occupant* as `ch` (`src/comm.c:715`), so `ch.hp = ch.hp - damage` really does damage the player. |

## Table 3 — `room/archive/` (36 files)

Room scripts get `ch == me == the occupant` in every trigger, including `onpulse`, because
`src/comm.c:715` iterates characters and passes each as both arguments.

| path | what it does in play | triggers | target vnum → name (zone) | C special on target | missing | notes |
|---|---|---|---|---|---|---|
| room/archive/alltalismans.lua | The Galeru column chamber: once all four elemental talismans are in their four chambers, any player entering this room is struck by four beams of coloured light and teleported to the Temple of Elements (1389). | onpulse | room 1372 "The Elemental Chamber" and rooms 1360/1364/1380/1384, talismans 1300-1303, destination 1389 — **all exist** (z13) | **ASSIGNROOM(1372, elements_galeru_column)**; the four chambers carry `elements_load_cylinders` | – | One of the few archive scripts that would work unchanged today; it cross-loads four other rooms with `load_room()` and counts talismans by vnum. |
| room/archive/bank.lua | A room-based bank: `balance`, `deposit <n>` and `withdraw <n>` against the character's stored bank balance, with messages in the room ("$n makes a bank transaction"). | oncmd | any bank room; C's bank is an **object** special (`ASSIGNOBJ(8034, bank)` KD ATM, `ASSIGNOBJ(18224, bank)` Kir-Oshi) | the C `bank` object special on 8034/18224 | – | The room alternative to the object ATM — evidence that the same feature was built twice. Uses `ch.bank` (`char_to_table` exposes it and `table_to_char` writes it back). |
| room/archive/citadel.lua | The aboleth trap: getting the moonstone amulet (10208) in room 10239 floods the chamber — the room's sector type is changed to underwater *and saved* — and spawns the aboleth (10204) plus three eels (10218) with "It's a trap, and there's no way out!"; a second hook drains the water and removes the water mobs when another player leaves room 6230 heading north. | onget, oncmd, enter | rooms 10239 and 6230, objs 10208, mobs 10204/10218 — **all absent** (z102 and z62 gone) | none | – | The only script that permanently rewrites a room: `room.sect = SECT_UNDERWATER` + `save_room(room)` (l.7-8), then reverses it from the other side of the zone. |
| room/archive/cylinders.lua | Keeps the four pillars of light in sync with the elemental talismans: taking a talisman collapses its cylinder with "The green cylinder of light slowly sinks back into the pillar.", and dropping the correct talisman into the correct chamber extends it again. | onget, ondrop | rooms 1360/1364/1380/1384, talismans 1300-1303, cylinders 1304-1307 — **all exist** (z13) | **ASSIGNROOM(1360/1364/1380/1384, elements_load_cylinders)** | – | The cleanest of the four Elemental Temple scripts: the room↔talisman↔cylinder mapping is a 3-line table (l.5-7). `ondrop` proves that room scripts do receive the drop hook (`src/act.item.c:506`). |
| room/archive/darts.lua | A wall-dart trap: any non-immortal in the room is told "Tiny darts shoot from holes in the wall and strike you!" and loses 20-40 hp every pulse. | onpulse | rooms inside the ziggurat (zone 22) — **the whole zone is absent** | `zigg_darts` is **declared in `spec_assign.c` but never `ASSIGNROOM`'d** | – | Pure period trap design: no saving throw, no mitigation, flat damage per pulse. |
| room/archive/elemental_room.lua | Generic elemental-plane punishment: fire burns you alive, earth drops semi-molten sand (10% of the time), wind peels flesh from bone, water fills your lungs, otherwise "The forces of nature slowly rip you apart" — then "You are DYING!" and 50-100 damage per pulse; immortals see "You ignore the forces of nature...". | onpulse | any room whose `sect` is SECT_FIRE/EARTH/WIND/WATER (z13-style planes) | `elemental_room` is **declared in `spec_assign.c` but never `ASSIGNROOM`'d** | – | Sector-driven, so it is portable to any elemental zone — the archive's most reusable room script. |
| room/archive/falldeck1.lua | Ship rigging: anything dropped in this room falls to the main deck (room 19144) with a message in both rooms. | onpulse | room 19144 "Main Deck" — **exists** (z191, Phantom Pirate Ship) | none | – | One of only two scripts that move objects between rooms with `objfrom(obj,"room")`/`objto(obj,"room", vnum)` — the "ship physics" of a multi-deck zone. |
| room/archive/falldeck2.lua | The same thing for the other rigging, sending objects to room 19127. | onpulse | room 19127 "Stairwell" — **exists** | none | – | Byte-different from `falldeck1.lua` only in the destination vnum and the source comment. |
| room/archive/falldown.lua | Player falls to the room named by the script's `argument` string, taking 10-50 damage; the second variant (`code_two`) additionally destroys the heaviest non-key item they carry ("You slowly get up and check your possessions. Something is missing!"). | code_one, code_two | destination room passed as `argument` (unknown) | none | – | Neither function is reachable: the C dispatcher has no `code_one`/`code_two` fname, so the "argument is a room vnum" contract documented here has no implementation in this tree. |

| room/archive/falycion_trap.lua | The spider-web highway: moving in a cardinal direction out of one of six surface rooms gives a 30% chance of being caught in an unseen web, dragged underground, paralysed and dumped in the matching Falycion (spider tunnel) room — and a skeleton containing a scroll of recall is planted in a random room of the zone so the victim can escape. | oncmd | source rooms 6806/6821/6835/7031/7038 (**absent**, z68) and 7064 (**exists**, z70); destinations 1801/1802/1807/1890/1898/1882 and the random range 1801-1899 — **all absent** (z18 gone); skeletons 1800 and scrolls 8052 | none (the C `carrion` room special on 14305 is unrelated) | – | The most thought-through trap in the archive: deliberate geographic pairing between surface room and destination, plus a fair-play escape mechanism (`obj/archive/falycion_skeleton.lua`). It is also the archive's only `spell(me, …, SPELL_PARALYSE)` user outside combat. |
| room/archive/fly_exit_up.lua | Blocks `up` in the Temple of Elements for anyone who cannot fly (and for all NPCs): "You try and jump up there but it's just too high." | oncmd | room 1389 "The Temple of Elements" — **exists** (z13) | **ASSIGNROOM(1389, fly_exit_up)** | – | One of the handful of archive scripts whose C twin is already assigned; it duplicates `fly_exit_up` in `spec_procs3.c`. |
| room/archive/francesca.lua | Taking the Tear of Seas (obj 19131) in room 19121 makes Francesca (mob 19126) hunt the new owner — but only if they now hold exactly one Tear (carried *or* worn). | onget | room 19121 — **exists** (z191); obj 19131 — **absent as an object**; mob 19126 — **absent** (19126 is a room, "Stairwell") | none | – | A "cursed treasure" mechanic implemented with `inworld("mob", 19126)` + `set_hunt`, i.e. the ghost is assumed to exist somewhere in the world already. |
| room/archive/galeru_dead.lua | The Galeru chamber: while the rainbow snake Galeru (mob 1315) is *not* present, everyone entering is made dizzy and teleported back to the zone's start (1395). It is implemented twice — once in `onpulse` and once in `enter`. | onpulse, enter | mob 1315 "Galeru, the master of elements" — **exists**; rooms 1394/1395 — **exist** | mob 1315 → `magic_user`; room 1394 → `elements_galeru_alive` | – | l.24 documents the reason for the duplicate: "This second function is needed to prevent players avoiding the room pulse." The pair is the clearest statement in the archive of how fragile pulse-only logic was felt to be. |
| room/archive/inn_music.lua | Plays one of six random MIDI files when a player enters an inn, at 20% volume, tagged "Inn". | enter | any inn room | none | msp | Half of a two-file jukebox: `msp_off.lua` stops playback when the player leaves. The MSP call signature (`msp("music", file, 20, -1, 100, "Inn")`) is fully specified even though no C implementation exists in this tree. |
| room/archive/jail_noentry.lua | An invisible doorway guard that stops non-immortals from *entering* a jail, with the forbidden direction hard-coded per jail zone: 5333 east, 8055 south, 18286 up, 21233 north. | oncmd | 8055, 18286, 21233 **exist**; 5333 **absent** (Mist Keep) | the C `jail` special is `ASSIGNROOM(8118, jail)` | – | Implemented as a room script rather than a mob, so no guard NPC is needed — the *room* refuses the movement. |
| room/archive/jail_noexit.lua | The same guard for *leaving*: 5365 west, 8062 north, 18290 down, 21226 south. | oncmd | 8062, 18290, 21226 **exist**; 5365 **absent** | as above | – | Together the two files define the jail geography of all four cities. |
| room/archive/keep_conversation.lua | Silences the whole room while the Keep conversation or the summoning ceremony is running: any command from a non-immortal is swallowed with "You really should be paying attention to the conversation!" or "A summoning ceremony is underway, you should be concentrating!". | oncmd | the Grey Keep rooms (zone 14, **absent**) | none | – | A script whose job is to *disable other scripted systems* — it reads the same `keep_conv_over`/`keep_ceremony` globals that `bane.lua`, `valoran.lua` and `daemonic_focus.lua` set. |
| room/archive/keep_portal.lua | A two-room shimmering blue portal loop (1414 ↔ 1423): `enter portal` steps through with a flash. | oncmd | rooms 1414 and 1423 — **both absent** | none | – | Identical in structure to `portal.lua` and `zigg_portal.lua` — three copies of one portal script survive, one per zone that used it. |
| room/archive/load_recall.lua | A safety net: on entering, if the player has no scroll of recall anywhere — including inside worn and carried containers, and inside a container inside a container — and does not know the word-of-recall spell, one magically appears in their inventory and is flagged ITEM_NODROP. | enter | scroll 8052 — **exists** (z80) | none | get_spell | The deepest inventory walk in the archive (l.37-53) and a clear statement of a design rule ("nobody gets permanently stranded"). Called by `citadel.lua`'s `enter` hook. |
| room/archive/loseitems.lua | Any non-corpse item dropped in these rooms falls into the ocean and is destroyed ("$p falls into the ocean and is lost forever."). | ondrop | rooms 19133, 19134, 19135 — **all exist** (z191 "Ship's Cabin") | none | – | Deliberately spares corpses so a player's gear can still be recovered by looting their own corpse. |
| room/archive/master_column.lua | The master column of the Elemental Temple: entering it teleports the player to the branch plane matching the *first* elemental talisman they are missing (or to the Galeru chamber if they hold all four), with a "your vision fades" message naming the missing element. | enter | room 1315 and destinations 1320/1331/1342/1353/1372 — **all exist** (z13) | **ASSIGNROOM(1315, elements_master_column)** | – | The talisman order earth→air→fire→water is implied only by the `new_loc` table (l.5) and the `obj_name` list (l.7) — the only place this ordering is recorded in the archive. |

| room/archive/mastiff.lua | Opening the south panel in room 19195 lets the mastiff (mob 19120) in from the neighbouring room; it walks north through the opening and attacks the first character in the room who has no leader. | oncmd (+mastiff via create_event) | rooms 19195 and 19192, mob 19120 — **all absent** (19120 is now the object "an old journal") | none | create_event | Neat trick: the script does not load a mob, it *moves an existing one* with `action(mastiff, "north")` (l.44), then hand-picks a victim. |
| room/archive/moonsun_forge.lua | The celestial forge: put the moonstone amulet (10208) and the sunstone amulet (10215) into the forge (10212) and `close` it, and both amulets are consumed; moments later the forge glows and a celestial key (10216) appears inside it. | oncmd (+make_key) | room 10245, forge 10212, amulets 10208/10215, key 10216 — **all absent** (z102 gone) | none | create_event | Two latent bugs: `if (not moonstone and not sunstone)` (l.43) should be `or`, and `extobj(moonstone)` would be called with `nil` if only one amulet were present. |
| room/archive/msp_off.lua | Stops any MSP music playing when the player enters this room. | enter | any room (used as a "leaving the inn" hook) | none | msp | Two lines. |
| room/archive/platforms.lua | The four elemental platforms: entering one makes you dizzy and teleports you back to the zone's start (1314). | enter | rooms 1326/1337/1348/1359 (Earthly/Aerial/Fiery/Aquatic Planes) and destination 1314 — **all exist** (z13) | **ASSIGNROOM(1326/1337/1348/1359, elements_platforms)** | – | The "you must complete the puzzle in order" enforcement mechanism of the Elemental Temple. |
| room/archive/portal.lua | A four-room portal loop (2278↔2378, 2279↔2379): `enter portal` teleports to the partner room with "a strange feeling overcomes you". | oncmd | rooms 2278/2279/2378/2379 — **all absent** (ziggurat, z22) | none | – | Note the doubled typo "and and" in l.18, present in `zigg_portal.lua` too — evidence they were copied from one another by hand. |
| room/archive/sacrifice.lua | The Serapis sacrifice altar: `put <item> in basin` records who is offering; each pulse the basin's contents are consumed (a bit of ash excepted), and if any offered item met the criteria (cost over 2000 and a rare percentage load) the sun god (10205) is loaded wearing a trident (10210) and drops the Sun talisman (10215) — otherwise he declares "You dare to offend the gods with your trite offerings?! Prepare to die!" and attacks the offerer. | oncmd, onpulse | room 10221, basin 10207, mob 10205, trident 10210, talisman 10215, reagent exception 1225 — **all absent except obj 1225** (z102 gone) | none | – | A brutal, well-realised risk/reward ritual. It uses the global `basin_char` to remember the offerer across the two triggers, and it burns *every* offering whether or not it qualified. |
| room/archive/start_room.lua | The death/rebirth room: on entry every command except `recall` is swallowed for non-immortals, and each pulse the player is read the game's founding speech — "now is not your time to die… Prove your worth and I may well grant you eternal life. Trust no one, for all here are but dark pawns above which you must struggle to prove yourself. All here strive to be a king...at any cost." — and then recalled to their starting temple. | oncmd, onpulse | room 8099 — **exists** (z80) | **ASSIGNROOM(8099, start_room)** | – | The origin of the game's own name, and the only script in the archive that is *lore primary source*: "all here are but dark pawns". |
| room/archive/tattoo_rune.lua | The rune-rubbing half of the tattoo system: `use <note>` on a rune room makes a rubbing (obj 1209) encoding the design type in `val[1]` and its gold price in `val[2]`, adds a descriptive extra ("This one depicts Jehduti, the Moon God.") and destroys the note. | oncmd | rune rooms 1402, 2627, 4692, 5631, 6536, 9132, 10214 — **only 2627 (z25), 4692 (z46) and 9132 (z91) survive**; obj 1209 — **absent** | none | – | The design table (l.20-28) is the authoritative list of the seven tattoos, their prices (5-40, ×1000 by the tattooist) and their room vnums, and the only place these are tied to geography. |
| room/archive/temple_music.lua | Plays one of six random temple MIDI files when a player enters a temple. | enter | any temple room | none | msp | Identical to `inn_music.lua` with different file names and the tag "Temple". |
| room/archive/thrall_key.lua | Guarantees the dungeon key: if a player enters room 20097 without the thrall key (obj 20023), it is loaded onto the thrall (mob 20051) standing there — or, if the thrall is missing, into the player's own hand with "A key magically appears in your hand!". | enter | rooms 20097 and 20071, obj 20023 — **exist** (z200, Dark Mage's Tower); mob 20051 — **absent** (only a room 20051 exists) | none | – | A blunt but effective anti-softlock script; a sibling of `load_recall.lua`. |
| room/archive/wight_entrance.lua | With DETECT MAGIC, the stone in room 3703 shimmers and reveals a passage to Wight Island; `look stone` describes it, and `down`/`enter` carries the player into the island (room 3709). Offers an MXP link to `look stone`. | oncmd, enter (+entrance via create_event) | room 3703 and destination 3709 — **absent** (zone 37 gone) | none | create_event, mxp, and `teleport` — l.45 calls `teleport(me, 3709)` where the API is `tport` | Two independent breakages: the detection gate is fine, but the timed `entrance` message needs `create_event`, and the movement uses a function name that **does not exist anywhere in this C build**. |

| room/archive/zigg_dt.lua | The ziggurat death trap: anyone in rooms 2228/2328 whose total worn + carried weight exceeds 120 crashes through the floor into room 2232. | onpulse | rooms 2228/2328 and destination 2232 — **all absent** (z22 gone) | `zigg_dt` is **declared in `spec_assign.c` but never `ASSIGNROOM`'d** | – | Exempts immortals and anyone with AFF_FLY — the only "weight limit" mechanic in the archive, paired with `zigg_warning.lua`. |
| room/archive/zigg_portal.lua | The ziggurat portal loop (2278↔2378, 2279↔2379). | oncmd | rooms 2278/2279/2378/2379 — **all absent** | `zigg_portal` declared but never assigned | – | A near-copy of `room/archive/portal.lua`, including the "and and" typo. |
| room/archive/zigg_recess.lua | The ziggurat key puzzle: `put <key> in recess` compares the room and the item against a six-entry combination table and, on a match, silently opens the matching secret door on **both** sides of the wall with grinding-stone messages. Three mechanism types are handled: a trapdoor under an altar, a wall sinking in the west, and a wall sinking in the north. | oncmd | rooms 2211/2311 (trapdoor), 2222/2322 (west wall), 2231/2331 (north wall) and keys 2200/2210/2209 — **all absent** | `zigg_recesses` declared but never assigned | – | The best-engineered mechanical puzzle in the archive: `exit_flags(..., "remove", EX_LOCKED/EX_CLOSED)` applied to both rooms, plus a `room_add = 100` trick that serves the mirrored second half of the dungeon from the same code. |
| room/archive/zigg_warning.lua | The warning one room before the weight trap (rooms 2224/2324): if your gear weighs more than 120, "Your weight causes one of the sandstone blocks to move markedly!". | enter | rooms 2224/2324 — **absent** | none | – | Does the same weight arithmetic as `zigg_dt.lua` — duplicated logic, in true MUD fashion. |

## Synthesis

### Themes

* **Combat AI / monster abilities (~40 scripts).** Two families: generic class AIs
  (`cleric.lua`, `magic_user.lua`, `fighter.lua`, `paladin.lua`, `sorcery.lua`) and 5-20 line
  per-monster gimmicks (`anhkheg`, `drake`, `gazer`, `griffin`, `porcupine`, `thornslinger`,
  `weatherworker`, `caerroil`, `bradle`, `snake`, `paralyse`, `fire_ant`, `ettin`, `kelpie`,
  `mindflayer`, `memory_moss`, `troll`, `werewolf`, `beholder`, `triflower`). The class AIs are
  the only scripts written for reuse — `magic_user.lua` is `dofile`d by `dracula.lua` and
  `medusa.lua`, `sorcery.lua` by `bane`/`pyros`/`valoran`, `fighter.lua` and `breed_killer.lua`
  by `cityguard.lua`, and `cityguard.lua` back by `take_jail.lua`.
* **Ambience / `sound` (~20).** A single random-emote hook was used as the whole atmosphere
  layer: `citizen.lua` has eight variants, `puff.lua` twenty-one, `tyr.lua` four. `puff`,
  `bhang`, `seiji`, `mime`, `singingdrunk`, `cuchi` and `quanlo` are jokes or in-jokes,
  occasionally naming a real person (`aki_kuroda`).
* **Shops, services and the economy (~28).** Banking (`banker`, `bank`), refunds and starter
  kits (`clerk`), appraisal (`identifier`), curse removal (`remove_curse`), enchantment
  (`enchanter`), forging (`crystal_forger`, `dragon_forger`, `blacksmith`), item crafting
  (`head_shrinker`), pets (`pet_store`), mounts (`stable`, `mount`), hirelings (`mercenary`),
  teleport fares (`town_teleport`), shop policy (`shop_give`, `shopkeeper`), training
  (`teacher`, `spell_scroll`), alchemy recipes (`mix_lookup`) and tattoos (`tattoo`,
  `tattoo_rune`).
* **Quests and puzzles (~20).** `merchant_inn`/`merchant_walk` (an escort across two thirds of
  the world), `prisoner` (rescue), `spiritwood_tree` (treasure-map hunt), `falycion_trap` +
  `falycion_skeleton` (a spider dungeon with a fairness valve), the Elemental Temple sequence
  (`master_column`, `platforms`, `cylinders`, `alltalismans`, `minion`, `fly_exit_up`,
  `galeru_dead`), the ziggurat (`darts`, `zigg_warning`, `zigg_dt`, `zigg_recess`,
  `zigg_portal`), `citadel` (the aboleth trap) and `moonsun_forge`.
* **The Grey Keep ritual (11 files).** `enchanter` → `keep_sorcerer`/`barrier` →
  `ceremonial_scroll` → `daemonic_focus` → `pyros`, wrapped in `bane`, `valoran`,
  `keep_conversation`, `keep_portal`, `horn`, `phoenix`, `tinderbox`, `souleater`. The most
  ambitious content in the archive; it reads like a designed *instance*: reagent collection, a
  spell-assignment crafting mini-game, a five-player positional ritual, an area-death phase and
  an RNG banishment with a wandering-boss failure state.
* **City life and justice.** `cityguard`, `guard_captain`, `take_jail`, `jailguard`,
  `jail_noentry`, `jail_noexit`, `clerk`, `banker`, `janitor`, `donation`, `citizen`, `beggar`,
  `elven_prostitute`, `towncrier`, `shopkeeper`, `town_teleport`, `merchant_inn`.

### Evidence of coherent, cross-script systems

1. **The Grey Keep daemon-summoning ritual.** Eleven files share the globals
   `keep_conv_state`, `keep_people`, `keep_focus_owner`, `keep_ceremony`, `keep_pyros`,
   `keep_bane_dead` and `keep_val_dead`. Two mob scripts (`bane.lua`, `valoran.lua`) run a
   conversation state machine by ping-ponging timers; `ceremonial_scroll.lua` encodes a
   four-reagent recipe into `focus.val[1..4]`; `daemonic_focus.lua` gates the ritual on five
   candles and five occupied rooms and then kills everyone in the centre room; `pyros.lua` and
   the `death` hooks reset the world for the next attempt. Even the onboarding is scripted —
   `enchanter.lua` explains where to buy the focus and tells you to study sorcery first.
2. **The golem crystal-mining economy (zone 117).** A three-stage production line:
   `golem_miner` mines chunks (obj 11701), `golem_to_crate` moves them into crates (obj 11702)
   while the crate weighs under 20, `golem_from_crate` empties crates and tips the chunks into a
   bucket at the mine entrance (room 11708) where they vanish; `crystal_forger` then sells
   chunks back as crystal gear for gold. It has both a **sink** (the bucket destroys chunks
   forever) and a **faucet** (chunks → gear), making it the only scripted economy in the set.
3. **The bread chain.** `farmer_wheat` (Mist Keep, mob 5305) leaves wheat (obj 5300) in the
   fields; `miller` (plains of Dor-Sefrith) pays 30 gold for wheat and produces sacks of flour
   (obj 15100) on the ground; `baker_flour` pays 50 gold for flour, produces dough (obj 8015)
   and bakes bread (obj 8010) for one gold; `baker_dough` pays 20 gold for dough and hands over
   bread. The dialogue names both ends ("the baker in Kir Morthis", "the flour mill on the
   plains of Dor-Sefrith") — a deliberate cross-continent supply chain.
4. **The wood economy.** `wood_axe` lets a low-level player chop wood (obj 1221) in forest
   sectors; `forester` pays 50 gold for it. Two files are the entire system.
5. **Newbie onboarding.** `creation.lua` asks three questions and sets attributes, alignment,
   gold and equipment; `clerk.lua` turns the resulting lifestyle into a letter (1245/1246/1247)
   and issues a backpack, a class/race starter kit and a **bond certificate stamped with the
   player's id**; `banker.lua` cashes that bond at level 5; `teacher.lua` and `spell_scroll.lua`
   are the skill/spell purchase path; `load_recall` guarantees an escape item. Five files
   describe an onboarding pipeline that is entirely absent from the C tree.
6. **The tattoo system.** `tattoo_rune.lua` (rubbing designs off runes in seven specific
   rooms), `tattoo.lua` (a conversational parlour with a 7-design catalogue, a colour-matching
   rule and a price of 1000× the design's value), `mix_lookup.lua` (manufacturing the dyes),
   plus a per-character `GET_TATTOO` bit and four C `tattoo` specials. The most cross-cutting
   feature in the archive: lore (`Jehduti, the Moon God`, `Aethen, the Sun God`), economy,
   geography and a persistent character flag.
7. **The Elemental Temple (zone 13).** `master_column`, `platforms`, `cylinders`,
   `alltalismans`, `minion`, `fly_exit_up` and `galeru_dead` reference only vnums that still
   exist, and every one has an equivalent C special (`elements_master_column`,
   `elements_platforms`, `elements_load_cylinders`, `elements_galeru_column`, `elements_minion`,
   `fly_exit_up`, `elements_galeru_alive`).
8. **The jail system.** `take_jail` (arrest) + `jailguard` (release/timer) + `jail_noentry` and
   `jail_noexit` (per-city door directions) + `guard_captain` (which polices the guards
   themselves), with a hard-coded city→cell map covering all four cities.

### Which look finished, and which look broken

* **Finished and self-contained:** the short combat scripts (`anhkheg`, `bradle`, `caerroil`,
  `drake`, `ettin`, `gazer`, `griffin`, `kelpie`, `paralyse`, `porcupine`, `snake`,
  `thornslinger`, `weatherworker`, `troll`, `werewolf`, `fire_ant`); all `sound`-only ambience;
  `alltalismans`, `cylinders`, `platforms`, `master_column`, `galeru_dead`, `fly_exit_up`,
  `darts`, `loseitems`, `falldeck1`/`falldeck2`, `elemental_room`, `mymic`, `thief`,
  `eq_thief`, `brain_eater`, `guardian`, `head_shrinker`, `identifier`, `remove_curse`,
  `souleater`, `francesca`, `thrall_key`, `zigg_recess`, `tattoo_rune`, `bank`, `shop_give`.
* **Broken because the engine lacks the API.** `create_event` — `autodraw`, `bane`, `clerk`,
  `merchant_inn`, `merchant_walk`, `remove_curse`, `sandstorm`, `take_jail`, `tattoo`,
  `valoran`, `zen_master`, `ceremonial_scroll`, `daemonic_focus`, `mastiff`, `moonsun_forge`,
  `wight_entrance`. `mxp` — `autodraw`, `merchant_inn`, `tattoo`, `teacher`, `town_teleport`,
  `wight_entrance`. `msp` — `inn_music`, `temple_music`, `msp_off`.
  `get_group_lvl`/`get_group_pts`/`skill_group` — `teacher`, `spell_scroll`. `get_spell` —
  `spell_scroll`, `load_recall`. `get_dye` — `tattoo`. `teleport` (the API is `tport`) —
  `wight_entrance`.
* **Broken because the entry point is never called:** anything whose only real entry point is
  `code` (`autodraw`, `stable`, `teacher`, `tattoo`, `mix_lookup`), `code_one`/`code_two`
  (`falldown`), or `question1`-`question3` (`creation`). No `run_script` call in `src/` passes
  any of those names.
* **Dependent on vnums that no longer exist** (per row above): the whole zone-14 Keep, the fire
  ants (z17), the ziggurat (z22), the patterns (z30), Wight Island (z37), Mist Keep (z53), the
  aboleth citadel and the Serapis temple (z102), and the zones behind most ranged vnums such as
  422-435.
* **Internally buggy:** `guard_captain` (writes a number into `room.char[i]`),
  `neckbreak`/`strike` (`ch` is the mob in a pulse trigger), `breed_killer` (`obj_list` searches
  the wrong inventory), `prisoner` (`NILL` typo), `moonsun_forge` (`and` instead of `or`),
  `puff` (unreachable case 39), `cleric` (self-contradicting heal thresholds),
  `head_shrinker` (double-counts necklace weight), `sandstorm` (destination vnum mask assumes a
  fully-populated zone).

### Surprises, ranked

1. **`create_event` is a dead stub** (`src/scripts.c:1616`), yet it is the backbone of every
   timed script in the archive. A generation of content was written against an API the shipped
   server did not have.
2. **`cuchi.lua` grants implementor level to a character named `Orodreth`** (l.16-17) — a
   name-based privilege check, not even an account flag.
3. **`start_room.lua` contains the sentence that named the game**: "Trust no one, for all here
   are but dark pawns above which you must struggle to prove yourself."
4. **Mist Keep (zone 53) has been erased from the world** — mobs, objects, rooms and zone file
   all gone — taking its jail, bakery, farm and teleport destination with it. "Kir Morthis"
   appears nowhere in the current world files (it is now Alaozar); only these scripts remember
   either name.
5. **The Elemental Temple is the counter-example to the whole archive**: its scripts survive
   *because* they were superseded by C specials, and every vnum still lines up.
6. **The archive is one SVN commit** (rev 1424, 2008-04-27, author `jravn`) covering all 165
   files, so the metadata dates the *archiving*, not the writing. The scripts predate April 2008,
   and their Lua-3-era idioms (`room.obj`, `foreachi`, `NIL`, `call(fn, …)`) suggest some are
   from the late 1990s. `room/30/*` was committed a few weeks later (rev 1523, 2008-06-20).
7. **Several C specials and Lua scripts were written for the same target** — the silver horn
   (obj 14415) is both `ASSIGNOBJ(14415, horn)` and `obj/archive/horn.lua`. This is a migration
   in progress, abandoned in flight.
8. **`aurumvorax.lua` still reads `room.obj` (singular)** — a fossil of a much older API than
   the one in this tree, and direct evidence of how long the script system had been in use.
9. **Most of the named targets are gone, and the attrition is not random.** 79 of the 155 vnums
   named in the scripts' own comments (51%) no longer exist anywhere in `lib/world`. Whole zones
   were deleted; the surviving scripts cluster around the four cities that remain
   (Kir Drax'in 80, Kir-Oshi 182, Xixieqi 48, Alaozar 212), the Elemental Temple and the Grey
   Keep.
10. **Two Pyroses.** Mob 7920 in the random-load zone is "Pyros, the Daemon Lord", while the
    Keep's summoned Pyros is mob 1410 — and the head-shrinker script's comment points at 7920
    while the demon's own script points at 1410. The same character existed in two zones; only
    the random-load one survives.

## Method and confidence

**Checked mechanically (so these claims should be solid):**

* **Inventory.** `find` over `lib/scripts` gave 179 `.lua` files: 167 archived (115 mob, 14 obj,
  36 room) plus the 2 in `room/30`, `globals.lua`, the four attached scripts
  (`mob/122/healer.lua` → mob 12220 "an old ssauran healer", bitmask 24 = `MS_ONGIVE|MS_SOUND`;
  `mob/144/hisc.lua` → mob 14412 "Hisczdrezzez", 512 = `MS_ONCMD`;
  `mob/212/blacksmith.lua` → mob 21210 "Hashkar the blacksmith", 24;
  `mob/212/highpriest.lua` → mob 21221 "the High Priest of Alaozar", 24), the unattached
  `mob/144/gatekeeper.lua`, and 5 other live library/example files
  (`mob/{assembler,dog,never_die,no_move}.lua`, `obj/portal.lua`, `room/pattern_tport.lua`).
  Reported rows = 115 + 14 + 36 + 2, verified by re-grepping the finished report against the
  filesystem. (`lib/world/mob/43.mob:116` carries a `Script: 0 0` stub on mob 4306.)
* **Every script was read in full** (not summarised) before being described.
* **SVN metadata** was parsed from `.svn/entries` (SVN 1.6 XML working copy) for all 167 files:
  165 at `committed-rev 1424`, `2008-04-27`, author `jravn`; `room/30/*` at rev 1523,
  `2008-06-20`, `jravn`; root `lib/scripts` at rev 1525 from `file:///var/svn/trunk/2.2/...`.
  I also md5-compared every working file against its `.svn/text-base` copy: **nothing** in the
  oracle working copy has been locally modified.
* **API surface.** I read `boot_lua()` and `run_script()` (`src/scripts.c:1701-1820`), the
  `cmdlib` table (`src/scripts.c:1609-1664`), all `run_script` call sites in `src/*.c`, the
  `MS_*`/`RS_*`/`OS_*` bitmasks (`src/structs.h:663-685`) and the Lua 4.0.1 library
  registrations (`src/lua/src/lib/{lbaselib,lmathlib,lstrlib}.c`, all registered as **globals**
  via `luaL_openl`) — so plain `sort`, `foreachi`, `getn`, `tremove`, `mod`, `exp`, `format`,
  `strfind` are *not* missing.
* **Missing-function lists** were produced by a script that extracts every `name(` call from
  each file, subtracts functions defined in the file, trigger names, Lua 4.0.1 base/math/str
  globals and the `cmdlib` set. The result set was then read by hand to check each hit.
* **Vnum resolution.** I built an index of every `#vnum` entry in `lib/world/{mob,obj,wld}`
  (1320 mobs, 1674 objects, 10049 rooms) plus zone names from `lib/world/zon/*.zon`. The counts
  match the raw `#vnum` line counts exactly, so the index is complete. Zone absence was
  confirmed by `ls lib/world/{mob,obj,wld,zon}` — there is no file for zones 4, 14, 17, 18, 22,
  30, 37, 53, 61, 62, 68, 101, 102 or 120.
* **C specials** were parsed out of `src/spec_assign.c` by regex (280 `ASSIGNMOB`/`ASSIGNOBJ`/
  `ASSIGNROOM` lines) after stripping comments, so "has a C special / does not" is machine-checked.
* **Go-copy comparison** used byte comparison plus a normalised (comments and whitespace
  removed) comparison of all 115 same-named pairs, plus `diff` inspection of the 15 pairs with
  real code differences.

**Inferred, and marked as such:**

* Where a script does not name its own vnum (e.g. `anhkheg.lua`), the target I list is a
  **guess** derived from the mob's name/location; those rows say so. `forester.lua`'s comment
  names "mob 9160", which is a room — I report the comment, not a corrected vnum.
* The claim that zone **14** was the *old* Grey Keep (renumbered to zone 144) is an inference:
  no `14.zon` exists, the Keep scripts use vnums 14xx, `144.zon` is "The Grey Keep (Nimos)", and
  mob 14401/14405/14411/14415/14420/14432 carry the same C special names (`never_die`,
  `teleport_victim`, `teleporter`, `brain_eater`) as the Keep scripts' intent. It fits, but I
  cannot prove the renumbering.
* "Lua-3-era idiom" (`room.obj`, `foreachi`, pervasive `NIL`, `call(fn, …)`) is my reading of
  the code style; the scripts could equally be Lua 4 code written by an older hand.
* I did **not** verify which of these files were ever attached to a live mob in *this* world
  build; only four `Script:` lines exist in the whole world
  (`mob/122.mob:319`, `mob/144.mob:202`, `mob/212.mob:166`, `mob/212.mob:343`, plus a stub
  `Script: 0 0` at `mob/43.mob:116`), and none of them points at an archived file.
* `create_event`'s absence is certain (commented out at `src/scripts.c:1616`); whether it
  worked in a later branch is **not** something I can check — the comment there reads
  "XXX not supported yet - needs 3.0 event system", which hints at a later codebase I do not
  have access to.
* Nothing was executed. All descriptions of play behaviour are read from the source; no live
  server or Lua interpreter was run, and the network was not used.

**Known limits.** Vnum→name lookups are ambiguous across the mob/obj/room namespaces; where a
vnum exists in more than one namespace I list all of them. A few zone numbers are used by
several content authors, so "zone name" is the zone file's own title, not an attribution of the
script. Line numbers cited are from the oracle tree as it stands at
`2d244524b66c09480cdf5660dc48781574b2bbea`'s sibling checkout and can shift if that tree moves.

## Optional: C archive vs. the Go port's copy (read-only)

`/home/zach/darkpawns/lib/world/scripts` is a *modified* copy and was **not** used as a source
for any claim above. What I found when comparing it (115 same-named pairs, plus structural
inspection):

* **Structure.** The Go tree has **no `obj/` or `room/` directory at all** — all 14 object and
  all 38 room/pattern archived scripts were dropped. It holds 149 `.lua` files: 115 flat
  `mob/*.lua` copies plus 30 copies filed under per-vnum directories
  (`mob/11700/`, `mob/11701/`, `mob/11702/`, `mob/1314/`, `mob/144/`, `mob/145/`, `mob/15100/`,
  `mob/19114/`, `mob/19118/`, `mob/212/`, `mob/21245/`, `mob/4209/`, `mob/5510/`, `mob/7903/`,
  `mob/7917/`, `mob/7920/`, `mob/8015/`, `mob/9111/`, `mob/9147/`, `mob/122/`), plus
  `globals.lua`. The vnum-directory names are the wiring: it is done by *where the file sits*,
  not by a `Script:` line, and it duplicates one script across several mob directories where the
  C comments named only one (e.g. `golem_miner.lua` exists under 11700, 11701 **and** 11702,
  because the C file's comment says 11702 while its logic is written from the miner's side).
* **Provenance comments were added to every file**, e.g. `-- Source: scripts_full_dump.txt
  ./mob/archive/aurumvorax.lua` or
  `-- Source: /home/zach/.openclaw/workspace/darkpawns/docs/scripts_full_dump.txt lines 419-452`
  — i.e. the Go copies derive from a *text dump*, not from the C tree directly.
* **Real code differences exist in 15 of the 115 pairs** (comments and whitespace normalised).
  The rest differ only by added headers, blank lines and trailing newlines. A "DEAD-LETTER"
  marker was added where the target vnum no longer exists (`aki_kuroda.lua`, `bane.lua`).

| script | how the Go copy differs |
|---|---|
| `aki_kuroda.lua` | Purely annotative: adds `-- DEAD-LETTER: Mob 12915 does not exist in ported world`, a file name line and a prose summary. Code identical. |
| `aurumvorax.lua` | Adds only a `-- Source:` line and drops the trailing newline. Code identical. |
| `bane.lua` | Two diagnostic headers and **real changes**: every `create_event(me, ch, …)` inside a pulse/timer callback becomes `create_event(me, NIL, …)`, with a comment claiming this fixes a nil-`ch` crash in the timer context. The cooperative state-machine logic with `valoran.lua` is retained unchanged. |
| `pyros.lua` | One line: the C `call(fight, room.char, "x")` becomes `call(fight, ch, "x")`. |
| `cityguard.lua` | The most heavily rewritten: `bribe` and `fight` are removed entirely and `onpulse_pc` is reimplemented with new text ("$n shouts 'Halt, outlaw!'") and a four-argument call `action(me, "kill", NIL, ch)` — a shape the C `lua_action` does not accept. |
| `clerk.lua` | Entitlement tables renamed (`equip_invent`/`equip_pack` → `class`/`race`) and the **player-id ownership check** on the lifestyle letters (C l.22-28) removed. |
| `banker.lua` | Substantially rewritten (36 → 22 normalised code lines): the payout formula `1500/obj.val[1]` becomes a hard-coded 100/500/1000 read from `obj.value[1]`, and both the level-5 gate and the player-id anti-forgery check are gone. Calls `extobj(obj, "destroy")` with an extra argument the C binding ignores. |
| `merchant_inn.lua` | 112 → 114 code lines; the `mxp`/`create_event` calls are still there, so it needs the Go engine's extra bindings. |
| `teacher.lua` | Adds an `-- Engine gaps: skill_group(), get_group_lvl(), get_group_pts(), set_skill(), mxp(), exp(), format()` header (note `exp`/`format` are in fact Lua 4.0 globals) and reflows comments; logic identical. |
| `troll.lua` / `mymic.lua` | Line-by-line provenance annotations ("`-- troll.lua line 5: regen to level*2`"); `mymic.lua` gains `-- TODO: requires steal(ch, obj) implementation`. |

Two observations worth recording, both **outside** the file-level comparison and clearly marked
as beyond scope: the Go engine registers the functions the C build never had — I saw
`create_event`, `mxp`, `get_group_lvl` and `skill_group` bound in `pkg/scripting/engine.go:646-675`
— which is why these scripts are usable there at all; and the Go `globals.lua` moves the
constant block out of `function default()` to top level, where the C build relies on
`boot_lua()` calling `default()` (`src/scripts.c:1713`). Neither claim was verified beyond
reading those lines, and the Go tree should not be treated as evidence about the C game.


