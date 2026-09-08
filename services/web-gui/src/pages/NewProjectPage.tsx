import { useContext, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ScriptEditor } from "../components/ScriptEditor";
import { NarrationPanel } from "../components/NarrationPanel";
import { RenderQualityPicker } from "../components/RenderQualityPicker";
import { ContentLanguagePicker } from "../components/ContentLanguagePicker";
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

  // A draft that already started a saga belongs to an existing project.
  // Reusing it here would re-run the saga against the same project_id and
  // overwrite the video that draft produced.
  useEffect(() => {
    if (draft.hasSubmitted) dispatch({ type: "RESET" });
  }, [draft.hasSubmitted, dispatch]);

  const scriptValidation = validateScript(draft.scriptContent);
  const isScriptEmpty = draft.scriptContent.trim().length === 0;
  const canSubmit = !isScriptEmpty && scriptValidation.isValid;

  const submitHint = error
    ? error
    : isScriptEmpty
      ? "Dán hoặc tạo script Manim để bắt đầu"
      : !scriptValidation.isValid
        ? scriptValidation.message
        : `Sẵn sàng render — ${scriptValidation.narrationCount} đoạn lời thoại`;

  function scrollToScript() {
    document.getElementById("script-validation")?.scrollIntoView({ behavior: "smooth", block: "center" });
    document.getElementById("script-editor")?.scrollIntoView({ behavior: "smooth", block: "center" });
  }

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
      dispatch({ type: "MARK_SUBMITTED" });
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
        {/* The project's first decision — it drives everything below it. */}
        <ContentLanguagePicker
          value={draft.voiceLanguage}
          onChange={(lang) => dispatch({ type: "SET_VOICE_LANGUAGE", payload: lang })}
        />

        <div className={styles.layout}>
          <ScriptEditor
            value={draft.scriptContent}
            onChange={(value) => dispatch({ type: "SET_SCRIPT", payload: value })}
            contentLanguage={draft.voiceLanguage}
          />

          <div className={styles.sidebar}>
            <NarrationPanel
              voiceLanguage={draft.voiceLanguage}
              ttsEnabled={draft.ttsEnabled}
              onTtsEnabledChange={(enabled) => dispatch({ type: "SET_TTS_ENABLED", payload: enabled })}
              voiceId={draft.voiceId}
              onVoiceIdChange={(voiceId) => dispatch({ type: "SET_VOICE_ID", payload: voiceId })}
              subtitlesEnabled={draft.subtitlesEnabled}
              onSubtitlesEnabledChange={(enabled) =>
                dispatch({ type: "SET_SUBTITLES_ENABLED", payload: enabled })
              }
              subtitleStyle={draft.subtitleStyle}
              onSubtitleStyleChange={(patch) => dispatch({ type: "SET_SUBTITLE_STYLE", payload: patch })}
            />

            {/*
              Quality already defaults to what a published video needs, and
              music is optional — neither is worth the vertical space that used
              to push the submit button off screen.
            */}
            <details className={`${glass.card} ${styles.advanced}`} data-testid="advanced-settings">
              <summary className={styles.advancedSummary}>
                <span className={glass.cardTitle}>Tuỳ chọn nâng cao</span>
                <span className={styles.advancedHint}>Chất lượng video, nhạc nền</span>
              </summary>
              <div className={styles.advancedBody}>
                <RenderQualityPicker
                  value={draft.renderQuality}
                  onChange={(quality) => dispatch({ type: "SET_RENDER_QUALITY", payload: quality })}
                />

                <BackgroundMusicPicker
                  projectId={draft.projectId}
                  value={draft.backgroundMusicPath}
                  volume={draft.backgroundMusicVolume}
                  onVolumeChange={(v) => dispatch({ type: "SET_BACKGROUND_MUSIC_VOLUME", payload: v })}
                  onChange={(path) => dispatch({ type: "SET_BACKGROUND_MUSIC", payload: path })}
                />
              </div>
            </details>
          </div>
        </div>
      </AppShell>

      {/*
        The submit button used to be the last card in a sidebar taller than the
        viewport — its sticky positioning could never engage, so the primary
        action of the page was permanently below the fold. It now rides a bar
        pinned to the bottom of the window.
      */}
      <div className={styles.submitBar}>
        <div className={styles.submitBarInner}>
          <p
            className={`${styles.submitHint} ${error || (!canSubmit && !isScriptEmpty) ? styles.submitHintError : ""}`}
            role={error ? "alert" : "status"}
          >
            {submitHint}
          </p>
          {!canSubmit && !isScriptEmpty && (
            <button type="button" className={glass.ghostBtn} onClick={scrollToScript}>
              Xem lỗi
            </button>
          )}
          <button
            type="button"
            data-testid="new-project-submit-button"
            className={glass.btnPrimary}
            disabled={!canSubmit || isSubmitting}
            onClick={handleSubmit}
          >
            {isSubmitting ? "Đang gửi..." : "Bắt đầu render"}
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M5 12h14M13 6l6 6-6 6" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  );
}
