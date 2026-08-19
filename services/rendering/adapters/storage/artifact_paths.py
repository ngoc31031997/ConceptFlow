"""Shared-volume path convention for Rendering animation artifacts
(Low-Level Design). Mirrors TTS Service's artifact_paths.py.

Pure filesystem helpers, no framework dependency — constructed directly
by the application layer rather than injected (dependency-injection.md).
"""

from __future__ import annotations

import os
import subprocess

SHARED_VOLUME_ROOT = "/shared"


def compute_animation_path(project_id: str, scene_index: int) -> str:
    """Conventional path: /shared/{project_id}/animations/{scene_index}.mp4"""
    return os.path.join(SHARED_VOLUME_ROOT, project_id, "animations", f"{scene_index}.mp4")


def animation_exists(animation_path: str) -> bool:
    return os.path.isfile(animation_path)


def ensure_parent_dir(animation_path: str) -> None:
    os.makedirs(os.path.dirname(animation_path), exist_ok=True)


def read_duration_seconds(animation_path: str) -> float:
    """Read the duration of an existing .mp4 file via ffprobe (parsing the
    container without a library is non-trivial; ffprobe is bundled with
    the ffmpeg system dependency already required for Manim)."""
    result = subprocess.run(
        [
            "ffprobe",
            "-v",
            "error",
            "-show_entries",
            "format=duration",
            "-of",
            "default=noprint_wrappers=1:nokey=1",
            animation_path,
        ],
        capture_output=True,
        text=True,
        check=True,
    )
    return round(float(result.stdout.strip()), 2)
