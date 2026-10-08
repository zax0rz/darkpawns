---
title: "Welcome Back, Serapis"
date: 2026-10-08
description: "Room 8008 had a prayer that made three staff names immortal. We ported it to Go faithfully, then wrote a test proving it worked."
draft: true
textKind: "original"
source: "src/spec_procs.c and src/spec_assign.c in the C oracle; git history (ae08ddadd, #745, #1720); player rumor as recounted by Zach; Zach's own account"
voiceLayer: "mythic-admin"
ogImage: "/images/blog/og-welcome-back-serapis.png"
ogImageAlt: "Welcome Back, Serapis: an engraving of the god Serapis enthroned before a red sun, beside the Dark Pawns blog title"
---

<figure class="frontispiece" style="margin: 0 auto var(--space-lg); text-align: center;">
  <picture>
    <source media="(max-width: 640px)" srcset="/images/blog/welcome-back-serapis-tall.webp" width="900" height="1125" />
    <img src="/images/blog/welcome-back-serapis.webp" width="1500" height="643" alt="Engraving-style illustration of the god Serapis enthroned on a stone plinth, wearing a tall crown and holding a forked staff, with the three-headed dog Cerberus at his knee and a red sun behind him over a mountain landscape" style="display: block; width: 100%; height: auto; margin: 0 auto;" />
  </picture>
</figure>

Type `pray immortality` in the [Temple of the Cross](https://darkpawns.org/rooms/8008/) and nothing happens. Unless your name is Serapis.

## A prayer at the Temple of the Cross

Room 8008 sits in Kir Drax'in, just south of the temple altar. Its special procedure, `pray_for_items`, was a staff tool, and most players never saw it work. Gear lost the everyday way, to death and a looting mob, you got back by paying [the Mortician](https://darkpawns.org/mobs/8095/) to retrieve your corpse. When staff wanted to replace someone's gear, an immortal could load the items into the room that follows the temple in the world file and tag each one `item_for_` plus that player's name. The next time the player prayed in the temple, the altar found everything tagged for them, made a copy of each one in front of them ("$p slowly fades into existence."), and charged them what the items cost. Nothing in the world files carries that tag, so unless staff had set something up for you, a prayer there was just a prayer.

Pray for one word, though, and the altar never searches at all. Here is the start of that branch in [`src/spec_procs.c`](https://github.com/zax0rz/darkpawns/blob/631fa67ee819479caf6f3140d15db26840aa9c83/src/spec_procs.c#L2083-L2101), verbatim:

```c
  if (!strcmp(buf,"immortality"))
     {
        if (    (!strcmp(GET_NAME(ch),"Serapis")) ||
        (!strcmp(GET_NAME(ch),"Orodreth")) )
    {
                GET_LEVEL(ch) = 40;
                send_to_char("Welcome back ",ch);
                send_to_char(GET_NAME(ch),ch);
                send_to_char(".\n\r",ch);
                send_to_char("You feel the power pulse through your veins again!\n\r",ch);
        }
        if ((!strcmp(GET_NAME(ch),"Frontline")))
        {
                GET_LEVEL(ch) = 39;
```

Level 40 is the top of the Dark Pawns ladder, the implementor. A character named Serapis or Orodreth who prays for immortality becomes one. A character named Frontline gets 39. There is no password and no check that you are the person who first played the name: just the name and the room.

For Serapis, the whole exchange looked like this:

```text
pray immortality
Welcome back Serapis.
You feel the power pulse through your veins again!
```

After Frontline the real names stop. Before the source was released, somebody replaced the rest with placeholders like `"this is not here"` and left each level where it was ([the full list](https://github.com/zax0rz/darkpawns/blob/631fa67ee819479caf6f3140d15db26840aa9c83/src/spec_procs.c#L2083-L2126)):

| Name in the code | Level granted |
| --- | --- |
| Serapis | 40 |
| Orodreth | 40 |
| Frontline | 39 |
| "this is not here" | 36, then 35, then 31 |
| "neither is this" | 36 |
| "no entry here" | 31 |
| "neither here" | 31 |

Nobody can make a character with spaces in its name, so those lines can never fire. One placeholder sits in three of the checks, so a character named "this is not here" would be welcomed back three times and end at 31.

There's a second door in Orodreth's own zone, The Checker Board. [Cuchi](https://darkpawns.org/mobs/18306/), his cat, has her own special procedure. Pat her while your name is Orodreth and she purrs at you, and your level is set to implementor.

## Where it came from

Nobody who wrote it has said why. Among old players there are two rumors, and I can't settle either.

The first is that it was a recovery tool. The story goes that early on, someone got hold of another player's password and used the access to delete the player files, staff characters included, and staff gave themselves a way back in. The code fits. It says "Welcome back", and it lives in the same procedure staff already used to hand back lost equipment.

The second rumor is shorter: Frontline added it after releasing the source, to mess with whoever ran it next.

Either way, the backdoor got used. Former players have run [darkpawns.net](https://www.darkpawns.net/) since 2019 on their own modified copy of the original code, and the rumor there is that a brand new character named Serapis turned up one day with implementor powers. Nobody has confirmed whether it was the original Serapis or a stranger borrowing the name, and either one would only have needed a new character and a walk to room 8008. Their staff found the code afterward and pulled it out of their copy.

## We ported it faithfully

Dark Pawns is being rebuilt in Go from the original C source, with one rule above the rest: the new server has to send players the same text the old one did. To check that, we keep the original server running beside the new one. A test is a script both servers play through, and the test passes when every line they send back matches.

The prayer came across on April 24, 2026, in the [commit](https://github.com/zax0rz/darkpawns/commit/ae08ddadd) that ported `spec_procs.c` to Go. Nobody flagged it. The file is long, and the branch sits inside a procedure that was already a staff tool for handing out equipment.

On August 29, a pass that tightened `pray_for_items` against the C ([#745](https://github.com/zax0rz/darkpawns/pull/745)) added a test for exactly this branch. It creates a brand new character named Serapis on both servers, walks to room 8008, prays for immortality and types `score`. Both servers said "Welcome back Serapis." and both showed level 40. The test passed, and we marked the case proven.

Cuchi got the same care that day ([#736](https://github.com/zax0rz/darkpawns/pull/736)). Two of our test scripts later started leaning on her: make a character named Orodreth, pat the cat, and you have an immortal to set up a test room with.

## Found and closed

I found it through the cat. Porting the Lua scripts, I came across a copy of Cuchi's script and mentioned it to the darkpawns.net crew. They told me about the backdoor and the Serapis who showed up on their server. I went looking, and it was in the C and in our Go, line for line, with a passing test attached.

[#1720](https://github.com/zax0rz/darkpawns/pull/1720) closed both doors on September 30, and the fix went live that day. Serapis, Orodreth and Frontline are still names anyone can take. Praying for immortality under one of them now gets a different message and no levels. Orodreth can still pat Cuchi and she still purrs, but the level stays where it was. Everyone else who pats her gets ten gold coins from the gods, as in the original, so Orodreth is now the one person she gives nothing.

This is one of the few places the port departs from the original on purpose, and the change is written down as an approved difference with its own test. The two test scripts that used Cuchi broke the moment the door shut, and [had to find another way](https://github.com/zax0rz/darkpawns/pull/1726) to make an immortal.

Our test did its job. It proved the new server does what the old one did, and the old one handed level 40 to anybody named Serapis. A test that checks the copy against the original can't tell you the original was a mistake. Somebody has to read it.

## If you run the old code

If you run your own copy of the original source, search `spec_procs.c` for `Serapis`, `Orodreth` and `Frontline`. The prayer is in `pray_for_items` and the cat is in `cuchi`.

If you would rather play the original than patch it, [darkpawns.net](https://www.darkpawns.net/) has been running it since 2019, minus this prayer, and their [Discord](https://discord.gg/aMmBUaAcJ) is public.

And if you are Serapis: welcome back. [Make a character](https://darkpawns.org/play/). You'll have to level like the rest of us.
