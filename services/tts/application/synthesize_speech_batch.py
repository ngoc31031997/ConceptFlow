"""SynthesizeSpeechBatchUseCase — synthesizes every scene in a project,
fail-fast on the first error (mirrors ClassifyScenesBatchUseCase, Unit 2).

Used by the synthesize_speech AMQP command handler (ADR-0014): the command
carries every scene for a project in one batch, matching how
classify_scenes is already handled at Content Plugin Service.
"""

from __future__ import annotations

from dataclasses import dataclass

from application.synthesize_speech import SynthesizeSpeechUseCase
from domain.errors import EmptyTextError, TTSEngineError, UnsupportedLanguageError
from domain.models import SpeechRequest, SpeechResult


@dataclass(frozen=True)
class SceneSpeechRequest:
    scene_index: int
    narration_text: str
    language: str


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

    def execute(self, project_id: str, scenes: list[SceneSpeechRequest]) -> BatchSynthesisOutcome:
        results: list[SceneSpeechResult] = []
        for scene in scenes:
            request = SpeechRequest(
                project_id=project_id,
                scene_index=scene.scene_index,
                text=scene.narration_text,
                language=scene.language,
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
        return BatchSynthesisSuccess(results=results)
