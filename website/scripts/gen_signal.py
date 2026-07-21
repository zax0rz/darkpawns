#!/usr/bin/env python3
"""Fetch a sanitized "signal" snapshot (online player count + recent narrative
events) from the Dark Pawns admin API and emit website/data/signal.json.

Runs at build time (make build-site / deploy-site) so the homepage Plates band
can show live-ish world state without exposing admin credentials to browsers.

Auth contract (pkg/admin):
  POST /admin/login {"player_name","password"} -> {"token","role"}
  GET  /admin/players   (Bearer) -> [{"name","level","room"}, ...]
  GET  /admin/narrative (Bearer) -> recent narrative events

Env:
  DP_ADMIN_USER      builder-role player name  (required; absent = skip quietly)
  DP_ADMIN_PASSWORD  builder-role password     (required; absent = skip quietly)
  DP_API_BASE        default https://darkpawns.labz0rz.com

  Any of the above may also live in the repo-root .env file (gitignored);
  real environment variables always win over .env values.

Privacy: only the player COUNT is published, never names. Narrative events are
truncated and capped.
"""

import json
import os
import sys
import urllib.request
from datetime import datetime, timezone
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent.parent
OUT_PATH = Path(__file__).resolve().parent.parent / "data" / "signal.json"
MAX_EVENTS = 5
MAX_EVENT_LEN = 160
TIMEOUT = 10
UA = "gen_signal/1.0 (darkpawns build)"


def _load_dotenv(path):
    """Populate os.environ from a KEY=VALUE .env file without overriding real env."""
    try:
        lines = path.read_text().splitlines()
    except OSError:
        return
    for line in lines:
        line = line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, _, value = line.partition("=")
        key = key.strip().removeprefix("export ").strip()
        value = value.strip().strip('"').strip("'")
        if key and key not in os.environ:
            os.environ[key] = value


_load_dotenv(REPO_ROOT / ".env")

API_BASE = os.environ.get("DP_API_BASE", "https://darkpawns.labz0rz.com")


def _post_json(url, payload):
    req = urllib.request.Request(
        url,
        data=json.dumps(payload).encode(),
        headers={"Content-Type": "application/json", "User-Agent": UA},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=TIMEOUT) as resp:
        return json.loads(resp.read())


def _get_json(url, token):
    req = urllib.request.Request(url, headers={"Authorization": f"Bearer {token}", "User-Agent": UA})
    with urllib.request.urlopen(req, timeout=TIMEOUT) as resp:
        return json.loads(resp.read())


def _event_texts(raw):
    """Pull displayable strings out of the narrative feed, tolerating shape drift."""
    if isinstance(raw, dict):
        for key in ("events", "entries", "items", "narrative"):
            if isinstance(raw.get(key), list):
                raw = raw[key]
                break
    if not isinstance(raw, list):
        return []
    texts = []
    for item in raw:
        if isinstance(item, str):
            texts.append(item)
        elif isinstance(item, dict):
            for key in ("text", "summary", "description", "message", "content"):
                if isinstance(item.get(key), str) and item[key].strip():
                    texts.append(item[key].strip())
                    break
    cleaned = []
    for t in texts:
        t = " ".join(t.split())
        if not t:
            continue
        cleaned.append(t[: MAX_EVENT_LEN - 1] + "…" if len(t) > MAX_EVENT_LEN else t)
    return cleaned[:MAX_EVENTS]


def main():
    user = os.environ.get("DP_ADMIN_USER")
    password = os.environ.get("DP_ADMIN_PASSWORD")
    if not user or not password:
        print("gen_signal: DP_ADMIN_USER/DP_ADMIN_PASSWORD not set — skipping signal snapshot")
        return 0

    try:
        login = _post_json(f"{API_BASE}/admin/login", {"player_name": user, "password": password})
        token = login["token"]
        players = _get_json(f"{API_BASE}/admin/players", token)
        try:
            narrative = _get_json(f"{API_BASE}/admin/narrative?limit={MAX_EVENTS * 3}", token)
        except Exception as exc:  # narrative feed is a bonus, never fatal
            print(f"gen_signal: narrative fetch failed ({exc}); continuing without events")
            narrative = []
    except Exception as exc:
        # Deploys must not fail because the game server was unreachable; the
        # Plates band simply renders without signal data.
        print(f"gen_signal: admin API unreachable ({exc}); skipping signal snapshot")
        return 0

    payload = {
        "generated_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        "players_online": len(players) if isinstance(players, list) else 0,
        "events": _event_texts(narrative),
    }
    OUT_PATH.parent.mkdir(parents=True, exist_ok=True)
    OUT_PATH.write_text(json.dumps(payload, indent=2) + "\n")
    print(f"gen_signal: wrote {OUT_PATH} ({payload['players_online']} online, {len(payload['events'])} events)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
