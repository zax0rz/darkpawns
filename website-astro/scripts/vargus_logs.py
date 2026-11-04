#!/usr/bin/env python3
"""Build the Vargus log pages from the raw archive.

The raw archive is read-only and lives outside the repository. It is read
through $DP_VARGUS_RAW (default ~/dp-community-revival/vargus/raw). The raw
files are zMUD captures: CRLF, latin-1, no ANSI. Nothing here is allowed to
put a raw file, or an unscrubbed line of one, into the site tree.

One capture file is one page, whole. What this writes, all under
website-astro/src/content/logs/:

  MANIFEST.tsv   one row per in-scope raw file, accounting for every line
  <slug>.log     the scrubbed capture, byte-for-byte reproducible
  <slug>.md      its frontmatter: file name, recorder, characters, redactions

Nothing on a page is written by us: the title is the file name Vargus gave
the capture, and every other field is read from the capture itself.

Run it with the interpreter isolated, because the inputs are untrusted:

    python3 -I website-astro/scripts/vargus_logs.py build
    python3 -I website-astro/scripts/vargus_logs.py check

`check` rebuilds everything in memory and fails if the committed files
differ, which is the proof that this script reproduces them exactly.
"""

from __future__ import annotations

import argparse
import datetime as dt
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

RAW_ROOT = Path(os.environ.get("DP_VARGUS_RAW", str(Path.home() / "dp-community-revival" / "vargus" / "raw")))
SCOPE_DIRS = ("Logs/Main Collection", "Logs/Exploration & Testing")

# Files in scope that are not published, with the reason the manifest records.
EXCLUDE = {
    # Same length and sha256 as random.txt: one capture under two names.
    "Logs/Main Collection/random12100.txt": "byte-identical duplicate of random.txt",
}

# A file whose name is a slur keeps it out of the URL and the title; the
# page says the name was withheld.
WITHHELD_NAMES = {
    "Logs/Main Collection/nimosladysolo1026002gaydeath.txt": ("nimosladysolo1026002-2", "nimosladysolo1026002, second file"),
}

# ---------------------------------------------------------------- redaction

EMAIL = re.compile(r"[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}")
IPV4 = re.compile(r"\b(?:\d{1,3}\.){3}\d{1,3}\b")
# perform_tell, act.comm.c:875: the receiver's line is "<name> tells you, '...'".
TELL_RECEIVED = re.compile(r"^\s*.+? tells you, '")
# perform_tell, act.comm.c:887: the sender's line is "You tell <name>, '...'".
# It can follow a prompt on the same line ("546H 268M 188V > You tell Mar,
# 'yo?'"). do_gsay (act.comm.c:848) and do_ctell (act.comm.c:1535) are public
# and stay.
TELL_SENT = re.compile(r"(?:^\s*|> ?)You tell (?!the group\b|your clan\b).+?, '")
# do_spec_comm, act.comm.c:1007-1013: whisper and ask are addressed to one
# character, and the room sees only "$n whispers something to $N."
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
# Slurs and sexual violence. ARCHIVE-POLICY: do not republish slurs or sexual
# material. The whole line goes, like a tell, so line numbers still match the
# capture. Each pattern is a whole word: "spic" alone once matched every
# "spice merchant" in the archive.
REMOVED_WORDS = re.compile(
    r"\b(?:nigg(?:a|er)s?|fag(?:got)?s?|retard(?:s|ed)?|dykes?|kikes?|spics?|chinks?"
    r"|wetbacks?|trann(?:y|ies)|beaners?|rap(?:e|ed|es|ing|ist))\b",
    re.IGNORECASE,
)

TELL_MARKER = "[private tell removed]"
MAIL_MARKER = "[mail removed]"
LINE_MARKER = "[line removed]"
COUNTS = ("emails", "ips", "passwords", "tells", "mail", "lines")


def collect_secrets(files: list[RawFile]) -> set[str]:
    """Every password the archive shows, typed at a prompt or sent by
    auto-login. A player used one password on several characters, so each is
    removed wherever it appears, and none is ever written into this script."""
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


def scrub(f: RawFile, secrets: set[str]) -> tuple[str, dict]:
    """The scrubbed capture and its redaction tally. Emails, addresses and
    passwords are replaced where they stand; a private message, a mail line
    or a line with a slur is replaced whole, so the line count never changes."""
    tally = dict.fromkeys(COUNTS, 0)
    secret_res = [re.compile(rf"(?<![A-Za-z0-9]){re.escape(s)}(?![A-Za-z0-9])") for s in secrets]
    out = []
    in_mail = False
    for raw in f.lines:
        if MAIL_HEADER.search(raw):
            in_mail = True
        line = raw
        if in_mail:
            line = MAIL_MARKER
            tally["mail"] += 1
            # The mail block runs to the blank line that closes it.
            if raw.strip() == "":
                in_mail = False
        elif TELL_RECEIVED.match(raw) or SPEC_RECEIVED.match(raw) or TELL_SENT.search(raw) or SPEC_SENT.search(raw):
            line = TELL_MARKER
            tally["tells"] += 1
        elif REMOVED_WORDS.search(raw):
            line = LINE_MARKER
            tally["lines"] += 1
        else:
            line, n = PASSWORD.subn(r"\1[password removed]", line)
            tally["passwords"] += n
            for rx in secret_res:
                line, n = rx.subn("[password removed]", line)
                tally["passwords"] += n
            line, n = EMAIL.subn("[email removed]", line)
            tally["emails"] += n
            line, n = IPV4.subn("[address removed]", line)
            tally["ips"] += n
        out.append(line)
    return "\n".join(out) + "\n", tally


# ----------------------------------------------------------- name discovery

GROUP_ROW = re.compile(r"^\s*\[(?:\d+)H (?:\d+)M (?:\d+)V\] \[[^\]]+\] ([A-Za-z][A-Za-z']*)(?: \(Head of group\))?\s*$")
CHANNEL_ROW = re.compile(
    r"^(?:You|([A-Za-z][A-Za-z']*)) "
    r"(?:tell|tells|holler|hollers|shout|shouts|gossip|gossips|auction|auctions"
    r"|congrat|congrats|newbie|newbies|say|says|ask|asks|tell the group|tells the group"
    r"|tell your clan|tells your clan)\b"
)
ARRIVAL_ROW = re.compile(r"^([A-Za-z][A-Za-z']*) (?:arrives|has arrived|walks in|has left the game|is dead!|clambers to|staggers in)\b")
# do_score, act.informative.c: "This ranks you as <name> <title>".
SCORE_RANK = re.compile(r"^This ranks you as ([A-Za-z][A-Za-z']*)\b")
THIRD_PERSON_VARGUS = re.compile(r"^(?:.*> )?Vargus\b(?!:)")


def load_mob_names() -> set[str]:
    """Names the world file says are mobs: a mob's short description (s) and
    keywords (k). A player name never matches one."""
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


def cast_for(lines: list[str], mobs: set[str]) -> list[tuple[str, int]]:
    """Every player name in the capture, in order of first appearance, with
    the line it first appears on: the group roster, who speaks on a channel,
    who arrives, who the recorder ranks as. Mob names are dropped."""
    seen: list[tuple[str, int]] = []
    names: set[str] = set()
    for i, text in enumerate(lines):
        name = None
        for rx in (GROUP_ROW, CHANNEL_ROW, ARRIVAL_ROW, SCORE_RANK):
            m = rx.match(text)
            if m:
                name = m.group(1)
                break
        if name and name.lower() not in mobs and name not in names:
            names.add(name)
            seen.append((name, i + 1))
    return seen


def recorders_for(lines: list[str]) -> list[str]:
    """Whose screen this was: every name the score command ranks the recorder
    as, most often first. Without a score, the archive is Vargus's, unless the
    capture shows Vargus from outside ("Vargus slashes ...", ssyzyilsolo1031.txt:17),
    in which case the capture does not say whose screen it was."""
    tally: dict[str, int] = {}
    for line in lines:
        m = SCORE_RANK.match(line)
        if m:
            tally[m.group(1)] = tally.get(m.group(1), 0) + 1
    if tally:
        return [name for name, _ in sorted(tally.items(), key=lambda kv: -kv[1])]
    third_person = sum(1 for line in lines if THIRD_PERSON_VARGUS.match(line))
    return [] if third_person >= 3 else ["Vargus"]


# ------------------------------------------------------------------- dates
# The files lost their timestamps before the donation, so a date comes only
# from what the files say. Three witnesses, in order of trust:
#   1. a note the recorder typed at the top ("Date: 10/26/2000",
#      nimoslady-solo-102600.txt:1), or the date on a mail the capture shows
#      (Hatsumi Bounty.txt:2), which the session can only follow;
#   2. the date the recorder put in the filename, month-day-year with the
#      month unpadded ("amber121200" is 12/12/00, "xedsolo90800" 9/08/00);
#   3. the score screen's "You have been playing for N days and M hours"
#      (do_score). It only climbs, and never faster than the clock, so one
#      character's dated logs settle which reading of an ambiguous filename
#      is possible ("11100" is 11/1/00 for Jyan's 2d22h, dagsolo11100.txt:127,
#      but 1/11/01 for Farius, who is past his 12/12/00 time, and the year
#      typed in January can lag), and bracket an undated log between two.

HEADER_DATE = re.compile(r"^Date: (\d{1,2})/(\d{1,2})/(\d{4})\s*$")
MAIL_DATE = re.compile(r"^Date: \w{3} (\w{3}) +(\d{1,2}) [\d:]+ (\d{4})\s*$")
PLAYED = re.compile(r"^You have been playing for (\d+) days? and (\d+) hours?\.")
FILENAME_DIGITS = re.compile(r"(\d{3,6})")
SAME_SESSION = re.compile(r"pov[a-z]$")   # searchforcyral, ...povj, ...povv: one search, three screens
MONTHS = "Jan Feb Mar Apr May Jun Jul Aug Sep Oct Nov Dec".split()
YEARS = (2000, 2001)                       # every dated file falls in these
BRACKET_DAYS = 14                          # widest range worth printing as an estimate


def filename_readings(name: str) -> tuple[str, bool, list[dt.date]]:
    """Every date the filename's number can mean: (digits, year written?, dates)."""
    m = FILENAME_DIGITS.search(name)
    if not m:
        return "", False, []
    digits, out = m.group(1), set()
    for i in (1, 2):
        month = digits[:i]
        for j in (i + 1, i + 2):
            day, year = digits[i:j], digits[j:]
            if month[0] == "0" or not day or year not in ("", "00", "01"):
                continue
            for y in ([2000 + int(year)] if year else YEARS):
                try:
                    out.add(dt.date(y, int(month), int(day)))
                except ValueError:
                    pass
    written = len(digits) >= 5
    if written:   # a January session can carry last year's number
        out |= {d.replace(year=d.year + 1) for d in out if d.month == 1 and d.year + 1 in YEARS}
    return digits, written, sorted(out)


def played_hours(lines: list[str]) -> int | None:
    for line in lines:
        m = PLAYED.match(line)
        if m:
            return int(m.group(1)) * 24 + int(m.group(2))
    return None


def long_date(d: dt.date) -> str:
    return f"{d.day} {d.strftime('%B')} {d.year}"


def date_range(lo: dt.date, hi: dt.date) -> str:
    if lo.year != hi.year:
        return f"Between {long_date(lo)} and {long_date(hi)}"
    if lo.month != hi.month:
        return f"Between {lo.day} {lo.strftime('%B')} and {long_date(hi)}"
    return f"Between {lo.day} and {long_date(hi)}"


def date_files(files: list[RawFile]) -> dict[str, dict]:
    """raw path -> {text, basis, iso?} for every capture the files can date."""
    played, who, readings, out = {}, {}, {}, {}
    for f in files:
        rec = recorders_for(f.lines)
        hours = played_hours(f.lines)
        if len(rec) == 1 and hours is not None and any(SCORE_RANK.match(line) for line in f.lines):
            played[f.rel], who[f.rel] = hours, rec[0]
        readings[f.rel] = filename_readings(f.name)
        for line in f.lines[:5]:
            m = HEADER_DATE.match(line)
            if m:
                d = dt.date(int(m.group(3)), int(m.group(1)), int(m.group(2)))
                out[f.rel] = {"text": long_date(d), "iso": d.isoformat(),
                              "basis": "a note the recorder typed at the top of the file"}
            m = MAIL_DATE.match(line)
            if m and f.rel not in out:
                d = dt.date(int(m.group(3)), MONTHS.index(m.group(1)) + 1, int(m.group(2)))
                out[f.rel] = {"text": f"On or after {long_date(d)}",
                              "basis": "the date on a mail in the capture, which itself is private"}

    def session(rel):
        digits = readings[rel][0]
        return (SAME_SESSION.sub("", re.sub(r"\d+.*$", "", Path(rel).stem.lower())), digits) if digits else rel

    def possible(rel, day):
        """No dated log of the same character contradicts `day`: play time
        runs forward, a day of label slack either side, and never faster
        than the clock."""
        if rel not in played:
            return True
        p = played[rel]
        for r, d in out.items():
            if r == rel or "iso" not in d or r not in played or who[r] != who[rel]:
                continue
            a, q = dt.date.fromisoformat(d["iso"]), played[r]
            gap = (a - day).days
            if (gap < -1 and q > p) or (gap > 1 and q < p) or abs(q - p) > (abs(gap) + 1) * 24:
                return False
        return True

    changed = True
    while changed:
        changed = False
        fits = {rel: [d for d in dates if possible(rel, d)]
                for rel, (_, _, dates) in readings.items() if rel not in out and dates}
        groups: dict = {}
        for rel, days in fits.items():
            groups.setdefault(session(rel), []).append(rel)
        for members in groups.values():
            common = set.intersection(*(set(fits[r]) for r in members))
            if len(common) != 1:
                continue
            (d,) = common
            for rel in members:
                digits, written, dates = readings[rel]
                basis = f"the recorder's filename ({digits})"
                if len(dates) > 1:
                    basis += (f", with the year from {who[rel]}'s play time" if not written and rel in played
                              else f", read against {who[rel]}'s play time" if rel in played else "")
                    if len(members) > 1:
                        basis += ", and the same session's other captures"
                out[rel] = {"text": long_date(d), "iso": d.isoformat(), "basis": basis}
                changed = True

    for rel, p in played.items():
        if rel in out:
            continue
        mine = [(dt.date.fromisoformat(d["iso"]), played[r]) for r, d in out.items()
                if "iso" in d and r in played and who[r] == who[rel]]
        before = [d for d, q in mine if q <= p]
        after = [d for d, q in mine if q >= p]
        if before and after:
            lo, hi = max(before), min(after)
            basis = f"{who[rel]}'s play time against dated logs, an estimate"
            if lo == hi:
                out[rel] = {"text": f"About {long_date(lo)}", "basis": basis}
            elif lo < hi and (hi - lo).days <= BRACKET_DAYS:
                out[rel] = {"text": date_range(lo, hi), "basis": basis}
    return out


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
        self.name = Path(rel).name


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


def slug_and_title(f: RawFile) -> tuple[str, str]:
    if f.rel in WITHHELD_NAMES:
        return WITHHELD_NAMES[f.rel]
    return re.sub(r"[^a-z0-9]+", "-", Path(f.rel).stem.lower()).strip("-"), f.name


def yaml_scalar(value) -> str:
    if isinstance(value, (int, float)):
        return str(value)
    return json.dumps(str(value), ensure_ascii=False)


def render_entry(f: RawFile, slug: str, title: str, recorders: list[str],
                 cast: list[tuple[str, int]], tally: dict, date: dict | None) -> str:
    first_line = dict(cast)
    ordered = [r for r in recorders if r in first_line] + [n for n, _ in cast if n not in recorders]
    lines = [
        "---",
        f"title: {yaml_scalar(title)}",
        f"nameWithheld: {'true' if f.rel in WITHHELD_NAMES else 'false'}",
        f"lines: {f.total}",
        f"log: {yaml_scalar(slug + '.log')}",
        "recordedBy: [" + ", ".join(yaml_scalar(r) for r in recorders) + "]",
        *(["date:", *[f"  {k}: {yaml_scalar(date[k])}" for k in ("text", "iso", "basis") if k in date]] if date else []),
        "redactions:",
        *[f"  {key}: {tally[key]}" for key in COUNTS],
        "cast: [" + ", ".join(yaml_scalar(n) for n in ordered) + "]",
        "appearances:" + ("" if ordered else " {}"),
        *[f"  {yaml_scalar(n)}: {first_line[n]}" for n in ordered],
        "---",
        "",
    ]
    return "\n".join(lines)


# ------------------------------------------------------------------- build

MANIFEST_COLUMNS = ["raw_path", "bytes", "lines", "sha256", "slug", "excluded", *COUNTS]


def build(check_only: bool = False) -> int:
    files = load_files()
    secrets = collect_secrets(files)
    mobs = load_mob_names()
    known = {f.rel for f in files}
    for rel in list(EXCLUDE) + list(WITHHELD_NAMES):
        if rel not in known:
            raise SystemExit(f"vargus-logs: unknown file {rel!r}")

    dates = date_files([f for f in files if f.rel not in EXCLUDE])
    expected: dict[str, str] = {}
    rows = []
    for f in files:
        if f.rel in EXCLUDE:
            rows.append([f.rel, str(len(f.data)), str(f.total), f.sha, "-", EXCLUDE[f.rel], *["0"] * len(COUNTS)])
            continue
        slug, title = slug_and_title(f)
        text, tally = scrub(f, secrets)
        expected[f"{slug}.log"] = text
        expected[f"{slug}.md"] = render_entry(f, slug, title, recorders_for(f.lines), cast_for(f.lines, mobs), tally, dates.get(f.rel))
        rows.append([f.rel, str(len(f.data)), str(f.total), f.sha, slug, "-", *[str(tally[k]) for k in COUNTS]])
    expected["MANIFEST.tsv"] = "\t".join(MANIFEST_COLUMNS) + "\n" + "\n".join("\t".join(r) for r in rows) + "\n"

    present = {p.name for p in CONTENT_DIR.iterdir() if p.suffix in (".log", ".md", ".tsv")}
    if check_only:
        problems = [n for n, body in expected.items()
                    if not (CONTENT_DIR / n).exists() or (CONTENT_DIR / n).read_text(encoding="utf-8") != body]
        problems += sorted(present - set(expected))
        for name in problems:
            print(f"  stale: {name}", file=sys.stderr)
        if problems:
            print(f"vargus-logs: check FAILED, {len(problems)} stale files", file=sys.stderr)
            return 1
        print(f"vargus-logs: check OK, {len(rows) - len(EXCLUDE)} captures reproduce")
        return 0

    for name, body in expected.items():
        (CONTENT_DIR / name).write_text(body, encoding="utf-8")
    for name in present - set(expected):
        (CONTENT_DIR / name).unlink()
    print(f"vargus-logs: {len(rows)} raw files, {len(rows) - len(EXCLUDE)} published, {len(EXCLUDE)} excluded")
    return 0


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("command", choices=["build", "check"])
    args = parser.parse_args()
    return build(check_only=args.command == "check")


if __name__ == "__main__":
    sys.exit(main())
