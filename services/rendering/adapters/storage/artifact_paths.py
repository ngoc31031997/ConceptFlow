"""Shared-volume path convention for Rendering's output video (Manim-script
input mode — one project renders to one silent video, later muxed with
narration audio by Video Assembly).

Pure filesystem helpers, no framework dependency — constructed directly
by the application layer rather than injected (dependency-injection.md).
"""

from __future__ import annotations

import json
import os

SHARED_VOLUME_ROOT = "/shared"


def compute_video_path(project_id: str) -> str:
    """Conventional path: /shared/{project_id}/video/rendered.mp4"""
    return os.path.join(SHARED_VOLUME_ROOT, project_id, "video", "rendered.mp4")


def compute_timing_path(project_id: str) -> str:
    """Sidecar for the timing that CR-002 added: /shared/{id}/video/timing.json

    Kept next to the video rather than only on the event, because render() is
    idempotent — a redelivered command finds the .mp4 already there and skips
    the Manim run. Without this file that fast path would have no offsets to
    report and would silently emit a desynchronised event, which is precisely
    the failure CR-002 removes.
    """
    return os.path.join(SHARED_VOLUME_ROOT, project_id, "video", "timing.json")


def video_exists(video_path: str) -> bool:
    return os.path.isfile(video_path)


def read_timing(timing_path: str) -> dict | None:
    """Returns None when the sidecar is missing or unreadable, so the caller
    can fall back to re-rendering rather than trusting partial data."""
    try:
        with open(timing_path, encoding="utf-8") as f:
            data = json.load(f)
    except (OSError, ValueError):
        return None
    if not isinstance(data, dict) or "wait_offsets" not in data:
        return None
    return data


def write_timing(timing_path: str, wait_offsets: list[float], video_duration_seconds: float) -> None:
    os.makedirs(os.path.dirname(timing_path), exist_ok=True)
    with open(timing_path, "w", encoding="utf-8") as f:
        json.dump(
            {"wait_offsets": wait_offsets, "video_duration_seconds": video_duration_seconds}, f
        )


def ensure_parent_dir(video_path: str) -> None:
    os.makedirs(os.path.dirname(video_path), exist_ok=True)
