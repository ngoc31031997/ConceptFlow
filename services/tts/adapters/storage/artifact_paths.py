"""Shared-volume path convention for TTS audio artifacts (Low-Level Design Question 5).

Pure filesystem helpers, no framework dependency — constructed directly by
the application layer rather than injected (dependency-injection.md).
"""

from __future__ import annotations

import hashlib
import os
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
