"""Turn lines into speech sounds with misaki, called the way Kokoro calls it (FR-532).

Reads a JSON list of lines on standard input and writes a JSON list of their speech
sounds, in the same order, on standard output. Pass --british for British English;
American otherwise. The Go half of the sounds tool owns the script, its spellings and
the saved file, so this half only speaks to misaki.
"""

import argparse
import json
import sys

from misaki import en, espeak


def sounds(maker: en.G2P, text: str) -> str:
    """Join as Kokoro does: a word with no sounds adds nothing; its space follows."""
    _, tokens = maker(text)
    joined = "".join(
        (token.phonemes or "") + (" " if token.whitespace else "") for token in tokens
    )
    return joined.strip()


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--british", action="store_true")
    british = parser.parse_args().british
    maker = en.G2P(
        trf=False,
        british=british,
        fallback=espeak.EspeakFallback(british=british),
        unk="",
    )
    sys.stdin.reconfigure(encoding="utf-8")
    sys.stdout.reconfigure(encoding="utf-8")
    lines = json.load(sys.stdin)
    json.dump([sounds(maker, line) for line in lines], sys.stdout, ensure_ascii=False)


if __name__ == "__main__":
    main()
