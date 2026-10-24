---
title: "Mickey and Mallory Never Loaded"
date: 2026-10-24
description: "Dark Pawns has a help entry for NBK and a spec proc for Mickey and Mallory Knox, and no mob in the world ever ran it. So I gave them bodies on a scratch copy of the original server and let them kill a warrior named Wayne."
draft: true
textKind: "original"
source: "src/spec_procs.c, src/spec_assign.c, src/db.c and lib/world/ at commit 29994e5; lib/text/help/socials.hlp; archive record topic-t10; session logs captured 2026-10-24 from a scratch build of src/ with two stand-in mobs (docs/research/nbk-session/)"
voiceLayer: "mythic-admin"
---

Type `help nbk` in Dark Pawns and this is the whole entry:

```
NBK

Natural Born Killer
```

It sits in the socials help file, right after the one for KISS ([read it here](https://darkpawns.org/help/socials/nbk/)). Players wore it in their titles. A who list quoted in a [dp-players.com thread from April 2004](https://darkpawns.org/archive/topic-t10/) has three of them on one screen:

> [ 30 Wa ] Altie a Warrior [NBK] (invis) (quest) (OUTLAW)
> [ 26 Wa ] Sparahawk the village pot farmer. [NBK] (quest)
> [ 22 Mu ] Nicolae the village tobacco farmer. [NBK]

An [NBK] in your title meant you were open to player killing, or wanted everyone in Kir Drax'in to think you were.

The tag comes from Oliver Stone's 1994 movie [Natural Born Killers](https://en.wikipedia.org/wiki/Natural_Born_Killers): Woody Harrelson and Juliette Lewis as Mickey and Mallory Knox, two lovers on a killing spree that television turns into celebrities. The game has the Knoxes too. They live in the C source as two spec procs, and as far as I can tell no mob ever ran them.

## What the code wanted them to do

A spec proc is a C function attached to one kind of mob (a monster or NPC). The server calls it every four seconds while the mob is idle and again on every round of combat, which comes around every two seconds.

[Mickey's proc](https://github.com/zax0rz/darkpawns/blob/29994e5c8dc59e8de2cc22e65eb4fd5d172658ce/src/spec_procs.c#L1266-L1371) is a short play in three beats. When he sees a player in the room, he wakes them up and says "$N, I'm gonna kill you." (The game fills in your name for $N.) On the next beat he asks "And you wanna know why?" On the third he checks the room. If it's flagged peaceful, he says "I'll tell you later, $N" and walks out the door to room 11369, an [Edge Of A Canyon](https://darkpawns.org/rooms/11369/) that the code comments call "some room in SE desert." The exit south of that room is named "A Fall to Your Death." Anywhere else, he says "Shit $N, I'm a natural born killer." and hits you. (In the source that line ends with a period, a closing quote and another period, and the game prints all three.)

To remember who he was talking to, Mickey writes your player ID number into his own alignment, the number that tracks how good or evil a character is. If a different player walks in, the numbers don't match, and he starts the speech over for the new person.

Once the fight starts, each round he has a one in two chance of saying something:

> I'll always love you Mal, no matter what!
> Do you believe in fate?
> You're not centered.
> When they come and ask you who did this, tell them it was Mickey and Mallory Knox!
> It's not nice to point.

The diner massacre in the movie ends with the Knoxes leaving one witness alive to tell people who did it. [unsourced: ask Zach to confirm which of the other lines are the film's dialogue, or cut this sentence]

[Mallory's proc](https://github.com/zax0rz/darkpawns/blob/29994e5c8dc59e8de2cc22e65eb4fd5d172658ce/src/spec_procs.c#L1374-L1422) only talks in combat, and she talks almost every round. The server rolls a ten-sided die. On six through ten she shouts "You stupid biiitch!" The rest of the die gets "How do you like me now?", "That was the worst head I've ever got.", a two-line "How sexy am i now, fucker?" and "How sexy am i NOW?", one roll of silence, and "$n giggles madly." Then she finds Mickey, wherever he is in the world, and sets him hunting whoever she's fighting. If he's already hunting someone, she leaves him to it. The comment on that line reads "if mickey's busy, catch him later :)".

## Why nobody ever met them

Procs get attached to mobs in `src/spec_assign.c`. [Lines 102 and 103](https://github.com/zax0rz/darkpawns/blob/29994e5c8dc59e8de2cc22e65eb4fd5d172658ce/src/spec_assign.c#L102-L103) declare `SPECIAL(mickey)` and `SPECIAL(mallory)`, and no line in the file attaches either one.

The code names Mickey's vnum (the ID number a mob has in the world files) as 7969. Mob 7969 is [a black-cloaked wanderer](https://darkpawns.org/mobs/7969/): "A mysterious man in a long black cloak wanders about." He gets the generic spellcaster proc on [line 267](https://github.com/zax0rz/darkpawns/blob/29994e5c8dc59e8de2cc22e65eb4fd5d172658ce/src/spec_assign.c#L267). Mallory has no number at all.

7969 belongs to zone 79, which the zone file names "BFM/Random Load/Redferne's". Every time that zone resets, each mob numbered 7900 to 7998 it loads gets dropped into a random room somewhere in the world ([src/db.c](https://github.com/zax0rz/darkpawns/blob/29994e5c8dc59e8de2cc22e65eb4fd5d172658ce/src/db.c#L2113-L2129)), skipping death traps, private rooms and city streets. The next two mobs in the file tell you what kind of neighborhood that is. Mob 7970 is [the Unicorn](https://darkpawns.org/mobs/7970/): "There is rumored to be only a single one in the entire world." Mob 7971 is [Eric Draven](https://darkpawns.org/mobs/7971/), The Crow, "dressed like a mime from hell." So Mickey was written to be a wandering encounter you could run into anywhere. The repository doesn't say whether a Mickey ever stood at 7969 before the wanderer did.

## So I gave them bodies

I built the original C server from the repository's `src/` on a scratch copy and gave it two stand-in mobs: Mickey Knox at 7969 and Mallory Knox at 7972. Two added lines in `spec_assign.c` attach their procs. Everything about the bodies is mine: the names on them, levels 20 and 18, their hit points, how hard they punch, and the HUNTER flag Mickey needs to actually walk to you when Mallory calls. (Without it, her call gives him a target he never moves toward.) Everything they say, and every hit, miss and death message, is the original game.

The player is a level 20 human warrior in steel plate. I named him Wayne, after Wayne Gale, the tabloid TV host Robert Downey Jr. plays in the movie, so Mickey's third line comes out "Shit Wayne, I'm a natural born killer." An implementor character carried everyone to the canyon with `transfer` before each scene; I cut that setup out. What's left is what the server sent Wayne with `color complete` on, prompt and all. Wayne's commands appear after the prompt, where a client would show them.

### Wayne meets Mickey

Mickey waited two rooms west of where Wayne started.

<figure class="session">
<pre class="mud-log"><span class="c32">262</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; <span class="in">west</span>
<span class="c36">Edge Of A Canyon</span>
   You stand on the very edge of a canyon.  The horizon on the other side is
dotted with trees and other signs of flourishing life, but the deadly drop to
the south makes it far out of your reach.
<span class="c36">[ Exits: north east south west ]</span>
<span class="c32"></span><span class="c33"></span>
<span class="c32">262</span>H <span class="c32">100</span>M <span class="c32">126</span>V &gt; <span class="in">west</span>
<span class="c36">Edge Of A Canyon</span>
   You stand on the very edge of a canyon.  The horizon on the other side is
dotted with trees and other signs of flourishing life, but the deadly drop to
the south makes it far out of your reach.
<span class="c36">[ Exits: north east south west ]</span>
<span class="c32"></span><span class="c33">Mickey Knox is here.
</span>
<span class="c32">262</span>H <span class="c32">100</span>M <span class="c32">126</span>V &gt; 
Mickey Knox says, 'Wayne, I'm gonna kill you.'

<span class="c32">262</span>H <span class="c32">100</span>M <span class="c32">126</span>V &gt; 
Mickey Knox asks, 'And you wanna know why?'

<span class="c32">262</span>H <span class="c32">100</span>M <span class="c32">126</span>V &gt; 
Mickey Knox says, 'Shit Wayne, I'm a natural born killer.'.
<span class="c31">Mickey Knox hits you very hard.
</span><span class="c33">You slash Mickey Knox hard.
</span><span class="c33">You barely slash Mickey Knox.
</span><span class="c31">Mickey Knox hits you hard.
</span><span class="c31">Mickey Knox hits you hard.
</span>Mickey Knox states, 'It's not nice to point.'

<span class="c32">233</span>H <span class="c32">100</span>M <span class="c32">126</span>V &gt; 
<span class="c33">You slash Mickey Knox.
</span><span class="c33">Mickey Knox is cooled by the breeze of your missed slash at him.
</span><span class="c31">Mickey Knox hits you very hard.
</span><span class="c31">Mickey Knox hits you very hard.
</span>
<span class="c32">209</span>H <span class="c32">100</span>M <span class="c32">126</span>V &gt; 
<span class="c33">You barely slash Mickey Knox.
</span><span class="c33">You slash Mickey Knox hard.
</span><span class="c31">Mickey Knox hits you hard.
</span><span class="c31">Mickey Knox hits you hard.
</span>Mickey Knox shouts, 'I'll always love you Mal, no matter what!'

<span class="c33">192</span>H <span class="c32">100</span>M <span class="c32">126</span>V &gt; 
<span class="c33">You slash Mickey Knox hard.
</span><span class="c31">Mickey Knox hits you hard.
</span><span class="c31">Mickey Knox hits you extremely hard.
</span>
<span class="c33">167</span>H <span class="c32">100</span>M <span class="c32">126</span>V &gt; 
<span class="c33">You slash Mickey Knox hard.
</span><span class="c33">You slash Mickey Knox.
</span><span class="c31">Mickey Knox hits you very hard.
</span><span class="c31">Mickey Knox hits you very hard.
</span>
<span class="c33">143</span>H <span class="c32">100</span>M <span class="c32">126</span>V &gt; 
<span class="c33">You slash Mickey Knox hard.
</span><span class="c33">You slash Mickey Knox hard.
</span><span class="c31">Mickey Knox hits you very hard.
</span><span class="c31">Mickey Knox hits you very hard.
</span>Mickey Knox shouts, 'When they come and ask you who did this, tell them it was Mickey and Mallory Knox!'

<span class="c33">118</span>H <span class="c32">100</span>M <span class="c32">126</span>V &gt; 
<span class="c33">You slash Mickey Knox hard.
</span><span class="c31">Mickey Knox misses a wild punch at you.
</span><span class="c31">Mickey Knox hits you hard.
</span>
<span class="c33">111</span>H <span class="c32">100</span>M <span class="c32">126</span>V &gt; 
<span class="c33">You barely slash Mickey Knox.
</span><span class="c33">You slash Mickey Knox hard.
</span><span class="c31">Mickey Knox hits you hard.
</span><span class="c31">Mickey Knox hits you hard.
</span>
<span class="c33">94</span>H <span class="c32">100</span>M <span class="c32">126</span>V &gt; 
<span class="c33">You scratch Mickey Knox as you slash him.
</span><span class="c31">Mickey Knox hits you very hard.
</span><span class="c31">Mickey Knox hits you very hard.
</span>Mickey Knox shouts, 'When they come and ask you who did this, tell them it was Mickey and Mallory Knox!'

<span class="c31">72</span>H <span class="c32">100</span>M <span class="c32">126</span>V &gt; 
<span class="c33">You slash Mickey Knox.
</span><span class="c33">You barely slash Mickey Knox.
</span><span class="c31">Mickey Knox hits you hard.
</span><span class="c31">You wish that your wounds would stop BLEEDING so much!</span>
<span class="c31">Mickey Knox hits you very hard.
</span><span class="c31">You wish that your wounds would stop BLEEDING so much!</span>

<span class="c31">49</span>H <span class="c32">100</span>M <span class="c32">126</span>V &gt; 
<span class="c33">Your desperate slash at Mickey Knox is easily avoided by him.
</span><span class="c33">You slash Mickey Knox hard.
</span><span class="c31">Mickey Knox hits you hard.
</span><span class="c31">You wish that your wounds would stop BLEEDING so much!</span>
<span class="c31">Mickey Knox hits you very hard.
</span><span class="c31">You wish that your wounds would stop BLEEDING so much!</span>

<span class="c31">30</span>H <span class="c32">100</span>M <span class="c32">126</span>V &gt; 
<span class="c33">You slash Mickey Knox hard.
</span><span class="c33">You slash Mickey Knox.
</span><span class="c31">Mickey Knox hits you hard.
</span><span class="c31">You wish that your wounds would stop BLEEDING so much!</span>
<span class="c31">Mickey Knox hits you very hard.
</span><span class="c31">You wish that your wounds would stop BLEEDING so much!</span>

<span class="c31">6</span>H <span class="c32">100</span>M <span class="c32">126</span>V &gt; 
<span class="c33">You slash Mickey Knox.
</span><span class="c33">You barely slash Mickey Knox.
</span><span class="c31">Mickey Knox hits you very hard.
</span>You are mortally wounded, and will die soon, if not aided.
<span class="c31">Mickey Knox hits you with a blow that breaks your spine!
</span>You are dead!  Sorry...

<span class="c31">-25</span>H <span class="c32">100</span>M <span class="c32">126</span>V &gt; </pre>
<figcaption>Verbatim server output recorded 2026-10-24 with stand-in mobs, setup commands cut</figcaption>
</figure>

Wayne started at 262 hit points and lasted thirteen rounds.

### Wayne picks a fight with Mallory

For the second scene Mallory stood alone on the canyon edge at 11369, and Mickey waited three rooms west.

<figure class="session">
<pre class="mud-log"><span class="c32">262</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; <span class="in">look</span>
<span class="c36">Edge Of A Canyon</span>
   You stand on the very edge of a canyon.  The horizon on the other side is
dotted with trees and other signs of flourishing life, but the deadly drop to
the south makes it far out of your reach.
<span class="c36">[ Exits: north east south west ]</span>
<span class="c32"></span><span class="c33">Mallory Knox is here.
</span>
<span class="c32">262</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; <span class="in">kill mallory</span>
<span class="c33">You slash Mallory Knox.
</span>
<span class="c32">262</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; 
<span class="c31">Mallory Knox barely hits you.
</span><span class="c31">Mallory Knox hits you hard.
</span>Mallory Knox says, 'That was the worst head I've ever got.'
<span class="c33">You barely slash Mallory Knox.
</span>
<span class="c32">250</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; 
<span class="c31">Mallory Knox hits you very hard.
</span><span class="c31">Mallory Knox hits you hard.
</span>Mallory Knox says, 'That was the worst head I've ever got.'
<span class="c33">You slash Mallory Knox.
</span><span class="c33">You slash Mallory Knox hard.
</span>
<span class="c32">230</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; 
<span class="c31">Mallory Knox hits you hard.
</span><span class="c31">You barely avoid Mallory Knox's fist as she takes a swing at you!
</span>Mallory Knox asks, 'How sexy am i now, fucker?'
Mallory Knox asks, 'How sexy am i NOW?'
<span class="c33">You try to slash Mallory Knox but miss by a mile.
</span><span class="c33">You slash Mallory Knox.
</span>
<span class="c32">223</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; 
Mickey Knox arrives from the west.
Mickey Knox starts following you.
<span class="c31">Mickey Knox hits you very hard.
</span><span class="c31">Mickey Knox hits you very hard.
</span><span class="c31">Mickey Knox hits you very hard.
</span>Mickey Knox shouts, 'I'll always love you Mal, no matter what!'
<span class="c31">Mallory Knox hits you hard.
</span><span class="c31">Mallory Knox barely hits you.
</span>Mallory Knox giggles madly.
<span class="c33">You slash Mallory Knox.
</span><span class="c33">You barely slash Mallory Knox.
</span>
<span class="c33">177</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; 
<span class="c31">Mickey Knox hits you very hard.
</span><span class="c31">Mickey Knox hits you very hard.
</span><span class="c31">Mallory Knox barely hits you.
</span><span class="c31">Mallory Knox hits you hard.
</span>Mallory Knox shouts, 'You stupid biiitch!'
<span class="c33">You slash Mallory Knox.
</span><span class="c33">You slash Mallory Knox.
</span>
<span class="c33">142</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; 
<span class="c31">Mickey Knox hits you very hard.
</span><span class="c31">Mickey Knox hits you very hard.
</span><span class="c31">Mallory Knox hits you hard.
</span><span class="c31">Mallory Knox hits you.
</span>Mallory Knox shouts, 'You stupid biiitch!'
<span class="c33">You slash Mallory Knox.
</span><span class="c33">You slash Mallory Knox.
</span>
<span class="c33">103</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; 
<span class="c31">Mickey Knox hits you hard.
</span><span class="c31">Mickey Knox hits you very hard.
</span>Mickey Knox shouts, 'I'll always love you Mal, no matter what!'
<span class="c31">Mallory Knox hits you.
</span><span class="c31">Mallory Knox hits you hard.
</span>Mallory Knox shouts, 'You stupid biiitch!'
<span class="c33">You slash Mallory Knox.
</span><span class="c33">You slash Mallory Knox hard.
</span>
<span class="c31">65</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; 
<span class="c31">Mickey Knox hits you hard.
</span><span class="c31">Mickey Knox hits you hard.
</span>Mickey Knox says, 'You're not centered.'
<span class="c31">Mallory Knox hits you hard.
</span><span class="c31">You wish that your wounds would stop BLEEDING so much!</span>
<span class="c31">Mallory Knox hits you hard.
</span><span class="c31">You wish that your wounds would stop BLEEDING so much!</span>
Mallory Knox shouts, 'You stupid biiitch!'
<span class="c33">You slash Mallory Knox.
</span><span class="c33">You barely slash Mallory Knox.
</span>
<span class="c31">50</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; 
<span class="c31">Mickey Knox hits you very hard.
</span><span class="c31">You wish that your wounds would stop BLEEDING so much!</span>
<span class="c31">Mickey Knox hits you very hard.
</span><span class="c31">You wish that your wounds would stop BLEEDING so much!</span>
<span class="c31">Mallory Knox hits you hard.
</span><span class="c31">You wish that your wounds would stop BLEEDING so much!</span>
<span class="c31">Mallory Knox hits you hard.
</span><span class="c31">You wish that your wounds would stop BLEEDING so much!</span>
Mallory Knox shouts, 'You stupid biiitch!'
<span class="c33">You slash Mallory Knox hard.
</span><span class="c33">You slash Mallory Knox.
</span>
<span class="c31">9</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; 
<span class="c31">Mickey Knox hits you very hard.
</span>You are incapacitated an will slowly die, if not aided.
<span class="c31">Mickey Knox hits you with such force that your neck snaps in two!
</span>You are dead!  Sorry...

<span class="c31">-24</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; </pre>
<figcaption>Verbatim server output recorded 2026-10-24 with stand-in mobs, setup commands cut</figcaption>
</figure>

Mickey walked in four rounds into the fight, started following Wayne and hit him three times before he said a word. Mallory shouted "You stupid biiitch!" five times, and the game spelled "an" for "and" on Wayne's way out.

### The take I didn't plan

One run ended before Mickey got past his first line. A bulette faded into view and took Wayne from 262 hit points to dead in two rounds.

<figure class="session">
<pre class="mud-log"><span class="c32">262</span>H <span class="c32">100</span>M <span class="c32">126</span>V &gt; <span class="in">west</span>
<span class="c36">Edge Of A Canyon</span>
   You stand on the very edge of a canyon.  The horizon on the other side is
dotted with trees and other signs of flourishing life, but the deadly drop to
the south makes it far out of your reach.
<span class="c36">[ Exits: north east south west ]</span>
<span class="c32"></span><span class="c33">Mickey Knox is here.
</span>
<span class="c32">262</span>H <span class="c32">100</span>M <span class="c32">126</span>V &gt; 
The bulette slowly fades into existence.
<span class="c31">The bulette DESTROYS you with its ungodly hit!!
</span>Mickey Knox says, 'Wayne, I'm gonna kill you.'
<span class="c33">You barely slash the bulette.
</span><span class="c33">You miss the bulette by an inch!
</span><span class="c31">The bulette DESTROYS you with its ungodly hit!!
</span><span class="c31">The bulette EVISCERATES you with its incredible hit!!
</span><span class="c31">The bulette DESTROYS you with its ungodly hit!!
</span>
<span class="c33">88</span>H <span class="c32">100</span>M <span class="c32">126</span>V &gt; 
<span class="c33">You scratch the bulette as you slash it.
</span><span class="c33">You try to slash the bulette but miss by a mile.
</span><span class="c31">The bulette DESTROYS you with its ungodly hit!!
</span><span class="c31">You wish that your wounds would stop BLEEDING so much!</span>
<span class="c31">The bulette DESTROYS you with its ungodly hit!!
</span>You are mortally wounded, and will die soon, if not aided.
<span class="c31">The bulette punches your head with a blow that crushes your skull!
</span>You are dead!  Sorry...

<span class="c31">-136</span>H <span class="c32">100</span>M <span class="c32">126</span>V &gt; </pre>
<figcaption>Verbatim server output recorded 2026-10-24 (the bulette is the game's own)</figcaption>
</figure>

## What's still missing

What's missing is the mob entries themselves. Whoever wrote these procs gave Mickey a speech and a room to hide in, and gave Mallory her lines, but their descriptions, their stats and the place they were meant to load aren't in any copy of the world files I have. If you played Dark Pawns and remember meeting Mickey or Mallory Knox, or you have an old backup with them in it, [tell me](https://darkpawns.org/contact/).
