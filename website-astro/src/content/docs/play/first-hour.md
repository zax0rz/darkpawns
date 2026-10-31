---
title: "Your first hour"
description: "Your first hour in Dark Pawns: gear up, find your guild, walk to the Guard Barracks in Kir Drax'in, win a first fight, and keep your stuff when you quit."
type: guide
audience: players
verified: v0.1.8
---

You've made a character and you're standing in a temple with a pack full of starter gear and zero skills practiced. Everything outside the city walls wants you dead, and half the things inside the walls aren't fond of you either. Here's how to survive your first hour.

> You will start out in the Temple of your home town. The first thing you should do is find your GUILD and PRACTICE your starting skills.
>
> `help NEWBIE`

## Before you start

- A new character, standing at the temple altar of your hometown.
- Three in-game commands worth reading before you wander off: `help newbie`, `help map`, and `help faq`.

![A new character at the Temple Altar in the web client, with vitals below the game text.](/media/docs/web-client-first-steps.png)

## Gear up

1. Type `wear all`.

   You equip everything in your inventory that fits a body slot. Some things don't belong on your head: you can't wear bread, so the game leaves it in your pack for later.

:::tip[Turn on the survival switches]
Type `auto` to see your automatic exploration and combat toggles. Turning on `auto exits`, `auto loot`, and `auto gold` prints the room's exits under every room description (saving you from typing `exits`) and loots coins and gear from corpses automatically after every kill.
:::

## Find your guild and practice

1. Type `score` to check your class, hometown, and how many practice sessions you have.
2. Type `help guild` to see where the guilds sit in Kir Drax'in.
3. Walk to your class's guild and type `practice` to see what your guildmaster teaches, then `practice <skill>` to train it.

In Kir Drax'in, warriors train on the Street of Swords, mages on the Street of Wizardry, clerics inside the Temple of the Cross, thieves on the southern wall road, and psionicists and ninjas on the Avenue of the Gods. Guild guards only let in their own class.

Lost already? Type `recall` to snap back to your hometown temple, or ask for directions with `newbie <your question>`.

## Walk to the Guard Barracks

`help NEWBIE` calls the Guard Barracks in Kir Drax'in "one of the best places to start". If Kir Drax'in is your hometown, follow this walk from the Temple Altar:

This map traces the walk from the Kir Drax'in Temple Altar to the basic training stairs.

```mermaid
flowchart LR
  A["Temple Altar"] -->|South| B["Temple of the Cross"] -->|South| C["Temple Square"] -->|West| D["Avenue of the Gods"] -->|North| E["Street of Swords"] -->|West| F["Guard Barracks"] -->|South door| G["New Recruit Training Area"]
```

1. Go south into the Temple of the Cross, then south again to Temple Square.
2. Go west to the Avenue of the Gods, then north onto the Street of Swords.
3. Go west into the Guard Barracks.
4. Open the south door (`open door south`) and go south into the New Recruit Training Area.

Started in Kir-Oshi or Alaozar instead? Your city has its own guilds and training grounds. `help map` and the `newbie` channel will point you there.

## Win your first fight

1. From the New Recruit Training Area, go east, then north to the stairway.
2. Go down the stairs into **Kir Drax'in Guard Basic Training - Level One**.
3. Type `look` and spot a grey guard trainee.
4. Type `consider trainee` to size up its strength, health, and level before swinging.
5. If you like your odds, type `kill trainee` and watch your hit points.

If things get uncomfortable, type `flee` to bolt for a random exit. It beats dying, though your pride and experience bar take a hit:

> Fleeing from battle will also cost you some experience points, but hey, thats what you get for wimping out.
>
> `help FLEE`

Stay in the training centre until you have a few levels under your belt. As `help FAQ` warns: *"Killing mobs in the city when you are a newbie is NEVER a good idea, your best bet is to stay down in the centre for awhile."* When you level up, walk back to your guild and `practice` again.

:::note[When you die (and you will)]
When you die, you wake up naked in the temple while your equipment and gold stay on your corpse where you fell (`help DEATH`). Walk back and type `get all corpse`, or pay a Mortician to `retrieve` it. Every death also carries a chance to lose 1 point of Constitution; drop below 1 Constitution and you are dead forever.
:::

## Keep your stuff when you quit

Dark Pawns has no rent, which is the good news. The bad news is that your gear only saves if you `quit` inside your hometown's safe area: the Temple of the Unholy (Temple and Altar) in Kir Drax'in, the public temple in Kir-Oshi, or the ziggurat in Alaozar (`help QUIT`).

:::danger[Never type reallyquit if you like your gear]
If you type `quit` outside a safe room, the game refuses so you don't lose your items by accident. As `help QUIT` puts it: *"If you have to type REALLYQUIT to leave the game, your items did not save, and were dropped on the ground in the room you quit out in."* Type `recall` first, then `quit`.
:::

## Get a party together

Soloing works at level 1, and stops working fast:

> A level 1 player fighting a level 1 mob is about even. A level 15 player fighting a level 15 mob doesn't probably stand a chance.
>
> `help MULTI`

Dark Pawns lets you run up to three characters at once so you can field your own tank, healer, and damage dealer. [Connect a client](/docs/play/connect-client/) shows how to set up all three.

## Did it work?

`score` shows experience and gold you didn't have an hour ago, `practice` lists skills you've started training, and you can walk from the Temple Altar to the Guard Barracks without checking the map.

## If it didn't work

- **You can't find the barracks.** Type `recall` to return to the temple, check `help map`, and ask on the `newbie` channel.
- **You're broke and starving.** `help FAQ` has two words for you: `eat` and `carve`. If you're just hungry, there's a bakery one east and one north of Market Square.
- **You died in the street.** Walk back to your corpse, type `get all corpse`, then head down to the Guard Barracks where the trainees won't one-shot you.

## Related

- [Connect a client](/docs/play/connect-client/)
- [Color](/docs/play/color/)
- [Commands](/docs/reference/commands/)
- [Glossary](/docs/reference/glossary/)
