---
title: "And You Wanna Know Why?"
date: 2026-10-24
description: "Mickey and Mallory Knox were in Dark Pawns. The source code that survived kept their speeches and lost their bodies, so I gave them new ones on the original server and let them kill a warrior named Wayne."
draft: true
textKind: "original"
source: "src/spec_procs.c, src/spec_assign.c, src/db.c and lib/world/ at commit 29994e5; lib/text/help/socials.hlp; archive record topic-t10; session logs captured 2026-10-24 from a scratch build of src/ with two stand-in mobs (docs/research/nbk-session/)"
voiceLayer: "mythic-admin"
ogImage: "/images/blog/og-mickey-and-mallory.png"
ogImageAlt: "And You Wanna Know Why?: Mickey Knox's two lines from the game on a black terminal panel, prompt numbers in green and his hit in red"
---

Back in the 90s, I got looted by one of the mobs named either Mickey or Mallory Knox. Their mob files have vanished, but their personalities are still in the code.

The Knoxes came from Oliver Stone's 1994 movie [Natural Born Killers](https://en.wikipedia.org/wiki/Natural_Born_Killers), Woody Harrelson and Juliette Lewis as two lovers on a killing spree that television turns into celebrities. I think I might have had it on VHS. A tag players wore in their titles came from the same movie. Type `help nbk` in Dark Pawns and this is the whole entry ([read it here](https://darkpawns.org/help/socials/nbk/)):

```
NBK

Natural Born Killer
```

Players put [NBK] in their titles to say they were open to player killing, or just to feel like bad asses. One `who` list, quoted in a [dp-players.com thread from April 2004](https://darkpawns.org/archive/topic-t10/), has five of them across its two pages:

> [ 30 Wa ] Altie a Warrior [NBK] (invis) (quest) (OUTLAW)\
> [ 26 Wa ] Sparahawk the village pot farmer. [NBK] (quest)\
> [ 30 Cl ] Alisande the Mass Mudder [NBK] [!PAW] (invis)\
> [ 22 Mu ] Nicolae the village tobacco farmer. [NBK]\
> [ 30 Th ] Aidan the Mass Murderer [NBK] [!PAW] (invis) (OUTLAW)

author's note: aidan is me

So I gave them new bodies on the original server and let them do what the code says they do.

## Mickey's speech

Each of them is a spec proc, a bit of C attached to a mob (a monster, in MUD terms). [Mickey's](https://github.com/zax0rz/darkpawns/blob/29994e5c8dc59e8de2cc22e65eb4fd5d172658ce/src/spec_procs.c#L1266-L1371) is a speech in three beats, four seconds apart, aimed at the first player he sees. ($N is your name.)

> $N, I'm gonna kill you.
>
> And you wanna know why?
>
> Shit $N, I'm a natural born killer.

And then he goes serial killer on you.

In a room flagged peaceful, where nobody can start a fight, the punchline changes to "I'll tell you later, $N" and he "walks out the door and vanishes." He turns up at his hideout, an [Edge Of A Canyon](https://darkpawns.org/rooms/11369/) the code calls "some room in SE desert," with a [death trap](https://darkpawns.org/rooms/11376/) one step south.

He remembers who he's threatening by writing your player ID into his own alignment, the number that says how good or evil a character is. If someone else walks in, the speech starts over for them.

Once the fight starts he has a one in two chance each round of saying something:

> I'll always love you Mal, no matter what!\
> Do you believe in fate?\
> You're not centered.\
> When they come and ask you who did this, tell them it was Mickey and Mallory Knox!\
> It's not nice to point.

[Mallory](https://github.com/zax0rz/darkpawns/blob/29994e5c8dc59e8de2cc22e65eb4fd5d172658ce/src/spec_procs.c#L1374-L1422) has no speech. She only talks once you're fighting her, and then almost every round. Half the time she shouts "You stupid biiitch!" Otherwise it's "How do you like me now?", "That was the worst head I've ever got.", "How sexy am i now, fucker?" (then "How sexy am i NOW?") or a giggle. Then she sets Mickey hunting you, wherever he is. The comment on that line: "if mickey's busy, catch him later :)".

## What the code kept

Spec procs get handed to mobs in `src/spec_assign.c`, and that file [declares both of theirs](https://github.com/zax0rz/darkpawns/blob/29994e5c8dc59e8de2cc22e65eb4fd5d172658ce/src/spec_assign.c#L102-L103) without ever handing them to anyone. The code [names 7969 as Mickey's number](https://github.com/zax0rz/darkpawns/blob/29994e5c8dc59e8de2cc22e65eb4fd5d172658ce/src/spec_procs.c#L1267) and never gives Mallory one. Today [mob 7969](https://darkpawns.org/mobs/7969/) is "a black-cloaked wanderer" running the generic spellcaster proc.

7969 sits in zone 79, which its zone file calls "BFM/Random Load/Redferne's". On every reset the game drops each of that zone's mobs into [a random room somewhere in the world](https://github.com/zax0rz/darkpawns/blob/29994e5c8dc59e8de2cc22e65eb4fd5d172658ce/src/db.c#L2113-L2129), so Mickey was written to turn up anywhere. His neighbors in the file are [the Unicorn](https://darkpawns.org/mobs/7970/) ("There is rumored to be only a single one in the entire world") and [Eric Draven](https://darkpawns.org/mobs/7971/), The Crow, "dressed like a mime from hell."

Neither proc takes anything off your corpse. Dark Pawns did have a flag for that, [written by Serapis in October 1996](https://github.com/zax0rz/darkpawns/blob/29994e5c8dc59e8de2cc22e65eb4fd5d172658ce/src/fight.c#L1100-L1108): a mob carrying it strips your corpse, puts on your gear and brags about it on gossip ("$N was tough, but had good eq..."). Their flags went with their mob files, so maybe Mickey and Mallory had it. Or maybe I'm misremembering, because it's been 30 years.

## Running them again

I built the original server from the repository's `src/` and gave it two stand-in mobs, Mickey Knox and Mallory Knox, with two added lines attaching their procs. The bodies are mine: their levels, hit points and damage, and the looting flag from the last section, which is a guess. Everything they say, every hit and miss, and everything after the kill comes from the original game.

My test character is Wayne, a level 20 warrior in steel plate, named after Wayne Gale, the TV host Robert Downey Jr. plays in the movie. An immortal carried everyone into place before each scene and I cut that part. The rest is exactly what the server sent, with Wayne's commands after the prompt where a client would show them.

### Mickey finds Wayne

Mickey waited two rooms west of Wayne on the canyon edge. Wayne's wimpy setting made him flee once he got low on hit points.

<figure class="session">
<pre class="mud-log"><span class="c32">262</span>H <span class="c32">100</span>M <span class="c32">126</span>V &gt; <span class="in">west</span>
<span class="c36">Edge Of A Canyon</span>
   You stand on the very edge of a canyon.  The horizon on the other side is
dotted with trees and other signs of flourishing life, but the deadly drop to
the south makes it far out of your reach.
<span class="c36">[ Exits: north east south west ]</span>
<span class="c33">Mickey Knox is here.</span>
<span></span>
<span class="c32">262</span>H <span class="c32">100</span>M <span class="c32">118</span>V &gt; 
Mickey Knox says, 'Wayne, I'm gonna kill you.'
<span></span>
<span class="c32">262</span>H <span class="c32">100</span>M <span class="c32">118</span>V &gt; 
Mickey Knox asks, 'And you wanna know why?'
<span></span>
<span class="c32">262</span>H <span class="c32">100</span>M <span class="c32">118</span>V &gt; 
Mickey Knox says, 'Shit Wayne, I'm a natural born killer.'.
<span class="c31">Mickey Knox hits you hard.</span>
<span class="c33">You slash Mickey Knox.</span>
<span class="c33">You miss Mickey Knox by an inch!</span>
<span class="c31">Mickey Knox hits you very hard.</span>
<span class="c31">Mickey Knox hits you very hard.</span>
<span class="c31">Mickey Knox hits you hard.</span>
Mickey Knox shouts, 'I'll always love you Mal, no matter what!'
<span></span>
<span class="c32">221</span>H <span class="c32">100</span>M <span class="c32">118</span>V &gt; 
<span class="c33">You slash Mickey Knox hard.</span>
<span class="c33">You slash Mickey Knox.</span>
<span class="c31">Mickey Knox hits you hard.</span>
<span class="c31">You duck under Mickey Knox's fist as he takes a swing at you.</span>
Mickey Knox asks, 'Do you believe in fate?'
<span></span>
<span class="c32">213</span>H <span class="c32">100</span>M <span class="c32">118</span>V &gt; 
<span class="c33">You slash Mickey Knox.</span>
<span class="c33">You slash Mickey Knox.</span>
<span class="c31">Mickey Knox hits you hard.</span>
<span class="c31">Mickey Knox hits you very hard.</span>
<span></span>
<span class="c33">192</span>H <span class="c32">100</span>M <span class="c32">118</span>V &gt; 
<span class="c33">You slash Mickey Knox hard.</span>
<span class="c31">Mickey Knox hits you very hard.</span>
<span class="c31">Mickey Knox hits you very hard.</span>
Mickey Knox shouts, 'When they come and ask you who did this, tell them it was Mickey and Mallory Knox!'
<span></span>
<span class="c33">94</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; 
<span class="c33">You barely slash Mickey Knox.</span>
<span class="c33">Your desperate slash at Mickey Knox is easily avoided by him.</span>
<span class="c31">Mickey Knox hits you hard.</span>
<span class="c31">Mickey Knox hits you very hard.</span>
Mickey Knox states, 'It's not nice to point.'
<span></span>
<span class="c31">74</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; 
<span class="c33">You miss Mickey Knox by an inch!</span>
<span class="c33">You slash Mickey Knox hard.</span>
<span class="c31">Mickey Knox hits you hard.</span>
<span class="c31">Mickey Knox hits you very hard.</span>
<span class="c31">You wish that your wounds would stop BLEEDING so much!</span>
<span></span>
<span class="c31">26</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; 
<span class="c33">You barely slash Mickey Knox.</span>
<span class="c33">You slash Mickey Knox hard.</span>
<span class="c31">Mickey Knox hits you very hard.</span>
<span class="c31">You wish that your wounds would stop BLEEDING so much!</span>
<span class="c31">Mickey Knox hits you very hard.</span>
<span class="c31">You wish that your wounds would stop BLEEDING so much!</span>
You wimp out, and attempt to flee!
<span class="c36">Edge Of A Desert</span>
   The desert ends here at a canyon to the south that drops thousands of feet
down into a dried river bed far below.  
<span class="c36">[ Exits: north east south west ]</span>
You flee head over heels.
<span></span>
<span class="c31">2</span>H <span class="c32">100</span>M <span class="c32">126</span>V &gt; 
Mickey Knox arrives from the south.
Mickey Knox starts following you.
<span class="c31">You barely avoid Mickey Knox's fist as he takes a swing at you!</span>
<span class="c31">You wish that your wounds would stop BLEEDING so much!</span>
You wimp out, and attempt to flee!
<span class="c36">Edge Of A Desert</span>
   The desert ends here at a canyon to the south that drops thousands of feet
down into a dried river bed far below.  
<span class="c36">[ Exits: north east south west ]</span>
You flee head over heels.
Mickey Knox arrives from the west.
<span class="c31">Mickey Knox tries to hit you but you easily avoid the blow.</span>
<span class="c31">You wish that your wounds would stop BLEEDING so much!</span>
You wimp out, and attempt to flee!
<span class="c36">Edge Of A Desert</span>
   The desert ends here at a canyon to the south that drops thousands of feet
down into a dried river bed far below.  
<span class="c36">[ Exits: north east south west ]</span>
You flee head over heels.
<span></span>
<span class="c31">2</span>H <span class="c32">100</span>M <span class="c32">110</span>V &gt; 
Mickey Knox arrives from the east.
<span class="c31">You recieve a punch in your kidneys which ends your worldly misery -- R.I.P.!</span>
You are dead!  Sorry...
<span class="c33">Mickey Knox gossips, 'Hey Wayne, was that suicide or did you try to fight back?'</span>
<span></span>
<span class="c31">-14</span>H <span class="c32">100</span>M <span class="c32">110</span>V &gt; </pre>
<figcaption>Verbatim server output recorded 2026-10-24, some rounds cut</figcaption>
</figure>

### The peaceful room

The same speech in [The Dragon's Claw](https://darkpawns.org/rooms/4874/), a weapon shop flagged peaceful:

<figure class="session">
<pre class="mud-log"><span class="c32">262</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; <span class="in">north</span>
<span class="c36">The Dragon's Claw</span>
   You stand in an arsenal.  Blades, bows, staves, and other deadly items
line the walls and rest on the floor as a large yellow-skinned man stares
cautiously at you from behind the counter against the northern wall.
<span class="c36">[ Exits: north south ]</span>
<span class="c33">Mickey Knox is here.</span>
<span class="c33">The weaponer stands here, chiseling a whetstone from granite.</span>
<span></span>
<span class="c32">262</span>H <span class="c32">100</span>M <span class="c32">132</span>V &gt; 
Mickey Knox says, 'Wayne, I'm gonna kill you.'
<span></span>
<span class="c32">262</span>H <span class="c32">100</span>M <span class="c32">132</span>V &gt; 
Mickey Knox asks, 'And you wanna know why?'
<span></span>
<span class="c32">262</span>H <span class="c32">100</span>M <span class="c32">132</span>V &gt; 
Mickey Knox says, 'I'll tell you later, Wayne'.
Mickey Knox walks out the door and vanishes.
<span></span>
<span class="c32">262</span>H <span class="c32">100</span>M <span class="c32">132</span>V &gt; </pre>
<figcaption>Verbatim server output recorded 2026-10-24</figcaption>
</figure>

### Mallory

Mallory alone at Mickey's hideout, Mickey three rooms west. Wayne swings first:

<figure class="session">
<pre class="mud-log"><span class="c32">262</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; <span class="in">kill mallory</span>
<span class="c33">You slash Mallory Knox hard.</span>
<span></span>
<span class="c32">262</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; 
<span class="c31">Mallory Knox hits you hard.</span>
<span class="c31">Mallory Knox hits you hard.</span>
Mallory Knox shouts, 'You stupid biiitch!'
<span class="c33">You barely slash Mallory Knox.</span>
<span class="c33">You slash Mallory Knox.</span>
<span></span>
<span class="c32">233</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; 
<span class="c31">Mickey Knox tells you, 'Run to your momma, pansy!'</span>
<span class="c31">Mallory Knox hits you hard.</span>
<span class="c31">Mallory Knox hits you hard.</span>
Mallory Knox says, 'That was the worst head I've ever got.'
<span class="c33">You barely slash Mallory Knox.</span>
<span class="c33">You slash Mallory Knox hard.</span>
<span></span>
<span class="c32">217</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; 
<span class="c31">Mallory Knox misses a wild punch at you.</span>
<span class="c31">Mallory Knox hits you hard.</span>
Mallory Knox asks, 'How do you like me now?'
<span class="c33">You slash Mallory Knox.</span>
<span class="c33">You slash Mallory Knox hard.</span>
<span></span>
<span class="c32">208</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; 
Mickey Knox arrives from the west.
Mickey Knox starts following you.
<span class="c31">Mickey Knox hits you hard.</span>
<span class="c31">Mickey Knox hits you very hard.</span>
<span class="c31">Mickey Knox hits you very hard.</span>
Mickey Knox asks, 'Do you believe in fate?'
<span class="c31">Mallory Knox tries to hit you but you easily avoid the blow.</span>
<span class="c31">Mallory Knox hits you hard.</span>
Mallory Knox asks, 'How do you like me now?'
<span class="c33">You scratch Mallory Knox as you slash her.</span>
<span></span>
<span class="c33">167</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; 
<span class="c31">Mickey Knox hits you hard.</span>
<span class="c31">Mickey Knox hits you extremely hard.</span>
<span class="c31">Mallory Knox hits you hard.</span>
<span class="c31">Mallory Knox barely hits you.</span>
Mallory Knox shouts, 'You stupid biiitch!'
<span class="c33">You slash Mallory Knox hard.</span>
<span class="c33">You slash Mallory Knox hard.</span>
<span></span>
<span class="c33">130</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; 
<span class="c31">Mickey Knox hits you very hard.</span>
<span class="c31">Mickey Knox hits you very hard.</span>
Mickey Knox states, 'It's not nice to point.'
<span class="c31">Mallory Knox hits you hard.</span>
<span class="c31">Mallory Knox hits you hard.</span>
Mallory Knox asks, 'How sexy am i now, fucker?'
Mallory Knox asks, 'How sexy am i NOW?'
<span class="c33">You slash Mallory Knox.</span>
<span class="c33">You slash Mallory Knox hard.</span>
<span></span>
<span class="c33">90</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; 
<span class="c31">Mickey Knox hits you hard.</span>
<span class="c31">Mickey Knox hits you hard.</span>
<span class="c31">Mallory Knox hits you hard.</span>
<span class="c31">Mallory Knox hits you hard.</span>
<span class="c31">You wish that your wounds would stop BLEEDING so much!</span>
Mallory Knox says, 'That was the worst head I've ever got.'
<span class="c33">You slash Mallory Knox hard.</span>
<span></span>
<span class="c31">58</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; 
<span class="c31">Mickey Knox hits you very hard.</span>
<span class="c31">You wish that your wounds would stop BLEEDING so much!</span>
<span class="c31">Mickey Knox hits you very hard.</span>
<span class="c31">You wish that your wounds would stop BLEEDING so much!</span>
Mickey Knox says, 'You're not centered.'
<span class="c31">Mallory Knox hits you hard.</span>
<span class="c31">You wish that your wounds would stop BLEEDING so much!</span>
<span class="c31">Mallory Knox hits you hard.</span>
<span class="c31">You wish that your wounds would stop BLEEDING so much!</span>
Mallory Knox giggles madly.
<span class="c33">You slash Mallory Knox hard.</span>
<span class="c33">You slash Mallory Knox.</span>
<span></span>
<span class="c31">17</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; 
<span class="c31">Mickey Knox hits you very hard.</span>
<span class="c31">You wish that your wounds would stop BLEEDING so much!</span>
<span class="c31">Mickey Knox hits you very hard.</span>
You are mortally wounded, and will die soon, if not aided.
<span class="c31">Mallory Knox hits you with such force that your neck snaps in two!</span>
You are dead!  Sorry...
<span class="c33">Mallory Knox gossips, 'Hey Wayne, was that suicide or did you try to fight back?'</span>
<span></span>
<span class="c31">-21</span>H <span class="c32">100</span>M <span class="c32">134</span>V &gt; </pre>
<figcaption>Verbatim server output recorded 2026-10-24, some rounds cut</figcaption>
</figure>

That tell comes from [the game's hunting code](https://github.com/zax0rz/darkpawns/blob/29994e5c8dc59e8de2cc22e65eb4fd5d172658ce/src/graph.c#L349-L401), which gives any mob tracking you a small chance of a taunt with every step, from "Run to your momma, pansy!" to putting your corpse up for auction before you're dead.

## Did you meet them?

If you played Dark Pawns and ran into Mickey or Mallory, tell me about it: a story about what they did to you, a log of an actual encounter, an old backup with their mob files in it, anything. [Send it here](https://darkpawns.org/contact/).
