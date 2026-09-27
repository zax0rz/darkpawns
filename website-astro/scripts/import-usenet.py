#!/usr/bin/env python3
"""Build curated Dark Pawns Usenet archive entries from reviewed extracts."""

from __future__ import annotations

import argparse
import collections
import datetime as dt
import html
import json
import pathlib
import re

POST_RE = re.compile(r"^--- #(\d+) ([^|]+?) \| (.+?) ---$", re.MULTILINE)
EMAIL_RE = re.compile(r"(?<![\w.])[\w.+-]+@[\w.-]+\.[A-Za-z]{2,}(?![\w.])")
OBFUSCATED_EMAIL_RE = re.compile(r"\S+@\S+")
PERSONAL_URL_RE = re.compile(r"https?://\S*~\S+")

RECORDS = [
    {"slug": "usenet-1996-12-03-looking-for-players", "file": "mention_rec_games_mud_misc_PDU4MnJpZSQ3MWJA.txt", "description": "Oddity announces Dark Pawns at knight.ufp.org, describing its CircleMUD base, original world, remorts, races, diseases and player killing.", "crossposts": ["mention_rec_games_mud_admin_PDU4MnBscCQ3MWJA.txt"]},
    {"slug": "usenet-1996-12-23-beta-ad", "file": "thread_rec_games_mud_diku_PDU5bmZzdiRmNDNA.txt", "description": "Serapis advertises the final beta at www.augusta.net; a reply two days later supplies the earliest surviving public response to Dark Pawns."},
    {"slug": "usenet-1997-03-22-palios-mudlist", "file": "mention_rec_games_mud_admin_PDMzMzU3NUNFLjdE.txt", "description": "Palio's public MUD directory lists Dark Pawns among CircleMUD servers at pawns.guru.org and 204.116.88.40.", "excerpt": "mudlist"},
    {"slug": "usenet-1997-05-19-grand-opening", "file": "thread_rec_games_mud_diku_PDVsczE2NCRqcGEk.txt", "description": "The grand-opening announcement schedules Dark Pawns for May 21, 1997 after two years of testing and development.", "crossposts": ["thread_rec_games_mud_announce_PDVsc3FnOCRlcTdA.txt"]},
    {"slug": "usenet-1997-06-08-open-now", "file": "thread_rec_games_mud_diku_PDVuaDVsOSRxNTck.txt", "description": "A June 1997 advertisement says Dark Pawns is open at pawns.guru.org and inventories the game's races, remorts and custom systems."},
    {"slug": "usenet-1997-06-26-advertisement", "file": "thread_rec_games_mud_announce_PDVvdmltayQzbWNA.txt", "description": "Serapis advertises Dark Pawns as a CircleMUD 3.00 beta 11 game with free rent, diseases, languages, mounts and ranged combat."},
    {"slug": "usenet-1997-08-05-moved", "file": "thread_rec_games_mud_admin_PDg3MDg3NjkxNi4x.txt", "description": "A move notice gives Dark Pawns' August 1997 host, address and feature list after the server changed locations."},
    {"slug": "usenet-1997-10-01-advertisement", "file": "thread_rec_games_mud_diku_PDYxMHM4aSQzMDAk.txt", "description": "An October 1997 advertisement describes the rules, races, diseases, remort classes and player-killing policy at pawns.guru.org.", "crossposts": ["thread_rec_games_mud_announce_PDYxMnV0cCQzOWJx.txt"]},
    {"slug": "usenet-1998-01-13-recruiting", "file": "thread_rec_games_mud_announce_PDY5amM3YSQzODU0.txt", "description": "Dark Pawns recruits new players in January 1998 and describes a game developed for more than three years."},
    {"slug": "usenet-1998-03-22-advertisement", "file": "thread_rec_games_mud_diku_PDZmNjdqcCQxODJt.txt", "description": "A March 1998 advertisement records the no-rent hometown rule and the game's then-current systems and address."},
    {"slug": "usenet-1998-09-13-still-alive", "file": "thread_rec_games_mud_announce_PG10Mi4wLTIwNDU2.txt", "description": "The 'still alive' advertisement places Dark Pawns at mud.darkrune.org and says the world had been played and developed for more than four years."},
    {"slug": "usenet-1998-11-18-darkrune-signature", "file": "mention_rec_games_mud_admin_PDM2MzlEQkYyLjgz.txt", "description": "A post by Stephen C. Thompson in a MUD administration thread carries a contemporary Dark Pawns signature for mud.darkrune.org.", "posts": [31], "completeness": "partial", "note": "Only post 31 is reproduced. The 147-post thread concerned character ownership and linked-list algorithms; this post is retained for its contemporary Dark Pawns signature."},
    {"slug": "usenet-1998-12-31-server-move", "file": "thread_rec_games_mud_announce_PG10Mi4wLTIzODA3.txt", "description": "A December 1998 move notice announces the new pawns.wolfpaw.net host and port 4300."},
    {"slug": "usenet-1999-02-11-license-dispute", "file": "thread_rec_games_mud_diku_PDdhMWZpMCQ3ZmUk.txt", "description": "A Dark Pawns advertisement becomes a seventeen-post public dispute over DikuMUD credits, source lineage and license compliance.", "warning": "This thread contains hostile accusations and period-typical insults. It is preserved as a disputed public record, not as an editorial verdict."},
    {"slug": "usenet-1999-02-12-license-follow-up", "file": "thread_rec_games_mud_diku_PDM2QzY1RUQ1Ljc3.txt", "description": "A three-post follow-up continues the Dark Pawns license dispute and the attempted verification of its credits screen.", "warning": "This thread contains hostile accusations. It is preserved as a disputed public record, not as an editorial verdict."},
    {"slug": "usenet-1999-05-03-version-2-2", "file": "thread_rec_games_mud_diku_PDdnbmxrNSR0Mmgk.txt", "description": "Frontline announces Dark Pawns 2.2, calling it the largest source update in the game's five-year history.", "crossposts": ["thread_rec_games_mud_misc_PDdnbmxsayR0MnYk.txt", "thread_rec_games_mud_announce_PG10Mi4wLTkwNTYt.txt"]},
    {"slug": "usenet-2000-06-18-hall-of-shame", "file": "mention_rec_games_mud_misc_PDM5NGU3OGEwXzFA.txt", "description": "AxL lists Dark Pawns in a public accusation that several DikuMUD derivatives were not displaying the required credits.", "excerpt": "hall-of-shame", "completeness": "partial", "note": "The introduction and Dark Pawns line are reproduced. Other named MUDs and all contact addresses are omitted.", "warning": "This is a public accusation from 2000, preserved as evidence in the license dispute. The archive does not present it as an independently established verdict."},
    {"slug": "usenet-2002-12-14-eighth-year", "file": "thread_rec_games_mud_announce_PG10Mi4wLTg4NzYt.txt", "description": "A 2002 advertisement by Zach describes Dark Pawns as entering its eighth year of continuous operation with more than 8,000 rooms."},
]


def parse_extract(path: pathlib.Path) -> dict:
    text = path.read_text(encoding="utf-8")
    header, body = text.split("\n\n", 1)
    values = {}
    for line in header.splitlines():
        if ":" in line and not line.startswith(" "):
            key, value = line.split(":", 1)
            values[key] = value.strip()
        elif line.lstrip().startswith("URL:"):
            values["URL"] = line.split("URL:", 1)[1].strip()
    matches = list(POST_RE.finditer(body))
    posts = []
    for index, match in enumerate(matches):
        end = matches[index + 1].start() if index + 1 < len(matches) else len(body)
        posts.append({"number": int(match.group(1)), "author": match.group(2).strip(), "date": match.group(3).strip(), "body": body[match.end():end].strip()})
    return {"subject": values["Subject"], "group": values["Group"], "message_id": values["Message-ID (root)"], "url": values["URL"], "posts": posts}


def parse_date(value: str) -> dt.datetime:
    return dt.datetime.strptime(value, "%b %d, %Y %H:%M UTC").replace(tzinfo=dt.timezone.utc)


def excerpt_posts(record: dict, parsed: dict) -> list[dict]:
    if record.get("posts"):
        wanted = set(record["posts"])
        return [post for post in parsed["posts"] if post["number"] in wanted]
    if record.get("excerpt") == "mudlist":
        post = parsed["posts"][0].copy()
        line = next(line for line in post["body"].splitlines() if "Dark Pawns" in line)
        post["body"] = "Type: Circle\n\n" + line.strip()
        return [post]
    if record.get("excerpt") == "hall-of-shame":
        post = parsed["posts"][0].copy()
        intro = post["body"].split("The List: (6)", 1)[0].strip()
        dark_pawns = next(line for line in post["body"].splitlines() if line.strip().startswith("Dark Pawns"))
        post["body"] = intro + "\n\nThe List: (excerpt)\n\n" + dark_pawns.rstrip()
        return [post]
    return parsed["posts"]


def quoted(value: str) -> str:
    return json.dumps(value, ensure_ascii=False)


def render_record(record: dict, parsed: dict, source_root: pathlib.Path) -> str:
    posts = excerpt_posts(record, parsed)
    dates = [parse_date(post["date"]) for post in posts]
    first, last = min(dates), max(dates)
    date_label = first.strftime("%B %-d, %Y")
    if first.date() != last.date():
        if first.year == last.year and first.month == last.month:
            date_label = f"{first.strftime('%B %-d')}-{last.strftime('%-d, %Y')}"
        elif first.year == last.year:
            date_label = f"{first.strftime('%B %-d')}-{last.strftime('%B %-d, %Y')}"
        else:
            date_label += "-" + last.strftime("%B %-d, %Y")
    participants = collections.Counter(post["author"] for post in posts)
    completeness = record.get("completeness", "complete")
    lines = ["---", f"title: {quoted(parsed['subject'])}", f"description: {quoted(record['description'])}", 'kind: "usenet-thread"', f"sortDate: {first.date().isoformat()}", f"dateLabel: {quoted(date_label)}", f"publishedAt: {first.date().isoformat()}", 'sourceSite: "Usenet"', f"sourceUrl: {quoted(parsed['url'])}", f"captureUrl: {quoted(parsed['url'])}", "recoveredAt: 2026-09-25", f"textKind: {quoted('edited-excerpt' if completeness == 'partial' else 'verbatim')}", 'source: "UsenetArchives capture identified by captureUrl and Message-ID"', 'voiceLayer: "frontline"', f"board: {quoted(parsed['group'])}", f"messageId: {quoted(parsed['message_id'])}", f"postCount: {len(posts)}", f"completeness: {quoted(completeness)}"]
    if record.get("note"):
        lines.append(f"completenessNote: {quoted(record['note'])}")
    if record.get("warning"):
        lines.append(f"contentWarning: {quoted(record['warning'])}")
    lines.append("participants:")
    for author, count in participants.items():
        lines.extend([f"  - name: {quoted(author)}", '    role: "unknown"', f"    posts: {count}"])
    if record.get("crossposts"):
        lines.append("crossposts:")
        for filename in record["crossposts"]:
            other = parse_extract(source_root / "threads" / filename)
            lines.extend([f"  - newsgroup: {quoted(other['group'])}", f"    messageId: {quoted(other['message_id'])}", f"    captureUrl: {quoted(other['url'])}"])
    lines.extend(["draft: false", "---", "", "*Transcript note: recovered from UsenetArchives. Email addresses and personal contact URLs are redacted; spelling, punctuation and the remaining text are preserved.*", ""])
    for post in posts:
        body = EMAIL_RE.sub("[email redacted]", post["body"])
        body = OBFUSCATED_EMAIL_RE.sub("[email redacted]", body)
        body = PERSONAL_URL_RE.sub("[personal URL redacted]", body)
        body = "\n".join(line.rstrip() for line in body.splitlines())
        lines.extend([f"### {post['author']} — {post['date']}", "", '<pre class="usenet-transcript">' + html.escape(body) + "</pre>", ""])
    return "\n".join(lines).rstrip() + "\n"


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("source", type=pathlib.Path)
    parser.add_argument("output", type=pathlib.Path)
    args = parser.parse_args()
    args.output.mkdir(parents=True, exist_ok=True)
    for record in RECORDS:
        parsed = parse_extract(args.source / "threads" / record["file"])
        target = args.output / f"{record['slug']}.md"
        target.write_text(render_record(record, parsed, args.source), encoding="utf-8")
        print(target)


if __name__ == "__main__":
    main()
