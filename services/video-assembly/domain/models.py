"""Domain value objects for the Video Assembly Service.

Manim-script input mode: Rendering produces one silent video for the whole
project (not per-scene clips), so assembly's job is to lay each narration
segment onto that video at its own offset, and optionally overlay
background music. There is no per-scene clip:audio pairing or
format-consistency check (only one video, no clips to compare).

The video's timeline is animation time *plus* narration time, so laying the
audio end to end would make every segment after the first play early, by the
total animation time that had run before it. Each segment therefore carries
the offset Rendering measured for it.
"""

from __future__ import annotations

from dataclasses import dataclass


@dataclass(frozen=True)
class SubtitleCue:
    """One narration line and the window it stays on screen.

    Timings come from the Orchestrator — real synthesized-audio durations when
    narration is on, estimated reading time when it is off — so subtitles stay
    in step with the animation either way.
    """

    scene_index: int
    text: str
    start_time: float
    end_time: float

    def shifted_by(self, seconds: float) -> SubtitleCue:
        """Returns a copy moved later in the timeline by `seconds` (ADR-0027).

        This is the ONLY place a subtitle timestamp gets shifted. The .ass and
        .srt serializers both receive already-shifted cues and never see a
        raw offset themselves — two serializers each doing their own shift
        arithmetic is how one of them ends up silently wrong; the burn-in
        track would still look right while the caption track drifted by
        exactly the offset, with no test positioned to catch it.
        """
        if not seconds:
            return self
        return SubtitleCue(
            scene_index=self.scene_index,
            text=self.text,
            start_time=self.start_time + seconds,
            end_time=self.end_time + seconds,
        )


@dataclass(frozen=True)
class SubtitleStyle:
    """Creator-chosen subtitle appearance."""

    font_size: str = "medium"  # small | medium | large
    text_color: str = "#FFFFFF"
    background_opacity: float = 0.6  # 0.0 = no box behind the text
    position: str = "bottom"  # bottom | top
    # One of adapters/assembly/subtitle_file.SUBTITLE_FONTS. The default is the
    # font every subtitle used before this field existed.
    font_family: str = "DejaVu Sans"


@dataclass(frozen=True)
class NarrationSegment:
    """One narration clip and where it belongs in the rendered video.

    start_time is the offset Rendering measured for the matching
    `self.wait(AUTO)` — not the running total of the
    preceding narration durations.
    """

    audio_path: str
    start_time: float


@dataclass(frozen=True)
class VideoAssemblyRequest:
    """Input to video assembly: one project's rendered (silent) video, its
    narration segments with their offsets, plus optional background music and
    subtitles.

    narration_segments is empty when the Creator disabled narration —
    the result is then a silent video, or one carrying background music alone.

    video_duration_seconds is Rendering's measured length of video_path. It is
    what assembly pads against so a final narration that runs past the last
    frame is not cut off; 0.0 means "unknown", in which case no
    padding is attempted.
    """

    project_id: str
    video_path: str
    narration_segments: list[NarrationSegment]
    video_duration_seconds: float = 0.0
    background_music_path: str | None = None
    # Creator-chosen music level, 0.0-1.0.
    background_music_volume: float = 0.2
    subtitle_cues: list[SubtitleCue] | None = None
    subtitle_style: SubtitleStyle | None = None
    # How subtitle_cues get delivered (see ADR-0027):
    #   off      — no subtitles at all
    #   track    — .srt only, for upload as a YouTube caption track
    #   burn_in  — .ass only, painted into the video frames
    #   both     — both, with the risk of doubled text a Creator who picks
    #              this has been warned about
    # Defaults to "burn_in" rather than the GUI's "track" default so a
    # command that carries no subtitle_mode keeps being burned in.
    subtitle_mode: str = "burn_in"
    # The channel's fixed intro/outro, resolved by
    # application/assemble_video.py from the command's intro_asset_id/
    # outro_asset_id (opaque ids Orchestrator read from its own
    # channel_asset_pointers projection) via ChannelAssetsRepository.
    # None/0.0 when the Creator toggled the intro/outro off, or Orchestrator
    # found no active asset for the project's render_quality — in which case
    # assembly proceeds exactly as it did before this CR.
    intro_video_path: str | None = None
    # Real duration of intro_video_path (ffprobe'd when the asset was
    # registered — see adapters/persistence/channel_assets.py). Only this
    # field feeds FfmpegVideoAssembler's effective_lead_in; outro needs no
    # equivalent because it is appended at the end, not spliced before the
    # narration timeline.
    intro_duration_seconds: float = 0.0
    outro_video_path: str | None = None


@dataclass(frozen=True)
class ChannelAsset:
    """One row of `channel_assets` — the channel-wide intro/outro
    currently (or formerly, when superseded) active for one (kind,
    render_quality) pair.

    Lives at video-assembly because this is the service that already owns
    `lead_in`/ghép (assembly), and is the one Creator-uploaded files get
    ffmpeg-normalized by (adapters/messaging/consumer.py's
    NormalizeChannelAssetCommandHandler).
    """

    id: str
    kind: str
    render_quality: str
    source_hash: str
    video_path: str
    music_path: str | None
    version: int
    duration_seconds: float
    # Hash of the music file baked into video_path, when one has been
    # uploaded. Kept apart from source_hash — which always describes
    # the VIDEO source — so each upload path caches against its own input.
    music_source_hash: str | None = None


@dataclass(frozen=True)
class VideoAssemblyResult:
    """Output of video assembly, returned as the event payload.

    Deliberately minimal (Functional Design Rule 6) — Publisher Service only
    needs video_path, and GUI preview can read other metadata directly from
    the file.
    """

    video_path: str
    # Set only when subtitle_mode produced a .srt — None when
    # subtitles are off or burn-in only. Flows to Publisher the same way
    # thumbnail_path does.
    caption_path: str | None = None
