"""Shared-volume path convention for Rendering's output video (Manim-script
input mode — one project renders to one silent video, later muxed with
narration audio by Video Assembly).

Pure filesystem helpers, no framework dependency — constructed directly
by the application layer rather than injected (dependency-injection.md).
"""

from __future__ import annotations

import os

SHARED_VOLUME_ROOT = "/shared"


def compute_video_path(project_id: str) -> str:
    """Conventional path: /shared/{project_id}/video/rendered.mp4"""
    return os.path.join(SHARED_VOLUME_ROOT, project_id, "video", "rendered.mp4")


def video_exists(video_path: str) -> bool:
    return os.path.isfile(video_path)


def ensure_parent_dir(video_path: str) -> None:
    os.makedirs(os.path.dirname(video_path), exist_ok=True)
