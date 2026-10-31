---
title: "Players can't connect"
description: "Find whether a connection failure comes from the world, the listening address, a firewall, or the client, then choose the next check."
type: troubleshooting
audience: operators
verified: v0.1.8
---

Use this page when a player can't reach a running Dark Pawns server. Start with the address and port the server actually reports, then check the path from the player's network.

![The operations log with synthetic warning and error entries from the local preview.](/media/docs/admin-operations.png)

## The player gets a timeout

**Why it happens:** The host may be offline, the listener may be bound to loopback, or a firewall or router may block the port.

**Fix it:**

1. On the server machine, open `http://127.0.0.1:4350/health` in a browser, replacing `4350` with the configured HTTP port. A response confirms the local HTTP listener answers.
2. For telnet, run `telnet 127.0.0.1 7777` from the server machine, replacing `7777` with the configured telnet port. A game prompt confirms that local listener answers.
3. Check the listener's bind address in **Staff tools → Configuration**. A loopback-only bind accepts connections from the server machine, not remote players.
4. If public access is intended, first choose the listener and interface you mean to expose. Then follow [Portable hosting and adoption](https://github.com/zax0rz/darkpawns/blob/main/docs/portable-hosting.md) to configure the host firewall and router for that listener.

## The web client opens, but can't reach the game

**Why it happens:** The HTTP page is available while the game's web socket or its configured origin is unreachable.

**Fix it:**

1. Open `http://127.0.0.1:4350/health` on the server machine, replacing `4350` with the configured HTTP port.
2. If the health check answers, open the web client from that same address and try again.
3. Check browser origins in **Staff tools → Configuration** if the web client is hosted at a different address.
4. Check the server log for a listener or connection error.

## A telnet client can't connect

**Why it happens:** Telnet may be disabled, bound to loopback, or configured on a different port.

**Fix it:**

1. Read the telnet listener status in **Staff tools → Configuration**.
2. Give the player the configured telnet address and port.
3. If TLS telnet is enabled, give the player its TLS port and a client that supports it.

## The connection opens and then closes

**Why it happens:** The listener accepted the connection, but the game may have rejected the login or stopped during startup.

**Fix it:**

1. Check the server log and current process status.
2. Ask the player which client they used and what appeared before disconnect.
3. Compare the attempt time with the server log.

## Related

- [Install Dark Pawns](/docs/host/install/)
- [Your first login](/docs/host/first-login/)
- [Configuration settings](/docs/host/admin-configuration/)
