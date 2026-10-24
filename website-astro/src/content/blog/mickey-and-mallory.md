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

Spec procs get handed to mobs in `src/spec_assign.c`, and that file [declares both of theirs](https://github.com/zax0rz/darkpawns/blob/29994e5c8dc59e8de2cc22e65eb4fd5d172658ce/src/spec_assign.c#L102-L103) without ever handing them to anyone. The code names 7969 as Mickey's number. Today [mob 7969](https://darkpawns.org/mobs/7969/) is "a black-cloaked wanderer" running the generic spellcaster proc, and Mallory never had a number at all.

7969 sits in zone 79, which its zone file calls "BFM/Random Load/Redferne's". On every reset the game drops each of that zone's mobs into [a random room somewhere in the world](https://github.com/zax0rz/darkpawns/blob/29994e5c8dc59e8de2cc22e65eb4fd5d172658ce/src/db.c#L2113-L2129), so Mickey was written to turn up anywhere. His neighbors in the file are [the Unicorn](https://darkpawns.org/mobs/7970/) ("There is rumored to be only a single one in the entire world") and [Eric Draven](https://darkpawns.org/mobs/7971/), The Crow, "dressed like a mime from hell."

Neither proc takes anything off your corpse. Dark Pawns did have a flag for that, [written by Serapis in October 1996](https://github.com/zax0rz/darkpawns/blob/29994e5c8dc59e8de2cc22e65eb4fd5d172658ce/src/fight.c#L1100-L1108): a mob carrying it strips your corpse, puts on your gear and brags about it on gossip ("$N was tough, but had good eq..."). Their flags went with their mob files, so maybe Mickey and Mallory had it. Or maybe I'm misremembering, because it's been 30 years.

<!-- Section 4, pending: Running them again. Stand-in caveat. Takes: Mickey's speech, flee and hunt (wimpy), loot and brag (witness view), the peaceful room at The Dragon's Claw, Mallory plus Mickey hunting in, the bulette outtake, and the side quest: Wayne walking off the edge at 11369, the silent rim, and the imp watching his gear vanish in 11376. -->

<!-- Section 5, pending: Close. Anyone who met them, or has a backup with their mob entries. -->
