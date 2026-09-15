"""Make the card a link to the Bridge Talk site unfurls into.

Discord, Slack and the social sites show the picture a page names in og:image when
its address is posted. This one carries the application's icon large on the site's
dark ground, with rings of sound leaving it, so the card reads as a voice rather
than as a badge.

Nothing on the card is written here. The words are read from the page's own og:title,
og:description and og:url; the colours from the dark block of the site's stylesheet;
the picture from the icon's master. So the card cannot say something the site does
not, nor wear a colour the site does not.

Run it when the icon, the page's wording or the site's palette changes:

    python tools/gensocialcard.py

It is not part of the build. The card is committed beside the page it describes.
"""

from __future__ import annotations

import html
import math
import os
import pathlib
import random
import re
import sys

try:
    from PIL import Image, ImageDraw, ImageFilter, ImageFont
except ImportError:  # pragma: no cover - a plain message beats a traceback
    sys.exit("Pillow is required: python -m pip install pillow")

REPO = pathlib.Path(__file__).resolve().parent.parent
ICON = REPO / "assets" / "application-icon.png"
PAGE = REPO / "docs" / "index.html"
STYLES = REPO / "docs" / "styles.css"
CARD = REPO / "docs" / "social-card.png"
FONTS = pathlib.Path(os.environ.get("WINDIR", r"C:\Windows")) / "Fonts"

# The faces the site's stylesheet names: its monospaced headings and its interface text.
MONO_BOLD = FONTS / "consolab.ttf"
MONO = FONTS / "consola.ttf"
SANS = FONTS / "segoeui.ttf"

# WIDTH and HEIGHT are the 1.91 to 1 size the readers show a large card at; the page
# declares the same two numbers, which main() checks rather than trusts.
WIDTH = 1200
HEIGHT = 630

# SCALE draws the card at twice its size then shrinks it once, which smooths every edge
# the drawing calls would otherwise leave jagged.
SCALE = 2

# MARGIN keeps everything clear of the edge a reader may crop.
MARGIN = 64

# ICON_SIDE is the icon's longer side: most of the card's height, so the icon leads.
ICON_SIDE = 470
ICON_CENTRE_X = MARGIN + ICON_SIDE // 2
ICON_CENTRE_Y = HEIGHT // 2

# GUTTER separates the icon from the words beside it.
GUTTER = 36
TEXT_LEFT = ICON_CENTRE_X + ICON_SIDE // 2 + GUTTER
TEXT_WIDTH = WIDTH - MARGIN - TEXT_LEFT

# The largest size each line may take; each shrinks until it fits the words' column.
TITLE_LARGEST = 92
STRAPLINE_LARGEST = 22
DESCRIPTION_LARGEST = 30
FOOTER_LARGEST = 20

# STRAPLINE_SPACING opens the capitals out, as the sibling cards do.
STRAPLINE_SPACING = 5

# DESCRIPTION_LINES is as many lines as the description may wrap onto before it shrinks.
DESCRIPTION_LINES = 4
# LEADING is a line's height as a multiple of its type size.
LEADING = 1.3

# The vertical rhythm of the words' column, top to bottom.
TITLE_TOP = 118
AFTER_TITLE = 18
AFTER_STRAPLINE = 34

# FOOTER_FACTS follow the address: the one platform built, the price and the licence,
# GPL-3.0 in LICENSE.
FOOTER_FACTS = ("Windows", "free", "open source")
FACT_SEPARATOR = " \u00b7 "

# RULE_HEIGHT is the orange line along the top the sibling cards carry.
RULE_HEIGHT = 4

# The rings of sound spreading from behind the icon: how many, where the first sits,
# how far apart, how thick, how wide an arc each sweeps in degrees and how strong the
# nearest is. Each ring further out is fainter.
RING_COUNT = 9
RING_FIRST = 250
RING_STEP = 62
RING_WIDTH = 5
RING_SWEEP = 110
RING_ALPHA = 150
RING_BLUR = 6

# The glow behind the icon: its radius, strength and softness.
GLOW_RADIUS = 300
GLOW_ALPHA = 70
GLOW_BLUR = 90

# The title's own glow, as the site's headings carry one.
TITLE_GLOW_ALPHA = 150
TITLE_GLOW_BLUR = 14

# The stars: how many, a fixed seed so two runs write the same card, their largest
# radius and the range of their strength.
STAR_COUNT = 170
STAR_SEED = 1
STAR_LARGEST = 2
STAR_FAINTEST = 30
STAR_BRIGHTEST = 150

OPAQUE = 255
TRANSPARENT = (0, 0, 0, 0)

# What the page and the stylesheet are read with.
TITLE_SEPARATOR = ": "
CSS_COMMENT = re.compile(r"/\*.*?\*/", re.DOTALL)
DARK_BLOCK = re.compile(
    r"@media \(prefers-color-scheme: dark\)\s*\{\s*:root\s*\{(.*?)\}", re.DOTALL
)
CSS_HEX = re.compile(r"--([\w-]+)\s*:\s*(#[0-9a-fA-F]{6})\s*;")
NEEDED_TOKENS = ("surface", "text", "muted", "accent")
URL_SCHEME = re.compile(r"^https?://")

Colour = tuple[int, int, int]


def px(length: float) -> int:
    """Return a length on the card as pixels on the canvas it is drawn at."""
    return round(length * SCALE)


def hex_colour(value: str) -> Colour:
    """Read a #rrggbb value."""
    return (int(value[1:3], 16), int(value[3:5], 16), int(value[5:7], 16))


def dark_palette() -> dict[str, Colour]:
    """Read the dark theme's colour tokens from the site's stylesheet."""
    text = CSS_COMMENT.sub("", STYLES.read_text(encoding="utf-8"))
    block = DARK_BLOCK.search(text)
    if block is None:
        sys.exit(f"no dark colour block in {STYLES}")
    tokens = dict(CSS_HEX.findall(block.group(1)))
    missing = [name for name in NEEDED_TOKENS if name not in tokens]
    if missing:
        sys.exit(f"{STYLES} declares no dark {', '.join(missing)}")
    return {name: hex_colour(tokens[name]) for name in NEEDED_TOKENS}


def meta(page: str, name: str) -> str:
    """Read one Open Graph property's content from the page."""
    found = re.search(rf'<meta property="{re.escape(name)}" content="([^"]*)"', page)
    if found is None:
        sys.exit(f"{PAGE} declares no {name}")
    return html.unescape(found.group(1))


def spaced_length(font: ImageFont.FreeTypeFont, text: str, spacing: int) -> float:
    """Measure text drawn with extra space between its letters."""
    return sum(font.getlength(letter) for letter in text) + spacing * (len(text) - 1)


def fitting(path: pathlib.Path, largest: int, fits) -> ImageFont.FreeTypeFont:
    """Return the largest size of a face, no larger than largest, that fits."""
    for size in range(px(largest), 0, -1):
        font = ImageFont.truetype(str(path), size)
        if fits(font):
            return font
    sys.exit(f"nothing fits at any size of {path.name}")


def wrapped(font: ImageFont.FreeTypeFont, text: str, width: int) -> list[str]:
    """Break text into lines no wider than width, at spaces."""
    lines: list[str] = []
    for word in text.split():
        trial = f"{lines[-1]} {word}" if lines else word
        if lines and font.getlength(trial) <= width:
            lines[-1] = trial
        else:
            lines.append(word)
    return lines


def draw_spaced(draw, xy, text, font, fill, spacing) -> None:
    """Draw text with extra space between its letters."""
    x, y = xy
    for letter in text:
        draw.text((x, y), letter, font=font, fill=fill)
        x += font.getlength(letter) + spacing


def glow(canvas: Image.Image, accent: Colour) -> None:
    """Lay a soft orange light behind where the icon sits."""
    layer = Image.new("RGBA", canvas.size, TRANSPARENT)
    radius = px(GLOW_RADIUS)
    centre = (px(ICON_CENTRE_X), px(ICON_CENTRE_Y))
    ImageDraw.Draw(layer).ellipse(
        (
            centre[0] - radius,
            centre[1] - radius,
            centre[0] + radius,
            centre[1] + radius,
        ),
        fill=accent + (GLOW_ALPHA,),
    )
    canvas.alpha_composite(layer.filter(ImageFilter.GaussianBlur(px(GLOW_BLUR))))


def stars(canvas: Image.Image, text: Colour) -> None:
    """Scatter faint stars across the ground, the same ones every run."""
    layer = Image.new("RGBA", canvas.size, TRANSPARENT)
    draw = ImageDraw.Draw(layer)
    chance = random.Random(STAR_SEED)
    for _ in range(STAR_COUNT):
        x = chance.uniform(0, canvas.width)
        y = chance.uniform(0, canvas.height)
        radius = px(chance.uniform(STAR_LARGEST / SCALE, STAR_LARGEST))
        alpha = chance.randint(STAR_FAINTEST, STAR_BRIGHTEST)
        draw.ellipse(
            (x - radius, y - radius, x + radius, y + radius), fill=text + (alpha,)
        )
    canvas.alpha_composite(layer)


def rings(canvas: Image.Image, accent: Colour) -> None:
    """Draw rings of sound spreading right from behind the icon, fading as they go."""
    layer = Image.new("RGBA", canvas.size, TRANSPARENT)
    draw = ImageDraw.Draw(layer)
    centre = (px(ICON_CENTRE_X), px(ICON_CENTRE_Y))
    for index in range(RING_COUNT):
        radius = px(RING_FIRST + index * RING_STEP)
        alpha = round(RING_ALPHA * (1 - index / RING_COUNT))
        draw.arc(
            (
                centre[0] - radius,
                centre[1] - radius,
                centre[0] + radius,
                centre[1] + radius,
            ),
            start=-RING_SWEEP / 2,
            end=RING_SWEEP / 2,
            fill=accent + (alpha,),
            width=px(RING_WIDTH),
        )
    canvas.alpha_composite(layer.filter(ImageFilter.GaussianBlur(px(RING_BLUR))))
    canvas.alpha_composite(layer)


def rule(canvas: Image.Image, accent: Colour) -> None:
    """Draw the orange line along the top, strongest at its middle."""
    draw = ImageDraw.Draw(canvas)
    for x in range(canvas.width):
        alpha = round(OPAQUE * math.sin(math.pi * x / canvas.width))
        blended = tuple(
            round(ground + (colour - ground) * alpha / OPAQUE)
            for ground, colour in zip(canvas.getpixel((x, 0))[:3], accent)
        )
        draw.line((x, 0, x, px(RULE_HEIGHT) - 1), fill=blended + (OPAQUE,))


def icon(canvas: Image.Image) -> None:
    """Place the icon large on the left, shrunk from its master and never enlarged."""
    art = Image.open(ICON).convert("RGBA")
    box = art.getbbox()
    if box is not None:
        art = art.crop(box)
    side = px(ICON_SIDE)
    if max(art.size) < side:
        sys.exit(f"{ICON.name} is smaller than the card needs; it is never enlarged")
    scale = side / max(art.size)
    art = art.resize(
        (round(art.width * scale), round(art.height * scale)), Image.LANCZOS
    )
    canvas.alpha_composite(
        art,
        (px(ICON_CENTRE_X) - art.width // 2, px(ICON_CENTRE_Y) - art.height // 2),
    )


def words(canvas: Image.Image, page: str, palette: dict[str, Colour]) -> None:
    """Write the name, the strapline, the description and the footer."""
    width = px(TEXT_WIDTH)
    left = px(TEXT_LEFT)
    name, separator, strapline = meta(page, "og:title").partition(TITLE_SEPARATOR)
    if not separator:
        sys.exit(
            f"og:title holds no {TITLE_SEPARATOR!r} between the name and strapline"
        )
    strapline = strapline.upper()
    description = meta(page, "og:description")
    address = URL_SCHEME.sub("", meta(page, "og:url")).rstrip("/")
    facts = FACT_SEPARATOR + FACT_SEPARATOR.join(FOOTER_FACTS)

    title_font = fitting(MONO_BOLD, TITLE_LARGEST, lambda f: f.getlength(name) <= width)
    strap_font = fitting(
        MONO,
        STRAPLINE_LARGEST,
        lambda f: spaced_length(f, strapline, px(STRAPLINE_SPACING)) <= width,
    )
    body_font = fitting(
        SANS,
        DESCRIPTION_LARGEST,
        lambda f: len(wrapped(f, description, width)) <= DESCRIPTION_LINES,
    )
    foot_font = fitting(
        MONO, FOOTER_LARGEST, lambda f: f.getlength(address + facts) <= width
    )

    y = px(TITLE_TOP)
    halo = Image.new("RGBA", canvas.size, TRANSPARENT)
    ImageDraw.Draw(halo).text(
        (left, y), name, font=title_font, fill=palette["accent"] + (TITLE_GLOW_ALPHA,)
    )
    canvas.alpha_composite(halo.filter(ImageFilter.GaussianBlur(px(TITLE_GLOW_BLUR))))

    draw = ImageDraw.Draw(canvas)
    draw.text((left, y), name, font=title_font, fill=palette["accent"])
    y += title_font.getbbox(name)[3] + px(AFTER_TITLE)

    draw_spaced(
        draw, (left, y), strapline, strap_font, palette["muted"], px(STRAPLINE_SPACING)
    )
    y += strap_font.getbbox(strapline)[3] + px(AFTER_STRAPLINE)

    line_height = round(body_font.size * LEADING)
    for line in wrapped(body_font, description, width):
        draw.text((left, y), line, font=body_font, fill=palette["text"])
        y += line_height

    foot_y = px(HEIGHT - MARGIN) - foot_font.getbbox(address)[3]
    draw.text((left, foot_y), address, font=foot_font, fill=palette["accent"])
    draw.text(
        (left + foot_font.getlength(address), foot_y),
        facts,
        font=foot_font,
        fill=palette["muted"],
    )


def main() -> int:
    page = PAGE.read_text(encoding="utf-8")
    declared = (meta(page, "og:image:width"), meta(page, "og:image:height"))
    if declared != (str(WIDTH), str(HEIGHT)):
        sys.exit(f"the page declares the card as {declared}, not ({WIDTH}, {HEIGHT})")
    if not meta(page, "og:image").endswith("/" + CARD.name):
        sys.exit(f"the page's og:image does not name {CARD.name}")

    palette = dark_palette()
    canvas = Image.new("RGBA", (px(WIDTH), px(HEIGHT)), palette["surface"] + (OPAQUE,))
    glow(canvas, palette["accent"])
    stars(canvas, palette["text"])
    rings(canvas, palette["accent"])
    rule(canvas, palette["accent"])
    icon(canvas)
    words(canvas, page, palette)

    card = canvas.resize((WIDTH, HEIGHT), Image.LANCZOS).convert("RGB")
    card.save(CARD, "PNG", optimize=True)
    where = CARD.relative_to(REPO).as_posix()
    print(f"{where}: {WIDTH}x{HEIGHT}, {CARD.stat().st_size:,} bytes")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
