"""Static language -> Piper voice model path mapping (Low-Level Design Question 4)."""

from __future__ import annotations

VOICE_MODEL_DIR = "/app/voices"

VOICE_MODEL_PATHS: dict[str, str] = {
    "vi": f"{VOICE_MODEL_DIR}/vi.onnx",
    "en": f"{VOICE_MODEL_DIR}/en.onnx",
}


def get_voice_model_path(language: str) -> str:
    """Raises KeyError if the language has no registered voice model."""
    return VOICE_MODEL_PATHS[language]
