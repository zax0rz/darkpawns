#!/usr/bin/env python3
"""Lint new Dark Pawns website copy against the brand voice guide.

Hard rules are deterministic and block publication. Heuristics only warn and
always require human judgment. Existing hard-rule debt is recorded in a
baseline so a new violation cannot hide behind old copy.
"""

from __future__ import annotations

import argparse
import collections
import dataclasses
import hashlib
import json
import re
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[2]
SITE = ROOT / "website-astro"
BASELINE = SITE / "voice-lint-baseline.json"

# (root, glob) pairs, relative to the repository. The console and the
# self-hosted front door carry user-facing copy too, and nothing was reading
# them: every piece of overwrought chrome removed from the admin panel on
# 2026-09-16 was invisible to this linter because it only ever looked at
# website-astro.
SCAN_ROOTS = (
    ("website-astro", "src/components/**/*.astro"),
    ("website-astro", "src/layouts/**/*.astro"),
    ("website-astro", "src/pages/**/*.astro"),
    ("website-astro", "src/content/**/*.md"),
    ("admin-ui", "src/**/*.tsx"),
    ("admin-ui", "index.html"),
    ("web/public", "*.html"),
)

PROVENANCE_COLLECTIONS = ("src/content/archive/", "src/content/blog/")
PRESERVED_TEXT_KINDS = {"verbatim", "transcription", "edited-excerpt"}

BANNED_PHRASES = (
    "a labor of love",
    "a living, breathing world",
    "a rich tapestry",
    "a testament to",
    "at its core",
    "delve into",
    "embark on",
    "from humble beginnings",
    "more than a game",
    "seamlessly",
    "whether you are a veteran or a newcomer",
    "where tradition meets",
)

# DESIGN.md reserves the mythic register for game content: lore, rooms, classes.
# The interface's own chrome stays factual and dry. That rule existed only as
# prose, and nothing enforced it, which is how the admin console came to greet
# people with "Mythic Administrative Console", ask for "Operator Credentials"
# under a "PASSPHRASE" label, and report "TRANSMITTING ENCRYPTED TELEMETRY"
# while signing in. Of the six strings removed on 2026-09-16, five broke no
# encoded rule; only an em dash in the page title was catchable.
#
# These are drawn from that drift rather than imagined. A warning, not an error:
# any of them can be right in game content, and the check cannot tell which
# surface it is reading. If one is legitimate, baseline it with a reason.
CHROME_REGISTER = (
    "mythic",
    "cognitive",
    "telemetry",
    "autonomous entities",
    "establish link",
    "operator credentials",
    "transmitting",
    "encrypted",
)

SYNTHETIC_WORDS = (
    "beloved",
    "curated",
    "deeply significant",
    "dynamic",
    "elevated",
    "groundbreaking",
    "iconic",
    "immersive",
    "remarkable",
    "robust",
    "timeless",
    "vibrant",
)


@dataclasses.dataclass(frozen=True)
class Finding:
    path: str
    line: int
    column: int
    rule: str
    severity: str
    message: str
    excerpt: str

    @property
    def fingerprint(self) -> str:
        normalized = re.sub(r"\s+", " ", self.excerpt.strip().lower())
        digest = hashlib.sha256(normalized.encode()).hexdigest()[:16]
        return f"{self.path}|{self.rule}|{digest}"



# Files that are code end to end. Skipping <style>/<script> blocks buys nothing
# here, so instead of reading whole lines these are reduced to the copy a reader
# actually sees: JSX text nodes and string literals that read as prose.
CODE_SUFFIXES = {".tsx", ".ts", ".jsx", ".js"}
MARKUP_SUFFIXES = {".html"}

_JSX_TEXT = re.compile(r">([^<>{}]+)<")
_STRING = re.compile(r"""(['"])([^'"\n]{4,200})\1""")
_TAG = re.compile(r"<[^>]+>")
# A class attribute is never prose. Trying to tell "bg-paper" from "self-hosted"
# lexically is a losing game; where the string sits says it outright.
_CLASS_ATTR = re.compile(r"""class(?:Name)?\s*=\s*(?:(['"])[^'"]*\1|\{[^{}]*\})""")
# No hyphens: "border-t", "bg-paper" and "w-1.5" are utilities, not words, and
# a class list assigned to a variable is not inside a class attribute to strip.
# Prose survives the stricter test because a genuine compound is rare enough to
# stay a minority of the line.
_WORD = re.compile(r"^[A-Za-z][A-Za-z'’]*$")


def reads_as_prose(text: str) -> bool:
    """True when a person would read this as words rather than as data.

    A word count alone is not enough. SVG path data ("M4.5 6.5 L6.5 8"),
    Tailwind class lists ("w-1.5 h-1.5 border-t border-l") and URLs all clear a
    three-token bar, and then their dots split into short sentences and trip the
    cadence heuristic. Prose is mostly plain words; these are mostly not.
    """
    tokens = text.split()
    if len(tokens) < 3:
        return False
    words = [t for t in tokens if _WORD.match(t.strip(".,:;!?()"))]
    if len(words) < 3:
        return False
    return len(words) / len(tokens) >= 0.6


def readable_text(line: str, suffix: str) -> str:
    """The copy a reader actually sees on this line, or "" if there is none."""
    if suffix in MARKUP_SUFFIXES:
        stripped = _TAG.sub(" ", _CLASS_ATTR.sub(" ", line))
        return stripped.strip() if reads_as_prose(stripped) else ""

    line = _CLASS_ATTR.sub(" ", line)
    parts: list[str] = []
    for match in _JSX_TEXT.finditer(line):
        text = match.group(1).strip()
        if reads_as_prose(text):
            parts.append(text)
    for _, text in _STRING.findall(line):
        text = text.strip()
        if reads_as_prose(text):
            parts.append(text)
    return "  ".join(parts)


def split_frontmatter(lines: list[str]) -> tuple[set[int], set[int]]:
    """Return line numbers belonging to frontmatter and body."""
    all_lines = set(range(1, len(lines) + 1))
    if not lines or lines[0].strip() != "---":
        return set(), all_lines
    for index, line in enumerate(lines[1:], start=2):
        if line.strip() == "---":
            frontmatter = set(range(1, index + 1))
            return frontmatter, all_lines - frontmatter
    return set(), all_lines


def code_region_lines(lines: list[str]) -> set[int]:
    """Line numbers inside <style> or <script> elements.

    The prose heuristics read a line as a sentence. CSS selector chains and JS
    property access split on '.' into short fragments, so `.status.online` and
    `response.ok` register as three short sentences and trip trailer-rhythm
    forever. Component files under src/components and src/pages are code with
    prose in them, not prose with code in it.

    Hard rules still run in here: a <script> can hold user-facing strings, and a
    banned dash in one of those ships to a reader like any other copy.
    """
    inside: set[int] = set()
    open_tag: str | None = None
    for number, line in enumerate(lines, start=1):
        if open_tag is None:
            match = re.search(r"<\s*(style|script)\b[^>]*>", line, re.IGNORECASE)
            if match:
                open_tag = match.group(1).lower()
                inside.add(number)
                if re.search(rf"</\s*{open_tag}\s*>", line[match.end():], re.IGNORECASE):
                    open_tag = None
        else:
            inside.add(number)
            if re.search(rf"</\s*{open_tag}\s*>", line, re.IGNORECASE):
                open_tag = None
    return inside


def frontmatter_value(lines: list[str], key: str) -> str | None:
    frontmatter, _ = split_frontmatter(lines)
    for number in sorted(frontmatter):
        match = re.match(rf"^{re.escape(key)}:\s*[\"']?([^\"'#]+)", lines[number - 1])
        if match:
            return match.group(1).strip()
    return None


def iter_files() -> list[Path]:
    files: set[Path] = set()
    for root, pattern in SCAN_ROOTS:
        files.update((ROOT / root).glob(pattern))
    return sorted(path for path in files if path.is_file())


def display_path(path: Path) -> str:
    """Path as reported in findings and fingerprints, relative to the repo.

    Falls back to SITE for tests, which lint a file in a temporary directory.
    """
    resolved = path.resolve()
    for base in (ROOT, SITE):
        try:
            return resolved.relative_to(base).as_posix()
        except ValueError:
            continue
    return resolved.name


def lint_file(path: Path) -> list[Finding]:
    relative = display_path(path)
    lines = path.read_text(encoding="utf-8").splitlines()
    frontmatter, body = split_frontmatter(lines)
    text_kind = frontmatter_value(lines, "textKind")
    allowed_lines = frontmatter | body
    if "src/content/help/" in relative or text_kind in PRESERVED_TEXT_KINDS:
        allowed_lines = frontmatter
    # Heuristics are prose-shaped and misfire on code; hard rules are not.
    prose_lines = allowed_lines - code_region_lines(lines)
    extracts_prose = path.suffix in CODE_SUFFIXES | MARKUP_SUFFIXES

    findings: list[Finding] = []
    if any(c in relative for c in PROVENANCE_COLLECTIONS):
        required = ("textKind", "source", "voiceLayer")
        for key in required:
            if frontmatter_value(lines, key) is None:
                findings.append(
                    Finding(relative, 1, 1, "provenance", "error", f"required frontmatter field {key!r} is missing", lines[0] if lines else "")
                )
    for number, line in enumerate(lines, start=1):
        if number not in allowed_lines:
            continue

        if extracts_prose:
            line = readable_text(line, path.suffix)
            if not line:
                continue

        for character, name in (("—", "em dash"), ("–", "en dash")):
            start = 0
            while (column := line.find(character, start)) >= 0:
                findings.append(
                    Finding(relative, number, column + 1, "dash-ban", "error", f"{name} is banned in new output copy; restructure the sentence", line)
                )
                start = column + 1

        lowered = line.lower()
        for phrase in BANNED_PHRASES:
            column = lowered.find(phrase)
            if column >= 0:
                findings.append(
                    Finding(relative, number, column + 1, "launch-copy", "error", f"generic launch phrase: {phrase!r}", line)
                )

        for phrase in CHROME_REGISTER:
            # Body copy only. Frontmatter is schema, and `voiceLayer:
            # mythic-admin` is a field value naming the register, not a surface
            # written in it.
            if number not in prose_lines or number not in body:
                break
            column = lowered.find(phrase)
            if column >= 0:
                findings.append(
                    Finding(relative, number, column + 1, "chrome-register", "warning", f"{phrase!r} is the game's voice, not the interface's; chrome stays factual", line)
                )

        for word in SYNTHETIC_WORDS:
            if number not in prose_lines:
                break
            match = re.search(rf"\b{re.escape(word)}\b", lowered)
            if match:
                findings.append(
                    Finding(relative, number, match.start() + 1, "synthetic-importance", "warning", f"show the evidence instead of calling it {word!r}", line)
                )

        # Three short declarative fragments in a row often produce the fake
        # trailer cadence described in the guide. This is intentionally only
        # a warning because terse technical prose can match it legitimately.
        if number in body and number in prose_lines:
            plain = re.sub(r"[`*_>#\[\]()]", "", line).strip()
            sentences = [part.strip() for part in re.split(r"[.!?]+", plain) if part.strip()]
            if len(sentences) >= 3 and all(len(part.split()) <= 7 for part in sentences[:3]):
                findings.append(
                    Finding(relative, number, 1, "trailer-rhythm", "warning", "three short sentences may be manufacturing a mic-drop cadence", line)
                )

    return findings


def load_baseline() -> collections.Counter[str]:
    if not BASELINE.exists():
        return collections.Counter()
    data = json.loads(BASELINE.read_text(encoding="utf-8"))
    return collections.Counter(data.get("findings", {}))


def write_baseline(findings: list[Finding]) -> None:
    known = collections.Counter(f.fingerprint for f in findings)
    payload = {
        "description": "Known voice-lint debt. New errors fail and new warnings remain visible.",
        "findings": dict(sorted(known.items())),
    }
    BASELINE.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--write-baseline", action="store_true", help="record current hard-rule debt")
    parser.add_argument("--quiet-warnings", action="store_true", help="hide non-blocking editorial warnings")
    args = parser.parse_args()

    findings = [finding for path in iter_files() for finding in lint_file(path)]
    if args.write_baseline:
        write_baseline(findings)
        print(f"Wrote {BASELINE.relative_to(ROOT)}")
        return 0

    remaining = load_baseline()
    new_errors: list[Finding] = []
    warnings: list[Finding] = []
    for finding in findings:
        if remaining[finding.fingerprint] > 0:
            remaining[finding.fingerprint] -= 1
        elif finding.severity == "warning":
            warnings.append(finding)
        else:
            new_errors.append(finding)

    stale_baseline = sum(remaining.values())

    shown = new_errors + ([] if args.quiet_warnings else warnings)
    for finding in shown:
        print(
            f"{finding.path}:{finding.line}:{finding.column}: "
            f"{finding.severity}: [{finding.rule}] {finding.message}"
        )

    if new_errors:
        print(f"\nvoice-lint: {len(new_errors)} new error(s), {len(warnings)} warning(s)")
        return 1
    if stale_baseline:
        print(
            f"voice-lint: baseline has {stale_baseline} stale finding(s); "
            "remove the violations, then regenerate the baseline"
        )
        return 1
    print(f"voice-lint: clean ({len(warnings)} editorial warning(s))")
    return 0


if __name__ == "__main__":
    sys.exit(main())
