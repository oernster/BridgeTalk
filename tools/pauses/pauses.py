"""Find the break before a final commander in each of a voice's joined lines (FR-551).

Reads one JSON request on standard input: every setting, then the WAV files of one
voice's lines. Writes a JSON list on standard output answering each file in order with
the sample the pause goes at and the length in seconds of the final voiced stretch;
both are null where no break is found. The Go half of the pauses tool owns every
setting, the lines and the file it writes, so this half only speaks to Praat.

The break is the last unvoiced stretch before the final voiced stretch of at least the
request's final frames, reading voicing with Praat in frames of the request's time
step and counting an unvoiced gap of up to its bridged frames as voiced. The pause
goes at the middle of the quietest window within the break.
"""

from __future__ import annotations

import itertools
import json
import sys
from dataclasses import dataclass
from typing import Any

import numpy as np
import numpy.typing as npt
import parselmouth
import soundfile

# The middle of a frame or a window, as a share of its length.
MIDDLE = 0.5

Samples = npt.NDArray[np.float32]
Voicing = npt.NDArray[np.bool_]


@dataclass(frozen=True, slots=True)
class Settings:
    """The request's settings.

    A frame in seconds and in samples, Praat's pitch band in hertz, the gap bridged and
    the final voiced stretch in frames, the quiet window in samples.
    """

    time_step: float
    frame: int
    pitch_floor: float
    pitch_ceiling: float
    bridged: int
    final: int
    quiet: int

    @classmethod
    def read(cls, request: dict[str, Any]) -> Settings:
        """Take every setting from the request; a missing one fails the run."""
        return cls(
            time_step=float(request["time_step"]),
            frame=int(request["frame"]),
            pitch_floor=float(request["pitch_floor"]),
            pitch_ceiling=float(request["pitch_ceiling"]),
            bridged=int(request["bridged"]),
            final=int(request["final"]),
            quiet=int(request["quiet"]),
        )


def voiced_frames(samples: Samples, rate: int, settings: Settings) -> Voicing:
    """Read a frame as voiced where the Praat frame nearest its middle has a pitch.

    A frame whose nearest Praat frame lies more than a time step away is unvoiced.
    """
    sound = parselmouth.Sound(samples.astype(np.float64), sampling_frequency=rate)
    track = sound.to_pitch(
        settings.time_step, settings.pitch_floor, settings.pitch_ceiling
    )
    times = np.array(track.xs())
    frequency = track.selected_array["frequency"]
    count = len(samples) // settings.frame
    middles = (np.arange(count) + MIDDLE) * settings.time_step
    right = np.clip(np.searchsorted(times, middles), 1, len(times) - 1)
    left = right - 1
    nearer_right = np.abs(times[right] - middles) < np.abs(times[left] - middles)
    nearest = np.where(nearer_right, right, left)
    close = np.abs(times[nearest] - middles) <= settings.time_step
    return close & (frequency[nearest] > 0)


def final_break(voiced: Voicing, settings: Settings) -> tuple[int, int, int] | None:
    """Answer the break's first frame, the final voiced stretch's first frame and the
    frame after that stretch.

    None where nothing is voiced, no stretch is long enough or nothing is voiced before
    the final stretch.
    """
    filled = voiced.copy()
    marks = np.nonzero(filled)[0]
    if marks.size == 0:
        return None
    for before, after in itertools.pairwise(marks):
        if 1 < after - before <= settings.bridged + 1:
            filled[before + 1 : after] = True
    runs: list[tuple[int, int]] = []
    start: int | None = None
    for index, value in enumerate(np.append(filled, False)):
        if value and start is None:
            start = index
        elif not value and start is not None:
            runs.append((start, index))
            start = None
    long_runs = [run for run in runs if run[1] - run[0] >= settings.final]
    if not long_runs:
        return None
    first, stop = long_runs[-1]
    voiced_before = np.nonzero(filled[:first])[0]
    if voiced_before.size == 0:
        return None
    return int(voiced_before[-1]) + 1, first, stop


def quietest(samples: Samples, first: int, end: int, width: int) -> int:
    """Answer the middle sample of the quietest window of width samples.

    The window starts from first and ends by end; the earliest wins where two tie.
    """
    last = end - width
    best, best_rms = first, float("inf")
    for index in range(first, max(first, last) + 1):
        rms = float(np.sqrt(np.mean(samples[index : index + width] ** 2)))
        if rms < best_rms:
            best, best_rms = index, rms
    return best + int(width * MIDDLE)


def answer(path: str, settings: Settings) -> dict[str, int | float | None]:
    """Answer one file's cut and final voiced stretch; None for both without a break."""
    samples, rate = soundfile.read(path, dtype="float32")
    found = final_break(voiced_frames(samples, rate, settings), settings)
    if found is None:
        return {"cut": None, "final": None}
    first, start, stop = found
    cut = quietest(
        samples, first * settings.frame, start * settings.frame, settings.quiet
    )
    return {"cut": cut, "final": (stop - start) * settings.time_step}


def main() -> None:
    """Answer every file the request names, in order."""
    sys.stdin.reconfigure(encoding="utf-8")
    sys.stdout.reconfigure(encoding="utf-8")
    request = json.load(sys.stdin)
    settings = Settings.read(request)
    json.dump([answer(path, settings) for path in request["files"]], sys.stdout)


if __name__ == "__main__":
    main()
