#!/usr/bin/env python3
"""Render the Dark Pawns pawn mark to the raster icons the site serves.

The mark is defined once, as SVG, in `website/static/favicon.svg`. The favicon,
the Apple touch icon and the maskable launcher icon are the same drawing at
different sizes, so they are generated from that geometry here rather than kept
as hand-made files that drift apart. Run it after changing favicon.svg:

    python3 website-astro/scripts/generate-icons.py

Outputs, all under website/static/:
    favicon.ico         16/32/48 px, transparent-free oxblood block
    apple-touch-icon.png  180x180 px, full bleed (iOS rounds the corners)
    icon-512.png        512x512 px, maskable (background fills the canvas)
"""
from pathlib import Path

from PIL import Image, ImageDraw

# DESIGN.md palette: the oxblood stamp on cream paper.
OXBLOOD = (0xA8, 0x20, 0x1A, 0xFF)
PAPER = (0xEF, 0xE7, 0xD6, 0xFF)

# The pawn, in favicon.svg's 100x100 viewBox. The group is drawn at
# translate(9 9) scale(0.82); the shapes below are in the group's own space.
GROUP_OFFSET = 9.0
GROUP_SCALE = 0.82
SUPERSAMPLE = 4


def _map(x: float, y: float, size: int) -> tuple[float, float]:
    """A group-space point to a pixel in a `size` px square."""
    unit = size / 100.0
    return ((GROUP_OFFSET + GROUP_SCALE * x) * unit, (GROUP_OFFSET + GROUP_SCALE * y) * unit)


def render(size: int, corner_radius: float) -> Image.Image:
    """The mark at `size` px, drawn 4x and downsampled for clean edges."""
    big = size * SUPERSAMPLE
    image = Image.new("RGBA", (big, big), (0, 0, 0, 0))
    draw = ImageDraw.Draw(image)

    unit = big / 100.0
    draw.rounded_rectangle(
        [0, 0, big - 1, big - 1],
        radius=corner_radius * unit,
        fill=OXBLOOD,
    )

    draw.ellipse([_map(50 - 12, 23 - 12, big), _map(50 + 12, 23 + 12, big)], fill=PAPER)
    draw.rectangle([_map(37, 38, big), _map(37 + 26, 38 + 5, big)], fill=PAPER)
    draw.polygon([_map(43, 46, big), _map(57, 46, big), _map(62, 75, big), _map(38, 75, big)], fill=PAPER)
    draw.rectangle([_map(31, 77, big), _map(31 + 38, 77 + 6, big)], fill=PAPER)
    draw.rectangle([_map(26, 85, big), _map(26 + 48, 85 + 7, big)], fill=PAPER)

    return image.resize((size, size), Image.LANCZOS)


def main() -> None:
    static = Path(__file__).resolve().parents[2] / "website" / "static"

    render(256, corner_radius=4).save(
        static / "favicon.ico",
        format="ICO",
        sizes=[(16, 16), (32, 32), (48, 48)],
    )
    render(180, corner_radius=0).save(static / "apple-touch-icon.png", format="PNG")
    render(512, corner_radius=0).save(static / "icon-512.png", format="PNG")

    for name in ("favicon.ico", "apple-touch-icon.png", "icon-512.png"):
        print(f"wrote website/static/{name}")


if __name__ == "__main__":
    main()
