import { useContext, useState } from "react";
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
} from "../context/ProjectDraftContext";
import { createProjectDraft, saveWizardSettings } from "../api/client";
import styles from "./WizardSteps.module.css";

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
 * định nên đi thẳng qua được. Project đã được tạo ở Bước 1; đổi engine ở đây
 * chỉ cần echo lại, không cần await.
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

  function handleEngineChange(engine: "manim" | "remotion") {
    dispatch({ type: "SET_RENDER_ENGINE", payload: engine });
    if (draft.projectId) {
      void createProjectDraft(draft.projectId, "", draft.voiceLanguage, engine).catch(() => {});
    }
  }

  async function handleContinue() {
    setSaving(true);
    setSaveError(null);
    try {
      // Lưu lên server trước khi đi tiếp: cấu hình nằm trong hàng project, nên
      // mở lại ở máy khác hay sau restart vẫn đúng, và saga đọc lại đúng giá trị này.
      await saveWizardSettings(draft.projectId, {
        voiceLanguage: draft.voiceLanguage,
        renderEngine: draft.renderEngine,
        videoFont: draft.videoFont,
        ttsEnabled: draft.ttsEnabled,
        voiceId: draft.voiceId,
        subtitleMode: draft.subtitleMode,
        subtitleStyle:
          draft.subtitleMode === "burn_in" || draft.subtitleMode === "both"
            ? {
                font_family: draft.subtitleStyle.fontFamily,
                font_size: draft.subtitleStyle.fontSize,
                text_color: draft.subtitleStyle.textColor,
                background_opacity: draft.subtitleStyle.backgroundOpacity,
                position: draft.subtitleStyle.position,
              }
            : undefined,
        renderQuality: draft.renderQuality,
        videoFormatId: draft.videoFormatId,
        videoOutputMode: draft.videoOutputMode,
        backgroundMusicPath: draft.backgroundMusicPath,
        backgroundMusicVolume: draft.backgroundMusicVolume,
      });
      // CR-028 FR86.1 — cấu hình này thành mặc định cho project kế tiếp.
      saveLastUsedSettings(draft);
      navigate("/create/script/outline");
    } catch {
      setSaveError("Không lưu được cấu hình — kiểm tra kết nối rồi thử lại.");
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
        subtitle="Chốt ngôn ngữ, công cụ render, giọng đọc và hình ảnh trước khi vào dàn ý. Mọi mục đã có sẵn lựa chọn hợp lý — bấm Tiếp tục ngay cũng được."
      >
        <div className={styles.settingsRow}>
          <ContentLanguagePicker
            value={draft.voiceLanguage}
            onChange={(lang) => dispatch({ type: "SET_VOICE_LANGUAGE", payload: lang })}
          />
        </div>

        <div className={styles.settingsRow}>
          <RenderEnginePicker value={draft.renderEngine} onChange={handleEngineChange} />
        </div>

        {isRemotion && (
          <div className={styles.settingsRow}>
            <VideoFontPicker
              value={draft.videoFont}
              onChange={(font) => dispatch({ type: "SET_VIDEO_FONT", payload: font })}
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
            />
          </div>
        )}

        <div className={styles.settingsLayout}>
          <NarrationPanel
            voiceLanguage={draft.voiceLanguage}
            ttsEnabled={draft.ttsEnabled}
            onTtsEnabledChange={(enabled) => dispatch({ type: "SET_TTS_ENABLED", payload: enabled })}
            voiceId={draft.voiceId}
            onVoiceIdChange={(voiceId) => dispatch({ type: "SET_VOICE_ID", payload: voiceId })}
            subtitleMode={draft.subtitleMode}
            onSubtitleModeChange={(mode) => dispatch({ type: "SET_SUBTITLE_MODE", payload: mode })}
            subtitleStyle={draft.subtitleStyle}
            onSubtitleStyleChange={(patch) => dispatch({ type: "SET_SUBTITLE_STYLE", payload: patch })}
          />

          <div className={styles.settingsRow}>
            <VideoFormatPicker
              formats={formats}
              value={draft.videoFormatId}
              onChange={(formatId) => dispatch({ type: "SET_VIDEO_FORMAT", payload: formatId })}
            />

            <Disclosure
              title="Chất lượng video"
              hint={`Hiện tại: ${RENDER_QUALITY_LABELS[draft.renderQuality] ?? draft.renderQuality}`}
              testId="settings-render-quality"
            >
              <RenderQualityPicker
                value={draft.renderQuality}
                onChange={(quality) => dispatch({ type: "SET_RENDER_QUALITY", payload: quality })}
              />
            </Disclosure>

            <VideoOutputModePicker
              value={draft.videoOutputMode}
              onChange={(mode) => dispatch({ type: "SET_VIDEO_OUTPUT_MODE", payload: mode })}
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
                onVolumeChange={(v) => dispatch({ type: "SET_BACKGROUND_MUSIC_VOLUME", payload: v })}
                onChange={(path) => dispatch({ type: "SET_BACKGROUND_MUSIC", payload: path })}
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
            ? "Chế độ gọi API đang bật — các tab 1a–1c sẽ có nút chạy bằng AI."
            : "Chế độ copy prompt ra ngoài — đổi sang gọi API ở đây hoặc ở bất kỳ tab nào.")
        }
        onBack={() => navigate("/")}
        onNext={handleContinue}
        nextLabel={saving ? "Đang lưu..." : "Sang 1a. Dàn ý"}
        nextTestId="script-authoring-settings-next"
      />
    </div>
  );
}
