#!/usr/bin/env python3
"""Generate the public project ledger from Git and, when available, GitHub."""

from __future__ import annotations

import json
import os
import subprocess
import urllib.request
from collections import Counter
from datetime import UTC, date, datetime, timedelta
from pathlib import Path


SITE = Path(__file__).resolve().parents[1]
ROOT = SITE.parent
OUTPUT = SITE / "src" / "generated" / "project-activity.json"
REPOSITORY = "zax0rz/darkpawns"


def git(*args: str) -> str:
    return subprocess.check_output(
        ["git", *args], cwd=ROOT, text=True, stderr=subprocess.DEVNULL
    ).strip()


def monday(day: date) -> date:
    return day - timedelta(days=day.weekday())


def weekly_activity(today: date) -> list[dict[str, object]]:
    first_week = monday(today) - timedelta(weeks=11)
    dates = git(
        "log",
        "--first-parent",
        f"--since={first_week.isoformat()}",
        "--format=%cs",
        "HEAD",
    ).splitlines()
    counts = Counter(monday(date.fromisoformat(value)) for value in dates if value)
    return [
        {
            "week": (first_week + timedelta(weeks=offset)).isoformat(),
            "commits": counts[first_week + timedelta(weeks=offset)],
        }
        for offset in range(12)
    ]


def local_commits() -> list[dict[str, str]]:
    rows = git(
        "log",
        "--first-parent",
        "-5",
        "--format=%H%x09%cs%x09%s",
        "HEAD",
    ).splitlines()
    return [
        {
            "sha": sha,
            "date": committed,
            "title": title,
            "url": f"https://github.com/{REPOSITORY}/commit/{sha}",
        }
        for sha, committed, title in (row.split("\t", 2) for row in rows)
    ]


def github_prs() -> list[dict[str, object]] | None:
    token = os.environ.get("GITHUB_TOKEN")
    if not token:
        return None

    request = urllib.request.Request(
        f"https://api.github.com/repos/{REPOSITORY}/pulls?state=closed&sort=updated&direction=desc&per_page=20",
        headers={
            "Accept": "application/vnd.github+json",
            "Authorization": f"Bearer {token}",
            "User-Agent": "darkpawns-site-build",
            "X-GitHub-Api-Version": "2022-11-28",
        },
    )
    try:
        with urllib.request.urlopen(request, timeout=15) as response:
            records = json.load(response)
    except Exception:
        return None

    return [
        {
            "number": record["number"],
            "title": record["title"],
            "url": record["html_url"],
            "mergedAt": record["merged_at"],
        }
        for record in records
        if record["merged_at"]
    ][:5]


def main() -> None:
    today = datetime.now(UTC).date()
    revision_time = git("show", "-s", "--format=%cI", "HEAD")
    existing: dict[str, object] = {}
    if OUTPUT.exists():
        existing = json.loads(OUTPUT.read_text(encoding="utf-8"))

    pull_requests = github_prs()
    payload = {
        "generatedAt": revision_time,
        "repository": f"https://github.com/{REPOSITORY}",
        "activitySource": "local Git history, first-parent commits on the build revision",
        "weeks": weekly_activity(today),
        "recentCommits": local_commits(),
        "pullRequests": pull_requests if pull_requests is not None else existing.get("pullRequests", []),
        "pullRequestSource": "GitHub REST API; the checked-in result is retained when GITHUB_TOKEN is absent",
    }
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    OUTPUT.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
