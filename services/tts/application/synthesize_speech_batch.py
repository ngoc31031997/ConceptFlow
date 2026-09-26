"""SynthesizeSpeechBatchUseCase — synthesizes every scene in a project,
fail-fast on the first error (mirrors ClassifyScenesBatchUseCase, Unit 2).

Used by the synthesize_speech AMQP command handler (ADR-0014): the command
carries every scene for a project in one batch, matching how
classify_scenes is already handled at Content Plugin Service.
"""

from __future__ import annotations

from dataclasses import dataclass
from typing import Callable

from application.synthesize_speech import SynthesizeSpeechUseCase
from domain.errors import EmptyTextError, TTSEngineError, UnsupportedLanguageError
from domain.models import SpeechRequest, SpeechResult


@dataclass(frozen=True)
class SceneSpeechRequest:
    scene_index: int
    narration_text: str
    language: str
    voice_id: str | None = None


@dataclass(frozen=True)
class SceneSpeechResult:
    scene_index: int
    audio_path: str
    duration_seconds: float


@dataclass(frozen=True)
class BatchSynthesisSuccess:
    results: list[SceneSpeechResult]


@dataclass(frozen=True)
class BatchSynthesisFailure:
    error_message: str


BatchSynthesisOutcome = BatchSynthesisSuccess | BatchSynthesisFailure


class SynthesizeSpeechBatchUseCase:
    def __init__(self, single_scene_use_case: SynthesizeSpeechUseCase) -> None:
        self._synthesize_speech = single_scene_use_case

    def execute(
        self,
        project_id: str,
        scenes: list[SceneSpeechRequest],
        on_scene_done: Callable[[int, int], None] | None = None,
        should_stop: Callable[[], bool] | None = None,
    ) -> BatchSynthesisOutcome:
        """CR-029: on_scene_done(scene_index, scene_total), called right after
        each scene's audio is ready, lets the caller publish a progress ping
        without this use case knowing anything about RabbitMQ/asyncio — it
        stays synchronous and testable exactly as before.

        should_stop, checked before each scene, lets the caller abandon the
        batch (the Creator cancelled): the remaining scenes are not spoken.
        """
        results: list[SceneSpeechResult] = []
        total = len(scenes)
        for scene in scenes:
            if should_stop is not None and should_stop():
                return BatchSynthesisFailure(error_message="cancelled")
            request = SpeechRequest(
                project_id=project_id,
                scene_index=scene.scene_index,
                text=scene.narration_text,
                language=scene.language,
                voice_id=scene.voice_id,
            )
            try:
                result: SpeechResult = self._synthesize_speech.synthesize(request)
            except EmptyTextError:
                return BatchSynthesisFailure(
                    error_message=f"empty_text: scene_index={scene.scene_index}"
                )
            except UnsupportedLanguageError as exc:
                return BatchSynthesisFailure(error_message=f"unsupported_language: {exc}")
            except TTSEngineError as exc:
                return BatchSynthesisFailure(error_message=f"tts_engine_failure: {exc}")
            results.append(
                SceneSpeechResult(
                    scene_index=scene.scene_index,
                    audio_path=result.audio_path,
                    duration_seconds=result.duration_seconds,
                )
            )
            if on_scene_done is not None:
                on_scene_done(len(results), total)
        return BatchSynthesisSuccess(results=results)
