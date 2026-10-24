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

[unsourced: Zach's opening. Draft line: "I remember getting looted by Mickey and Mallory Knox. I couldn't tell you where, or exactly when (the 1990s, I think)." Add one detail from that death if any survives, or keep it this short.]

The Knoxes came from Oliver Stone's 1994 movie [Natural Born Killers](https://en.wikipedia.org/wiki/Natural_Born_Killers), Woody Harrelson and Juliette Lewis as two lovers on a killing spree that television turns into celebrities. A tag players wore in their titles came from the same movie. Type `help nbk` in Dark Pawns and this is the whole entry ([read it here](https://darkpawns.org/help/socials/nbk/)):

```
NBK

Natural Born Killer
```

Players put [NBK] in their titles to say they were open to player killing, or just to look like bad asses. One `who` list, quoted in a [dp-players.com thread from April 2004](https://darkpawns.org/archive/topic-t10/), has five of them across its two pages:

> [ 30 Wa ] Altie a Warrior [NBK] (invis) (quest) (OUTLAW)
> [ 26 Wa ] Sparahawk the village pot farmer. [NBK] (quest)
> [ 30 Cl ] Alisande the Mass Mudder [NBK] [!PAW] (invis)
> [ 22 Mu ] Nicolae the village tobacco farmer. [NBK]
> [ 30 Th ] Aidan the Mass Murderer [NBK] [!PAW] (invis) (OUTLAW)

The source code that survived kept Mickey and Mallory's speeches and lost everything else about them: their descriptions, their stats, and the line that puts either of them in the world. So I gave them new bodies on the original server and let them do what the code says they do.

## Mickey's speech

Each of them is a spec proc, a C function attached to a mob (a monster, in MUD terms) that the server runs every four seconds while it stands around and again on every round of combat. [Mickey's](https://github.com/zax0rz/darkpawns/blob/29994e5c8dc59e8de2cc22e65eb4fd5d172658ce/src/spec_procs.c#L1266-L1371) is a speech in three beats, one beat every four seconds, aimed at the first player he can see. ($N is the game filling in your name.)

> $N, I'm gonna kill you.
>
> And you wanna know why?
>
> Shit $N, I'm a natural born killer.

Then he hits you. The second and third beats are one sentence with four seconds in the middle. (The third line ends `.'.` in the source, a period, the closing quote and another period, and the game prints all of it.)

In a room flagged peaceful, where nobody can start a fight, the third beat changes: "I'll tell you later, $N" and "$n walks out the door and vanishes." He reappears in room 11369, an [Edge Of A Canyon](https://darkpawns.org/rooms/11369/) the code comments call "some room in SE desert." The exit south of it is named "A Fall to Your Death." It is a death trap. Walk south off any of the ten rooms along that canyon edge and you land in room 11376, "A Step Into The Canyon," read one paragraph ("Mercifully, you begin to lose consciousness before you reach the thrusting spire below you"), and go straight back to the login menu, with no corpse and no death message. Everything you carried lands on the floor of 11376, and the room kills any mortal who walks in after it. Immortals survive the room but can't save your gear either: [the game turns every death trap into a dump](https://github.com/zax0rz/darkpawns/blob/29994e5c8dc59e8de2cc22e65eb4fd5d172658ce/src/spec_assign.c#L638-L641), so the first command anyone types in there, `get all` included, destroys whatever is on the floor. That's why players were terrified of them. (Fleeing will never throw you over the edge. [The game checks.](https://github.com/zax0rz/darkpawns/blob/29994e5c8dc59e8de2cc22e65eb4fd5d172658ce/src/act.offensive.c#L392-L395))

To keep track of whom he's threatening, Mickey writes your player ID number into his own alignment, the number that says how good or evil a character is. A different player walks in, the numbers stop matching, and the speech starts over for them.

Once the fight starts he has a one in two chance each round of saying something:

> I'll always love you Mal, no matter what!
> Do you believe in fate?
> You're not centered.
> When they come and ask you who did this, tell them it was Mickey and Mallory Knox!
> It's not nice to point.

[Mallory's proc](https://github.com/zax0rz/darkpawns/blob/29994e5c8dc59e8de2cc22e65eb4fd5d172658ce/src/spec_procs.c#L1374-L1422) has no speech. She only talks once you're fighting her, and she talks nearly every round. The server rolls a ten-sided die, and on six through ten she shouts "You stupid biiitch!" The other rolls get "How do you like me now?", "That was the worst head I've ever got.", a two-line "How sexy am i now, fucker?" and "How sexy am i NOW?", "$n giggles madly.", and one roll of silence. Then she finds Mickey wherever he is in the world and sets him hunting you. The comment on that line says what happens if he's already hunting someone else: "if mickey's busy, catch him later :)".

<!-- Section 3, pending: What survived and what didn't. The procs are declared in spec_assign.c and never assigned; 7969 is the black-cloaked wanderer; zone 79 random load and the neighbors (Unicorn, Eric Draven); the attitude flag (AGGR24, attitude_loot) as the likely source of the looting, labeled as inference. -->

<!-- Section 4, pending: Running them again. Stand-in caveat. Takes: Mickey's speech, flee and hunt (wimpy), loot and brag (witness view), the peaceful room at The Dragon's Claw, Mallory plus Mickey hunting in, the bulette outtake, and the side quest: Wayne walking off the edge at 11369, the silent rim, and the imp watching his gear vanish in 11376. -->

<!-- Section 5, pending: Close. Anyone who met them, or has a backup with their mob entries. -->
