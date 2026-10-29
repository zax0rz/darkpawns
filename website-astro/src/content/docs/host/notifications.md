---
title: "Notifications"
description: "Configure outbound game events, gossip mirroring, and health checks, then inspect delivery evidence and troubleshoot common failures."
type: reference
audience: operators
verified: v0.1.7
---

The Notifications page at `/admin/staff/notifications` is the single place for
every outbound channel this server owns. It needs a current HIGOD access level,
and every change is audited. Each channel is one block with the same parts: an
enable switch, that channel's settings, its write-only credentials, its delivery
evidence, and a test send.

| Channel | What it carries | Destination |
| --- | --- | --- |
| Discord alerts | Selected character observations | A Discord channel webhook |
| Pushover | The same observations, selected independently | A Pushover user or group key |
| Gossip mirror | Every public gossip line | A second Discord channel webhook |
| Healthchecks heartbeat | One check-in a minute | A hosted Healthchecks ping URL |

Nothing is sent until a channel is enabled, and a channel cannot be enabled
without credentials. Subscriptions start off.

## Events

Only **Discord alerts** and **Pushover** let you select events. Guests and
invisible staff are excluded from every observation. The gossip mirror sends
public gossip lines; the heartbeat sends its timed check-in.

| Event ID | Observation boundary | Message |
| --- | --- | --- |
| `player.created` | The character is saved for the first time | Avery created a character. |
| `player.login` | The character is admitted to the world | Avery entered the game. |
| `player.logout` | The character leaves the world | Avery left the game. |
| `player.death` | The death path, including raw kills | Avery died. |
| `player.level` | Each level gained through play | Avery reached level 20. |

A login is world entry, not authentication at the menu. A logout is world
departure, so a linkdead character that stays in the game has not logged out.
Creation never announces a host, an email address, a password or a location. A
message carries a character name and, for level gains, the new level: no mudlog
free text, no room detail, no IP address. Discord mentions are disabled in the
request payload, so a character named `everyone` cannot ping a channel.

## Credentials

Credentials are pasted into the channel block and are write-only: the API never
returns a saved secret, and the page only reports whether one exists and whether
it came from the console or the server environment. Replacing a credential takes
effect immediately, without a restart. Removing a credential also disables that
channel, keeps its event selections, and stops the environment fallback from
being used again.

Credential formats are checked against the provider:

- Pushover takes an application API token and a user or group key, both 30 characters.
- Discord takes a channel webhook URL on `discord.com` with no query string.
- Healthchecks takes a hosted ping URL beginning with `https://hc-ping.com/`.

Provider hosts and paths are restricted, HTTPS is required, and redirects are
refused. Upstream response bodies and credential-bearing URLs never reach the API
errors or the log.

## Where the settings live

`lib/data/notifications.json` holds the routing configuration and the credential
ciphertexts, written atomically with mode 0600. Each credential is stored as
AES-256-GCM ciphertext with a per-channel authenticated context and a fresh
nonce. The master key is a random 32-byte file created on the first credential
save, mode 0600, at `<config-path>.key` unless `DP_NOTIFICATION_KEY_PATH` names
another location. A missing, replaced or unreadable key fails closed.

## Test sends

Each block has its own test button, and a test runs while the channel is
disabled. That is deliberate: the moment you paste a webhook is the moment you
want to know it works. A test uses the saved credentials, so save or discard a
pending credential change first.

A test reports **queued**, not delivered. The provider's answer appears under
Delivery once the request completes, and a provider accepting a request is not
proof that a phone displayed it. The Discord alerts, Pushover and Healthchecks
tests send the usual payload. The gossip test posts one labelled line to the
gossip channel, because that channel is public and a bare test would read as
somebody's gossip.

## Delivery

Delivery is best effort. Events are not replayed after an outage or a restart, and
a test that arrives during a queue or rate-limit window is skipped rather than
queued behind a backlog. Each block counts what it skipped and shows the last
provider answer, so a silent channel is visible rather than assumed healthy.

The gossip mirror also honours Discord's rate-limit response: after a rejection it
waits for the delay the response asks for and drops lines arriving during that
cooldown. Nothing from Discord returns to the game.

## Gossip mirror

The mirror carries every public gossip line to its own Discord webhook, rendered
the way a level one mortal sees the channel. Nothing is selected: it is not an
event destination. Mirroring is off until it is enabled, credentials alone never
enable it, and removing the credentials disables it and discards queued lines.

While the mirror is active, the automatic MOTD notice tells players that gossip
echoes to Discord. It is on by default and can be turned off when your own MOTD
already discloses the mirror.

## Healthchecks heartbeat

The heartbeat sends one request a minute while the game loop is progressing, and
stops when the loop stalls or the process dies. [Healthchecks.io](https://healthchecks.io/) notices the missing
check-ins. Configure outage and recovery alerts in Healthchecks so they can reach
you when this server is down: a crashed process cannot send its own alert.

## Troubleshooting

| Symptom | Cause and fix |
| --- | --- |
| Setup needed, switch disabled | No usable credentials. Paste them in that channel's Credentials group, then enable the channel. |
| Save refused after another admin edited the page | The settings revision moved. Reload settings and reapply your change. |
| Credential save fails with a storage message | The master key is missing, replaced or unreadable, or the settings directory is not writable. Restore the matching key file from the same backup as the settings file, then check permissions. |
| Test queued but nothing arrives | Check Delivery for the provider's answer. A rejected request records the failure and increments the skipped count. |
| Healthchecks reports an outage while the game is running | The loop has not progressed for over 30 seconds, or the heartbeat is disabled. Check the game's own logs before the ping URL. |
| Gossip lines missing from Discord | The mirror is disabled, its credentials were removed, or Discord asked for a cooldown. Delivery shows the last failure and the skipped count. |

Keep the settings file and its key together in protected backups. Losing the key
loses the credentials; exposing both exposes them. This protects a leaked
settings file, not a compromised host or a full backup.
