"""Numbers for deciding whether the compile check needs its own container (CR-040 FR115).

The check shares a process, CPU limit and event loop with render. Nothing has
to be split until these numbers say so: FR115.3's trigger is p95 wait above a
threshold. Kept in memory and read from GET /metrics/checks — losing them on a
restart only resets a window, and no metrics stack is needed to read three
counters.
"""

from __future__ import annotations

import math
from collections import deque
from contextlib import contextmanager
from typing import Iterator

WINDOW = 500  # most recent checks kept for the percentile


def _p95(values: list[float]) -> float:
    if not values:
        return 0.0
    ordered = sorted(values)
    return ordered[max(0, math.ceil(0.95 * len(ordered)) - 1)]


class CheckMetrics:
    def __init__(self) -> None:
        self._active_commands = 0
        self.checks_total = 0
        self.checks_timed_out = 0
        self.checks_overlapping_render = 0
        self._waits: deque[float] = deque(maxlen=WINDOW)
        self._runs: deque[float] = deque(maxlen=WINDOW)
        self.max_wait_seconds = 0.0

    @property
    def renders_running(self) -> int:
        """Rendering commands in flight right now (the consumer takes one at a time)."""
        return self._active_commands

    @contextmanager
    def command_running(self) -> Iterator[None]:
        self._active_commands += 1
        try:
            yield
        finally:
            self._active_commands -= 1

    def record(self, *, wait_seconds: float, run_seconds: float, renders_running: int, timed_out: bool) -> None:
        self.checks_total += 1
        self._waits.append(wait_seconds)
        self._runs.append(run_seconds)
        self.max_wait_seconds = max(self.max_wait_seconds, wait_seconds)
        if timed_out:
            self.checks_timed_out += 1
        if renders_running > 0:
            self.checks_overlapping_render += 1

    def snapshot(self) -> dict:
        waits, runs = list(self._waits), list(self._runs)
        return {
            "checks_total": self.checks_total,
            "checks_timed_out": self.checks_timed_out,
            "checks_overlapping_render": self.checks_overlapping_render,
            "renders_running": self.renders_running,
            "wait_seconds": {"p95": round(_p95(waits), 3), "max": round(self.max_wait_seconds, 3)},
            "run_seconds": {"p95": round(_p95(runs), 3)},
            "window": len(waits),
        }
