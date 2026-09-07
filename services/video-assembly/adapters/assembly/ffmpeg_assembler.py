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
import os
import subprocess
from concurrent.futures import ThreadPoolExecutor
from concurrent.futures import TimeoutError as FutureTimeoutError

from adapters.assembly.subtitle_file import write_subtitle_file
from domain.errors import AssemblyEngineError
from domain.models import SubtitleStyle, VideoAssemblyRequest
from domain.ports import VideoAssemblerPort

logger = logging.getLogger(__name__)

# Sized from the Phase 0 benchmark (long-form-baseline.md): burning subtitles
# into a 215s 1080p60 video took 24.8s, i.e. roughly 0.115x the video's
# duration, so a 10-minute video lands near 70s. 900s leaves ~12x headroom —
# a timeout only matters when something has already gone wrong, so it is cheap
# to set it generously.
DEFAULT_ASSEMBLY_TIMEOUT_SECONDS = 900
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
        audio_map = None

        if n > 0:
            narration_inputs = "".join(f"[{i + 1}:a]" for i in range(n))
            filter_parts.append(f"{narration_inputs}concat=n={n}:v=0:a=1[narration]")
            audio_map = "[narration]"

        if request.background_music_path is not None:
            bg_index = n + 1
            cmd += ["-stream_loop", "-1", "-i", request.background_music_path]
            filter_parts.append(f"[{bg_index}:a]volume={BACKGROUND_MUSIC_VOLUME}[bg]")
            if audio_map is None:
                # Narration is disabled: background music is the only track, so
                # it must stop with the video rather than loop forever.
                audio_map = "[bg]"
            else:
                filter_parts.append("[narration][bg]amix=inputs=2:duration=first[aout]")
                audio_map = "[aout]"

        video_map = "0:v"
        video_codec = ["-c:v", "copy"]
        if request.subtitle_cues:
            subtitle_path = self._write_subtitles(request, output_path)
            filter_parts.append(f"[0:v]subtitles={_escape_filter_path(subtitle_path)}[vout]")
            video_map = "[vout]"
            # Burning subtitles paints new pixels, so the video stream has to be
            # re-encoded — it can no longer be stream-copied.
            video_codec = ["-c:v", "libx264", "-preset", "medium", "-crf", "23"]

        cmd += ["-filter_complex", ";".join(filter_parts)] if filter_parts else []
        cmd += ["-map", video_map]
        if audio_map is not None:
            cmd += ["-map", audio_map]
        else:
            cmd += ["-an"]
        cmd += [*video_codec, "-shortest", output_path]
        self._run_ffmpeg(cmd)

    @staticmethod
    def _write_subtitles(request: VideoAssemblyRequest, output_path: str) -> str:
        style = request.subtitle_style or SubtitleStyle()
        subtitle_path = os.path.join(os.path.dirname(output_path), f"{request.project_id}.ass")
        write_subtitle_file(request.subtitle_cues or [], style, subtitle_path)
        return subtitle_path

    @staticmethod
    def _run_ffmpeg(args: list[str]) -> None:
        result = subprocess.run([FFMPEG_BINARY, *args], capture_output=True, text=True)
        if result.returncode != 0:
            raise AssemblyEngineError(f"ffmpeg exited with code {result.returncode}: {result.stderr}")


def _escape_filter_path(path: str) -> str:
    """Inside a filtergraph, ffmpeg reads ':' as an option separator and '\\' as
    an escape, so a bare path breaks the `subtitles=` filter."""
    escaped = path.replace("\\", "\\\\").replace(":", "\\:").replace("'", "\\'")
    return f"'{escaped}'"
