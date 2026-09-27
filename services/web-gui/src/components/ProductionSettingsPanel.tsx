import { useEffect, useRef, useState } from "react";
import { ApiError, patchWizardSettings, type WizardSettingsPatch } from "../api/client";
import {
  defaultSubtitleStyle,
  rememberProductionSettings,
  type RenderQuality,
  type SubtitleMode,
  type SubtitleStyle,
} from "../context/ProjectDraftContext";
import type { Project } from "../types";
import { BackgroundMusicPicker } from "./BackgroundMusicPicker";
import { Disclosure } from "./Disclosure";
import { RenderQualityPicker } from "./RenderQualityPicker";
import { SubtitleSettings } from "./SubtitleSettings";
import { VideoFontPicker } from "./VideoFontPicker";
import glass from "../styles/glass.module.css";

/**
 * Which saga step's settings to show: "render" = render_scenes (quality, video
 * font), "merge" = assemble_video (subtitles, background music).
 */
export type ProductionStage = "render" | "merge";

interface ProductionSettingsPanelProps {
  project: Project;
  stages: ProductionStage[];
  /** One line under the title: why these settings are shown here. */
  hint: string;
  /** Called after each successful save, e.g. to refetch the project. */
  onSaved?: () => void;
}

const DEFAULT_MUSIC_VOLUME = 0.2;
const DEFAULT_VIDEO_FONT = "Be Vietnam Pro";

const RENDER_QUALITY_LABELS: Record<string, string> = {
  "480p15": "Test (480p15)",
  "720p30": "Nháp (720p30)",
  "1080p60": "Chuẩn (1080p60)",
  "4k60": "Cao (4K60)",
};

const SUBTITLE_MODE_LABELS: Record<SubtitleMode, string> = {
  off: "Tắt",
  track: "Phụ đề YouTube",
  burn_in: "Ghi cứng vào hình",
  both: "Cả hai",
};

function styleFromProject(project: Project): SubtitleStyle {
  const style = project.subtitle_style;
  if (!style) return defaultSubtitleStyle;
  return {
    // Projects saved before the font choice existed rendered in DejaVu Sans.
    fontFamily: style.font_family || "DejaVu Sans",
    fontSize: style.font_size,
    textColor: style.text_color,
    backgroundOpacity: style.background_opacity,
    position: style.position,
  };
}

function toWireStyle(style: SubtitleStyle) {
  return {
    font_family: style.fontFamily,
    font_size: style.fontSize,
    text_color: style.textColor,
    background_opacity: style.backgroundOpacity,
    position: style.position,
  };
}

function subtitleModeOf(project: Project): SubtitleMode {
  const mode = project.subtitle_mode as SubtitleMode | undefined;
  return mode && mode in SUBTITLE_MODE_LABELS ? mode : "track";
}

/**
 * Settings read only by the production steps, shown next to the step that
 * reads them rather than in step 2: at the review gate (before anything costly
 * runs) and on the failure of the step that reads them, so a bad choice can be
 * fixed and retried in place. The server enforces the same boundary
 * (domain.WizardPatchAllowed): render settings stay editable until
 * render_scenes has run, merge settings until assemble_video has run.
 *
 * Each change is its own PATCH, chained so the server applies them in the
 * order the Creator made them; a failed field is resent with the next one.
 */
export function ProductionSettingsPanel({ project, stages, hint, onSaved }: ProductionSettingsPanelProps) {
  const projectId = project.project_id;
  const isRemotion = project.render_engine === "remotion";

  const [renderQuality, setRenderQuality] = useState<RenderQuality>(project.render_quality ?? "1080p60");
  const [videoFont, setVideoFont] = useState(project.video_font || DEFAULT_VIDEO_FONT);
  const [subtitleMode, setSubtitleMode] = useState<SubtitleMode>(subtitleModeOf(project));
  const [subtitleStyle, setSubtitleStyle] = useState<SubtitleStyle>(styleFromProject(project));
  const [musicPath, setMusicPath] = useState<string | null>(project.background_music_path ?? null);
  const [musicVolume, setMusicVolume] = useState(project.background_music_volume || DEFAULT_MUSIC_VOLUME);
  const [saveError, setSaveError] = useState<string | null>(null);

  const queueRef = useRef<Promise<void>>(Promise.resolve());
  const failedRef = useRef<WizardSettingsPatch>({});
  const volumeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pendingVolumeRef = useRef<number | null>(null);
  const onSavedRef = useRef(onSaved);
  onSavedRef.current = onSaved;

  function send(fields: WizardSettingsPatch) {
    const run = queueRef.current.then(async () => {
      const body = { ...failedRef.current, ...fields };
      try {
        await patchWizardSettings(projectId, body);
        failedRef.current = {};
        setSaveError(null);
        onSavedRef.current?.();
      } catch (err) {
        // 409: the step reading this setting has already run (or is running) —
        // resending would only fail again, so drop it instead of queueing it.
        const conflict = err instanceof ApiError && err.status === 409;
        failedRef.current = conflict ? {} : body;
        setSaveError(
          conflict
            ? "Bước dùng cấu hình này đã chạy hoặc đang chạy, không đổi được nữa."
            : "Không lưu được cấu hình — kiểm tra kết nối rồi thử lại.",
        );
      }
    });
    queueRef.current = run;
  }

  function flushVolume() {
    if (volumeTimerRef.current) clearTimeout(volumeTimerRef.current);
    volumeTimerRef.current = null;
    if (pendingVolumeRef.current !== null) {
      const backgroundMusicVolume = pendingVolumeRef.current;
      pendingVolumeRef.current = null;
      rememberProductionSettings({ backgroundMusicVolume });
      send({ backgroundMusicVolume });
    }
  }
  useEffect(() => flushVolume, []); // eslint-disable-line react-hooks/exhaustive-deps

  function changeQuality(quality: RenderQuality) {
    setRenderQuality(quality);
    rememberProductionSettings({ renderQuality: quality });
    send({ renderQuality: quality });
  }

  function changeFont(font: string) {
    setVideoFont(font);
    rememberProductionSettings({ videoFont: font });
    send({ videoFont: font });
  }

  function changeSubtitleMode(mode: SubtitleMode) {
    setSubtitleMode(mode);
    rememberProductionSettings({ subtitleMode: mode });
    // The style only means something when subtitles are burned in; save it
    // with the mode so the server never holds a burn-in mode without a style.
    send({
      subtitleMode: mode,
      ...(mode === "burn_in" || mode === "both" ? { subtitleStyle: toWireStyle(subtitleStyle) } : {}),
    });
  }

  function changeSubtitleStyle(patch: Partial<SubtitleStyle>) {
    const next = { ...subtitleStyle, ...patch };
    setSubtitleStyle(next);
    rememberProductionSettings({ subtitleStyle: next });
    send({ subtitleStyle: toWireStyle(next) });
  }

  function changeMusic(path: string | null) {
    setMusicPath(path);
    rememberProductionSettings({ backgroundMusicPath: path });
    send({ backgroundMusicPath: path ?? "" });
  }

  function changeVolume(volume: number) {
    setMusicVolume(volume);
    pendingVolumeRef.current = volume;
    if (volumeTimerRef.current) clearTimeout(volumeTimerRef.current);
    volumeTimerRef.current = setTimeout(flushVolume, 400);
  }

  const showRender = stages.includes("render");
  const showMerge = stages.includes("merge");

  return (
    <section data-testid="production-settings" aria-label="Cấu hình dựng và ghép">
      <div className={glass.cardTitle}>Cấu hình dựng & ghép</div>
      <p className={glass.cardHint}>{hint}</p>

      {showRender && (
        <>
          <div className={glass.mtSm}>
            <Disclosure
              title="Chất lượng video"
              hint={`Hiện tại: ${RENDER_QUALITY_LABELS[renderQuality] ?? renderQuality}`}
              testId="production-render-quality"
            >
              {(close) => (
                <RenderQualityPicker
                  value={renderQuality}
                  onChange={(quality) => {
                    changeQuality(quality);
                    close();
                  }}
                />
              )}
            </Disclosure>
          </div>
          {isRemotion && (
            <div className={glass.mtSm}>
              <VideoFontPicker value={videoFont} onChange={changeFont} />
            </div>
          )}
        </>
      )}

      {showMerge && (
        <>
          <div className={glass.mtSm}>
            <Disclosure
              title="Phụ đề"
              hint={`Hiện tại: ${SUBTITLE_MODE_LABELS[subtitleMode]}`}
              testId="production-subtitles"
            >
              <SubtitleSettings
                mode={subtitleMode}
                onModeChange={changeSubtitleMode}
                style={subtitleStyle}
                onStyleChange={changeSubtitleStyle}
                ttsEnabled={project.tts_enabled ?? true}
              />
            </Disclosure>
          </div>
          <div className={glass.mtSm}>
            <Disclosure
              title="Nhạc nền"
              hint={musicPath ? "Hiện tại: đã chọn nhạc nền" : "Hiện tại: không có"}
              testId="production-background-music"
            >
              <BackgroundMusicPicker
                projectId={projectId}
                value={musicPath}
                volume={musicVolume}
                onVolumeChange={changeVolume}
                onChange={changeMusic}
              />
            </Disclosure>
          </div>
        </>
      )}

      {saveError && (
        <p className={glass.helperText} role="alert" data-testid="production-settings-error">
          {saveError}
        </p>
      )}
    </section>
  );
}
