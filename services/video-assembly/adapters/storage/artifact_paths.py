"""Shared-volume path convention for Video Assembly artifacts
(Low-Level Design Question 7, Infrastructure Design).

Pure filesystem helpers, no framework dependency — constructed directly by
the application/adapter layers rather than injected (dependency-injection.md).
"""

from __future__ import annotations

import os
import shutil

SHARED_VOLUME_ROOT = "/shared"


def video_output_path(project_id: str) -> str:
    """Conventional path: /shared/{project_id}/video/final.mp4"""
    return os.path.join(SHARED_VOLUME_ROOT, project_id, "video", "final.mp4")


def video_exists(video_path: str) -> bool:
    return os.path.isfile(video_path)


def tmp_dir(project_id: str) -> str:
    """Intermediate mux files live here, cleaned up after a successful assembly."""
    return os.path.join(SHARED_VOLUME_ROOT, project_id, "video", "_tmp")


def tmp_muxed_path(project_id: str, scene_index: int) -> str:
    return os.path.join(tmp_dir(project_id), f"scene_{scene_index}_muxed.mp4")


def tmp_concatenated_path(project_id: str) -> str:
    return os.path.join(tmp_dir(project_id), "concatenated.mp4")


def tmp_concat_list_path(project_id: str) -> str:
    return os.path.join(tmp_dir(project_id), "concat_list.txt")


def ensure_parent_dir(path: str) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)


def cleanup_tmp_dir(project_id: str) -> None:
    shutil.rmtree(tmp_dir(project_id), ignore_errors=True)


def file_exists(path: str) -> bool:
    return os.path.isfile(path)
