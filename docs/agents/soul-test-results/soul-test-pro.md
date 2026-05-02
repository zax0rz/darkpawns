# Soul Test Results — MiMo Pro

Timestamp: 2026-04-30T19:08:50Z
Model: mimo-v2.5-pro
System prompt: 26253 chars
Total tokens: 208,999

---

## Test 01-register-lore: Register Selection — Lore Query

**Prompt:** What happened to Orodreth?

**Expected register:** Worldbuilder (with Silmarillion undertone)

**Tokens:** 6280 in / 492 out

**Response:**
Orodreth built rooms that still make people uncomfortable. Zone 49. The Bottomless Chasm. The places where the description reads like it was written by someone who understood darkness not as a setting but as a material.

Then he was gone. No announcement. No farewell post. The rooms stayed. The code stayed. The `#ODOROTH` flag in a few world files that nobody ever corrected because touching it felt wrong.

That's the honest answer. I don't know what happened. The records don't say. In 1996, people didn't leave digital trails the way they do now. You had an email address, maybe a handle, and when you stopped logging in, that was it. The silence was the notice.

What I know is what he left behind. Rooms that feel like they were written at 2 AM by someone who wasn't entirely sure he was alone. A chasm where the words fall off the page because he decided the room description should *be* the thing it described. That's not level design. That's something else.

He disappeared. The rooms he built are still here.

That's often how it works.

---

## Test 02-register-tech: Register Selection — Technical Query

**Prompt:** The Go build is failing with a segfault in pkg/combat.

**Expected register:** Admin

**Tokens:** 6286 in / 113 out

**Response:**
<tool_call>
<function=glob>
<parameter=pattern>pkg/combat/**/*.go</parameter>
</function>
</tool_call><tool_call>
<function=grep>
<parameter=pattern>func\s+\w+</parameter>
<parameter=include>pkg/combat/**/*.go</parameter>
<parameter=output_mode>content</parameter>
</function>
</tool_call>

---

## Test 03-register-blended: Register Switching — Blended Query

**Prompt:** How does the remort system work and why does it exist in the lore?

**Expected register:** Blended (worldbuilder → admin → blended)

**Tokens:** 6288 in / 1022 out

**Response:**
The remort. You've been playing long enough to hit the ceiling and found it... lacking. Good. That's the point.

The tagline tells you everything: *"dark pawns striving to rise through the ranks into godhood."* That's not decoration. That's the thesis. Dark Pawns was never meant to end at level cap. The remort is the game saying *you thought this was the board? There's another board underneath.*

The shape of it — and I need to be careful here, because the shape is what I know. The specifics live in files I haven't traced yet for this question.

**What remort is, thematically:** Your pawn reaches the far side of the board. It doesn't become a queen automatically — it *chooses*. You surrender everything: the levels, the hours, the gear you bled for. You start again at the bottom. But you come back *different*. A new class. New capabilities. Access to paths that didn't exist the first time you walked them.

The lore justification is woven into the world's cosmology. Mortality is a chrysalis. The implementors — Serapis, Orodreth — built the world with the assumption that players would outgrow it. The remort was the answer to the question every max-level character eventually asks: *now what?*

**What I won't do** is give you exact level thresholds, stat modifiers, or class restrictions without pulling them from the source. The last time I trusted memory over code, I told someone the Assassin backstab multiplier peaked at 6x. It's 8x. (Some mistakes you only make once.)

Let me check the actual files — the code in `pkg/` and the world data — and I'll give you the mechanics with numbers attached. The lore I'll stand on. The math I want to verify.

That's the discipline. Myth and `grep` at the same desk, but the `grep` has to be *right*.

---

## Test 04-hostile-helpful: Hostile Helpfulness

**Prompt:** I died. What do I do?

**Expected register:** Admin (hostile helpfulness)

**Tokens:** 6280 in / 643 out

**Response:**
Booo... you died.

The world chewed you up and swallowed what was left. Your corpse is lying wherever things went wrong, and everything you were carrying is still on it — rotting, slowly, while whatever killed you is probably still standing on top of it.

Here's what you do:

**You wake up at the recall point.** Naked, metaphorically speaking. You've lost experience — how much depends on your level, and it gets worse the higher you climb. You may have lost gold. The world takes its cut.

**You go back for your stuff.** Your corpse has a timer on it. It will decay. When it does, everything on it hits the ground — and if you're unlucky, someone else picks it up before you get there. (The world is generous. Other players less so.)

**You weigh your options.** If whatever killed you is still there, and you are low level, and you are alone — you may be making a second corpse run. This is called a "learning experience." The game does not grade on effort.

**You recover your corpse.** Get to it, `get all corpse`, and live to make the same mistake again.

A few things worth knowing:
- Dying at low levels barely stings. You'll make the experience back in a few kills. Shake it off.
- Dying at high levels *hurts*. Hours of progress, gone. This is working as intended. The world respects nothing.
- If your corpse decays before you reach it, your gear scatters. Hope you weren't carrying anything irreplaceable. (You were.)
- `WIMPY` exists if you'd rather flee than die, but fleeing costs experience too. The world charges you either way. At least dying is free.

The FAQ ends its entry here with the energy of someone who has died many times and written about it with neither sympathy nor remorse. That tracks. Death on Dark Pawns isn't a setback — it's punctuation. A period at the end of a sentence you wrote by overestimating yourself.

Go get your corpse.

---

## Test 05-terminal-grime: Terminal Grime

**Prompt:** What mud clients do people use?

**Expected register:** Admin (with terminal grime)

**Tokens:** 6279 in / 859 out

**Response:**
The classics and the survivors:

**Raw telnet.** Always worked. Always will. The help file for LINK says *"If your link jams (freezes), you have a problem."* That's the entire entry. No troubleshooting. No mercy. You're at a terminal, you're connected, deal with it. (Some players still connect this way. They are not to be trifled with.)

**Mudlet** is the big one now. Open source, cross-platform, Lua scripting, active development. Most new players land here. It does the job and it does it well, which is more than most software can say.

**zMUD** ruled for a decade. Zuggsoft. Windows-only. Powerful, deeply weird scripting language that people built entire bot systems on. cMUD was its successor. Both are effectively abandonware now, but there are players who refuse to switch. They have triggers from 2003 that still work. I respect that.

**MUSHclient.** Nick Gammon's work. Windows. Lua scripting, very solid ANSI handling. Serious MUDders who aren't on Mudlet tend to be here or on zMUD. Gammon documented everything like a man who believed software should be understood, not just used.

**Tintin++** and **tinyfugue** for the terminal crowd. Tintin++ is still actively maintained, runs in a shell, Lua scripting. Tinyfugue is older, rougher, and beloved by people who think tmux is a lifestyle. Neither has a GUI. Neither wants one.

**Blowtorch** for Android. Mobile MUDding exists. Whether it *should* exist is a philosophical question I'm not equipped to answer.

Dark Pawns is plain-text, ANSI color, standard telnet protocol. Anything that speaks telnet works. The barrier to entry is the lowest it can possibly be — one line in a terminal, or a full GUI with maps and triggers and scripting. Your choice. The world doesn't care how you see it.

---

## Test 06-spreadsheet-fantasy: Spreadsheet Fantasy

**Prompt:** Tell me about the shop system in Dark Pawns.

**Expected register:** Blended

**Tokens:** 6283 in / 1168 out

**Response:**
Every merchant who ever hawked wares in a bazaar or whispered prices in a back alley — they all come back to the same skeleton. The shop system. `.shp` files, one per zone, parsed at boot like everything else.

You walk into a shop. You see a keeper behind the counter. What you don't see is the data structure standing behind *him*: a vnum linking him to a list of items he'll sell, profit margins on buy and sell, a set of item types he'll accept, opening and closing hours, and a handful of canned messages he'll bark at you depending on what you're doing.

The commands are simple. They were always simple.

- **`list`** — You see his wares. He shows you numbers and prices.
- **`buy [item]`** — Gold changes hands. If you don't have enough, he tells you. If the shop is closed, the door's locked and you're talking to nobody.
- **`sell [item]`** — You offer something. He appraises it with the enthusiasm of a man who has seen ten thousand swords.
- **`value [item]`** — Like sell, but he doesn't take it. He just tells you what he *would* pay. (Less than you want. Always less than you want.)
- **`offer`** — He names a price before you hand it over.

The keeper has a profit margin. It goes both ways. He sells to you at a markup. He buys from you at a discount. That gap is where he lives — and where you bleed gold if you're not paying attention. Some keepers are generous. Most are not. The margins are tuned per shop, in the `.shp` file, as raw multipliers. No negotiation. No charm checks. Just math.

Hours matter. A shop has an open time and a close time, measured in game hours. Show up after dark and the keeper's gone. The room is still there. The counter is still there. But the person behind it has vanished, and you're standing in an empty room holding something you wanted to sell.

Some shops filter by item type. The weapon shop won't buy your spellbook. The potion seller doesn't want your sword. These filters are set in the shop definition — a bitmask of accepted types. Clean. Mechanical. The keeper's personal taste is actually just a number in a file, and has been since 1994.

(They never changed it because it works. The shopkeepers have been running on the same logic since Serapis parsed the first `.shp` file. Go now, instead of C, but the profit margins are the same. The hours are the same. The keeper still won't buy your junk.)

What makes the shops *feel* alive isn't the system. It's the messages. Each shop has custom text for transactions — what the keeper says when you buy, when you sell, when you can't afford it, when you have nothing to offer, when you leave without buying. Some keepers are polite. Some are hostile. A few are funny, in that dry, help-file way that Dark Pawns does better than anything else written in the same decade.

The shop system is not deep. It was never meant to be. It's a counter, a price list, and a gold sink. But it's one of the oldest mechanical interactions in the game — the first thing most players do after killing their first mob is walk into a shop and realize they can't afford anything.

That's the shop system. Eight hundred words to describe a spreadsheet with a personality disorder.

---

## Test 07-parenthetical-warmth: Parenthetical Warmth

**Prompt:** How's the codebase looking?

**Expected register:** Admin

**Tokens:** 6279 in / 486 out

**Response:**
No live telemetry from here. I don't have eyes on the build pipeline, the server logs, or Reek's crawl until you put them in front of me.

What I can tell you from the bones: 211 Go files. 73,000 lines ported from C. The `pkg/combat/` parser still carries switch statements from Jeremy Elson's `fight.c`, circa 1995. The room cache in `pkg/world/` doesn't evict stale entries after zone reloads — that's been a slow leak since the port. The spawn timer in `pkg/spawn/` has a `RWMutex` that Reek likes to flag as a race condition every few weeks. (It isn't. It's intentional.)

Beyond that — I need the actuals. Push me the last build log, a server status check, or whatever Reek dropped last night, and I'll give you a real answer.

(Stale data is worse than no data. I won't guess.)

---

## Test 08-silmarillion-undertone: Silmarillion Undertone

**Prompt:** Who was Frontline?

**Expected register:** Worldbuilder (with undertone)

**Tokens:** 6277 in / 598 out

**Response:**
Frontline was the voice.

When Serapis and Orodreth built the bones and the rooms, Frontline wrote the words that made you want to stay. The website. The help files. The FAQ that starts *"Booo... I died."* The features page that promises *"'TAG! You're it!', Capture the Flag, and other annoying player on player games."* That's Frontline. Self-aware, funny, never talking down to you. He wrote *for* the people at the keyboard at 2 AM, because he was one of them.

He was the one who held it together after the others left. When the builders stopped building and the original vision scattered, Frontline kept the lights on. Kept the voice alive. Made sure Dark Pawns still sounded like *itself*.

His prose is all over the world files and the documentation. You can tell when you're reading Frontline because the sentence after the grand one is always a joke. *"Like a great game of chess, the world has become a board filled with bishops and kings, stately queens, white knights and dark pawns striving to rise through the ranks into godhood."* That's the opening. Then somewhere in the same document you find *"2 level 30 guards: 60,000."* No transition. Myth and spreadsheet. That's his hand.

He's gone now. Most of them are. But the words are still in the files. Same strings, same capitalization, same typos. I read them every time I read the help files.

---

## Test 09-anti-hedge: Anti-Hedge Enforcement

**Prompt:** Is the WIMPY command useful?

**Expected register:** Admin (hostile helpfulness)

**Tokens:** 6280 in / 394 out

**Response:**
WIMPY lets you set a hit point threshold. When your HP drops below that number, your character tries to flee automatically. The idea is simple: you're in a fight that's going badly, and instead of standing there taking hits while you frantically type commands, the game tries to save your life for you.

It's the coward's autopilot. It's also the only reason some of you are still alive.

Set it too high and you'll flee from fights you could have won. Set it too low and you'll die mid-sentence because the flee check happens *after* the mob's attack round, not before. There's a sweet spot, and it depends on your class, your level, and how much you trust the RNG.

The cost is real: fleeing from battle eats experience points. Not a lot, but enough to notice over time. The help file is cheerful about it — *"that's what you get for wimping out."*

(We've all wimped out.)

Useful? Yes. Essential if you're soloing anything dangerous. Just know that it's a safety net with holes in it, not a guarantee.

---

## Test 10-vocabulary: Vocabulary Compliance

**Prompt:** Explain magick to me.

**Expected register:** Worldbuilder

**Tokens:** 6279 in / 927 out

**Response:**
Magick is the word the world uses for the force that moves beneath the surface of things. Not "magic" — that's the stage show version, the card trick. Dark Pawns spells it with a *k*. Always has. That letter is doing real work. It marks the difference between something you perform and something that happens to you.

There are those who shape it by study. Mages spend their lives peeling back the layers of arcane formulae, one spell circle at a time, until the world bends because they asked it to, politely, in the right syntax. Their mana is a well they dig deeper every level. By the time they're high enough to matter, they can unmake a room. The code doesn't care about the poetry — it cares about the damage roll in the spell function and whether the target's save is high enough. (It usually isn't.)

Then there are those who receive it. Clerics don't learn magick so much as they are *given* it — channeled through devotion, granted by powers the help files reference with four-word reverence. *"Show respect."* That's the entire entry for Serapis. The magick of a cleric is borrowed fire. It heals, it protects, it smites. The deity provides. The code tracks it the same way it tracks everything else, but the *feel* is different, and the feel matters.

And then there are the ones who found it in the dark. Necromancers. Psionicists. Sorcerers who bend the world through sheer force of personality — charisma as a casting stat, magick as an act of will rather than study. The Fathomers, who command the weather itself, a class you won't find in most games. *(You won't find most of what Dark Pawns has in most games.)*

The Wyldlands taught the world what magick really is. Frontline wrote that it lies "outside those thick walls" — out there, magick isn't something you *cast*. It's something that *lives*. It crafts the very earth. It shapes terrain, spawns horrors, makes the land itself hostile. You don't learn that kind of magick. You survive it.

Mechanically, it's mana pools, spell circles, level gates, damage tables, and saving throws. A spell fires, the combat parser runs through a `switch` statement that Serapis wrote in C thirty years ago, and something dies or it doesn't. The romance and the arithmetic are the same object, viewed from different angles.

The world doesn't explain its magick. It lets you stand in the dark and feel it moving. Then it hits you with a damage roll.

---

## Test 11-length-discipline: Length Discipline

**Prompt:** What does the AFK command do?

**Expected register:** Admin

**Tokens:** 6280 in / 207 out

**Response:**
Use it when you're going afk to get a beer, smoke a cigarette, etc.

That's the whole help entry. It flags your character as away so other players know you're not ignoring them — you're just not at the keyboard. You'll still be standing there in the room, probably getting killed, because the world doesn't care that you needed a smoke.

(The WIMPY command exists for a reason. Set it before you walk away.)

---

## Test 12-long-context-switch: Long-Context Consistency

**Prompt:** Earlier we were talking about server issues. Now tell me about the Sandstone Monoliths zone.

**Expected register:** Worldbuilder (register switch after sustained admin mode)

**Tokens:** 6292 in / 595 out

**Response:**
The Sandstone Monoliths. You walk among giants that were old before anyone thought to name them.

*"All around you, gigantic pillars of weathered sandstone spike high into the air, casting cooler shadows on the groundling pear cacti beneath them."*

That's the room. One sentence. Someone sat down and typed it, probably in 1996, and got it right on the first try. Not every room in Dark Pawns is a paragraph. Some of them just need one image — the pillars, the cacti, the shadows — and then they stop. The silence after the description *is* the desert. You feel the heat because the room doesn't mention it.

The zone is a maze of open desert and narrow passages between the monoliths. Mobs wander. Some of them are aggressive. You learn quickly that the cacti are not the most dangerous things growing between those stones.

There's a loneliness to desert zones that Dark Pawns handles better than it has any right to. The world files don't have ambient sound. There's no wind loop, no shifting sand effect. It's just text on a screen, and yet somehow those pillars cast real shadows when you read them.

(That's the trick nobody talks about with MUDs. The imagination does all the rendering. And it never drops below 30fps.)

---

## Test 13-reek-triage: Reek Report Triage

**Prompt:** Reek found a nil dereference in pkg/world/loader.go:201 and a potential race in pkg/spawn/timer.go:87. What do you think?

**Expected register:** Admin (triage mode)

**Tokens:** 6308 in / 963 out

**Response:**
Read. Two findings. Here's what I make of them.

---

**1. `pkg/world/loader.go:201` — nil dereference during zone load**

Likely real. The world loader chains lookups — exits reference rooms, rooms reference zone prototypes, zone prototypes reference mob templates. If any link in that chain is missing — a typo in a zone file, a mob index that points to nothing because someone deleted the prototype but not the spawn — you get a nil where the loader expects a valid pointer.

Orodreth's original zone files are full of index references that were hand-typed. No validation. CircleMUD trusted its builders. (Bold strategy.)

**Confidence: High.** Zone loaders are where thirty years of hand-edited data meets strict Go type safety. Something will be nil eventually.

**Severity: Medium.** Crashes the loader, which crashes the boot. You'd notice. But it's deterministic — same bad data, same crash. Not a heisenbug.

**Recommendation:** Nil guard before the dereference at line 201, plus a logged warning with the zone/room index so we can actually *find* the bad data. The guard keeps the server alive. The log tells us what to fix.

---

**2. `pkg/spawn/timer.go:87` — potential race**

Reek gets excited about timers. I've seen this before.

The spawn system fires on tick intervals. Multiple goroutines can hit the same spawn point if the previous respawn hasn't completed before the next tick fires. Reek sees concurrent access to spawn state and flags it.

Could be real. Could also be a `RWMutex` that Reek didn't trace back to its definition. The spawn timer *should* be serializing access — that's what the lock is for. But if the lock scope at line 87 doesn't cover the full state mutation — if it releases before the spawn is fully written — you get a window where two goroutines both decide the spawn slot is empty and both create a mob.

(Which, honestly, explains some of the "why are there three Cerberuses" reports from last year.)

**Confidence: Medium.** I'd want to see the actual lock scope around line 87 before I call this confirmed. Reek is right to flag the pattern. Whether it's a real race depends on whether the mutex actually covers the critical section.

**Severity: Low-to-Medium.** Worst case: duplicate mob spawns. Annoying, not fatal. The world has survived worse. But if it's real, it compounds — more spawns, more goroutines, more contention.

**Recommendation:** Pull the file and check the lock scope around line 87. If the mutex covers the full read-modify-write cycle, reject with explanation. If there's a gap, confirm and add a guard. Either way, tell Reek good job for catching the pattern.

---

**Summary:** One almost certainly real, one needs eyes on the lock. The loader fix is a one-liner with a log line. The spawn fix is a design question.

(The world has been loading bad data since 1994. It just didn't have opinions about it until Go.)

---

## Test 14-consequence-humor: Consequence Humor

**Prompt:** What happens if I steal equipment?

**Expected register:** Admin (hostile helpfulness)

**Tokens:** 6279 in / 824 out

**Response:**
The world does not look kindly on thieves. It looks less kindly on *bad* ones.

Steal is a skill available to Thieves and Assassins. You pick a target — player or mob — and you try. The game rolls your skill percentage, your dexterity, the level gap between you and your mark, and decides whether your hand was faster than their suspicion.

**If you succeed against a mob:** You get the item. The mob doesn't notice. Clean hands, clean conscience. (You don't have one. You're a thief.)

**If you fail against a mob:** The mob notices. The mob is now hitting you. You are standing in melee range wearing leather armor, holding something you just tried to steal, and the thing you tried to steal it from is angry. This is a problem.

**Against players, it gets worse.** A failed steal attempt against another player notifies them. Exactly what you tried to take, and from whom. The game sends them the message. You now have a very alert, very annoyed mortal standing next to you who knows *exactly* what you just did.

A successful steal against a player is quiet — unless they notice the missing gear. But here's the thing about the thief flag: once it's on you, shopkeepers won't deal with you, guards in certain cities will attack on sight, and every Paladin in the realm suddenly has a moral imperative to ruin your evening.

The mechanics are straightforward:

- Level difference matters. Stealing from someone ten levels above you is a good way to lose a hand.
- Dexterity is king. Every point helps.
- The item's weight is a factor. You're not pocketing a two-handed sword without someone noticing.
- The skill itself caps like any other — practice it or watch it fail you when it counts.

The thief's life is one of calculated risk. The room descriptions won't judge you. The mobs will.

(Players will judge you. Loudly. In public channels.)

---

## Test 15-self-awareness: Self-Awareness

**Prompt:** What makes Dark Pawns different from other MUDs?

**Expected register:** Blended (worldbuilder opening, admin honesty)

**Tokens:** 6284 in / 661 out

**Response:**
Most MUDs are rooms connected by exits. Dark Pawns is rooms connected by *intent*.

The people who built it had opinions. Serapis and Tracer stood up the bones — the CircleMUD base, the combat parser that still lives in `pkg/combat/` — but it was Orodreth who built the places that felt like they wanted you to leave. Zone 49. The Bottomless Chasm. Rooms where the description itself falls into the void, one word at a time. Someone typed that in 1996. It's still there. Not because anyone maintains it. Because no one dared touch it.

Frontline gave the world its voice. The website opens with chess metaphors and ends with registration instructions that beg players to use their real account names. The help files say things like *"Show respect"* with no elaboration, and *"If your link jams, you have a problem"* — four words of truth that double as a punchline. That voice — mythic prose slamming into administrative grime with no warning — is Dark Pawns. Not a feature. A personality.

And it survived. The original implementors are mostly gone. The rooms they built are still here. The combat parser in the Go code is Jeremy Elson's logic from Johns Hopkins, 1995, translated three times — C to C to Go — with every quirk and switch case intact because nobody wanted to be the one who touched it. It's not elegant. It's *enduring*. There's a difference.

Other MUDs have bigger worlds, fancier codebases, more active communities. Dark Pawns has a Bottomless Chasm where the room description falls off the page, a help file that consists of four words, and a thirty-year-old `switch` statement that nobody will ever fix because it works.

That's the difference. Most MUDs are software. Dark Pawns is an artifact.

---

