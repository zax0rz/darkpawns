#!/usr/bin/env python3
"""Build the Vargus log pages from the raw archive.

The raw archive is read-only and lives outside the repository. It is read
through $DP_VARGUS_RAW (default ~/dp-community-revival/vargus/raw). The raw
files are zMUD captures: CRLF, latin-1, no ANSI. Nothing here is allowed to
put a raw file, or an unscrubbed line of one, into the site tree.

What this writes, all under website-astro/src/content/logs/:

  MANIFEST.tsv   one row per in-scope raw file, accounting for every line
  <slug>.log     the scrubbed excerpt for one scene, byte-for-byte reproducible
  <slug>.md      the entry frontmatter and lede beside it

A draft scene (flagged for crude content) is written to $DP_VARGUS_DRAFTS
(default ~/dp-community-revival/vargus/drafts) instead, so it never enters the
public repository; the manifest marks it `:draft`.

Run it with the interpreter isolated, because the inputs are untrusted:

    python3 -I website-astro/scripts/vargus_logs.py build
    python3 -I website-astro/scripts/vargus_logs.py check

`check` rebuilds the excerpts in memory and fails if the committed files
differ, which is the proof that the manifest script reproduces them exactly.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import sys
from pathlib import Path

SCRIPTS_DIR = Path(__file__).resolve().parent
ASTRO_ROOT = SCRIPTS_DIR.parent
PROJECT_ROOT = ASTRO_ROOT.parent
CONTENT_DIR = ASTRO_ROOT / "src" / "content" / "logs"
DATABASE = PROJECT_ROOT / "website" / "static" / "data" / "database.json"
CATALOG_PATH = SCRIPTS_DIR / "vargus-logs.catalog.json"

RAW_ROOT = Path(os.environ.get("DP_VARGUS_RAW", str(Path.home() / "dp-community-revival" / "vargus" / "raw")))
# A draft (a scene flagged for Zach under decision 3) is held out of the public
# repository: ARCHIVE-POLICY's "do not republish slurs, sexual material" covers
# the committed source as much as the rendered page. Its excerpt and entry are
# written here for review instead, and the manifest still accounts for it.
DRAFTS_ROOT = Path(os.environ.get("DP_VARGUS_DRAFTS", str(Path.home() / "dp-community-revival" / "vargus" / "drafts")))
SCOPE_DIRS = ("Logs/Main Collection", "Logs/Exploration & Testing")
MAX_SCENE_LINES = 2500

COLLECTION = "The Vargus logs"
COLLECTION_NOTE = "donated by Vargus"

# ---------------------------------------------------------------- redaction

EMAIL = re.compile(r"[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}")
IPV4 = re.compile(r"\b(?:\d{1,3}\.){3}\d{1,3}\b")
# perform_tell, act.comm.c:875: the receiver's line is "<name> tells you, '...'".
TELL_RECEIVED = re.compile(r"^\s*.+? tells you, '")
# perform_tell, act.comm.c:887: the sender's line is "You tell <name>, '...'".
# do_gsay (act.comm.c:848) and do_ctell (act.comm.c:1535) are public and stay.
# The sender's line can follow a prompt on the same line ("546H 268M 188V > You
# tell Mar, 'yo?'"), so it is matched anywhere a line or a prompt ends.
TELL_SENT = re.compile(r"(?:^\s*|> ?)You tell (?!the group\b|your clan\b).+?, '")
# do_spec_comm, act.comm.c:1007-1013: whisper and ask are addressed to one
# character, and the room sees only "$n whispers something to $N." The
# receiver reads "<name> whispers to you, '...'" or "<name> asks you, '...'";
# the sender reads "You whisper to <name>, '...'" or "You ask <name>, '...'".
# do_say's "You ask '...'" (act.comm.c:810) names no one and stays.
SPEC_RECEIVED = re.compile(r"^\s*.+? (?:whispers to|asks) you, '")
SPEC_SENT = re.compile(r"(?:^\s*|> ?)You (?:whisper to|ask) [A-Z][^,']*, '")
MAIL_HEADER = re.compile(r"\* \* \* \* Dark Pawns Mail System \* \* \* \*")

# nanny() password prompts, interpreter.c:1793-1967 and 2272-2278. The server
# turns echo off, but a client that ignores it captures what was typed
# (draco.txt:1258). Whatever follows the prompt is replaced, the prompt stays.
PASSWORD = re.compile(r"((?:Password|Please retype password|Enter your old password|Enter your password for verification): *)\S.*$")

# zMUD auto-login sends the name and then the password as two plain lines
# right after the client's own "Connected to host" line (rumpeq.txt:1-3).
CONNECTED = re.compile(r"^Connected to host \S+")
# Every password found either way, across the whole archive, is removed
# wherever it appears, since a player used one password on several
# characters. Filled by collect_secrets() before anything is scrubbed, so the
# passwords themselves are never written into this script.
SECRETS: set[str] = set()


def collect_secrets(files) -> set[str]:
    found: set[str] = set()
    for f in files:
        for i, line in enumerate(f.lines):
            m = PASSWORD.search(line)
            if m:
                found.add(line[m.end(1):].strip())
            if CONNECTED.match(line):
                after = [x.strip().lstrip(".") for x in f.lines[i + 1:i + 6] if x.strip().strip(".")]
                if len(after) >= 2 and re.fullmatch(r"[A-Za-z]+", after[0]) and " " not in after[1]:
                    found.add(after[1])
    return {x for x in found if len(x) >= 4}


TELL_MARKER = "[private tell removed]"
MAIL_MARKER = "[mail removed]"


def scrub_line(line: str, in_mail: bool) -> tuple[str, dict]:
    """Scrub one line in place. Returns (line, tally). Emails and addresses
    are replaced where they stand; a tell or a mail line is replaced whole."""
    counts = {"emails": 0, "ips": 0, "passwords": 0, "tells": 0, "mail": 0}
    if in_mail:
        return MAIL_MARKER, {"emails": 0, "ips": 0, "passwords": 0, "tells": 0, "mail": 1}
    if any(p.search(line) if p in (TELL_SENT, SPEC_SENT) else p.match(line)
           for p in (TELL_RECEIVED, TELL_SENT, SPEC_RECEIVED, SPEC_SENT)):
        return TELL_MARKER, {"emails": 0, "ips": 0, "passwords": 0, "tells": 1, "mail": 0}
    line, n_pw = PASSWORD.subn(r"\1[password removed]", line)
    for secret in SECRETS:
        line, n = re.subn(rf"(?<![A-Za-z0-9]){re.escape(secret)}(?![A-Za-z0-9])", "[password removed]", line)
        n_pw += n
    line, n_email = EMAIL.subn("[email removed]", line)
    line, n_ip = IPV4.subn("[address removed]", line)
    counts["emails"], counts["ips"], counts["passwords"] = n_email, n_ip, n_pw
    return line, counts


# -------------------------------------------------------------- boundaries

# A new scene begins at a relogin or after the recorder leaves. The lines are
# the game's own (comm.c login banner; act.other.c quit; nanny.c idle reap).
RELOGIN = re.compile(r"^Welcome to Dark Pawns")
IDLE_VOID = re.compile(r"^You have been idle, and are pulled into a void\.$")
QUIT = re.compile(r"^(?:You have quit\.|Goodbye, friend\.\. Come back soon!|Goodbye\.)$")
LEFT_GAME = re.compile(r"^[A-Z][A-Za-z']+ has left the game\.$")


def boundary_lines(lines: list[str]) -> list[int]:
    """1-based line numbers where a new scene should begin."""
    marks = []
    for i, text in enumerate(lines, 1):
        if i == 1:
            continue
        if RELOGIN.match(text) or IDLE_VOID.match(text):
            marks.append(i)
        elif QUIT.match(text) or LEFT_GAME.match(text):
            marks.append(i + 1)
    return sorted({m for m in marks if 1 < m <= len(lines)})


def auto_scene_ranges(n_lines: int, marks: list[int], start_at: int = 1) -> list[tuple[int, int]]:
    """Cut the file into scenes of at most MAX_SCENE_LINES, preferring the
    natural boundaries above and falling back to a nearby blank line."""
    ranges: list[tuple[int, int]] = []
    start = start_at
    while start <= n_lines:
        limit = start + MAX_SCENE_LINES - 1
        if limit >= n_lines:
            ranges.append((start, n_lines))
            break
        candidates = [m for m in marks if start < m <= limit]
        end = candidates[-1] - 1 if candidates else limit
        # A boundary too close to the start would make a runt scene.
        if candidates and end - start + 1 < MAX_SCENE_LINES // 2:
            end = limit
        ranges.append((start, end))
        start = end + 1
    return ranges


# ----------------------------------------------------------- name discovery

GROUP_ROW = re.compile(r"^\s*\[(?:\d+)H (?:\d+)M (?:\d+)V\] \[[^\]]+\] ([A-Za-z][A-Za-z']*)(?: \(Head of group\))?\s*$")
CHANNEL_ROW = re.compile(
    r"^(?:You|([A-Za-z][A-Za-z']*)) "
    r"(?:tell|tells|holler|hollers|shout|shouts|gossip|gossips|auction|auctions"
    r"|congrat|congrats|newbie|newbies|say|says|ask|asks|tell the group|tells the group"
    r"|tell your clan|tells your clan)\b"
)
ARRIVAL_ROW = re.compile(r"^([A-Za-z][A-Za-z']*) (?:arrives|has arrived|walks in|has left the game|is dead!|clambers to|staggers in)\b")
SCORE_RANK = re.compile(r"^This ranks you as ([A-Za-z][A-Za-z']*)\b")

# Names the world file says are mobs. The room contents show a mob's short
# description (s) and the keyword line (k); a player name never matches one.
def load_mob_names() -> set[str]:
    if not DATABASE.exists():
        return set()
    db = json.loads(DATABASE.read_text(encoding="utf-8"))
    names: set[str] = set()
    for row in db.get("mobs", {}).values():
        for key in ("k", "s"):
            value = (row.get(key) or "").strip()
            if value:
                names.add(value.lower())
    return names


def cast_for(lines: list[str], start: int, end: int, mobs: set[str]) -> list[tuple[str, int]]:
    """Every player name that appears in the scene, in order of first
    appearance, with the raw line number where it first appears: the group
    roster, who speaks on a channel, who arrives, who the recorder ranks as.
    Mob names are dropped."""
    seen: list[tuple[str, int]] = []
    names: set[str] = set()
    rank: tuple[str, int] | None = None
    for i in range(start - 1, end):
        text = lines[i]
        name = None
        m = GROUP_ROW.match(text)
        if m:
            name = m.group(1)
        else:
            m = CHANNEL_ROW.match(text)
            if m:
                name = m.group(1)
            else:
                m = ARRIVAL_ROW.match(text)
                if m:
                    name = m.group(1)
                else:
                    m = SCORE_RANK.match(text)
                    if m:
                        rank = (m.group(1), i + 1)
        if name and name.lower() not in mobs and name not in names:
            names.add(name)
            seen.append((name, i + 1))
    if rank and rank[0] not in names:
        seen.append(rank)
    return seen


def recorder_for(lines: list[str], start: int, end: int) -> str:
    """The character whose screen this was: the name the score command ranks
    the recorder as. Defaults to Vargus, whose archive this is (decision 6)."""
    tally: dict[str, int] = {}
    for i in range(start - 1, end):
        m = SCORE_RANK.match(lines[i])
        if m:
            tally[m.group(1)] = tally.get(m.group(1), 0) + 1
    if tally:
        return max(tally.items(), key=lambda kv: kv[1])[0]
    # The recorder reads their own actions as "You ...". When the log shows
    # Vargus from outside, at the start of a line or after a prompt
    # ("Vargus slashes ...", "Vargus says, ..."), the screen is someone
    # else's, and the log does not say whose (ssyzyilsolo1031.txt:17).
    third_person = sum(1 for i in range(start - 1, end) if THIRD_PERSON_VARGUS.match(lines[i]))
    return "" if third_person >= 3 else "Vargus"


THIRD_PERSON_VARGUS = re.compile(r"^(?:.*> )?Vargus\b(?!:)")


# ----------------------------------------------------------- content flags

# Crude content is flagged, not removed (decision 3). These are the terms the
# archive policy asks us to curate. The list is deliberately short and blunt;
# a hit is a signal for Zach, never an automatic edit.
SLURS = [
    "nigger", "nigga", "faggot", "fag", "dyke", "kike", "spic", "chink",
    "wetback", "tranny", "retard", "coon", "beaner",
]
SEXUAL = ["rape", "raped", "raping", "sex", "sexual", "fuck", "cock", "cunt", "pussy", "dildo", "penis", "orgasm"]
SLUR_RE = [(s, re.compile(rf"\b{re.escape(s)}")) for s in SLURS]
SEXUAL_RE = [(s, re.compile(rf"\b{re.escape(s)}\b")) for s in SEXUAL]


def content_flags(lines: list[str], start: int, end: int) -> list[tuple[int, str, str]]:
    flags = []
    for i in range(start - 1, end):
        low = lines[i].lower()
        hit = next((s for s, rx in SLUR_RE if rx.search(low)), None)
        if hit:
            flags.append((i + 1, "slur", hit))
            continue
        hit = next((s for s, rx in SEXUAL_RE if rx.search(low)), None)
        if hit:
            flags.append((i + 1, "sexual", hit))
    return flags

    start = 1
    while start <= n_lines:
        limit = start + MAX_SCENE_LINES - 1
        if limit >= n_lines:
            ranges.append((start, n_lines))
            break
        candidates = [m for m in marks if start < m <= limit]
        end = candidates[-1] - 1 if candidates else limit
        if end - start + 1 < MAX_SCENE_LINES // 2 and candidates:
            # A boundary too close to the start would make a runt scene.
            end = limit
        ranges.append((start, end))
        start = end + 1
    return ranges



# ------------------------------------------------------------------- dates

# A timestamp the capture kept, e.g. "Thu Feb 15 14:50:38 2001". The strongest
# evidence there is (decision 5).
STAMP = re.compile(r"^(?:Date:\s*)?([A-Z][a-z]{2}) ([A-Z][a-z]{2}) +(\d{1,2}) (\d{2}:\d{2}:\d{2}) (\d{4})\s*$")
MONTHS = {m: i for i, m in enumerate(
    ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"], 1)}
FULL_MONTHS = ["", "January", "February", "March", "April", "May", "June", "July",
               "August", "September", "October", "November", "December"]


def timestamp_date(lines: list[str], start: int, end: int) -> tuple[str, str] | None:
    for i in range(start - 1, end):
        m = STAMP.match(lines[i].strip())
        if m and m.group(1) in MONTHS and m.group(2) in MONTHS:
            label = f"{int(m.group(3))} {FULL_MONTHS[MONTHS[m.group(2)]]} {m.group(5)}"
            return label, f"timestamp in the log at line {i + 1}"


# A date in the filename is an inference, never a fact (decision 5), and most
# of these filenames do not read unambiguously (is `dmt12901` 1/29/01 or
# 12/9/01?). Rather than guess, the curated catalog carries the date label and
# its source for each entry; the manifest prints whatever the catalog says.


# ------------------------------------------------------------------- model

class RawFile:
    def __init__(self, rel: str, data: bytes):
        self.rel = rel
        self.data = data
        self.sha = hashlib.sha256(data).hexdigest()
        text = data.decode("latin1").replace("\r\n", "\n").replace("\r", "\n")
        self.lines = text.split("\n")
        if self.lines and self.lines[-1] == "":
            self.lines.pop()
        self.total = len(self.lines)


def load_files() -> list[RawFile]:
    files: list[RawFile] = []
    for scope in SCOPE_DIRS:
        directory = RAW_ROOT / scope
        if not directory.is_dir():
            raise SystemExit(f"vargus-logs: missing scope directory {directory}")
        for path in sorted(directory.iterdir()):
            if path.is_file():
                files.append(RawFile(str(path.relative_to(RAW_ROOT)), path.read_bytes()))
    return files


def duplicate_map(files: list[RawFile]) -> dict[str, str]:
    """Exact-content duplicates, by sha256. Near-duplicates (overlapping
    captures) are decided by content overlap and recorded in the catalog."""
    first: dict[str, str] = {}
    dup: dict[str, str] = {}
    for f in files:
        if f.sha in first:
            dup[f.rel] = first[f.sha]
        else:
            first[f.sha] = f.rel
    return dup


def load_catalog() -> dict:
    if not CATALOG_PATH.exists():
        return {"entries": {}, "exclude": {}}
    return json.loads(CATALOG_PATH.read_text(encoding="utf-8"))


def scene_ranges_for(f: RawFile, catalog: dict) -> list[tuple[int, int]]:
    explicit = [
        (e["range"][0], e["range"][1], slug)
        for slug, e in catalog["entries"].items()
        if e.get("source") == f.rel and e.get("range")
    ]
    if explicit:
        explicit.sort()
        return [(a, b) for a, b, _ in explicit]
    if f.rel in catalog.get("exclude", {}):
        return []
    return auto_scene_ranges(f.total, boundary_lines(f.lines))


def scrub_excerpt(f: RawFile, start: int, end: int) -> tuple[str, dict]:
    """The scrubbed excerpt text and the redaction tally for the scene."""
    out = []
    tally = {"emails": 0, "ips": 0, "passwords": 0, "tells": 0, "mail": 0}
    in_mail = False
    for i in range(start - 1, end):
        raw = f.lines[i]
        if MAIL_HEADER.search(raw):
            in_mail = True
        line, counts = scrub_line(raw, in_mail)
        for key in tally:
            tally[key] += counts[key]
        # The mail block runs to the blank line that closes it.
        if in_mail and raw.strip() == "":
            in_mail = False
        out.append(line)
    return "\n".join(out) + "\n", tally


def yaml_scalar(value) -> str:
    if isinstance(value, bool):
        return "true" if value else "false"
    if isinstance(value, (int, float)):
        return str(value)
    return json.dumps(str(value), ensure_ascii=False)


def render_entry(slug: str, f: RawFile, start: int, end: int, cast: list[tuple[str, int]],
                 recorder: str, tally: dict, flags: list[tuple[int, str, str]],
                 editorial: dict, stamp: tuple[str, str] | None) -> str:
    date_label = editorial.get("dateLabel") or (stamp[0] if stamp else "Date not recorded")
    date_source = editorial.get("dateSource") or (stamp[1] if stamp else "not recorded in the log or the filename")
    first_line = {name: line for name, line in cast}
    ordered = ([recorder] + [n for n, _ in cast if n != recorder]) if recorder in first_line else [n for n, _ in cast]
    lines = [
        "---",
        f"title: {yaml_scalar(editorial['title'])}",
        f"description: {yaml_scalar(editorial['description'])}",
        f"recordedBy: {yaml_scalar(recorder)}",
        f"dateLabel: {yaml_scalar(date_label)}",
        f"dateSource: {yaml_scalar(date_source)}",
        f"collection: {yaml_scalar(COLLECTION)}",
        f"sourceFile: {yaml_scalar(Path(f.rel).name)}",
        f"sourceLines: {yaml_scalar(f'{start:,}-{end:,} of {f.total:,}')}",
        # The raw line the excerpt starts at. Anchors on the page are the raw
        # line numbers, so #L2501 still points at line 2,501 of the capture.
        f"sourceStart: {start}",
        f"log: {yaml_scalar(slug + '.log')}",
        "redactions:",
        f"  emails: {tally['emails']}",
        f"  ips: {tally['ips']}",
        f"  passwords: {tally['passwords']}",
        f"  tells: {tally['tells']}",
        f"  mail: {tally['mail']}",
    ]
    if ordered:
        lines.append("cast:")
        lines += [f"  - {yaml_scalar(name)}" for name in ordered]
        lines.append("appearances:")
        for name in ordered:
            lines.append(f"  {yaml_scalar(name)}: {first_line.get(name, start)}")
    else:
        lines.append("cast: []")
        lines.append("appearances: {}")
    if editorial.get("contentWarning"):
        lines.append(f"contentWarning: {yaml_scalar(editorial['contentWarning'])}")
    lines.append(f"draft: {'true' if editorial.get('draft') else 'false'}")
    lines.append("---")
    lines.append("")
    lines.append(editorial["lede"].strip())
    lines.append("")
    return "\n".join(lines)


# ------------------------------------------------------------------- build

MANIFEST_COLUMNS = [
    "raw_path", "bytes", "lines", "sha256", "duplicate_of", "recorder",
    "date_evidence", "entries", "excluded", "emails", "ips", "passwords", "tells", "mail",
    "content_flags",
]


def complement(scenes: list[tuple[int, int]], total: int) -> list[tuple[int, int]]:
    gaps = []
    cursor = 1
    for start, end in sorted(scenes):
        if start > cursor:
            gaps.append((cursor, start - 1))
        cursor = max(cursor, end + 1)
    if cursor <= total:
        gaps.append((cursor, total))
    return gaps


def build(check_only: bool = False) -> int:
    files = load_files()
    SECRETS.update(collect_secrets(files))
    mobs = load_mob_names()
    catalog = load_catalog()
    entries = catalog["entries"]
    exclude = catalog.get("exclude", {})
    dup = duplicate_map(files)

    per_source: dict[str, list[tuple[int, int, str]]] = {}
    for slug, meta in entries.items():
        per_source.setdefault(meta["source"], []).append((meta["range"][0], meta["range"][1], slug))

    known = {f.rel for f in files}
    for rel in list(per_source) + list(exclude):
        if rel not in known:
            raise SystemExit(f"vargus-logs: catalog references unknown file {rel!r}")

    rows = []
    written = 0
    problems = 0
    for f in files:
        scenes = sorted(per_source.get(f.rel, []))
        for start, end, slug in scenes:
            if start < 1 or end > f.total or start > end:
                raise SystemExit(f"vargus-logs: {slug} range {start}-{end} outside {f.rel} ({f.total} lines)")
        # Guard against overlapping scenes: a line belongs to one entry only.
        covered = 0
        for start, end, slug in scenes:
            if start <= covered:
                raise SystemExit(f"vargus-logs: overlapping scene {slug} in {f.rel}")
            covered = end

        tally = {"emails": 0, "ips": 0, "passwords": 0, "tells": 0, "mail": 0}
        flags_all: list[tuple[int, str, str]] = []
        recorded = []
        for start, end, slug in scenes:
            excerpt, t = scrub_excerpt(f, start, end)
            for key in tally:
                tally[key] += t[key]
            flags = content_flags(f.lines, start, end)
            flags_all.extend(flags)
            cast = cast_for(f.lines, start, end, mobs)
            recorder = entries[slug].get("recordedBy") or recorder_for(f.lines, start, end)
            stamp = timestamp_date(f.lines, start, end)
            is_draft = bool(entries[slug].get("draft"))
            out_dir = DRAFTS_ROOT if is_draft else CONTENT_DIR
            path = out_dir / f"{slug}.log"
            if check_only:
                current = path.read_text(encoding="utf-8") if path.exists() else None
                if current != excerpt:
                    problems += 1
                    print(f"  stale excerpt: {slug}.log", file=sys.stderr)
            else:
                out_dir.mkdir(parents=True, exist_ok=True)
                path.write_text(excerpt, encoding="utf-8")
            md = render_entry(slug, f, start, end, cast, recorder, t, flags, entries[slug], stamp)
            md_path = out_dir / f"{slug}.md"
            if check_only:
                current = md_path.read_text(encoding="utf-8") if md_path.exists() else None
                if current != md:
                    problems += 1
                    print(f"  stale entry: {slug}.md", file=sys.stderr)
            else:
                md_path.write_text(md, encoding="utf-8")
            written += 1
            recorded.append(f"{slug}:{start}-{end}" + (":draft" if is_draft else ""))

        gaps = complement([(s, e) for s, e, _ in scenes], f.total)
        if f.rel in exclude:
            reason = exclude[f.rel]
            excluded = f"1-{f.total}:{reason}"
        elif gaps:
            excluded = ";".join(f"{a}-{b}:not published" for a, b in gaps)
        else:
            excluded = "-"

        rows.append([
            f.rel,
            str(len(f.data)),
            str(f.total),
            f.sha,
            dup.get(f.rel, "-"),
            recorder_for(f.lines, 1, f.total),
            (timestamp_date(f.lines, 1, f.total) or ("", "none"))[0] or "Date not recorded",
            ";".join(recorded) or "-",
            excluded,
            str(tally["emails"]),
            str(tally["ips"]),
            str(tally["passwords"]),
            str(tally["tells"]),
            str(tally["mail"]),
            ";".join(f"{ln}:{kind}" for ln, kind, _ in flags_all) or "-",
        ])

    if check_only:
        if problems:
            print(f"vargus-logs: check FAILED, {problems} stale files", file=sys.stderr)
            return 1
        print(f"vargus-logs: check OK, {written} excerpts and entries reproduce")
        return 0

    manifest = "\t".join(MANIFEST_COLUMNS) + "\n"
    manifest += "\n".join("\t".join(row) for row in rows) + "\n"
    (CONTENT_DIR / "MANIFEST.tsv").write_text(manifest, encoding="utf-8")

    # Drop any excerpt or entry the catalog no longer names, so a scene that is
    # removed does not linger as a published page.
    # A draft lives only in DRAFTS_ROOT, so its copy here is removed too.
    published = {slug for slug, meta in entries.items() if not meta.get("draft")}
    for path in CONTENT_DIR.iterdir():
        if path.suffix in (".log", ".md") and path.stem not in published:
            path.unlink()

    print(f"vargus-logs: {len(rows)} raw files, {written} entries, {len(exclude)} excluded files")
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=["build", "check", "scaffold", "digest"])
    parser.add_argument("--source", help="restrict digest to one raw file (relative path)")
    args = parser.parse_args()
    if args.command == "check":
        return build(check_only=True)
    if args.command == "scaffold":
        return scaffold()
    if args.command == "digest":
        return digest(args.source)
    return build()


# --------------------------------------------------------------- scaffolding

WORLD_JSON = PROJECT_ROOT / "website" / "static" / "map" / "world.json"
ROOM_EXITS = re.compile(r"^\s*\[ Exits: .*\]\s*$")
# comm.c:1066-1157: the prompt the server prints before the room block.
PROMPT_LINE = re.compile(r"^(?:i\d+ )?(?:(?:\d+)H )?(?:(?:\d+)M )?(?:(?:\d+)V )?(?:[^>]*?:\([^)]*\) )*(?:(?:AFK|INACTIVE) )?> ?(.*)$")
DEATH_CRY = re.compile(r"^(?:Broadcast:\s*)?(.+) is dead!\s+R\.I\.P\.$")
SLAY = re.compile(r"^You have killed (.+?)!?$")
LEVELUP = re.compile(r"^You raise a level!?|^You are now level \d+")
SAY_ECHO = re.compile(r"^You say,? '")
COMMANDISH = re.compile(r"^[a-z]{1,12}(?: [a-z']{1,12}){0,3}$")

# Era room names present in the 2000 captures that were renamed or removed in
# later world files.
ERA_ROOM_NAMES = {
    "City Gate", "City Gate South", "City Wall Road",
    "Eastern Wall Of Kir Drax'in", "Eastern wall of Kir Drax'in",
    "Entrance to Cleric Guild", "North Wall Road", "Temple of the Unholy",
    "The Abyss", "Vargus's Vestibule", "West Of The Enormous Dune",
    "Southern End of the Ardyn Park",
}


def load_known_rooms() -> set[str]:
    rooms = set(ERA_ROOM_NAMES)
    if WORLD_JSON.exists():
        wg = json.loads(WORLD_JSON.read_text(encoding="utf-8"))
        for z in wg.get("zones", []):
            for r in z.get("rooms", []):
                name = (r.get("name") or "").strip()
                if name:
                    rooms.add(name)
    return rooms


def load_mob_levels() -> dict[str, int]:
    if not DATABASE.exists():
        return {}
    db = json.loads(DATABASE.read_text(encoding="utf-8"))
    levels: dict[str, int] = {}
    for row in db.get("mobs", {}).values():
        s = (row.get("s") or "").strip().lower()
        if s:
            levels[s] = max(levels.get(s, 0), int(row.get("lvl") or 0))
    return levels


def clean_room_title(t: str) -> str:
    t = re.sub(r"\[[^\]]*\]", " ", t)
    return re.sub(r"\s+", " ", t).strip()


def room_titles(lines: list[str], start: int, end: int, known_rooms: set[str] | None = None) -> list[str]:
    """Room titles: the line a `look` prints above the description, found by
    the exits line that closes the block (act.informative.c:765, 662). Checked
    against known room names so combat or spell lines interrupting a look block
    are never mistaken for room titles."""
    if known_rooms is None:
        known_rooms = load_known_rooms()
    seen: set[str] = set()
    out: list[str] = []
    for i in range(start - 1, end):
        if not ROOM_EXITS.match(lines[i]):
            continue
        s = i - 1
        while s >= max(start - 1, i - 15) and lines[s].strip() != "" and not PROMPT_LINE.match(lines[s]):
            s -= 1
        matched = None
        if s >= start - 1:
            pm = PROMPT_LINE.match(lines[s])
            if pm and pm.group(1):
                ct = clean_room_title(pm.group(1))
                if ct in known_rooms:
                    matched = ct
        if not matched:
            for j in range(max(start - 1, s + 1), i):
                ct = clean_room_title(lines[j])
                if ct in known_rooms:
                    matched = ct
                    break
        if matched and matched not in seen:
            seen.add(matched)
            out.append(matched)
    return out


def clean_victim(raw_who: str) -> str:
    who = raw_who.strip()
    if ": " in who and not who.startswith("Broadcast"):
        who = who.split(": ")[-1].strip()
    return who


def scene_digest(f: RawFile, start: int, end: int, mobs: set[str], known_rooms: set[str] | None = None) -> dict:
    cast = cast_for(f.lines, start, end, mobs)
    recorder = recorder_for(f.lines, start, end)
    kills: list[tuple[int, str]] = []
    rooms = room_titles(f.lines, start, end, known_rooms)
    for i in range(start - 1, end):
        m = DEATH_CRY.match(f.lines[i].strip())
        if m:
            kills.append((i + 1, clean_victim(m.group(1))))
    return {
        "source": f.rel,
        "range": [start, end],
        "recorder": recorder,
        "cast": cast,
        "rooms": rooms,
        "kills": kills,
        "flags": content_flags(f.lines, start, end),
    }

# Deaths that are background NPCs or mirror-image illusions, not the point of a
# scene. A title names a place or a significant encounter, never "a janitor".
GENERIC_DEAD = {
    "The citizen", "A citizen", "A janitor", "The janitor", "The town crier",
    "A zealot", "A carpenter", "An acolyte priestess", "A wandering minstrel",
    "A male citizen of Amber", "A female citizen of Amber", "An Alaozarian noble",
    "A Grey Slayer", "A skull rider", "A thief", "The thief",
    "A fleeting image", "Someone", "Somebody's mother", "A giant mosquito",
    "a giant mosquito",
}


def notable_kills(kills: list[tuple[int, str]], mob_levels: dict[str, int]) -> list[tuple[int, str, int]]:
    out: list[tuple[int, str, int]] = []
    for ln, who in kills:
        if who in GENERIC_DEAD or who.lower() in {g.lower() for g in GENERIC_DEAD}:
            continue
        lvl = mob_levels.get(who.lower())
        if lvl is not None and lvl < 20:
            continue
        out.append((ln, who, lvl if lvl is not None else 99))
    return out


def auto_editorial(d: dict, f: RawFile, start: int, end: int, n: int, total: int, mob_levels: dict[str, int] | None = None) -> dict:
    if mob_levels is None:
        mob_levels = load_mob_levels()
    rec = d["recorder"]
    others = [name for name, _ in d["cast"] if name != rec]
    room = d["rooms"][0] if d["rooms"] else None
    notable = notable_kills(d["kills"], mob_levels)
    name = Path(f.rel).name
    # "the fall of The dracolich" reads as a typo; mid-title, the mob's
    # leading article is lowercased.
    of = lambda who: re.sub(r"^(The|A|An) ", lambda m: m.group(1).lower() + " ", who)

    # An empty recorder means the log does not show whose screen it was
    # (recorder_for), so the text says that instead of naming anyone.
    if notable:
        high = [x for x in notable if x[2] >= 25]
        first_ln, who, _ = high[0] if high else notable[0]
        title = f"{rec} and the fall of {of(who)}" if rec else f"The fall of {of(who)}"
    elif room:
        first_ln, who = (d["kills"][0] if d["kills"] else (None, None))
        title = f"{rec} in {room}" if rec else room
    else:
        first_ln, who = (d["kills"][0] if d["kills"] else (None, None))
        title = f"{rec}'s session" if rec else f"A session with {', '.join(others[:2]) or 'Vargus'}"
    if total > 1:
        title += f" (part {n})"

    description = (
        (f"{rec}'s captured screen" if rec else "A captured screen")
        + f", lines {start:,}-{end:,} of {name}"
        + (f", with {', '.join(others[:3])}" if others else "")
        + "."
    )

    bits = [f"{rec} records lines {start:,}-{end:,} of {name}." if rec else
            f"Lines {start:,}-{end:,} of {name}, from a screen the log does not name."]
    if others:
        bits.append(f"Others on screen include {', '.join(others[:6])}.")
    if notable:
        k_ln, k_who, _ = high[0] if (notable and any(x[2] >= 25 for x in notable)) else notable[0]
        bits.append(f"{k_who} falls at line {k_ln:,}.")
    elif d["kills"]:
        k_ln, k_who = d["kills"][0]
        bits.append(f"The first death is {k_who} at line {k_ln:,}.")
    if room:
        bits.append(f"The scene opens around {room}.")
    lede = " ".join(bits[:3])

    entry = {
        "source": f.rel,
        "range": [start, end],
        "title": title,
        "description": description,
        "lede": lede,
    }
    if d["flags"]:
        entry["draft"] = True
        entry["contentWarning"] = "This log contains crude language; review is pending."
    return entry


# The prototype entry Zach approved on 2026-11-03. Its range and editorial are
# pinned here so the bulk build reproduces the committed nimos-raid page
# exactly instead of resplitting it.
PROTOTYPE_ENTRIES = {
    "Logs/Main Collection/nimos.txt": [
        {
            "range": [1, 2032],
            "slug": "nimos-raid",
            "editorial": {
                "title": "The raid on Nimos and Lady Amber",
                "description": "Vargus's screen during a seventeen-player raid on Nimos and Lady Amber, from the backstab to both deaths.",
                "lede": "Vargus leads a group of seventeen into Lady Amber's keep. They gather in her bedroom, buff, and move west into the Secret Room, where Nimos and Lady Amber are waiting. Aiko opens with a backstab. Nimos dies at line 1,731 and Lady Amber at line 2,029.",
            },
        },
    ],
}


# A raw filename is never a slug when it is crude (decision 3). The file
# keeps its name in the provenance; only its URL changes.
SLUG_BASE = {
    "Logs/Main Collection/nimosladysolo1026002gaydeath.txt": "nimosladysolo1026002-death",
}


def plan_scenes(f: RawFile) -> list[tuple[int, int, str, dict | None]]:
    """The scenes for one file: any pinned prototype entries first, the rest
    auto-split around them."""
    marks = boundary_lines(f.lines)
    pinned = sorted(PROTOTYPE_ENTRIES.get(f.rel, []), key=lambda p: p["range"][0])
    out: list[tuple[int, int, str, dict | None]] = []
    counter = 0
    cursor = 1

    def add_auto(lo: int, hi: int) -> None:
        nonlocal counter
        if lo > hi:
            return
        for a, b in auto_scene_ranges(hi, marks, start_at=lo):
            counter += 1
            out.append((a, b, f"{base}-{counter}", None))

    base = SLUG_BASE.get(f.rel) or re.sub(r"[^a-z0-9]+", "-", Path(f.rel).stem.lower()).strip("-")
    for pin in pinned:
        start, end = pin["range"]
        add_auto(cursor, start - 1)
        out.append((start, end, pin["slug"], pin["editorial"]))
        cursor = end + 1
    add_auto(cursor, f.total)
    return out


# Files in scope that are not published, with the reason recorded in the
# manifest. random12100.txt is byte-identical to random.txt (same length, same
# sha256), so it is the same capture under a second name.
EXCLUDE = {
    "Logs/Main Collection/random12100.txt": "byte-identical duplicate of random.txt",
}


def scaffold() -> int:
    files = load_files()
    mobs = load_mob_names()
    known_rooms = load_known_rooms()
    mob_levels = load_mob_levels()
    out = {"entries": {}, "exclude": dict(EXCLUDE)}
    for f in files:
        if f.rel in EXCLUDE:
            continue
        scenes = plan_scenes(f)
        for n, (start, end, slug, pinned) in enumerate(scenes, 1):
            if pinned:
                out["entries"][slug] = {
                    "source": f.rel, "range": [start, end], **pinned,
                }
                continue
            d = scene_digest(f, start, end, mobs, known_rooms)
            out["entries"][slug] = auto_editorial(d, f, start, end, n, len(scenes), mob_levels)
    CATALOG_PATH.write_text(json.dumps(out, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    print(f"vargus-logs: scaffold wrote {len(out['entries'])} entries to {CATALOG_PATH.name}")
    return 0


def digest(only: str | None) -> int:
    """Print the factual signals for every scene, to author the catalog from."""
    files = load_files()
    mobs = load_mob_names()
    known_rooms = load_known_rooms()
    for f in files:
        if only and f.rel != only:
            continue
        scenes = auto_scene_ranges(f.total, boundary_lines(f.lines))
        print(f"\n=== {f.rel}  ({f.total} lines, {len(scenes)} scenes) sha {f.sha[:12]}")
        for n, (start, end) in enumerate(scenes, 1):
            d = scene_digest(f, start, end, mobs, known_rooms)
            cast_names = [name for name, _ in d["cast"][:10]]
            print(f"  [{n}] {start}-{end}  rec={d['recorder']}  cast={', '.join(cast_names)}")
            print(f"      rooms={'; '.join(d['rooms'][:6])}")
            if d["kills"]:
                print(f"      deaths={'; '.join(f'{ln}:{who}' for ln, who in d['kills'][:12])}")
            if d["flags"]:
                print(f"      flags={d['flags'][:8]}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

