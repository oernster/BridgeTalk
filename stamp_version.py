"""Stamp the version from VERSION into the Bridge Talk site.

A browser rendering the site cannot read VERSION, so every place the site shows a
version carries a delimited token: <!--VERSION-->x.y.z<!--/VERSION-->. This script
rewrites whatever sits between the delimiters. VERSION stays the one place a real
version string is written by hand.

The site only, deliberately: docs/**/*.html and docs/**/*.md. Markdown at the
repository root holds no version by rule, so it is never a target.

Each page's local stylesheet and script links carry their file's content hash too, as
styles.css?v=<hash>. GitHub Pages lets a browser keep a stylesheet for ten minutes, so
a freshly deployed page could otherwise be drawn with the old one; a changed file is a
new address instead. The hash is taken with CRLF read as LF, so a Windows checkout and
the LF blob GitHub serves agree. A link to a file that does not exist is a failure.

Idempotent. A file already carrying the current version is left alone rather than
rewritten, so a second run changes nothing and says so. Files are read and written as
bytes, so their line endings survive a stamp on any platform.

A missing or empty VERSION is a failure rather than a sentinel: the build calls this
script and has to stop instead of publishing a number nobody chose.
"""

from __future__ import annotations

import hashlib
import pathlib
import re
import sys

ROOT = pathlib.Path(__file__).resolve().parent
VERSION_FILE = ROOT / "VERSION"
SITE_DIR = ROOT / "docs"
SITE_PATTERNS = ("**/*.html", "**/*.md")
PAGE_PATTERN = "**/*.html"
OPEN_TOKEN = "<!--VERSION-->"
CLOSE_TOKEN = "<!--/VERSION-->"
TOKEN = re.compile(re.escape(OPEN_TOKEN) + r".*?" + re.escape(CLOSE_TOKEN), re.DOTALL)
# A stylesheet or script link: the path, any query it already carries (replaced), then
# any fragment (kept).
ASSET_LINK = re.compile(
    r"""(?P<attr>\b(?:href|src)=)(?P<quote>["'])(?P<path>[^"'?#]+\.(?:css|js))"""
    r"""(?:\?[^"'#]*)?(?P<fragment>#[^"']*)?(?P=quote)"""
)
ASSET_HASH_LENGTH = 10
ENCODING = "utf-8"
SUCCESS = 0
FAILURE = 1


def read_version() -> str | None:
    """The version VERSION holds; None when the file is missing or empty."""
    try:
        text = VERSION_FILE.read_text(encoding=ENCODING).strip()
    except OSError:
        return None
    return text or None


def site_files() -> list[pathlib.Path]:
    """Every site page that could carry a token, in a stable order."""
    found: set[pathlib.Path] = set()
    for pattern in SITE_PATTERNS:
        found.update(path for path in SITE_DIR.glob(pattern) if path.is_file())
    return sorted(found)


def stamp(path: pathlib.Path, version: str) -> bool:
    """Put this version in every token in one file; True when the file changed."""
    original = path.read_bytes().decode(ENCODING)
    stamped = TOKEN.sub(lambda _match: f"{OPEN_TOKEN}{version}{CLOSE_TOKEN}", original)
    if stamped == original:
        return False
    path.write_bytes(stamped.encode(ENCODING))
    return True


def asset_hash(path: pathlib.Path) -> str:
    """The content hash one stylesheet or script is linked by, CRLF read as LF."""
    if not path.is_file():
        raise FileNotFoundError(f"a site page links {path}, which does not exist")
    content = path.read_bytes().replace(b"\r\n", b"\n")
    return hashlib.sha256(content).hexdigest()[:ASSET_HASH_LENGTH]


def version_assets(page: pathlib.Path) -> bool:
    """Put the content hash on every local asset link in one page; True when it changed."""
    original = page.read_bytes().decode(ENCODING)

    def versioned(match: re.Match[str]) -> str:
        path = match["path"]
        if path.startswith("/") or ":" in path:
            return match[0]
        digest = asset_hash(page.parent / path)
        fragment = match["fragment"] or ""
        quote = match["quote"]
        return f"{match['attr']}{quote}{path}?v={digest}{fragment}{quote}"

    linked = ASSET_LINK.sub(versioned, original)
    if linked == original:
        return False
    page.write_bytes(linked.encode(ENCODING))
    return True


def report(heading: str, paths: list[pathlib.Path]) -> None:
    """Name each file one step touched, under its heading; silent when there are none."""
    if not paths:
        return
    print(heading)
    for path in paths:
        print(f"  {path.relative_to(ROOT).as_posix()}")


def main() -> int:
    """Stamp the whole site, naming each file actually touched."""
    version = read_version()
    if version is None:
        print(f"no version in {VERSION_FILE}; refusing to stamp", file=sys.stderr)
        return FAILURE
    if not SITE_DIR.is_dir():
        print(f"no site at {SITE_DIR}; nothing to stamp")
        return SUCCESS
    touched = [path for path in site_files() if stamp(path, version)]
    try:
        pages = sorted(path for path in SITE_DIR.glob(PAGE_PATTERN) if path.is_file())
        linked = [page for page in pages if version_assets(page)]
    except FileNotFoundError as missing:
        print(f"{missing}; refusing to stamp", file=sys.stderr)
        return FAILURE
    if not touched and not linked:
        print(f"nothing to stamp: the site is already at {version} with current links")
        return SUCCESS
    report(f"stamped {version} into:", touched)
    report("versioned the stylesheet and script links in:", linked)
    return SUCCESS


if __name__ == "__main__":
    sys.exit(main())
