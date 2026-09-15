"""Find where each of a voice's lines ending on a nasal ends on a burst (FR-555).

Reads one JSON request on standard input: every setting, then the WAV files of one
voice's lines. Writes a JSON list on standard output answering each file in order with
the sample its fade starts at; null where the line ends on no burst. The Go half of the
pauses tool owns every setting, the lines and the file it writes, so this half only
reads the sound.

A burst is a run of frames, counted from the line's first sample, each louder than the
request's loudness with at least its share of energy above its frequency, read through
a Hann window, whose last frame ends no more than its span before the end of the line's
last frame louder than that loudness. The fade starts at the first sample of the run.
"""

from __future__ import annotations

import json
import sys
from dataclasses import dataclass
from typing import Any

import numpy as np
import numpy.typing as npt
import soundfile

# Decibels are twenty times the base-ten logarithm of an amplitude ratio.
DECIBELS_PER_DECADE = 20

Samples = npt.NDArray[np.float32]


@dataclass(frozen=True, slots=True)
class Settings:
    """The request's settings.

    A frame in samples, the frequency in hertz a burst frame holds its share of energy
    above, the loudness in decibels a frame must exceed, that share and the most samples
    a burst may end before the line's last loud frame ends.
    """

    frame: int
    high_hz: float
    loud_db: float
    share: float
    within: int

    @classmethod
    def read(cls, request: dict[str, Any]) -> Settings:
        """Take every setting from the request; a missing one fails the run."""
        return cls(
            frame=int(request["frame"]),
            high_hz=float(request["high_hz"]),
            loud_db=float(request["loud_db"]),
            share=float(request["share"]),
            within=int(request["within"]),
        )


def fade_start(samples: Samples, rate: int, settings: Settings) -> int | None:
    """Answer the first sample of the last run of burst frames; None without a burst."""
    window = np.hanning(settings.frame)
    frequencies = np.fft.rfftfreq(settings.frame, 1 / rate)
    run_start: int | None = None
    burst_end: int | None = None
    loud_end = 0
    in_run = False
    for first in range(0, len(samples) - settings.frame + 1, settings.frame):
        frame = samples[first : first + settings.frame].astype(np.float64)
        rms = float(np.sqrt(np.mean(frame**2)))
        if rms <= 0 or DECIBELS_PER_DECADE * np.log10(rms) <= settings.loud_db:
            in_run = False
            continue
        loud_end = first + settings.frame
        power = np.abs(np.fft.rfft(frame * window)) ** 2
        high = float(power[frequencies >= settings.high_hz].sum())
        burst = high >= settings.share * float(power.sum())
        if burst:
            if not in_run:
                run_start = first
            burst_end = first + settings.frame
        in_run = burst
    if burst_end is None or loud_end - burst_end > settings.within:
        return None
    return run_start


def answer(path: str, settings: Settings) -> dict[str, int | None]:
    """Answer one file's fade start; None where it ends on no burst."""
    samples, rate = soundfile.read(path, dtype="float32")
    return {"start": fade_start(samples, rate, settings)}


def main() -> None:
    """Answer every file the request names, in order."""
    sys.stdin.reconfigure(encoding="utf-8")
    sys.stdout.reconfigure(encoding="utf-8")
    request = json.load(sys.stdin)
    settings = Settings.read(request)
    json.dump([answer(path, settings) for path in request["files"]], sys.stdout)


if __name__ == "__main__":
    main()
