"""FfmpegVideoAssembler — implements VideoAssemblerPort by shelling out to
the ffmpeg CLI binary in a single pass: place each narration segment at its
own offset on one track, mux that onto the (silent) video, and overlay
optional background music underneath — all via one filter_complex graph.

The offsets matter (CR-002). This used to `concat` the narration clips back
to back, which assumed the video was nothing but narration. It is not: a
Manim video is animation time *plus* narration time, so every segment after
the first played early by however much animation had run before it, and the
error accumulated — measured at 61.6s of drift by the end of a 3.6-minute
reference video. Each segment now gets an `adelay` to the offset Rendering
measured for it.

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

from adapters.assembly.subtitle_file import (
    DEFAULT_PLAY_RES_X,
    DEFAULT_PLAY_RES_Y,
    write_subtitle_file,
)
from domain.errors import AssemblyEngineError
from domain.models import NarrationSegment, SubtitleStyle, VideoAssemblyRequest
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

# Below this, padding the video is not worth re-encoding it for.
VIDEO_PAD_EPSILON_SECONDS = 0.05

# Upload-grade x264 settings (CR-004 FR12.2). YouTube transcodes whatever it is
# given, so the source has to carry spare quality into that second encode.
#
# The Phase 0 benchmark makes this nearly free: slow/crf18 measured 25.9s
# against medium/crf23's 24.8s on the same 215s 1080p60 clip — 4% slower for
# 28% more bitrate.
#
# yuv420p and +faststart are not optional extras: without the first some
# players and platforms refuse the file outright, and without the second the
# moov atom sits at the end, so every player must fetch the whole file before
# it can start.
VIDEO_ENCODE_ARGS = [
    "-c:v", "libx264",
    "-preset", "slow",
    "-crf", "18",
    "-pix_fmt", "yuv420p",
    "-profile:v", "high",
    "-bf", "2",
]
AUDIO_ENCODE_ARGS = ["-c:a", "aac", "-b:a", "192k", "-ar", "48000"]
CONTAINER_ARGS = ["-movflags", "+faststart"]


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
        segments = sorted(request.narration_segments, key=lambda s: s.start_time)
        n = len(segments)

        cmd = ["-y", "-i", request.video_path]
        for segment in segments:
            cmd += ["-i", segment.audio_path]

        target_duration = self._target_duration(request, segments)

        filter_parts: list[str] = []
        audio_map = None

        if n > 0:
            # One adelay per segment, then mix. `all=1` applies the delay to
            # every channel, so this works for mono and stereo alike, and
            # `normalize=0` is essential: amix otherwise divides each input's
            # volume by the number of inputs, which would make a 20-segment
            # narration 20x too quiet.
            for i, segment in enumerate(segments):
                delay_ms = int(round(segment.start_time * 1000))
                filter_parts.append(f"[{i + 1}:a]adelay={delay_ms}:all=1[na{i}]")
            mix_inputs = "".join(f"[na{i}]" for i in range(n))
            if n == 1:
                filter_parts.append("[na0]anull[narration]")
            else:
                filter_parts.append(f"{mix_inputs}amix=inputs={n}:normalize=0[narration]")
            audio_map = "[narration]"

        if request.background_music_path is not None:
            bg_index = n + 1
            cmd += ["-stream_loop", "-1", "-i", request.background_music_path]
            filter_parts.append(f"[{bg_index}:a]volume={BACKGROUND_MUSIC_VOLUME}[bg]")
            if audio_map is None:
                # Narration is disabled: background music is the only track. It
                # loops forever, so the output duration cap below is what stops
                # it.
                audio_map = "[bg]"
            else:
                # duration=first keeps the mix as long as the narration track,
                # which apad has already stretched to the full target.
                filter_parts.append("[narration][bg]amix=inputs=2:duration=first[aout]")
                audio_map = "[aout]"

        video_map = "0:v"
        video_codec = ["-c:v", "copy"]
        video_filters: list[str] = []

        if target_duration is not None:
            # CR-002 FR10.6: hold the last frame rather than truncating a final
            # narration that runs past the end of the animation.
            pad_seconds = target_duration - request.video_duration_seconds
            if pad_seconds > VIDEO_PAD_EPSILON_SECONDS:
                video_filters.append(f"tpad=stop_mode=clone:stop_duration={pad_seconds:.3f}")
            if audio_map is not None:
                # Cover any silence between the last narration and the end, so
                # the audio stream runs the full length of the video.
                filter_parts.append(f"{audio_map}apad=whole_dur={target_duration:.3f}[apadded]")
                audio_map = "[apadded]"

        if request.subtitle_cues:
            subtitle_path = self._write_subtitles(request, output_path)
            video_filters.append(f"subtitles={_escape_filter_path(subtitle_path)}")
            # Burning subtitles paints new pixels, so the video stream has to be
            # re-encoded — it can no longer be stream-copied.
            video_codec = list(VIDEO_ENCODE_ARGS)

        if video_filters:
            if video_codec == ["-c:v", "copy"]:
                # tpad also paints frames, so a stream copy is impossible here
                # too. (Without subtitles this is the only reason to re-encode.)
                video_codec = list(VIDEO_ENCODE_ARGS)
            filter_parts.append(f"[0:v]{','.join(video_filters)}[vout]")
            video_map = "[vout]"

        cmd += ["-filter_complex", ";".join(filter_parts)] if filter_parts else []
        cmd += ["-map", video_map]
        if audio_map is not None:
            cmd += ["-map", audio_map]
        else:
            cmd += ["-an"]
        cmd += video_codec
        if video_codec != ["-c:v", "copy"]:
            # Keyframe every 2 seconds, which is what YouTube's ingestion
            # guidance asks for. Only meaningful when actually encoding.
            cmd += ["-g", str(int(round(_probe_frame_rate(request.video_path) * 2)))]
        if audio_map is not None:
            cmd += AUDIO_ENCODE_ARGS
        cmd += CONTAINER_ARGS
        if target_duration is not None:
            # An explicit cap replaces the old `-shortest`, which would have cut
            # the closing narration off whenever it outlasted the animation.
            cmd += ["-t", f"{target_duration:.3f}"]
        else:
            cmd += ["-shortest"]
        cmd += [output_path]
        self._run_ffmpeg(cmd)

    @staticmethod
    def _target_duration(
        request: VideoAssemblyRequest, segments: list[NarrationSegment]
    ) -> float | None:
        """How long the finished video should be, or None when Rendering did
        not report a duration (a pre-CR-002 project), in which case the old
        `-shortest` behaviour is kept rather than guessing.

        It is the longer of the rendered animation and the end of the last
        narration — the latter can win because a narration segment's audio may
        run slightly past the `self.wait()` that was sized for it.
        """
        if request.video_duration_seconds <= 0:
            return None
        last_narration_end = max(
            (s.start_time + _probe_audio_duration(s.audio_path) for s in segments),
            default=0.0,
        )
        return max(request.video_duration_seconds, last_narration_end)

    @staticmethod
    def _write_subtitles(request: VideoAssemblyRequest, output_path: str) -> str:
        style = request.subtitle_style or SubtitleStyle()
        subtitle_path = os.path.join(os.path.dirname(output_path), f"{request.project_id}.ass")
        write_subtitle_file(
            request.subtitle_cues or [],
            style,
            subtitle_path,
            play_res=_probe_resolution(request.video_path),
        )
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


def _probe_audio_duration(audio_path: str) -> float:
    """Length of one narration clip. Returns 0.0 rather than raising: a clip
    whose duration cannot be read should not fail the whole assembly, it just
    means that clip cannot extend the target duration."""
    result = subprocess.run(
        [
            "ffprobe", "-v", "error",
            "-show_entries", "format=duration",
            "-of", "csv=p=0",
            audio_path,
        ],
        capture_output=True,
        text=True,
    )
    if result.returncode != 0:
        logger.warning("ffprobe could not read %s: %s", audio_path, result.stderr.strip())
        return 0.0
    try:
        return float(result.stdout.strip())
    except ValueError:
        logger.warning("ffprobe returned an unreadable duration for %s", audio_path)
        return 0.0


def _probe_frame_rate(video_path: str) -> float:
    """Frame rate of the rendered video, for sizing the keyframe interval.

    Falls back to 30 rather than raising: a missing frame rate should cost a
    slightly off keyframe interval, not the whole assembly.
    """
    result = subprocess.run(
        [
            "ffprobe", "-v", "error",
            "-select_streams", "v:0",
            "-show_entries", "stream=r_frame_rate",
            "-of", "csv=p=0",
            video_path,
        ],
        capture_output=True,
        text=True,
    )
    raw = result.stdout.strip()
    if result.returncode != 0 or not raw:
        logger.warning("ffprobe could not read the frame rate of %s", video_path)
        return 30.0
    try:
        # ffprobe reports it as a rational, e.g. "60/1".
        numerator, _, denominator = raw.partition("/")
        rate = float(numerator) / float(denominator or 1)
        return rate if rate > 0 else 30.0
    except (ValueError, ZeroDivisionError):
        logger.warning("unreadable frame rate %r for %s", raw, video_path)
        return 30.0


def _probe_resolution(video_path: str) -> tuple[int, int]:
    """Frame size of the rendered video, so ASS can declare a matching PlayRes.

    Falls back to 1080p rather than raising: a wrong subtitle scale is a
    cosmetic problem, an aborted assembly is not.
    """
    result = subprocess.run(
        [
            "ffprobe", "-v", "error",
            "-select_streams", "v:0",
            "-show_entries", "stream=width,height",
            "-of", "csv=p=0",
            video_path,
        ],
        capture_output=True,
        text=True,
    )
    raw = result.stdout.strip()
    if result.returncode != 0 or not raw:
        logger.warning("ffprobe could not read the resolution of %s", video_path)
        return (DEFAULT_PLAY_RES_X, DEFAULT_PLAY_RES_Y)
    try:
        width, height = (int(part) for part in raw.split(",")[:2])
        return (width, height) if width > 0 and height > 0 else (DEFAULT_PLAY_RES_X, DEFAULT_PLAY_RES_Y)
    except ValueError:
        logger.warning("unreadable resolution %r for %s", raw, video_path)
        return (DEFAULT_PLAY_RES_X, DEFAULT_PLAY_RES_Y)
