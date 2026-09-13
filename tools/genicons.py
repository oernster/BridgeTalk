"""Generate the icons from the master artwork in assets/.

Two kinds come out of it.

The nav-band icons: the masters are around 1300 pixels square and a couple of
megabytes each, which is right for artwork and wrong for a band that draws them
at seventy. Each is trimmed to its own content then centred on a square canvas,
so every icon carries the same optical weight, then written small enough to embed.

The muted icon is the exception: it has no master. It is DERIVED here by laying
muteslash.png over unmute.png, so the two states of the speaker cannot drift
apart. Every pixel the slash does not cover is the sounding artwork's own pixel.
Two hand-drawn masters were tried first and the speaker came out a different size
in each, which read as the icon jumping when the button toggled.

The application icon: assets/application-icon.png becomes a multi-size Windows
.ico beside it. That one file is the whole identity. It is the executable's icon,
which is in turn where the taskbar button, the tray icon and both shortcuts take
theirs, because each of those reads the icon out of the binary rather than
carrying a copy of its own.

Run it when a master changes:

    python tools/genicons.py

It is not part of the build. The output is committed, so a clone needs neither
Python nor Pillow to build the application.
"""

from __future__ import annotations

import pathlib
import sys

try:
    from PIL import Image
except ImportError:  # pragma: no cover - a plain message beats a traceback
    sys.exit("Pillow is required: python -m pip install pillow")

# SIZE is three times the largest size the band draws an icon at, so the artwork
# stays crisp on a high-density display without carrying detail nothing shows.
SIZE = 208

# PAD keeps the trimmed artwork off the edge of its square, so an icon with a
# glow does not look clipped against its neighbour.
PAD = 2

# APP_MASTER is the application's own identity rather than a band icon, so it is
# left out of the set the front end embeds.
APP_MASTER = "application-icon.png"

# SOUNDING_MASTER is the speaker artwork and SLASH_MASTER the bar laid over it to
# make MUTED_ICON. The slash is an overlay rather than an icon in its own right,
# so it is left out of the set the front end embeds as well.
SOUNDING_MASTER = "unmute.png"
SLASH_MASTER = "muteslash.png"
MUTED_ICON = "mute.png"

# ICO_SIZES are the sizes Windows chooses between: the small tray and menu sizes,
# the taskbar and shortcut sizes, then the large one Explorer uses in its biggest
# view. Leaving one out makes Windows scale a neighbour, which looks soft.
ICO_SIZES = [(16, 16), (24, 24), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)]

# HEADER_SIZE is the setup window's header mark and SETUP_ICON_SIZE its theme
# button. Each is roughly two and a half times the size it is drawn at, for the same
# reason the band icons are: crisp on a high-density display without carrying detail
# nothing shows. That page has no bundler, so it loads each file as it finds it;
# shipping the master there would put a megabyte and a half behind one badge.
HEADER_SIZE = 256
SETUP_ICON_SIZE = 128

REPO = pathlib.Path(__file__).resolve().parent.parent
MASTERS = REPO / "assets"
OUTPUT = REPO / "frontend" / "src" / "assets" / "icons"
SETUP = REPO / "installer" / "frontend" / "dist"
HEADER = SETUP / "icon.png"

# SETUP_ICONS are the band icons the setup window also needs. Its page has no
# bundler, so each is copied in at a size it can load as it finds it.
SETUP_ICONS = ("light-mode.png", "dark-mode.png")


def trimmed(master: pathlib.Path) -> Image.Image:
    """Open a master and crop away the transparent margin around its artwork."""
    image = Image.open(master).convert("RGBA")
    box = image.getbbox()
    return image.crop(box) if box is not None else image


def slashed(sounding: pathlib.Path, slash: pathlib.Path) -> Image.Image:
    """Return the sounding artwork with the slash laid over it.

    Nothing underneath is touched, so the muted state is the sounding state plus a
    bar rather than a second drawing of the same object. The bar is fitted inside
    the artwork's own bounding box rather than the canvas, which keeps it from
    reaching past the speaker: both states then share a canvas AND a bounding box,
    so the trim in render_image treats them alike without any nudging.
    """
    base = Image.open(sounding).convert("RGBA")
    bar = Image.open(slash).convert("RGBA")
    box = bar.getbbox()
    if box is not None:
        bar = bar.crop(box)
    left, top, right, bottom = base.getbbox() or (0, 0, base.width, base.height)
    width, height = right - left, bottom - top
    fit = min(width / bar.width, height / bar.height)
    bar = bar.resize(
        (max(1, round(bar.width * fit)), max(1, round(bar.height * fit))),
        Image.LANCZOS,
    )
    muted = base.copy()
    muted.alpha_composite(
        bar,
        (left + (width - bar.width) // 2, top + (height - bar.height) // 2),
    )
    return muted


def render_image(image: Image.Image, target: pathlib.Path) -> int:
    """Write one downscaled band icon from artwork in memory; return its byte size."""
    box = image.getbbox()
    art = image.crop(box) if box is not None else image
    inner = SIZE - 2 * PAD
    scale = min(inner / art.width, inner / art.height)
    scaled = art.resize(
        (max(1, round(art.width * scale)), max(1, round(art.height * scale))),
        Image.LANCZOS,
    )

    canvas = Image.new("RGBA", (SIZE, SIZE), (0, 0, 0, 0))
    canvas.paste(
        scaled,
        ((SIZE - scaled.width) // 2, (SIZE - scaled.height) // 2),
        scaled,
    )
    canvas.save(target, "PNG", optimize=True)
    return target.stat().st_size


def render(master: pathlib.Path, target: pathlib.Path) -> tuple[int, int]:
    """Write one downscaled band icon; return its source and output byte sizes."""
    image = Image.open(master).convert("RGBA")
    return master.stat().st_size, render_image(image, target)


def render_ico(master: pathlib.Path, target: pathlib.Path) -> int:
    """Write the multi-size Windows icon; return its byte size."""
    image = trimmed(master)
    # Square it before saving. An .ico entry is square by definition, so a source
    # that is not would be stretched into every size rather than padded once.
    side = max(image.width, image.height)
    square = Image.new("RGBA", (side, side), (0, 0, 0, 0))
    square.paste(image, ((side - image.width) // 2, (side - image.height) // 2), image)
    square.save(target, "ICO", sizes=ICO_SIZES)
    return target.stat().st_size


def main() -> int:
    skip = (APP_MASTER, SLASH_MASTER)
    masters = sorted(p for p in MASTERS.glob("*.png") if p.name not in skip)
    if not masters:
        sys.exit(f"no master artwork found in {MASTERS}")
    OUTPUT.mkdir(parents=True, exist_ok=True)

    total_in = total_out = 0
    for master in masters:
        source, written = render(master, OUTPUT / master.name)
        total_in += source
        total_out += written
        print(f"{master.name:<22} {source:>9,} -> {written:>7,} bytes")

    sounding = MASTERS / SOUNDING_MASTER
    slash = MASTERS / SLASH_MASTER
    for needed in (sounding, slash):
        if not needed.exists():
            sys.exit(f"no {needed.name} in {MASTERS}; the muted icon is made from it")
    total_in += slash.stat().st_size
    written = render_image(slashed(sounding, slash), OUTPUT / MUTED_ICON)
    total_out += written
    print(f"{MUTED_ICON:<22} {'derived':>9} -> {written:>7,} bytes")

    count = len(masters) + 1
    print(f"\n{count} band icons, {total_in:,} -> {total_out:,} bytes")

    app = MASTERS / APP_MASTER
    if not app.exists():
        sys.exit(f"\nno application icon at {app}")
    ico = app.with_suffix(".ico")
    written = render_ico(app, ico)
    source = app.stat().st_size
    print(f"\n{app.name:<22} {source:>9,} -> {written:>7,} bytes  ({ico.name})")

    # The front end wants it as well, for the crest above the About dialog.
    render(app, OUTPUT / APP_MASTER)
    crest = (OUTPUT / APP_MASTER).stat().st_size
    print(f"{'':<22} {'':>9} -> {crest:>7,} bytes  (About crest)")

    header = trimmed(app)
    side = max(header.size)
    canvas = Image.new("RGBA", (side, side), (0, 0, 0, 0))
    canvas.paste(
        header,
        ((side - header.width) // 2, (side - header.height) // 2),
        header,
    )
    canvas.resize((HEADER_SIZE, HEADER_SIZE), Image.LANCZOS).save(
        HEADER, "PNG", optimize=True
    )
    print(f"{'':<22} {'':>9} -> {HEADER.stat().st_size:>7,} bytes  (setup header)")

    for name in SETUP_ICONS:
        source = OUTPUT / name
        if not source.exists():
            sys.exit(f"expected {source} to have just been written")
        icon = Image.open(source).convert("RGBA")
        target = SETUP / name
        icon.resize((SETUP_ICON_SIZE, SETUP_ICON_SIZE), Image.LANCZOS).save(
            target, "PNG", optimize=True
        )
        print(f"{'':<22} {'':>9} -> {target.stat().st_size:>7,} bytes  (setup {name})")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
