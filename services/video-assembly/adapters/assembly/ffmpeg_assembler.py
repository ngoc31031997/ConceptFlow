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

from adapters.assembly.srt_file import write_srt_file
from adapters.assembly.subtitle_file import (
    DEFAULT_PLAY_RES_X,
    DEFAULT_PLAY_RES_Y,
    write_subtitle_file,
)
from domain.errors import AssemblyEngineError
from domain.models import (
    NarrationSegment,
    SubtitleCue,
    SubtitleStyle,
    VideoAssemblyRequest,
)
from domain.ports import VideoAssemblerPort

logger = logging.getLogger(__name__)

# Sized from the Phase 0 benchmark (long-form-baseline.md): burning subtitles
# into a 215s 1080p60 video took 24.8s, i.e. roughly 0.115x the video's
# duration, so a 10-minute video lands near 70s. 900s leaves ~12x headroom —
# a timeout only matters when something has already gone wrong, so it is cheap
# to set it generously.
DEFAULT_ASSEMBLY_TIMEOUT_SECONDS = 900
FFMPEG_BINARY = "ffmpeg"
DEFAULT_BACKGROUND_MUSIC_VOLUME = 0.2

# Sidechain ducking (CR-005 FR14.1). A flat mix leaves music competing with the
# narration when it is loud and leaves silence when it is not; ducking pulls the
# music down only while someone is speaking, which is what every produced video
# does. threshold/ratio/attack/release are conventional voice-over values.
DUCKING_FILTER = "sidechaincompress=threshold=0.05:ratio=8:attack=20:release=400"

# YouTube normalizes everything it serves to about -14 LUFS. A quieter master
# does not get left quiet — it gets turned up, along with its noise floor — so
# matching the target here is what keeps the channel from sounding weaker than
# everyone else's (CR-005 FR14.3).
LOUDNORM_FILTER = "loudnorm=I=-14:TP=-1.5:LRA=11"

# Optional silence before the first line and after the last (CR-005 FR14.5).
#
# Both default to OFF, which is a deliberate trade rather than an oversight.
# Padding either end means tpad, tpad repaints frames, and repainting frames
# rules out `-c:v copy` — which Phase 0 measured at 0.1s against 24.8s for a
# re-encode of the same clip. Paying 250x the assembly time for half a second
# of silence is a bad default, especially since the Creator already controls
# how their animation opens and closes.
#
# A Creator who wants that polish can turn it on with ASSEMBLY_LEAD_IN_SECONDS
# / ASSEMBLY_TAIL_SECONDS and accept the re-encode.
DEFAULT_LEAD_IN_SECONDS = 0.0
DEFAULT_TAIL_SECONDS = 0.0

# Auto-generated thumbnail candidate (CR-006 FR16). YouTube wants 1280x720 and
# under 2MB; q=3 lands comfortably inside that for a Manim frame, which is flat
# colour and text rather than photographic detail.
THUMBNAIL_WIDTH = 1280
THUMBNAIL_HEIGHT = 720
THUMBNAIL_QUALITY = 3

# Grabbed a quarter of the way in rather than at the start: the opening frames
# of a Manim video are usually a title card or an empty stage, which makes a
# poor thumbnail. A quarter in is reliably inside the actual content.
THUMBNAIL_POSITION_FRACTION = 0.25

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
    def __init__(
        self,
        timeout_seconds: int = DEFAULT_ASSEMBLY_TIMEOUT_SECONDS,
        lead_in_seconds: float = DEFAULT_LEAD_IN_SECONDS,
        tail_seconds: float = DEFAULT_TAIL_SECONDS,
    ) -> None:
        self._timeout_seconds = timeout_seconds
        self._lead_in_seconds = lead_in_seconds
        self._tail_seconds = tail_seconds
        self._executor = ThreadPoolExecutor(max_workers=2, thread_name_prefix="ffmpeg-assembly")

    def assemble(self, request: VideoAssemblyRequest, output_path: str) -> str | None:
        future = self._executor.submit(self._run_pipeline, request, output_path)
        try:
            return future.result(timeout=self._timeout_seconds)
        except FutureTimeoutError as exc:
            raise AssemblyEngineError(f"ffmpeg assembly timed out after {self._timeout_seconds}s") from exc
        except AssemblyEngineError:
            raise
        except Exception as exc:  # noqa: BLE001 — any engine failure becomes a domain error
            logger.exception("ffmpeg assembly failed")
            raise AssemblyEngineError(str(exc)) from exc

    def _run_pipeline(self, request: VideoAssemblyRequest, output_path: str) -> str | None:
        segments = sorted(request.narration_segments, key=lambda s: s.start_time)
        n = len(segments)

        # Lead-in shifts the video, the narration and the subtitles by the SAME
        # amount, so the alignment CR-002 established is untouched — narration i
        # still lands exactly on wait i, just half a second later in the file.
        # Shifting only the audio here would reintroduce the very desync that
        # CR-002 existed to remove.
        #
        # CR-023: effective_lead_in folds the channel intro's own duration into
        # that same single shift (D5) — a project with an intro spliced in
        # front needs narration/subtitles pushed later by the intro's length
        # too, on top of whatever ASSEMBLY_LEAD_IN_SECONDS already added. This
        # is the only source-of-truth change; every consumer below is
        # unchanged from when it read `lead_in`.
        base_lead_in = self._lead_in_seconds if request.video_duration_seconds > 0 else 0.0
        effective_lead_in = base_lead_in + (request.intro_duration_seconds or 0.0)

        cmd = ["-y", "-i", request.video_path]
        for segment in segments:
            cmd += ["-i", segment.audio_path]

        target_duration = self._target_duration(request, segments)
        if target_duration is not None:
            # Lead-in pushes everything later; the tail leaves the closing frame
            # up briefly instead of cutting hard (FR14.5).
            target_duration += effective_lead_in + self._tail_seconds

        filter_parts: list[str] = []
        audio_map = None

        if n > 0:
            # One adelay per segment, then mix. `all=1` applies the delay to
            # every channel, so this works for mono and stereo alike, and
            # `normalize=0` is essential: amix otherwise divides each input's
            # volume by the number of inputs, which would make a 20-segment
            # narration 20x too quiet.
            for i, segment in enumerate(segments):
                delay_ms = int(round((segment.start_time + effective_lead_in) * 1000))
                filter_parts.append(f"[{i + 1}:a]adelay={delay_ms}:all=1[na{i}]")
            mix_inputs = "".join(f"[na{i}]" for i in range(n))
            if n == 1:
                filter_parts.append("[na0]anull[narration]")
            else:
                filter_parts.append(f"{mix_inputs}amix=inputs={n}:normalize=0[narration]")
            audio_map = "[narration]"

        if request.background_music_path is not None:
            bg_index = n + 1
            volume = request.background_music_volume
            cmd += ["-stream_loop", "-1", "-i", request.background_music_path]
            filter_parts.append(f"[{bg_index}:a]volume={volume}[bg]")
            if audio_map is None:
                # Narration is disabled: background music is the only track. It
                # loops forever, so the output duration cap below is what stops
                # it. Nothing to duck against either.
                audio_map = "[bg]"
            else:
                # Duck the music against the narration before mixing, so the
                # voice stays intelligible without the music dropping out
                # entirely between lines.
                # ffmpeg consumes a filtergraph label exactly once, and the
                # narration is needed twice here: as the sidechain key and as a
                # voice in the final mix. `asplit` hands out the two copies --
                # reusing [narration] instead makes ffmpeg read the second use
                # as an input stream specifier and abort.
                filter_parts.append("[narration]asplit=2[navoice][nakey]")
                filter_parts.append(f"[bg][nakey]{DUCKING_FILTER}[ducked]")
                filter_parts.append("[navoice][ducked]amix=inputs=2:duration=first[aout]")
                audio_map = "[aout]"

        video_map = "0:v"
        video_codec = ["-c:v", "copy"]
        video_filters: list[str] = []

        if target_duration is not None:
            # CR-002 FR10.6: hold the last frame rather than truncating a final
            # narration that runs past the end of the animation.
            pad_seconds = target_duration - request.video_duration_seconds - effective_lead_in
            if effective_lead_in > VIDEO_PAD_EPSILON_SECONDS:
                video_filters.append(
                    f"tpad=start_mode=clone:start_duration={effective_lead_in:.3f}"
                )
            if pad_seconds > VIDEO_PAD_EPSILON_SECONDS:
                video_filters.append(f"tpad=stop_mode=clone:stop_duration={pad_seconds:.3f}")
            if audio_map is not None:
                # Cover any silence between the last narration and the end, so
                # the audio stream runs the full length of the video.
                filter_parts.append(f"{audio_map}apad=whole_dur={target_duration:.3f}[apadded]")
                audio_map = "[apadded]"

        # Shifted exactly once, here, regardless of which serializer(s) below
        # end up using the result — two serializers each applying lead_in
        # themselves is how one of them quietly ends up wrong (ADR-0027).
        cues = request.subtitle_cues or []
        if cues and effective_lead_in:
            cues = [cue.shifted_by(effective_lead_in) for cue in cues]

        caption_path: str | None = None
        if cues and request.subtitle_mode in ("burn_in", "both"):
            subtitle_path = self._write_ass(request, cues, output_path)
            video_filters.append(f"subtitles={_escape_filter_path(subtitle_path)}")
            # Burning subtitles paints new pixels, so the video stream has to be
            # re-encoded — it can no longer be stream-copied.
            video_codec = list(VIDEO_ENCODE_ARGS)
        if cues and request.subtitle_mode in ("track", "both"):
            caption_path = self._write_srt(request, cues, output_path)

        if video_filters:
            if video_codec == ["-c:v", "copy"]:
                # tpad also paints frames, so a stream copy is impossible here
                # too. (Without subtitles this is the only reason to re-encode.)
                video_codec = list(VIDEO_ENCODE_ARGS)
            filter_parts.append(f"[0:v]{','.join(video_filters)}[vout]")
            video_map = "[vout]"

        if audio_map is not None:
            # Last link in the audio chain, so it normalises whatever the mix
            # ended up being — narration alone, or narration plus ducked music.
            filter_parts.append(f"{audio_map}{LOUDNORM_FILTER}[anorm]")
            audio_map = "[anorm]"

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

        has_channel_assets = bool(request.intro_video_path or request.outro_video_path)
        # CR-023 D5: with an intro/outro to splice in, this pass produces only
        # the main segment, to a temp path — _concat_channel_assets below
        # joins it with the real intro/outro files afterward. Without either,
        # this pass's output IS the final file, exactly as before this CR.
        main_target = _main_segment_path(output_path) if has_channel_assets else output_path
        cmd += [main_target]
        self._run_ffmpeg(cmd)

        if has_channel_assets:
            self._concat_channel_assets(request, main_target, output_path)

        self._write_thumbnail_candidate(output_path, target_duration)
        return caption_path

    @staticmethod
    def _concat_channel_assets(request: VideoAssemblyRequest, main_path: str, output_path: str) -> None:
        """Splices the channel's real intro/outro files around the just-built
        main segment (CR-023 D5) via the concat demuxer, in one re-encode
        pass — accepted cost (LLD's "Rủi ro" section): the main segment may
        already have lost `-c:v copy` to tpad/subtitles, and concat forces a
        matching re-encode of the intro/outro anyway since nothing here tries
        to keep them byte-identical to the main segment's codec profile.
        """
        segments = [p for p in (request.intro_video_path, main_path, request.outro_video_path) if p]
        concat_list_path = f"{output_path}.concat.txt"
        with open(concat_list_path, "w", encoding="utf-8") as f:
            for segment_path in segments:
                f.write(f"file {_escape_concat_path(segment_path)}\n")

        try:
            FfmpegVideoAssembler._run_ffmpeg(
                [
                    "-y",
                    "-f", "concat",
                    "-safe", "0",
                    "-i", concat_list_path,
                    *VIDEO_ENCODE_ARGS,
                    *AUDIO_ENCODE_ARGS,
                    *CONTAINER_ARGS,
                    output_path,
                ]
            )
        finally:
            # Best-effort cleanup: in production ffmpeg always wrote main_path,
            # but nothing here should fail the whole assembly over a missing
            # temp file (mirrors _write_thumbnail_candidate's best-effort
            # stance elsewhere in this class).
            _remove_if_exists(concat_list_path)
            _remove_if_exists(main_path)

    def _write_thumbnail_candidate(self, video_path: str, duration: float | None) -> None:
        """Extracts a still the Creator can use as a thumbnail (CR-006 FR16.1).

        Deliberately no text overlay. The title does not exist yet at assembly
        time — it is drafted later, at publish — so anything burned in here
        would be guesswork. The Creator gets a usable frame and can replace it
        with their own image, which the existing upload path already supports.

        Best-effort: a missing thumbnail is a small inconvenience, while
        failing an assembled video over one would not be.
        """
        if not duration or duration <= 0:
            return

        thumbnail_dir = os.path.join(os.path.dirname(video_path), "..", "thumbnail")
        thumbnail_dir = os.path.normpath(thumbnail_dir)
        thumbnail_path = os.path.join(thumbnail_dir, "auto.jpg")
        if os.path.exists(thumbnail_path):
            return

        try:
            os.makedirs(thumbnail_dir, exist_ok=True)
            self._run_ffmpeg([
                "-y",
                "-ss", f"{duration * THUMBNAIL_POSITION_FRACTION:.3f}",
                "-i", video_path,
                "-frames:v", "1",
                "-vf", f"scale={THUMBNAIL_WIDTH}:{THUMBNAIL_HEIGHT}:force_original_aspect_ratio=increase,"
                       f"crop={THUMBNAIL_WIDTH}:{THUMBNAIL_HEIGHT}",
                "-q:v", str(THUMBNAIL_QUALITY),
                thumbnail_path,
            ])
        except AssemblyEngineError:
            logger.warning("Could not generate a thumbnail candidate for %s", video_path)

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
    def _write_ass(request: VideoAssemblyRequest, cues: list[SubtitleCue], output_path: str) -> str:
        """Burn-in track. `cues` are expected already shifted by any lead-in
        (see _run_pipeline) — this method does not touch timestamps."""
        style = request.subtitle_style or SubtitleStyle()
        subtitle_path = os.path.join(os.path.dirname(output_path), f"{request.project_id}.ass")
        write_subtitle_file(
            cues,
            style,
            subtitle_path,
            play_res=_probe_resolution(request.video_path),
        )
        return subtitle_path

    @staticmethod
    def _write_srt(request: VideoAssemblyRequest, cues: list[SubtitleCue], output_path: str) -> str:
        """Caption-track file for Publisher to upload (CR-015 FR38). `cues`
        are expected already shifted, same as _write_ass — no `SubtitleStyle`
        here, SRT has no styling fields (FR38.3)."""
        caption_path = os.path.join(os.path.dirname(output_path), f"{request.project_id}.srt")
        write_srt_file(cues, caption_path)
        return caption_path

    @staticmethod
    def _run_ffmpeg(args: list[str]) -> None:
        result = subprocess.run([FFMPEG_BINARY, *args], capture_output=True, text=True)
        if result.returncode != 0:
            raise AssemblyEngineError(f"ffmpeg exited with code {result.returncode}: {result.stderr}")


def _remove_if_exists(path: str) -> None:
    try:
        os.remove(path)
    except FileNotFoundError:
        pass


def _main_segment_path(output_path: str) -> str:
    """Temp path for the main-only segment when an intro/outro will be
    concatenated around it (CR-023 D5) — sits next to output_path so it is
    on the same filesystem/volume as the concat demuxer's other inputs."""
    return f"{output_path}.main.mp4"


def _escape_concat_path(path: str) -> str:
    """The concat demuxer reads each `file` line like a mini shell string —
    a literal single quote in the path has to be closed out and re-opened
    (ffmpeg's own documented idiom for this) or the list file is invalid."""
    escaped = path.replace("'", "'\\''")
    return f"'{escaped}'"


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
