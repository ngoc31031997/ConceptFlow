"""FfmpegVideoAssembler — implements VideoAssemblerPort by shelling out to
the ffmpeg CLI binary in a single pass: concatenate the ordered narration
audio_segments into one track, mux it onto the (silent) video, and overlay
optional background music underneath — all via one filter_complex graph.

Runs in a threadpool so it never blocks the asyncio event loop (NFR
Requirements, Performance), with a bounded timeout so a hung ffmpeg call
surfaces as a clear AssemblyEngineError instead of hanging the caller
forever.
"""

from __future__ import annotations

import logging
import subprocess
from concurrent.futures import ThreadPoolExecutor
from concurrent.futures import TimeoutError as FutureTimeoutError

from domain.errors import AssemblyEngineError
from domain.models import VideoAssemblyRequest
from domain.ports import VideoAssemblerPort

logger = logging.getLogger(__name__)

DEFAULT_ASSEMBLY_TIMEOUT_SECONDS = 180
FFMPEG_BINARY = "ffmpeg"
BACKGROUND_MUSIC_VOLUME = 0.2


class FfmpegVideoAssembler(VideoAssemblerPort):
    def __init__(self, timeout_seconds: int = DEFAULT_ASSEMBLY_TIMEOUT_SECONDS) -> None:
        self._timeout_seconds = timeout_seconds
        self._executor = ThreadPoolExecutor(max_workers=2, thread_name_prefix="ffmpeg-assembly")

    def assemble(self, request: VideoAssemblyRequest, output_path: str) -> None:
        future = self._executor.submit(self._run_pipeline, request, output_path)
        try:
            future.result(timeout=self._timeout_seconds)
        except FutureTimeoutError as exc:
            raise AssemblyEngineError(f"ffmpeg assembly timed out after {self._timeout_seconds}s") from exc
        except AssemblyEngineError:
            raise
        except Exception as exc:  # noqa: BLE001 — any engine failure becomes a domain error
            logger.exception("ffmpeg assembly failed")
            raise AssemblyEngineError(str(exc)) from exc

    def _run_pipeline(self, request: VideoAssemblyRequest, output_path: str) -> None:
        n = len(request.audio_segments)

        cmd = ["-y", "-i", request.video_path]
        for audio_path in request.audio_segments:
            cmd += ["-i", audio_path]

        filter_parts = []
        narration_inputs = "".join(f"[{i + 1}:a]" for i in range(n))
        filter_parts.append(f"{narration_inputs}concat=n={n}:v=0:a=1[narration]")

        if request.background_music_path is not None:
            bg_index = n + 1
            cmd += ["-stream_loop", "-1", "-i", request.background_music_path]
            filter_parts.append(f"[{bg_index}:a]volume={BACKGROUND_MUSIC_VOLUME}[bg]")
            filter_parts.append("[narration][bg]amix=inputs=2:duration=first[aout]")
            audio_map = "[aout]"
        else:
            audio_map = "[narration]"

        cmd += [
            "-filter_complex",
            ";".join(filter_parts),
            "-map",
            "0:v",
            "-map",
            audio_map,
            "-c:v",
            "copy",
            "-shortest",
            output_path,
        ]
        self._run_ffmpeg(cmd)

    @staticmethod
    def _run_ffmpeg(args: list[str]) -> None:
        result = subprocess.run([FFMPEG_BINARY, *args], capture_output=True, text=True)
        if result.returncode != 0:
            raise AssemblyEngineError(f"ffmpeg exited with code {result.returncode}: {result.stderr}")
