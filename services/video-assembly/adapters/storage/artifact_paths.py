"""Shared-volume path convention for Video Assembly artifacts.

Pure filesystem helpers, no framework dependency — constructed directly by
the application/adapter layers rather than injected (dependency-injection.md).
"""

from __future__ import annotations

import os

SHARED_VOLUME_ROOT = "/shared"


def video_output_path(project_id: str) -> str:
    """Conventional path: /shared/{project_id}/video/final.mp4"""
    return os.path.join(SHARED_VOLUME_ROOT, project_id, "video", "final.mp4")


def video_exists(video_path: str) -> bool:
    return os.path.isfile(video_path)


def ensure_parent_dir(path: str) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)


def file_exists(path: str) -> bool:
    return os.path.isfile(path)
