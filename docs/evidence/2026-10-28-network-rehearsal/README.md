# Network settings rehearsal: change, apply, verify, roll back

2026-10-28, local portable instance only (the production-shape bundle
v0.1.7-net.1 with a snapshot publisher key; first character Netowner,
implementor; telnet loopback on 7777). Everything driven from the browser
through the Configuration page's Network card.

## Scenario 1: telnet 7777 to 42069

`network-card.png` (the card: HTTP read-only with its how-to-change
explanation, observed listener beside each toggle), `network-draft-banner.png`
(save-as-draft banner: the port field shows the saved draft while observed
still names the running listener, the exposure diff table, who-is-online),
`network-confirm.png` (the confirm dialog: players by name, the diff, the
recovery-path sentence), `network-committed.png` (Committed; the page survived
the restart and reconnected by itself).

Durable proof: `ss -tln` shows `127.0.0.1:42069` listening, the card reports
configured 42069 with observed `127.0.0.1:42069`, and the receipt
`.config-apply.json` reads `state: committed, was_running: true`.

## Scenario 2: back to 7777

Same flow, committed; `ss -tln` shows `127.0.0.1:7777`.

## Scenario 3: deliberate bind failure rolls back

`python3 -m http.server 42069` holds the port; the card proposes 42069 and
applies. The launcher stops the world, starts it on the proposed
configuration, the server cannot bind (the game process fails readiness
before HTTP health), and the transaction restores the previous configuration
and restarts it. `network-rolled-back.png` shows the page's report; the
receipt (`apply-receipt-rolled-back.json`) records the failure; `ss -tln`
afterwards shows `127.0.0.1:7777` answering and `/health` returns 200. The
console never lied about exposure: configured and observed both read 7777
again.

## Notes from the rehearsal

Two real bugs were caught and fixed before this evidence was captured: the
save response could serialize an empty diff or warning list as JSON null
(crashing the card), and the apply transaction resolved release assets
against the working directory instead of the launcher's bundle when the
control server spawned it (now resolved from the running launcher binary,
matching `darkpawns restart`).
