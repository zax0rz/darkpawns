# Admin notifications — v1 and follow-ups

Accepted scope: Zach + Sol, 2026-10-19. This is a modern operator surface under
R4. Notification observation must not change telnet bytes, game state, RNG draws,
or persistence ordering. Existing mudlog text and its in-game recipients stay intact.

## V1

The staff Notifications page configures one Pushover destination and one Discord
channel webhook independently. Each has an enable switch, the same five event
choices, a test button, credential readiness, and last provider success/failure.
All destinations and selections default off. Tests may be sent while disabled.
Tests report **queued**, not delivered; successful delivery means the provider
accepted the request, not that a phone displayed it.

| Event ID | Observation boundary | Message |
|---|---|---|
| `player.created` | Character persistence succeeds, once per creation | Avery created a character. |
| `player.login` | Body admission to the world succeeds | Avery entered the game. |
| `player.logout` | Body leaves the world through extraction or cleanup | Avery left the game. |
| `player.death` | Guarded die path, or raw-kill extraction callback | Avery died. |
| `player.level` | Existing structured level-gained event | Avery reached level 20. |

A login is world entry, not authentication at the menu. Logout is world departure,
not necessarily a dropped TCP connection: linkdead bodies may remain in the game.
Returning to the menu after death is a departure. Guests and invisible staff do
not produce activity or milestone announcements. Creation does not announce a
host, email address, password, or location. Fixed templates carry only a sanitized
character name and optionally a level; no mudlog free text is forwarded. Discord
mentions are disabled in the request payload. Operators choose channel access in
Discord; neither provider implies public or private visibility.

Secrets live only in fixed server environment variables. Admin can read readiness
but cannot read or overwrite secrets. This preserves the existing mudlog-push
separation between operator secret provisioning and portal subscriptions. Provider
hosts and URL paths are restricted, HTTPS is mandatory, redirects are refused,
and upstream response bodies/credential-bearing URLs never enter API errors.

Settings use an atomic, mode-0600 JSON file, default `lib/data/notifications.json`
in a normal deployment. Revision checks reject concurrent stale edits. Saving
settings does disk I/O outside the publisher lock. One bounded worker queue per
provider prevents an unavailable destination from delaying another. Delivery has a
five-second deadline, a burst of five and a sustained limit of 30 queued messages
per minute per destination. Over-limit/full-queue messages are counted and dropped.
There is no persistent outbox, replay, or automatic retry in v1. Settings changes
invalidate queued jobs from older revisions; already in-flight requests can finish.
Counters and delivery times reset on server restart.

The previous `DISCORD_WEBHOOK_URL` sender is replaced, so there is one delivery
path. Operators migrating from it must provision `DP_NOTIFY_DISCORD_WEBHOOK` and
explicitly enable desired events in admin. The separate environment-only mudlog
ntfy observer remains independent; it forwards raw staff logs when explicitly
configured and is not a new public announcement provider.

### External monitoring

An optional hosted Healthchecks heartbeat runs every minute only if the completed
game-loop callback has advanced within 30 seconds. No progress means no success
ping. It starts off and requires a valid `DP_NOTIFY_HEARTBEAT_URL`. Healthchecks
sends outage/recovery alerts directly to the operator's Pushover independently of
the game. Configure a one-minute period and about two minutes of grace. Planned
maintenance is paused in Healthchecks itself. The portal exposes its own last
heartbeat request result, not Healthchecks' external incident state.

Provisioning an external account or altering production alerting is an operator
step; implementing the heartbeat does not enroll an instance automatically.

## V2 backlog — post-launch, or selected pre-launch additions

These are captured ideas, not promises or automatic additions to v1.

- **More providers:** ntfy (hosted or self-hosted), Telegram, WhatsApp, X. Keep
  provider delivery separate from event observation. Evaluate each service's
  authentication, costs, delivery constraints and public-posting semantics when
  implementing its adapter. Self-hosted ntfy on iOS needs the appropriate upstream
  wake-up relay for timely native push; self-hosting alone does not remove that.
- **Generic webhook:** custom automation destinations. Requires an explicit SSRF
  policy covering DNS resolution, private/link-local addresses, redirects, DNS
  rebinding, credential handling and outbound network restrictions. The v1
  allowlisted providers do not make arbitrary URLs safe.
- **Multiple named destinations per provider:** separate private staff/community
  channels and multiple operators, with independent events and enable switches.
- **Presets:** Operations (important errors/down/recovery), Milestones (creation,
  deaths, level-ups), Activity (adds login/logout), Custom. Volume selection is
  separate from notification urgency.
- **Structured fatal/error events:** establish a reliable error source beyond
  mudlog. Redact sensitive fields; group repeated errors; never expose raw logs in
  public destinations by default. A crashed process cannot deliver its own alert.
- **Public announcement policy:** explicit public/private destination labeling,
  per-event eligibility, player privacy choices and optional milestone thresholds
  or aggregation. Keep public templates separate from staff diagnostics.
- **Delivery controls:** exponential backoff respecting provider Retry-After,
  quiet hours, per-event rate caps, digesting, duplicate suppression and a durable
  outbox with explicit retention/replay rules. Avoid delivering stale login floods
  after an outage.
- **Delivery history and metrics:** bounded sanitized attempt history, queue depth,
  success/failure/drop counters and Prometheus collectors. Never store credentials
  or full provider responses in history.
- **Credential management:** operator-managed secret references for multiple
  destinations; evaluate encrypted write-only portal provisioning separately.
  Do not put secrets into ordinary settings JSON or audit descriptions.
- **Monitoring:** external public HTTPS/application-readiness and game-port probes;
  login-greeting synthetic check; separate successful-backup heartbeat; links to
  monitoring dashboards/status pages; maintenance windows and off-host monitoring.
  Self-hosted monitors must live outside the MUD host's failure domain.
- **Provider-neutral heartbeats:** support other hosted/self-hosted heartbeat
  destinations with the same outbound URL protections as generic webhooks.

## Validation boundaries

Unit and API tests cover opt-in routing, persistence/revisions, authentication,
CSRF, current stored level, missing credentials, queue limits, provider payloads,
secret redaction, redirects, failure/recovery, stalled heartbeat suppression,
creation idempotence, entry/extraction timing, guest/invisible-staff exclusion and
combat/environment/raw death observation. Provider requests use mocked transports;
no test sends to a real Pushover account, Discord channel or Healthchecks project.
Live credential provisioning and device receipt remain operator acceptance steps.
