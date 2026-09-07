"""AssembleVideoUseCase — business-logic-model.md.

Unlike Unit 2/3/4/5, there is no separate "batch" wrapper — assemble_video
is already a single operation over an entire project (Low-Level Design
Question 10), so this one use case is the whole application layer.
"""

from __future__ import annotations

from adapters.storage.artifact_paths import (
    ensure_parent_dir,
    file_exists,
    video_exists,
    video_output_path,
)
from domain.errors import MissingArtifactError
from domain.models import VideoAssemblyRequest, VideoAssemblyResult
from domain.ports import VideoAssemblerPort


class AssembleVideoUseCase:
    """Orchestrates validation, idempotency, and delegation to the assembler."""

    def __init__(self, assembler: VideoAssemblerPort) -> None:
        self._assembler = assembler

    def assemble(self, request: VideoAssemblyRequest) -> VideoAssemblyResult:
        output_path = video_output_path(request.project_id)

        if video_exists(output_path):
            # Idempotency (Business Rule 8): reuse the artifact from a prior call
            # instead of re-assembling.
            return VideoAssemblyResult(video_path=output_path)

        self._validate(request)

        ensure_parent_dir(output_path)
        self._assembler.assemble(request, output_path)
        return VideoAssemblyResult(video_path=output_path)

    @staticmethod
    def _validate(request: VideoAssemblyRequest) -> None:
        """Zero-trust validation (Business Rule 1): does not trust that
        upstream steps already validated this data."""
        if not request.video_path:
            raise MissingArtifactError("video_path must not be empty")
        if not file_exists(request.video_path):
            raise MissingArtifactError(f"missing video {request.video_path}")

        # An empty narration_segments list is valid since CR-001: the Creator
        # can disable narration, producing a silent video (or one with
        # background music only).
        for segment in request.narration_segments:
            if not segment.audio_path:
                raise MissingArtifactError("empty audio_path in narration_segments")
            if not file_exists(segment.audio_path):
                raise MissingArtifactError(f"missing audio {segment.audio_path}")
            if segment.start_time < 0:
                raise MissingArtifactError(
                    f"negative start_time {segment.start_time} for {segment.audio_path}"
                )

        if request.background_music_path and not file_exists(request.background_music_path):
            raise MissingArtifactError(f"missing background music {request.background_music_path}")
