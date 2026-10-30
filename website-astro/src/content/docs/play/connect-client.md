---
title: "Connect a client"
description: "Dark Pawns is played with three characters at once. Set up TinTin++, Mudlet, MUSHclient, the web client or telnet to run your whole party."
type: guide
audience: players
verified: v0.1.7
---

Dark Pawns is played as a party. You can run up to three characters at once, and you'll want all three: by the mid levels, a character on their own has a rough time. A MUD client keeps every character connected and lets you hop between them.

> Each player may have a Total of 3 characters at any given time.
>
> `help multi`

## Before you start

- **The address.** For darkpawns.org it's `darkpawns.org`, port **7778** (encrypted) or **7777** (classic telnet). Playing on someone else's server? Ask its operator.
- **Three character names.** You can create the characters as you connect.

:::note
Characters that lose their connection stay in the game for a while, and they still count toward your three. Log them out if you aren't using them.
:::

## TinTin++

TinTin++ calls each connection a session, and you switch between them by name. Type the commands below exactly as shown, curly braces included: TinTin++ uses them to group each argument.

1. Start TinTin++.
2. Open your first character's session: `#ssl {first} {darkpawns.org} {7778}`.
3. Log in, or create the character.
4. Open the second and third the same way: `#ssl {second} {darkpawns.org} {7778}`, then `#ssl {third} {darkpawns.org} {7778}`.
5. Type `#first`, `#second` or `#third` to switch.

No encryption support in your build? Use `#session` instead of `#ssl`, with port 7777. The [TinTin++ session manual](https://tintin.mudhalla.net/manual/session.php) covers the rest.

## Mudlet

Mudlet saves each connection as a profile, so make one per character.

1. In Mudlet's connection window, choose **New**.
2. Name the profile after the character. Enter `darkpawns.org` and port `7778`, and tick **Secure**.
3. Connect, then log in or create the character.
4. Repeat for the other two.
5. Switch between profiles, or turn on MultiView to see them side by side.

Mudlet's [connection window manual](https://wiki.mudlet.org/w/Manual%3AConnection_Window) covers profiles; its [profile preferences manual](https://wiki.mudlet.org/w/Manual%3AProfile_Preferences) covers MultiView.

![Mudlet connected to Dark Pawns, with the player dock, vitals and the Temple Altar map.](/media/docs/mudlet-dock.png)

## MUSHclient

MUSHclient saves each connection as a world, one per character.

1. Open **Connection → Quick Connect**.
2. Enter a world name, `darkpawns.org` and the classic port, `7777`. (Use 7778 instead if your MUSHclient setup supports encrypted connections.)
3. Choose **OK**, then log in or create the character.
4. Repeat for the other two, with a different world name each time.

Each world opens in its own window. The [getting started guide](https://www.mushclient.com/mushclient/doc/general/starting.html) walks through Quick Connect.

## The web client

No install needed. Each browser tab is one character.

1. Open the [web client](/play/).
2. At the login prompt, enter a character name and password, or type `guest` to look around without making one.
3. Open two more tabs at the same address for the rest of the party.

![A new character at the Temple Altar in the web client, with vitals below the game text.](/media/docs/web-client-first-steps.png)

## Plain telnet

It works, it's everywhere, and it does one character per window.

:::caution
Telnet sends your password unencrypted. Use port 7778 with a client that supports encryption when you can.
:::

1. Open a terminal.
2. Run `telnet darkpawns.org 7777`.
3. Log in, or create the character.
4. Do the same in two more windows.

No `telnet` command on your system? Any of the clients above will do.

## Did it work?

Each character has its own window, tab or session, and each one shows its own prompt. You can switch between them without anyone logging out.

## If it didn't work

- **The connection times out.** Check the address and port, then see [Players can't connect](/docs/host/players-cannot-connect/).
- **You connect but never get a login prompt.** Disconnect and try again. If it keeps happening, tell us in the [Discord](https://discord.gg/supVzU9yN) which client you used and when.
- **A second tab kicks out the first.** Open a fresh tab for each character rather than reusing one.

## Related

- [Your first hour](/docs/play/first-hour/)
- [Players can't connect](/docs/host/players-cannot-connect/)
