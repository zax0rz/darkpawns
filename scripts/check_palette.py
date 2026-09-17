#!/usr/bin/env python3
"""Enforce The One Oxblood Rule in the admin console's Tailwind classes.

DESIGN.md allows two creams, two inks, one oxblood, and a single functional
green that means "live". Tailwind ships a few hundred other colours, and any of
them is one keystroke away in a className.

This checks by allowlist rather than by blocklist, deliberately. The earlier
manual sweep enumerated hues by hand and so missed cyan, purple and violet: a
heap meter in `bg-cyan-500` and four categorical badges survived a pass that
believed it had converted everything. A blocklist only catches the colours
somebody already thought of.

Run via `make check-palette`, or as part of `make site-check`.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCAN_DIR = ROOT / "admin-ui" / "src"

# The palette, from website-astro/DESIGN.md.
ALLOWED_TOKENS = {
    "paper",
    "paper-deep",
    "ink",
    "ink-muted",
    "rule",
    "accent",
    "accent-deep",
    "registration",
    "online",
    # Structural, not colour.
    "transparent",
    "current",
    "inherit",
}

# Colour-bearing Tailwind utilities. `decoration` and `outline` included because
# they are as visible as a border.
PREFIXES = (
    "bg",
    "text",
    "border",
    "ring",
    "divide",
    "outline",
    "decoration",
    "from",
    "via",
    "to",
    "fill",
    "stroke",
    "shadow",
    "accent",
    "caret",
)

# A utility that carries a colour: prefix-token, optionally with an opacity
# suffix. Numeric Tailwind shades (bg-cyan-500) and bare names (bg-white) both
# land here and are judged against the allowlist.
UTILITY = re.compile(
    r"\b(" + "|".join(PREFIXES) + r")-([a-z]+(?:-[a-z]+)*(?:-\d{2,3})?)(?:/\d{1,3})?\b"
)

# Utilities whose second half is a size or a keyword, not a colour.
NON_COLOUR_VALUES = {
    "xs", "sm", "md", "lg", "xl", "2xl", "3xl", "4xl", "5xl", "6xl", "7xl",
    "none", "auto", "full", "px", "0", "1", "2", "4", "8",
    "center", "left", "right", "justify", "start", "end",
    "wrap", "nowrap", "balance", "pretty", "clip", "ellipsis",
    "solid", "dashed", "dotted", "double", "hidden", "collapse",
    "y", "x", "t", "b", "l", "r", "se", "so",
    "base", "inner", "opacity",
}


def offenders_in(text: str) -> list[str]:
    found: list[str] = []
    for match in UTILITY.finditer(text):
        prefix, value = match.group(1), match.group(2)
        if value in NON_COLOUR_VALUES or value in ALLOWED_TOKENS:
            continue
        # Arbitrary values (bg-[#fff]) are caught by the bracket, not here.
        if value.isdigit():
            continue
        found.append(f"{prefix}-{value}")
    return found


def main() -> int:
    problems: list[tuple[str, int, str]] = []
    for path in sorted(SCAN_DIR.rglob("*.tsx")) + sorted(SCAN_DIR.rglob("*.ts")):
        for number, line in enumerate(path.read_text(encoding="utf-8").splitlines(), 1):
            if "className" not in line and "'" not in line and '"' not in line:
                continue
            for bad in offenders_in(line):
                problems.append((path.relative_to(ROOT).as_posix(), number, bad))

    if problems:
        print("check-palette: colours outside the design system\n", file=sys.stderr)
        for rel, number, bad in problems:
            print(f"  {rel}:{number}: {bad}", file=sys.stderr)
        print(
            "\nThe palette is two creams, two inks, one oxblood and one green that\n"
            "means live. See The One Oxblood Rule in website-astro/DESIGN.md.",
            file=sys.stderr,
        )
        return 1

    print(f"check-palette: clean ({len(ALLOWED_TOKENS)} allowed tokens)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
