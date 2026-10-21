"""SynthesizeLinesUseCase — reads script lines outside a render saga (ADR-0040).

Unlike a scene batch, every line is answered on its own: a line that cannot be
read carries its reason and the others are still read. A line already read in
the same voice with the same wording is not read again (cache_hit).
"""

from __future__ import annotations

from collections.abc import Callable
from dataclasses import dataclass

from adapters.storage.artifact_paths import (
    audio_exists,
    compute_line_audio_path,
    compute_line_cache_key,
    ensure_parent_dir,
    is_valid_line_id,
    read_duration_seconds,
)
from adapters.tts_engines.voice_registry import resolve_voice_id
from domain.errors import EmptyTextError, TTSEngineError, UnsupportedLanguageError
from domain.ports import TTSEnginePort

SUPPORTED_LANGUAGES = ("vi", "en")


@dataclass(frozen=True)
class LineSpeechRequest:
    line_id: str
    text: str
    language: str
    voice_id: str | None = None
    # Handed back unchanged so the caller can tell which wording was read.
    ref: str = ""


@dataclass(frozen=True)
class LineSpeechResult:
    line_id: str
    ref: str
    audio_path: str = ""
    duration_seconds: float = 0.0
    cache_key: str = ""
    cache_hit: bool = False
    error: str = ""


class SynthesizeLinesUseCase:
    def __init__(self, engine: TTSEnginePort) -> None:
        self._engine = engine

    def execute(
        self,
        project_id: str,
        lines: list[LineSpeechRequest],
        should_stop: Callable[[], bool] | None = None,
    ) -> list[LineSpeechResult]:
        """should_stop, checked before each line, lets the caller abandon the
        rest (the Creator cancelled): the remaining lines come back as errors
        saying so, never as silent successes."""
        results: list[LineSpeechResult] = []
        for line in lines:
            if should_stop is not None and should_stop():
                results.append(LineSpeechResult(line.line_id, line.ref, error="cancelled"))
                continue
            results.append(self._one(project_id, line))
        return results

    def _one(self, project_id: str, line: LineSpeechRequest) -> LineSpeechResult:
        fail = lambda why: LineSpeechResult(line.line_id, line.ref, error=why)  # noqa: E731
        if not is_valid_line_id(line.line_id):
            return fail(f"mã câu không hợp lệ: {line.line_id!r}")
        text = line.text.strip()
        try:
            if not text:
                raise EmptyTextError
            if line.language not in SUPPORTED_LANGUAGES:
                raise UnsupportedLanguageError(line.language, list(SUPPORTED_LANGUAGES))
        except EmptyTextError:
            return fail("câu trống")
        except UnsupportedLanguageError as exc:
            return fail(str(exc))

        voice_id = resolve_voice_id(line.voice_id, line.language)
        cache_key = compute_line_cache_key(voice_id, text)
        path = compute_line_audio_path(project_id, line.line_id, cache_key)
        hit = audio_exists(path)
        try:
            if not hit:
                ensure_parent_dir(path)
                self._engine.synthesize(text, voice_id, path)
            duration = read_duration_seconds(path)
        except (TTSEngineError, OSError) as exc:
            return fail(f"không đọc được câu: {exc}")
        return LineSpeechResult(line.line_id, line.ref, path, duration, cache_key, hit)
