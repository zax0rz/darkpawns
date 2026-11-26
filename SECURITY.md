# Security policy

## Supported versions

Security fixes go into the latest release only. Before 1.0, that means the
newest `v0.1.x` tag. If you run an older version, update with
`darkpawns update` before reporting.

## Reporting a vulnerability

Please report privately through GitHub:
[open a private security advisory](https://github.com/zax0rz/darkpawns/security/advisories/new).
Don't open a public issue, post in Discord, or say it in game.

Include what you can:

- the version (`darkpawns --version`) or commit;
- what an attacker can do, and what access they need to start with;
- steps to reproduce, or a proof of concept;
- whether you've tested it against darkpawns.org or only your own copy.

We'll reply on the advisory, work out a fix with you there, and ship it in a
new release. You'll be credited in the release notes unless you'd rather not
be.

## Scope

In scope:

- the game server and its listeners (telnet, TLS telnet, WebSocket, the web
  client, the admin console);
- accounts, passwords and sessions;
- the installer, the `darkpawns` launcher, updates, and release signing;
- the Lua script sandbox;
- the production world at darkpawns.org.

Out of scope:

- **Gameplay that matches the original.** Dark Pawns reproduces the 1990s C
  game byte for byte, quirks included. If a trick or exploit works the same
  way in the original, it's part of the game: open an ordinary issue if you
  think it's a porting mistake.
- What immortals and admins can do by design.
- Problems that need someone who already controls the host, the instance
  directory, or an admin account.
- Missing hardening with no attack behind it (headers, banner text, rate
  limits on their own).

## Testing darkpawns.org

Real people play there. Please don't:

- run denial-of-service or load tests, or automated scanners;
- log in as, read, or change anyone else's character or account;
- spam other players or channels.

A self-hosted copy (see the [install guide](https://darkpawns.org/docs/host/install/))
behaves the same and is the better place to test. Good-faith research that
follows this page won't be treated as an attack.
