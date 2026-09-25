# 2026-09-24: reconnecting, and a session keyed by name

Field notes from DP-1325, which ports C's `perform_dupe_check`. Claims are
backed by the ledger row named at the end.

## 1. The takeover that tore itself down

Before DP-1325, logging in to a character that was already in the game went
through `Manager.Register`. That closed the old session, removed the
character from the world, and reloaded it from the store. C instead
reattaches the new descriptor to the live body. Reading that path turned up
a worse problem: the old connection's teardown still ran afterwards, and it
called `Unregister(name)`. The manager keys sessions by character name, so
that removed whatever session held the name, which by then was the new
login. A relog could be undone by the death of the connection it replaced.

C has no such hazard, because `perform_dupe_check` works on descriptor and
character pointers and marks the old descriptor `CON_CLOSE`. The port's name
key made "the session for this character" and "this session" the same
question, and the teardown asked the wrong one. The fix gives the transport
its own identity-checked `UnregisterSession`, and a superseded flag so the
old session never goes linkdead on the new session's body.

## 2. The echo that kept the player silent

C's CON_PASSWORD starts with `echo_on()`: `IAC WONT ECHO`, then CR LF
(`comm.c:954-967`). The port only turned echo back on at the next non-secret
entry prompt. The MOTD is one of those, so a normal login never showed the
gap, but a reconnect has no entry prompt after the password. Without the
fix, a telnet client would still believe the server was echoing, and the
reconnected player's typing would be invisible. The oracle showed it as a
missing line break after `Password:`. The symptom a player would have hit
is much larger than that diff.

## 3. The same prompt gap again

The observer's view differed from C by one blank line, between "has lost his
link." and "has reconnected.". That's the leftover of the prompt C sends
after async output, which the normalizer otherwise hides (RO-011). It's
DP-1307, the third time today it has blocked a proof: groinrip's dead
victim, this observer, and the Mudlet playtest where it was first reported.

(PF-021)
