"""Shared-volume path convention for Video Assembly artifacts.

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


def caption_output_path(project_id: str) -> str:
    """Conventional path for the CR-015 caption track: same directory and
    stem as the video, .srt extension — mirrors video_output_path so the two
    artifacts are found the same way."""
    return os.path.join(SHARED_VOLUME_ROOT, project_id, "video", "final.srt")


def normalized_channel_asset_path(kind: str, render_quality: str) -> str:
    """Conventional path for a Creator-uploaded intro/outro after
    NormalizeChannelAssetCommandHandler transcodes it (CR-023 D8):
    /shared/channel-assets/{kind}/{render_quality}/normalized.mp4

    Deliberately keyed by render_quality too (unlike rendering's
    compute_channel_asset_path for the Manim-rendered default, which is a
    single file shared across qualities) — an uploaded file gets a real,
    quality-specific transcode here, one per render_quality.
    """
    return os.path.join(SHARED_VOLUME_ROOT, "channel-assets", kind, render_quality, "normalized.mp4")


def channel_asset_with_music_path(kind: str, render_quality: str, version: int) -> str:
    """Where NormalizeChannelAssetCommandHandler writes the intro/outro clip
    after muxing the Creator's music bed into it (CR-023 D5 — the music is
    baked in at asset-build time, so per-project assembly never has to know
    about it):
    /shared/channel-assets/{kind}/{render_quality}/with_music_v{version}.mp4

    Versioned because the input of that mux is the currently active asset's
    own file — an unversioned name would have ffmpeg read and write the same
    path on the second music upload.
    """
    return os.path.join(
        SHARED_VOLUME_ROOT, "channel-assets", kind, render_quality, f"with_music_v{version}.mp4"
    )


def clip_output_path(project_id: str, slug: str, preset: str) -> str:
    """Conventional path for a CR-007 vertical clip:
    /shared/{project_id}/clips/{slug}_{preset}.mp4 — D7's download route reads
    the same shared volume, so the shape here is the contract with it."""
    return os.path.join(SHARED_VOLUME_ROOT, project_id, "clips", f"{slug}_{preset}.mp4")


def video_exists(video_path: str) -> bool:
    return os.path.isfile(video_path)


def ensure_parent_dir(path: str) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)


def file_exists(path: str) -> bool:
    return os.path.isfile(path)


def _safe_project_id(project_id: str) -> str:
    """A project id is a single path segment. Anything else could make the
    purge delete outside the project's own directory."""
    if not project_id or project_id in (".", "..") or "/" in project_id or "\\" in project_id:
        raise ValueError(f"unsafe project_id {project_id!r}")
    return project_id


def _remove(path: str) -> None:
    if os.path.isdir(path) and not os.path.islink(path):
        shutil.rmtree(path, ignore_errors=True)
    else:
        try:
            os.remove(path)
        except FileNotFoundError:
            pass


def _rmdir_if_empty(path: str) -> None:
    try:
        os.rmdir(path)
    except OSError:
        pass  # not empty (another service's files) or already gone


def purge_project_artifacts(project_id: str) -> None:
    """CR-040 FR114.2: remove what Video Assembly owns for a deleted project —
    final.mp4, final.srt and the clips directory. Never touches rendered.mp4 or
    timing.json (rendering's) or audio (tts's). Idempotent."""
    pid = _safe_project_id(project_id)
    project_dir = os.path.join(SHARED_VOLUME_ROOT, pid)
    video_dir = os.path.join(project_dir, "video")
    _remove(os.path.join(video_dir, "final.mp4"))
    _remove(os.path.join(video_dir, "final.srt"))
    _remove(os.path.join(project_dir, "clips"))
    _rmdir_if_empty(video_dir)
    _rmdir_if_empty(project_dir)
