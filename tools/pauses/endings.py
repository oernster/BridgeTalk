"""Find where the hiss starts in each of a voice's lines ending on a nasal (FR-555).

Reads one JSON request on standard input: every setting, then the WAV files of one
voice's lines. Writes a JSON list on standard output answering each file in order with
the sample its hiss starts at; null where the line ends on no burst. The Go half of the
pauses tool owns every setting, the lines, the fade and the file it writes, so this half
only reads the sound.

A burst is a run of frames, counted from the line's first sample, each louder than the
request's loudness with at least its share of energy above its frequency, read through
a Hann window, whose last frame ends no more than its span before the end of the line's
last frame louder than that loudness. The hiss starts at the first of the shorter hiss
frames read the same way, from one frame before the run's first frame up to it; at the
run's first sample where none of them is a burst frame.
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

    A frame and a hiss frame in samples, the frequency in hertz a burst frame holds its
    share of energy above, the loudness in decibels a frame must exceed, that share and
    the most samples a burst may end before the line's last loud frame ends.
    """

    frame: int
    hiss_frame: int
    high_hz: float
    loud_db: float
    share: float
    within: int

    @classmethod
    def read(cls, request: dict[str, Any]) -> Settings:
        """Take every setting from the request; a missing one fails the run."""
        return cls(
            frame=int(request["frame"]),
            hiss_frame=int(request["hiss_frame"]),
            high_hz=float(request["high_hz"]),
            loud_db=float(request["loud_db"]),
            share=float(request["share"]),
            within=int(request["within"]),
        )


@dataclass(frozen=True, slots=True)
class Reading:
    """What one frame is: louder than the loudness; a burst frame besides."""

    loud: bool
    burst: bool


def read_frame(frame: Samples, rate: int, settings: Settings) -> Reading:
    """Read one frame: loud where it beats the loudness, a burst where its share is high too."""
    values = frame.astype(np.float64)
    rms = float(np.sqrt(np.mean(values**2)))
    if rms <= 0 or DECIBELS_PER_DECADE * np.log10(rms) <= settings.loud_db:
        return Reading(loud=False, burst=False)
    power = np.abs(np.fft.rfft(values * np.hanning(len(values)))) ** 2
    frequencies = np.fft.rfftfreq(len(values), 1 / rate)
    high = float(power[frequencies >= settings.high_hz].sum())
    return Reading(loud=True, burst=high >= settings.share * float(power.sum()))


def run_start(samples: Samples, rate: int, settings: Settings) -> int | None:
    """Answer the first sample of the last run of burst frames; None without a burst."""
    start: int | None = None
    burst_end: int | None = None
    loud_end = 0
    in_run = False
    for first in range(0, len(samples) - settings.frame + 1, settings.frame):
        reading = read_frame(samples[first : first + settings.frame], rate, settings)
        if not reading.loud:
            in_run = False
            continue
        loud_end = first + settings.frame
        if reading.burst:
            if not in_run:
                start = first
            burst_end = first + settings.frame
        in_run = reading.burst
    if burst_end is None or loud_end - burst_end > settings.within:
        return None
    return start


def hiss_start(samples: Samples, rate: int, settings: Settings) -> int | None:
    """Answer the sample the hiss starts at; None without a burst."""
    run = run_start(samples, rate, settings)
    if run is None:
        return None
    for first in range(max(0, run - settings.frame), run, settings.hiss_frame):
        hiss_frame = samples[first : first + settings.hiss_frame]
        if read_frame(hiss_frame, rate, settings).burst:
            return first
    return run


def answer(path: str, settings: Settings) -> dict[str, int | None]:
    """Answer where one file's hiss starts; None where it ends on no burst."""
    samples, rate = soundfile.read(path, dtype="float32")
    return {"hiss": hiss_start(samples, rate, settings)}


def main() -> None:
    """Answer every file the request names, in order."""
    sys.stdin.reconfigure(encoding="utf-8")
    sys.stdout.reconfigure(encoding="utf-8")
    request = json.load(sys.stdin)
    settings = Settings.read(request)
    json.dump([answer(path, settings) for path in request["files"]], sys.stdout)


if __name__ == "__main__":
    main()
