"""Shared-volume path convention for TTS audio artifacts (Low-Level Design Question 5).

Pure filesystem helpers, no framework dependency — constructed directly by
the application layer rather than injected (dependency-injection.md).
"""

from __future__ import annotations

import os
import wave

SHARED_VOLUME_ROOT = "/shared"


def compute_audio_path(project_id: str, scene_index: int, language: str) -> str:
    """Conventional path: /shared/{project_id}/audio/{scene_index}_{language}.wav"""
    return os.path.join(SHARED_VOLUME_ROOT, project_id, "audio", f"{scene_index}_{language}.wav")


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
