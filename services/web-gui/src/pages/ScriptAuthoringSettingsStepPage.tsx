import { useContext, useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { ContentLanguagePicker } from "../components/ContentLanguagePicker";
import { RenderEnginePicker } from "../components/RenderEnginePicker";
import { VideoFontPicker } from "../components/VideoFontPicker";
import { AuthoringModeBar } from "../components/AuthoringModeBar";
import { AuthoringModelPicker } from "../components/AuthoringModelPicker";
import { useLlmStatus } from "../hooks/useLlmStatus";
import { useAuthoringMode } from "../hooks/useAuthoringMode";
import { useAuthoringModels } from "../hooks/useAuthoringModels";
import { NarrationPanel } from "../components/NarrationPanel";
import { RenderQualityPicker } from "../components/RenderQualityPicker";
import { VideoOutputModePicker } from "../components/VideoOutputModePicker";
import { VideoFormatPicker } from "../components/VideoFormatPicker";
import { BackgroundMusicPicker } from "../components/BackgroundMusicPicker";
import { Disclosure } from "../components/Disclosure";
import { useVideoFormats } from "../hooks/useVideoFormats";
import { WizardNav } from "../components/WizardNav";
import {
  ProjectDraftContext,
  ProjectDraftDispatchContext,
  saveLastUsedSettings,
  type SubtitleMode,
  type SubtitleStyle,
} from "../context/ProjectDraftContext";
import { patchWizardSettings, type WizardSettingsPatch } from "../api/client";
import styles from "./WizardSteps.module.css";

function toWireStyle(style: SubtitleStyle) {
  return {
    font_family: style.fontFamily,
    font_size: style.fontSize,
    text_color: style.textColor,
    background_opacity: style.backgroundOpacity,
    position: style.position,
  };
}

const RENDER_QUALITY_LABELS: Record<string, string> = {
  "480p15": "Test (480p15)",
  "720p30": "Nháp (720p30)",
  "1080p60": "Chuẩn (1080p60)",
  "4k60": "Cao (4K60)",
};

/**
 * Bước 2 — toàn bộ cấu hình: ngôn ngữ, render engine, cách làm (manual/AI),
 * và giọng đọc/phụ đề/định dạng/chất lượng/nhạc nền. Gộp từ "Cách làm" và
 * "Cấu hình" cũ (bước "Xem lại" đã bỏ) — chốt hết trước khi viết script, vì
 * định dạng video quyết định số beat mà script phải theo. Mọi mục đều có mặc
 * định nên đi thẳng qua được. Mỗi lần đổi một mục là một PATCH riêng lên
 * project (đã tạo ở Bước 1); "Tiếp tục" chỉ chốt bước.
 */
export function ScriptAuthoringSettingsStepPage() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();
  const llm = useLlmStatus();
  const { mode: authoringMode, setMode: setAuthoringMode } = useAuthoringMode(draft.projectId);
  const { models: authoringModels, setModels: setAuthoringModels } = useAuthoringModels(draft.projectId);

  const formats = useVideoFormats();

  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  const isRemotion = draft.renderEngine === "remotion";

  // PATCH nối đuôi nhau để server nhận đúng thứ tự Creator đổi. Field nào lưu
  // lỗi được giữ ở `failedRef` và gửi lại cùng lần PATCH kế tiếp.
  const queueRef = useRef<Promise<void>>(Promise.resolve());
  const failedRef = useRef<WizardSettingsPatch>({});
  const volumeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pendingVolumeRef = useRef<number | null>(null);

  function send(fields: WizardSettingsPatch): Promise<boolean> {
    const projectId = draft.projectId;
    if (!projectId) return Promise.resolve(true);
    const run = queueRef.current.then(async () => {
      const body = { ...failedRef.current, ...fields };
      try {
        await patchWizardSettings(projectId, body);
        failedRef.current = {};
        setSaveError(null);
        return true;
      } catch {
        failedRef.current = body;
        setSaveError("Không lưu được cấu hình — kiểm tra kết nối rồi thử lại.");
        return false;
      }
    });
    queueRef.current = run.then(() => undefined);
    return run;
  }

  function flushVolume() {
    if (volumeTimerRef.current) clearTimeout(volumeTimerRef.current);
    volumeTimerRef.current = null;
    if (pendingVolumeRef.current !== null) {
      const backgroundMusicVolume = pendingVolumeRef.current;
      pendingVolumeRef.current = null;
      void send({ backgroundMusicVolume });
    }
  }
  useEffect(() => flushVolume, []); // eslint-disable-line react-hooks/exhaustive-deps

  function handleEngineChange(engine: "manim" | "remotion") {
    dispatch({ type: "SET_RENDER_ENGINE", payload: engine });
    void send({ renderEngine: engine });
  }

  function handleSubtitleModeChange(mode: SubtitleMode) {
    dispatch({ type: "SET_SUBTITLE_MODE", payload: mode });
    // Phong cách chỉ có nghĩa khi phụ đề được đốt vào video; lưu cùng lúc với
    // chế độ để server không bao giờ giữ chế độ đốt mà thiếu style.
    void send({
      subtitleMode: mode,
      ...(mode === "burn_in" || mode === "both" ? { subtitleStyle: toWireStyle(draft.subtitleStyle) } : {}),
    });
  }

  function handleSubtitleStyleChange(patch: Partial<SubtitleStyle>) {
    dispatch({ type: "SET_SUBTITLE_STYLE", payload: patch });
    void send({ subtitleStyle: toWireStyle({ ...draft.subtitleStyle, ...patch }) });
  }

  function handleBackgroundMusicVolumeChange(v: number) {
    dispatch({ type: "SET_BACKGROUND_MUSIC_VOLUME", payload: v });
    pendingVolumeRef.current = v;
    if (volumeTimerRef.current) clearTimeout(volumeTimerRef.current);
    volumeTimerRef.current = setTimeout(flushVolume, 400);
  }

  async function handleContinue() {
    setSaving(true);
    try {
      flushVolume();
      // Các field đã lưu từng cái một; bấm tiếp chỉ chốt bước (và gửi lại
      // field nào trước đó lưu lỗi) rồi mới đi tiếp.
      if (await send({ confirm: true })) {
        // CR-028 FR86.1 — cấu hình này thành mặc định cho project kế tiếp.
        saveLastUsedSettings(draft);
        navigate("/create/script/outline");
      }
    } finally {
      setSaving(false);
    }
  }

  return (
    <div data-testid="script-authoring-settings-step-page">
      <AppShell
        currentStep={2}
        wide
        title={isRemotion ? "Bước 2 — Cấu hình Remotion" : "Bước 2 — Cấu hình Manim"}
        subtitle="Chọn ngôn ngữ, giọng đọc và chất lượng video. Các mục đã có sẵn giá trị phù hợp, bạn có thể bấm Tiếp tục ngay."
      >
        <div className={styles.settingsRow}>
          <ContentLanguagePicker
            value={draft.voiceLanguage}
            onChange={(lang) => {
              dispatch({ type: "SET_VOICE_LANGUAGE", payload: lang });
              void send({ voiceLanguage: lang });
            }}
          />
        </div>

        <div className={styles.settingsRow}>
          <RenderEnginePicker value={draft.renderEngine} onChange={handleEngineChange} />
        </div>

        {isRemotion && (
          <div className={styles.settingsRow}>
            <VideoFontPicker
              value={draft.videoFont}
              onChange={(font) => {
                dispatch({ type: "SET_VIDEO_FONT", payload: font });
                void send({ videoFont: font });
              }}
            />
          </div>
        )}

        <div className={styles.settingsRow}>
          <AuthoringModeBar
            llm={llm}
            mode={authoringMode}
            onModeChange={setAuthoringMode}
            projectId={draft.projectId}
          />
        </div>

        {authoringMode === "ai" && llm?.enabled && (
          <div className={styles.settingsRow}>
            <AuthoringModelPicker
              models={authoringModels}
              onChange={setAuthoringModels}
              options={llm.models ?? []}
              defaultModel={llm.default_model ?? ""}
            />
          </div>
        )}

        <div className={styles.settingsLayout}>
          <NarrationPanel
            voiceLanguage={draft.voiceLanguage}
            ttsEnabled={draft.ttsEnabled}
            onTtsEnabledChange={(enabled) => {
              dispatch({ type: "SET_TTS_ENABLED", payload: enabled });
              void send({ ttsEnabled: enabled });
            }}
            voiceId={draft.voiceId}
            onVoiceIdChange={(voiceId) => {
              dispatch({ type: "SET_VOICE_ID", payload: voiceId });
              void send({ voiceId: voiceId ?? "" });
            }}
            subtitleMode={draft.subtitleMode}
            onSubtitleModeChange={handleSubtitleModeChange}
            subtitleStyle={draft.subtitleStyle}
            onSubtitleStyleChange={handleSubtitleStyleChange}
          />

          <div className={styles.settingsRow}>
            <VideoFormatPicker
              formats={formats}
              value={draft.videoFormatId}
              onChange={(formatId) => {
                dispatch({ type: "SET_VIDEO_FORMAT", payload: formatId });
                void send({ videoFormatId: formatId });
              }}
            />

            <Disclosure
              title="Chất lượng video"
              hint={`Hiện tại: ${RENDER_QUALITY_LABELS[draft.renderQuality] ?? draft.renderQuality}`}
              testId="settings-render-quality"
            >
              <RenderQualityPicker
                value={draft.renderQuality}
                onChange={(quality) => {
                  dispatch({ type: "SET_RENDER_QUALITY", payload: quality });
                  void send({ renderQuality: quality });
                }}
              />
            </Disclosure>

            <VideoOutputModePicker
              value={draft.videoOutputMode}
              onChange={(mode) => {
                dispatch({ type: "SET_VIDEO_OUTPUT_MODE", payload: mode });
                void send({ videoOutputMode: mode });
              }}
            />

            <Disclosure
              title="Nhạc nền"
              hint={draft.backgroundMusicPath ? "Hiện tại: đã chọn nhạc nền" : "Hiện tại: không có"}
              testId="settings-background-music"
            >
              <BackgroundMusicPicker
                projectId={draft.projectId}
                value={draft.backgroundMusicPath}
                volume={draft.backgroundMusicVolume}
                onVolumeChange={handleBackgroundMusicVolumeChange}
                onChange={(path) => {
                  dispatch({ type: "SET_BACKGROUND_MUSIC", payload: path });
                  void send({ backgroundMusicPath: path ?? "" });
                }}
              />
            </Disclosure>
          </div>
        </div>
      </AppShell>

      <WizardNav
        isBlocked={!!saveError}
        nextDisabled={saving}
        hint={
          saveError ??
          (authoringMode === "ai" && llm?.enabled
            ? "AI sẽ giúp bạn ở các bước Kịch bản, Visual và Code."
            : "Bạn tự làm với ChatGPT, Claude… Có thể đổi sang AI làm giúp bất cứ lúc nào.")
        }
        onBack={() => navigate("/")}
        onNext={handleContinue}
        nextLabel={saving ? "Đang lưu..." : "Sang Kịch bản"}
        nextTestId="script-authoring-settings-next"
      />
    </div>
  );
}
