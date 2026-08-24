"""FfmpegVideoAssembler — implements VideoAssemblerPort by shelling out to
the ffmpeg CLI binary (Low-Level Design Question 3).

Runs the whole mux -> concat -> overlay pipeline in a threadpool so it
never blocks the asyncio event loop (NFR Requirements, Performance), with
a bounded timeout so a hung ffmpeg call surfaces as a clear
AssemblyEngineError instead of hanging the caller forever.

Sorts scenes by scene_index and validates the sequence itself (Business
Rule 2, Revision LLD Question 2) rather than trusting the array order the
command arrived in.
"""

from __future__ import annotations

import logging
import subprocess
from concurrent.futures import ThreadPoolExecutor
from concurrent.futures import TimeoutError as FutureTimeoutError

from adapters.assembly.ffprobe_inspector import MediaFormatInspector
from adapters.storage.artifact_paths import (
    cleanup_tmp_dir,
    ensure_parent_dir,
    tmp_concat_list_path,
    tmp_concatenated_path,
    tmp_muxed_path,
)
from domain.errors import AssemblyEngineError, InvalidSceneIndexError
from domain.models import VideoAssemblyRequest
from domain.ports import VideoAssemblerPort

logger = logging.getLogger(__name__)

DEFAULT_ASSEMBLY_TIMEOUT_SECONDS = 180
FFMPEG_BINARY = "ffmpeg"
BACKGROUND_MUSIC_VOLUME = 0.2


class FfmpegVideoAssembler(VideoAssemblerPort):
    def __init__(
        self, inspector: MediaFormatInspector, timeout_seconds: int = DEFAULT_ASSEMBLY_TIMEOUT_SECONDS
    ) -> None:
        self._inspector = inspector
        self._timeout_seconds = timeout_seconds
        self._executor = ThreadPoolExecutor(max_workers=2, thread_name_prefix="ffmpeg-assembly")

    def assemble(self, request: VideoAssemblyRequest, output_path: str) -> None:
        future = self._executor.submit(self._run_pipeline, request, output_path)
        try:
            future.result(timeout=self._timeout_seconds)
        except FutureTimeoutError as exc:
            raise AssemblyEngineError(f"ffmpeg assembly timed out after {self._timeout_seconds}s") from exc
        except (InvalidSceneIndexError, AssemblyEngineError):
            raise
        except Exception as exc:  # noqa: BLE001 — any engine failure becomes a domain error
            logger.exception("ffmpeg assembly failed")
            raise AssemblyEngineError(str(exc)) from exc

    def _run_pipeline(self, request: VideoAssemblyRequest, output_path: str) -> None:
        scenes = self._sorted_validated_scenes(request.scenes)

        self._inspector.validate_consistent([scene.clip_path for scene in scenes])

        ensure_parent_dir(tmp_muxed_path(request.project_id, 0))
        muxed_paths = [self._mux_scene(request.project_id, scene) for scene in scenes]

        concatenated_path = (
            output_path
            if request.background_music_path is None
            else tmp_concatenated_path(request.project_id)
        )
        self._concat(request.project_id, muxed_paths, concatenated_path)

        if request.background_music_path is not None:
            self._overlay_background_music(concatenated_path, request.background_music_path, output_path)

        cleanup_tmp_dir(request.project_id)

    @staticmethod
    def _sorted_validated_scenes(scenes):
        ordered = sorted(scenes, key=lambda scene: scene.scene_index)
        expected_indices = list(range(len(ordered)))
        actual_indices = [scene.scene_index for scene in ordered]
        if actual_indices != expected_indices:
            raise InvalidSceneIndexError(
                f"scene_index values must be a contiguous 0-based sequence, got {actual_indices}"
            )
        return ordered

    def _mux_scene(self, project_id: str, scene) -> str:
        output = tmp_muxed_path(project_id, scene.scene_index)
        self._run_ffmpeg(
            [
                "-y",
                "-i",
                scene.clip_path,
                "-i",
                scene.audio_path,
                "-map",
                "0:v",
                "-map",
                "1:a",
                "-c:v",
                "copy",
                "-c:a",
                "aac",
                output,
            ]
        )
        return output

    def _concat(self, project_id: str, muxed_paths: list[str], output_path: str) -> None:
        concat_list_path = tmp_concat_list_path(project_id)
        with open(concat_list_path, "w") as concat_list_file:
            for path in muxed_paths:
                concat_list_file.write(f"file '{path}'\n")

        self._run_ffmpeg(
            ["-y", "-f", "concat", "-safe", "0", "-i", concat_list_path, "-c", "copy", output_path]
        )

    def _overlay_background_music(
        self, video_path: str, background_music_path: str, output_path: str
    ) -> None:
        self._run_ffmpeg(
            [
                "-y",
                "-i",
                video_path,
                "-stream_loop",
                "-1",
                "-i",
                background_music_path,
                "-filter_complex",
                f"[1:a]volume={BACKGROUND_MUSIC_VOLUME}[bg];[0:a][bg]amix=inputs=2:duration=first[aout]",
                "-map",
                "0:v",
                "-map",
                "[aout]",
                "-c:v",
                "copy",
                "-shortest",
                output_path,
            ]
        )

    @staticmethod
    def _run_ffmpeg(args: list[str]) -> None:
        result = subprocess.run([FFMPEG_BINARY, *args], capture_output=True, text=True)
        if result.returncode != 0:
            raise AssemblyEngineError(f"ffmpeg exited with code {result.returncode}: {result.stderr}")
