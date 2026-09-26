"""Shared-volume path convention for TTS audio artifacts (Low-Level Design Question 5).

Pure filesystem helpers, no framework dependency — constructed directly by
the application layer rather than injected (dependency-injection.md).
"""

from __future__ import annotations

import hashlib
import os
import shutil
import wave

SHARED_VOLUME_ROOT = "/shared"


def compute_audio_path(project_id: str, scene_index: int, voice_id: str, text: str = "") -> str:
    """Conventional path:
    /shared/{project_id}/audio/{scene_index}_{voice_id}_{text_hash}.wav

    The text hash is not an optimisation — it is a correctness fix (CR-005
    FR13.6). The path used to be keyed on (project_id, scene_index, voice_id)
    alone, so editing a narration line and re-rendering the same project hit
    the idempotency check and reused the OLD audio: the video said the previous
    sentence while the subtitle showed the new one.

    Hashing also means a re-render only pays for the lines that actually
    changed, which matters once synthesis is metered (ADR-0023).

    text defaults to "" so a caller that only needs to locate an existing file
    keeps working; callers that synthesize must pass it.
    """
    digest = hashlib.sha256(text.encode("utf-8")).hexdigest()[:12]
    return os.path.join(
        SHARED_VOLUME_ROOT, project_id, "audio", f"{scene_index}_{voice_id}_{digest}.wav"
    )


def audio_exists(audio_path: str) -> bool:
    return os.path.isfile(audio_path)


def ensure_parent_dir(audio_path: str) -> None:
    os.makedirs(os.path.dirname(audio_path), exist_ok=True)


def read_duration_seconds(audio_path: str) -> float:
    """Read the duration of an existing .wav file from its header (Business Rule 5)."""
    with wave.open(audio_path, "rb") as wav_file:
        frames = wav_file.getnframes()
        rate = wav_file.getframerate()
    return round(frames / float(rate), 2)


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
    """CR-040 FR114.2: remove what TTS owns for a deleted project
    (/shared/{project_id}/audio) and nothing else. Idempotent."""
    project_dir = os.path.join(SHARED_VOLUME_ROOT, _safe_project_id(project_id))
    _remove(os.path.join(project_dir, "audio"))
    _rmdir_if_empty(project_dir)
