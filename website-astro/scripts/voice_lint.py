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

SCAN_GLOBS = (
    "src/components/**/*.astro",
    "src/layouts/**/*.astro",
    "src/pages/**/*.astro",
    "src/content/**/*.md",
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


def frontmatter_value(lines: list[str], key: str) -> str | None:
    frontmatter, _ = split_frontmatter(lines)
    for number in sorted(frontmatter):
        match = re.match(rf"^{re.escape(key)}:\s*[\"']?([^\"'#]+)", lines[number - 1])
        if match:
            return match.group(1).strip()
    return None


def iter_files() -> list[Path]:
    files: set[Path] = set()
    for pattern in SCAN_GLOBS:
        files.update(SITE.glob(pattern))
    return sorted(path for path in files if path.is_file())


def lint_file(path: Path) -> list[Finding]:
    relative = path.relative_to(SITE).as_posix()
    lines = path.read_text(encoding="utf-8").splitlines()
    frontmatter, body = split_frontmatter(lines)
    text_kind = frontmatter_value(lines, "textKind")
    allowed_lines = frontmatter | body
    if relative.startswith("src/content/help/") or text_kind in PRESERVED_TEXT_KINDS:
        allowed_lines = frontmatter

    findings: list[Finding] = []
    if relative.startswith(PROVENANCE_COLLECTIONS):
        required = ("textKind", "source", "voiceLayer")
        for key in required:
            if frontmatter_value(lines, key) is None:
                findings.append(
                    Finding(relative, 1, 1, "provenance", "error", f"required frontmatter field {key!r} is missing", lines[0] if lines else "")
                )
    for number, line in enumerate(lines, start=1):
        if number not in allowed_lines:
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

        for word in SYNTHETIC_WORDS:
            match = re.search(rf"\b{re.escape(word)}\b", lowered)
            if match:
                findings.append(
                    Finding(relative, number, match.start() + 1, "synthetic-importance", "warning", f"show the evidence instead of calling it {word!r}", line)
                )

        # Three short declarative fragments in a row often produce the fake
        # trailer cadence described in the guide. This is intentionally only
        # a warning because terse technical prose can match it legitimately.
        if number in body:
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
