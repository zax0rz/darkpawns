---
title: Connect a client
description: Connect to Dark Pawns in the browser, with Mudlet, or from a telnet client using the game's configured connection ports and settings.
type: guide
audience: players
verified: v0.1.7
---

Use the web client for a connection with no client setup. It connects when the page opens. At the game prompt, enter an existing character name, create a character, or type `guest` for an ephemeral visit. Mudlet and other telnet clients can connect directly to the game server when you have its address and port.

## Before you start

- A Dark Pawns character, or time to create one after connecting.
- For Mudlet or telnet, the server address and port from the operator.

## Use the web client

1. Open the [Dark Pawns web client](/play/).
2. Wait for the login prompt in the terminal.
3. Enter an existing character name, start character creation, or type `guest` for an ephemeral visit.

The **Reconnect** button appears after the connection drops. Choose it to reconnect.

TODO(media): show the web client connected at character entry, with the transcript and connection control visible.

## Connect with Mudlet

1. Create a new connection in Mudlet.
2. Enter the server address supplied by the operator.
3. Enter the raw telnet port supplied by the operator.
4. Connect, then enter your character name and password.

TODO(media): show the Mudlet connection profile fields for a Dark Pawns server.

## Connect with telnet

1. Open a terminal.
2. Run `telnet HOST PORT`, replacing `HOST` and `PORT` with the address and port supplied by the operator.
3. Enter your character name and password when the game asks.

## Check it worked

The game shows its login prompt. Entering a valid character name takes you to the password prompt or character entry flow.

## If it went wrong

- **The connection times out.** Check the server address and port with the operator, then see [Players can't connect](/docs/host/players-cannot-connect/).
- **The connection opens but shows no prompt.** Disconnect and try again. If it repeats, give the operator the client name and time of the attempt.
- **The browser client cannot connect.** Reload the web client and check the server status with the operator.

## Related

- [Your first hour](/docs/play/first-hour/)
- [Players can't connect](/docs/host/players-cannot-connect/)
