#!/usr/bin/env python3
"""Render the site header's lockup as the Mudlet dock's header image.

Mudlet cannot show the lockup the way the site does: a package loaded from
darkpawns.xml cannot install fonts, so its labels set DARK / PAWNS in whatever
serif the player has, and a label cannot draw SVG. So the dock shows the
lockup as one picture, rendered here from the header's own drawing and CSS.

The pawn is the five canonical shapes read from Header.astro. The lettering is
DM Serif Display, the font the site loads, laid out the way CSS lays out
`.brand`: pawn 3.2rem tall, a 0.5rem gap, the wordmark at 2rem with
line-height 0.82 and 0.01em letter-spacing, centred against the pawn. It is written at the CSS size and at twice it
(lockup.png, lockup@2x.png), on transparency, with 4x4 supersampled edges;
the dock shows it unscaled, and Qt picks the @2x file on high-density screens.

    python3 scripts/render_mudlet_lockup.py   # rewrite mudlet/lockup*.png

Rerun it when the mark changes. TestLockupPawnIsCanonical fails until you do.
Needs Pillow and fontTools; the font is fetched from Google Fonts on first run.
"""

from __future__ import annotations

import math
import re
import sys
import urllib.request
from pathlib import Path

from fontTools.ttLib import TTFont
from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parents[1]
HEADER = ROOT / "website-astro/src/components/Header.astro"
OUT = ROOT / "mudlet/lockup.png"
OUT_2X = ROOT / "mudlet/lockup@2x.png"
FONT_CACHE = Path.home() / ".cache/darkpawns/DMSerifDisplay-Regular.ttf"
FONT_CSS = "https://fonts.googleapis.com/css2?family=DM+Serif+Display"

INK = (0x1A, 0x16, 0x14, 255)
ACCENT = (0xA8, 0x20, 0x1A, 255)

# Header.astro's CSS, in CSS pixels (1rem = 16px). check_header() fails if
# the header stops saying this, so the two cannot drift apart silently.
REM = 16
PAWN_HEIGHT = 3.2 * REM
GAP = 0.5 * REM
FONT_SIZE = 2 * REM
LINE_HEIGHT = 0.82
LETTER_SPACING = 0.01
HEADER_CSS = [
    r"\.brand \{[^}]*gap: var\(--space-sm\)",
    r"\.pawn \{ height: 3\.2rem; width: auto;",
    r"font-size: 2rem;\s*line-height: 0\.82;\s*letter-spacing: 0\.01em;",
]

SUPER = 4  # supersampling per output pixel
PAD = 1  # transparent margin, in CSS pixels
FONT_RENDER = 16  # size multiple used only to measure the lettering


def check_header(text: str) -> None:
    for pattern in HEADER_CSS:
        if not re.search(pattern, text):
            sys.exit(f"{HEADER}: lockup CSS changed ({pattern!r}); update this script to match")


def read_pawn(text: str) -> tuple[list[float], list[tuple[str, list[float]]]]:
    svg = re.search(r'<svg class="pawn" viewBox="([^"]+)"[^>]*>(.*?)</svg>', text, re.S)
    if not svg:
        sys.exit(f"{HEADER}: no <svg class=\"pawn\">")
    shapes = []
    for tag, raw in re.findall(r"<(circle|rect|polygon)\b([^>]*)>", svg.group(2)):
        attrs = dict(re.findall(r'([a-zA-Z-]+)\s*=\s*"([^"]*)"', raw))
        keys = {"circle": ["cx", "cy", "r"], "rect": ["x", "y", "width", "height"]}.get(tag)
        values = [attrs[k] for k in keys] if keys else re.split(r"[ ,]+", attrs["points"].strip())
        shapes.append((tag, [float(v) for v in values]))
    return [float(v) for v in svg.group(1).split()], shapes


def font_file() -> Path:
    if not FONT_CACHE.exists():
        # Without a browser User-Agent the CSS API serves TrueType.
        with urllib.request.urlopen(FONT_CSS) as resp:  # noqa: S310 - fixed https URL
            css = resp.read().decode()
        url = re.search(r"url\((https://[^)]+\.ttf)\)", css)
        if not url:
            sys.exit("Google Fonts did not return a TrueType URL for DM Serif Display")
        FONT_CACHE.parent.mkdir(parents=True, exist_ok=True)
        with urllib.request.urlopen(url.group(1)) as resp:  # noqa: S310
            FONT_CACHE.write_bytes(resp.read())
    return FONT_CACHE


def main() -> int:
    header = HEADER.read_text(encoding="utf-8")
    check_header(header)
    (vx, vy, vw, vh), shapes = read_pawn(header)
    path = font_file()

    metrics = TTFont(path)
    upem = metrics["head"].unitsPerEm
    ascent = metrics["hhea"].ascent / upem * FONT_SIZE
    descent = -metrics["hhea"].descent / upem * FONT_SIZE
    font = ImageFont.truetype(str(path), round(FONT_SIZE * FONT_RENDER))

    # CSS inline layout: each line box is line-height tall, and the glyphs'
    # ascent+descent sit centred in it (the half-leading is negative here).
    line = LINE_HEIGHT * FONT_SIZE
    baseline = (line - (ascent + descent)) / 2 + ascent
    text_height = 2 * line
    pawn_width = PAWN_HEIGHT * vw / vh
    height = max(text_height, PAWN_HEIGHT)
    pawn_top = (height - PAWN_HEIGHT) / 2  # align-items: center
    text_top = (height - text_height) / 2
    text_left = pawn_width + GAP

    words = [("DARK", INK), ("PAWNS", ACCENT)]
    text_width = max(font.getlength(w) + len(w) * LETTER_SPACING * FONT_SIZE * FONT_RENDER for w, _ in words) / FONT_RENDER
    # Whole CSS pixels plus a pixel of margin, so the 2x image is exactly
    # twice the 1x one and both show at the same size.
    size = (math.ceil(text_left + text_width) + 2 * PAD, math.ceil(height) + 2 * PAD)

    for scale, out_path in ((1, OUT), (2, OUT_2X)):
        k = scale * SUPER  # canvas pixels per CSS pixel
        font = ImageFont.truetype(str(path), round(FONT_SIZE * k))
        spacing = LETTER_SPACING * FONT_SIZE * k
        canvas = Image.new("RGBA", (size[0] * k, size[1] * k), (0, 0, 0, 0))
        draw = ImageDraw.Draw(canvas)

        def px(x: float, y: float, k: float = k) -> tuple[float, float]:
            return (PAD + x) * k, (PAD + y) * k

        pscale = PAWN_HEIGHT / vh
        for tag, v in shapes:
            def at(x: float, y: float) -> tuple[float, float]:
                return px((x - vx) * pscale, pawn_top + (y - vy) * pscale)

            if tag == "circle":
                (x0, y0), (x1, y1) = at(v[0] - v[2], v[1] - v[2]), at(v[0] + v[2], v[1] + v[2])
                draw.ellipse([x0, y0, x1, y1], fill=INK)
            elif tag == "rect":
                (x0, y0), (x1, y1) = at(v[0], v[1]), at(v[0] + v[2], v[1] + v[3])
                draw.rectangle([x0, y0, x1 - 1, y1 - 1], fill=INK)
            else:
                draw.polygon([at(v[i], v[i + 1]) for i in range(0, len(v), 2)], fill=INK)

        for row, (word, colour) in enumerate(words):
            x, base = px(text_left, text_top + row * line + baseline)
            for i, char in enumerate(word):
                # Advance with kerning: where the glyph starts once its pair
                # with the previous glyph is applied, plus CSS letter-spacing.
                advance = font.getlength(word[: i + 1]) - font.getlength(char)
                draw.text((x + advance + i * spacing, base), char, font=font, fill=colour, anchor="ls")

        out = canvas.convert("RGBa").resize((size[0] * scale, size[1] * scale), Image.Resampling.BOX).convert("RGBA")
        out.save(out_path, optimize=True)
        print(f"{out_path.relative_to(ROOT)}: {out.width}x{out.height}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
