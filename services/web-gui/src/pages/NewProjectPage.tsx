import { useContext, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ScriptEditor } from "../components/ScriptEditor";
import { NarrationPanel } from "../components/NarrationPanel";
import { RenderQualityPicker } from "../components/RenderQualityPicker";
import { SubtitleStylePanel } from "../components/SubtitleStylePanel";
import { BackgroundMusicPicker } from "../components/BackgroundMusicPicker";
import { AppShell } from "../components/AppShell";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { startRenderSaga, ApiError } from "../api/client";
import glass from "../styles/glass.module.css";
import styles from "./NewProjectPage.module.css";
import { validateScript } from "../utils/scriptValidation";

export function NewProjectPage() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const scriptValidation = validateScript(draft.scriptContent);
  const canSubmit = draft.scriptContent.trim().length > 0 && scriptValidation.isValid;

  async function handleSubmit() {
    if (!canSubmit) return;
    setIsSubmitting(true);
    setError(null);
    try {
      const projectId = draft.projectId;
      await startRenderSaga({
        project_id: projectId,
        script_content: draft.scriptContent,
        voice_language: draft.voiceLanguage,
        background_music_path: draft.backgroundMusicPath ?? undefined,
        tts_enabled: draft.ttsEnabled,
        voice_id: draft.ttsEnabled ? (draft.voiceId ?? undefined) : undefined,
        subtitles_enabled: draft.subtitlesEnabled,
        subtitle_style: draft.subtitlesEnabled
          ? {
              font_size: draft.subtitleStyle.fontSize,
              text_color: draft.subtitleStyle.textColor,
              background_opacity: draft.subtitleStyle.backgroundOpacity,
              position: draft.subtitleStyle.position,
            }
          : undefined,
        render_quality: draft.renderQuality,
        background_music_volume: draft.backgroundMusicPath
          ? draft.backgroundMusicVolume
          : undefined,
      });
      navigate(`/projects/${projectId}/render`);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <div data-testid="new-project-page">
      <AppShell
        currentStep={1}
        wide
        title="Tạo video mới"
        subtitle="Dán script Manim của bạn (đánh dấu lời thoại bằng # NARRATION), chọn giọng đọc và phụ đề, rồi bắt đầu render tự động."
      >
        <div className={styles.layout}>
          <ScriptEditor
            value={draft.scriptContent}
            onChange={(value) => dispatch({ type: "SET_SCRIPT", payload: value })}
            contentLanguage={draft.voiceLanguage}
          />

          <div className={styles.sidebar}>
            <NarrationPanel
              voiceLanguage={draft.voiceLanguage}
              onVoiceLanguageChange={(lang) => dispatch({ type: "SET_VOICE_LANGUAGE", payload: lang })}
              ttsEnabled={draft.ttsEnabled}
              onTtsEnabledChange={(enabled) => dispatch({ type: "SET_TTS_ENABLED", payload: enabled })}
              voiceId={draft.voiceId}
              onVoiceIdChange={(voiceId) => dispatch({ type: "SET_VOICE_ID", payload: voiceId })}
              subtitlesEnabled={draft.subtitlesEnabled}
              onSubtitlesEnabledChange={(enabled) =>
                dispatch({ type: "SET_SUBTITLES_ENABLED", payload: enabled })
              }
            />

            <RenderQualityPicker
              value={draft.renderQuality}
              onChange={(quality) => dispatch({ type: "SET_RENDER_QUALITY", payload: quality })}
            />

            {draft.subtitlesEnabled && (
              <SubtitleStylePanel
                value={draft.subtitleStyle}
                onChange={(patch) => dispatch({ type: "SET_SUBTITLE_STYLE", payload: patch })}
              />
            )}

            <BackgroundMusicPicker
              projectId={draft.projectId}
              value={draft.backgroundMusicPath}
              volume={draft.backgroundMusicVolume}
              onVolumeChange={(v) => dispatch({ type: "SET_BACKGROUND_MUSIC_VOLUME", payload: v })}
              onChange={(path) => dispatch({ type: "SET_BACKGROUND_MUSIC", payload: path })}
            />

            <div className={`${glass.card} ${styles.ctaCard}`}>
              {error && (
                <p role="alert" className={glass.helperText} style={{ marginRight: 0 }}>
                  {error}
                </p>
              )}
              {!error && !canSubmit && draft.scriptContent.trim().length === 0 && (
                <p className={glass.helperText} style={{ marginRight: 0 }}>
                  Dán script Manim để tiếp tục
                </p>
              )}
              {!error && !canSubmit && draft.scriptContent.trim().length > 0 && (
                <p className={glass.helperText} style={{ marginRight: 0 }}>
                  Sửa lỗi định dạng script (xem cảnh báo phía trên) trước khi render
                </p>
              )}
              <button
                type="button"
                data-testid="new-project-submit-button"
                className={`${glass.btnPrimary} ${styles.btnPrimaryFull}`}
                disabled={!canSubmit || isSubmitting}
                onClick={handleSubmit}
              >
                Bắt đầu render
                <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M5 12h14M13 6l6 6-6 6" />
                </svg>
              </button>
            </div>
          </div>
        </div>
      </AppShell>
    </div>
  );
}
