# Network settings in the admin console

Status: proposal only. This does not enable admin network changes in v0.1.2.
The portable launcher owns `instance.json`; the game owns its listeners. A
browser must not become a second writer of that configuration.

## Operator surface

Place **Network** under Staff tools. Require the same implementor authority
used for deployment-level settings; ordinary builders cannot expose a listener
or restart the world. Show configured values separately from observed listeners.
A failed bind must never appear as a running listener (R5f).

Offer plaintext telnet, TLS telnet and WHOD controls:

- Enable/disable, port, and access: **This computer** or **Public**.
- This computer means `127.0.0.1`; Public means all interfaces. Display the
  resulting address and port before saving. An advanced address field can show
  an explicitly configured address without silently broadening it.
- For plaintext Public access, explain that passwords cross the network in
  clear text. Prefer TLS for remote players.
- Validate port range and collisions across HTTP, telnet, TLS telnet and WHOD.
  Availability probes are advisory; actual startup is the final bind check.

Show HTTP address/port and reverse-proxy status as read-only in the first version.
Keep the current browser management route stable. Do not infer public reachability
from a local bind: firewall, proxy, NAT and DNS remain operator responsibilities.

## Configuration authority and applying changes

The launcher supplies an explicit managed-instance capability when it starts the
server. Without it, the page is read-only and directs a checkout operator to
server flags. Do not guess an instance from the process working directory.

The authenticated admin API submits a typed proposed change to a private launcher
control channel. Extend the existing authenticated local control protocol; do
not expose its token or arbitrary file/process operations to the browser. The
launcher reloads the current configuration and checks a revision before validating
and atomically saving only network fields. Preserve JWT, credentials, unknown
future fields and service registration. A stale revision returns a conflict with
instructions to reload, rather than overwriting another operator's changes.

Saving a proposal does not alter a live socket. Show **Restart required**, a diff
of old/new exposure, connected-player count, and an explicit **Apply and restart**
action. The launcher owns the restart transaction, including external-supervisor
coordination. Do not restart from a game-loop callback or use an untracked goroutine
in the server that is about to exit. Ordinary TLS certificate renewal remains the
existing in-place reload; changing listener addresses/ports requires restart.

## Avoiding lockout and failed changes

Keep HTTP unchanged, including its proxy contract. Identify the current session's
transport: if the only connected telnet route is being disabled or moved, require
the operator to confirm the browser/admin recovery path first. Do not close a
listener merely because a draft was saved. A restart disconnects players; say so
before confirmation, and let the operator cancel.

Before applying, retain the prior configuration and a transaction receipt, and
validate certificate identity and all requested listeners. After restart, require
both owned process readiness and successful binding of the requested listeners;
HTTP health alone is insufficient. On failure, restore the previous configuration
and restart it. If recovery fails, retain the receipt and both configurations for
`doctor` and local CLI recovery; do not silently keep trying broader bind addresses.
The admin page polls the unchanged management URL and reports restored settings
or a recovery instruction. Do not claim that local success proves LAN/WAN access.

## TLS and settings that stay outside the browser

Use named certificate profiles configured by the operator outside the web UI.
The page may display a profile's public hostname, expiry and fingerprint and let
the operator select it. Never return a private key, accept arbitrary filesystem
paths, or read browser-supplied paths as the server account. Existing reload keeps
the working certificate if renewal files are temporarily incomplete.

Keep these JSON/service-file only initially: HTTP binding/port, custom bind
addresses, certificate/key paths and profile creation, filesystem permissions,
service accounts, environment files, proxy trust, startup registration and update
policy. `instance.json` remains the advanced operator path while the world is
stopped. None of this changes telnet dialogue or game rules (R1–R4).

## Decisions for Zach

- Proposed first UI scope: telnet/TLS/WHOD enablement, ports and loopback/public
  access; keep HTTP read-only to preserve the management route. Alternative:
  allow HTTP changes with a separate timed confirmation/recovery design.
- Proposed authority: implementor-only, rather than all builders or moderators,
  because these controls expose sockets and restart the entire world.
- Proposed apply model: explicit validated restart with previous-configuration
  recovery. Alternative: live socket replacement, which adds transition and
  concurrency behavior that needs separate fidelity and transport proof.
- Proposed TLS model: select preconfigured profiles, rather than upload keys or
  expose filesystem paths. Profile provisioning stays operator-managed.
- Defaults are unchanged by this design note. Each new UI default, limit and
  recovery behavior must be reviewed before implementation.
