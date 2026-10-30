---
title: Your first login
description: Finish the first-run setup, connect to your new Dark Pawns server, create a character, and open the admin console with the first character.
type: guide
audience: operators
verified: v0.1.7
---

This guide takes a fresh portable instance from setup to a running world and the first operator sign-in. The first successfully created character becomes the Implementor (level 40), with the highest staff access. Setup does not create a separate admin account. Create that character before allowing other people to connect.

## Before you start

- Install Dark Pawns. See [Install Dark Pawns](/docs/host/install/).
- A terminal on the machine that will run the instance.

## Create an instance

1. Run `darkpawns setup`.
2. Choose the instance folder and local web and telnet ports.
3. Review the settings and let setup create and start the instance.

Setup prints the local web client address when the world is ready. The default is `http://127.0.0.1:4350/`; the default telnet listener is `127.0.0.1:7777`. Both bind to loopback, so another machine cannot connect directly.

## Connect to the game

1. Open the web client at the local address printed by setup.
2. Create the first character. This character becomes the Implementor (level 40).

![Portable bundle setup starts an isolated local world.](/media/docs/install-and-setup.gif)

Setup prints the local web address and the first-character warning.

## Open the admin console

1. Open `/admin/` on the server's HTTP address.
2. Sign in with the Implementor character's name and password.

The admin console checks the character's current access level on each authenticated request. Some tools require level 40; see [The admin console](/docs/host/admin-console/) for page access gates.

## Check it worked

The first character has level 40, and `/admin/` opens the console for that character.

## If it went wrong

- **Setup cannot start the instance.** Read the terminal error and run `darkpawns doctor` from the installed launcher directory.
- **The web client cannot reach the game.** Check the server startup output and see [Players can't connect](/docs/host/players-cannot-connect/).
- **The console rejects the sign-in.** Confirm you are using the first character you created in the web client and its game password; the first character receives Implementor access.

## Related

- [Install Dark Pawns](/docs/host/install/)
- [The admin console](/docs/host/admin-console/)
- [Update your server](/docs/host/update-server/)
