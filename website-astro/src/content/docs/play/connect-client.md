---
title: "Connect a client"
description: "Run the three-character party Dark Pawns expects, with TinTin++, Mudlet, MUSHclient, the web client, or plain telnet across separate sessions."
type: guide
audience: players
verified: v0.1.7
---

Set up a three-character party: Dark Pawns is designed for multiplaying. A MUD client keeps each session open so you can switch among your characters. The game allows up to three characters at a time, and linkdead characters still count.

> “Once you get on your way, you'll probably want to check out information on MULTIplaying, which the mud is designed for.”
>
> Source: `help NEWBIE`

For the mid levels, `help MULTI` warns: “going solo might prove to be extremely difficult.” It sets the limit too: “Each player may have a Total of 3 characters at any given time.” If you're new, this guide also works while you create your first character; then open two more sessions for the party.

## Before you start

- The server address and raw telnet port from the operator.
- Up to three character names and passwords. You can create the characters after connecting.

## Open three TinTin++ sessions

TinTin++ calls each connection a session. Its manual uses `#session {name} {host} {port}` and lets you switch sessions by typing `#name`.

1. Start TinTin++.
2. At its prompt, enter `#session {first} {HOST} {PORT}`, replacing `HOST` and `PORT` with the operator's address and raw telnet port.
3. Enter your first character's name and password at the game prompts.
4. At the TinTin++ prompt, enter `#session {second} {HOST} {PORT}` and log in with your second character.
5. Repeat with `#session {third} {HOST} {PORT}` for your third character.
6. Type `#first`, `#second`, or `#third` to switch the active session.

The [TinTin++ session manual](https://tintin.mudhalla.net/manual/session.php) documents multiple simultaneous sessions and session switching.

## Open three Mudlet profiles

Mudlet saves each connection as a profile. Create one profile for each character, then connect them one at a time.

1. In Mudlet's connection window, choose **New**.
2. Name the profile for the character, then enter the server address and raw telnet port.
3. Connect and log in, or create that character.
4. Repeat for the other two characters, giving each one its own profile.
5. Switch profiles from Mudlet's connection window or use MultiView to show profiles side by side.

Mudlet's [Connection Window manual](https://wiki.mudlet.org/w/Manual%3AConnection_Window) covers profile setup; its [Profile Preferences manual](https://wiki.mudlet.org/w/Manual%3AProfile_Preferences) documents MultiView.

![Mudlet connected to a disposable local Dark Pawns world, showing the player dock, vitals, and Temple Altar map.](/media/docs/mudlet-dock.png)

## Open three MUSHclient worlds

MUSHclient saves each connection as a world. Use **Connection → Quick Connect** to make one for each character.

1. Open **Connection → Quick Connect**.
2. Enter a world name, the server address, and the raw telnet port.
3. Choose **OK**, then log in or create the character.
4. Repeat for the other two characters with a different world name for each.

MUSHclient opens each world in its own window. Its [getting started guide](https://www.mushclient.com/mushclient/doc/general/starting.html) documents Quick Connect, and the [Quick Connect dialog help](https://mushclient.com/mushclient/doc/dialog/IDR_MAINFRAME.html) lists the address and port fields.

## Use the web client

The web client opens one game session per browser tab. Open three tabs to keep three characters connected.

1. Open the [Dark Pawns web client](/play/).
2. Wait for the login prompt, then enter a character name and password, or type `guest` for an ephemeral visit.
3. Open two more tabs at the web client address and connect the other characters.

![A new Guest at the Temple Altar in the web client, with the connection status and vitals below the game text.](/media/docs/web-client-first-steps.png)

## Use plain telnet

Plain telnet works for a single session. Open three terminal windows and connect each one separately.

:::caution
Telnet sends passwords without encryption. Ask the operator for a TLS port and a compatible client if one is configured.
:::

1. Open a terminal with the `telnet` command available.
2. Run `telnet HOST PORT`, replacing `HOST` and `PORT` with the operator's address and raw telnet port.
3. Enter one character's name and password.
4. Repeat in two more terminal windows for the other characters.

Some systems don't include a telnet program. Use TinTin++, Mudlet, MUSHclient, or the web client if the command isn't available.

## Did it work?

You can switch among three connected characters, and each session shows its own game prompt. Linkdead characters count toward the three-character limit until they leave the game.

## If it didn't work

- **The connection times out.** Confirm the address and port with the operator, then see [Players can't connect](/docs/host/players-cannot-connect/).
- **You see a connection but no login prompt.** Disconnect and reconnect. If it repeats, tell the operator which client you used and when it happened.
- **The browser session replaces another.** Open a fresh tab for each character and connect them separately.

## Related

- [Your first hour](/docs/play/first-hour/)
- [Players can't connect](/docs/host/players-cannot-connect/)
