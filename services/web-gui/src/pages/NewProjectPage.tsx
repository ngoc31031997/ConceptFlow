import { useContext, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ScriptEditor } from "../components/ScriptEditor";
import { PluginSelector } from "../components/PluginSelector";
import { CategorySelector } from "../components/CategorySelector";
import { VoiceLanguageSelector } from "../components/VoiceLanguageSelector";
import { BackgroundMusicPicker } from "../components/BackgroundMusicPicker";
import { AppShell } from "../components/AppShell";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { startRenderSaga, ApiError } from "../api/client";
import glass from "../styles/glass.module.css";

export function NewProjectPage() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const canSubmit =
    draft.scriptContent.trim().length > 0 && draft.pluginId !== null && draft.categoryHint !== null;

  async function handleSubmit() {
    if (!canSubmit || draft.pluginId === null || draft.categoryHint === null) return;
    setIsSubmitting(true);
    setError(null);
    try {
      const projectId = crypto.randomUUID();
      await startRenderSaga({
        project_id: projectId,
        script_content: draft.scriptContent,
        plugin_id: draft.pluginId,
        category_hint: draft.categoryHint,
        voice_language: draft.voiceLanguage,
        background_music_path: draft.backgroundMusicPath ?? undefined,
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
        title="Tạo video mới"
        subtitle="Soạn nội dung script, chọn plugin và ngôn ngữ giọng đọc, rồi bắt đầu render tự động."
      >
        <ScriptEditor
          value={draft.scriptContent}
          onChange={(value) => dispatch({ type: "SET_SCRIPT", payload: value })}
        />

        <div style={{ display: "grid", gridTemplateColumns: "repeat(2, minmax(0, 1fr))", gap: 20 }}>
          <PluginSelector
            value={draft.pluginId}
            onChange={(pluginId) => dispatch({ type: "SET_PLUGIN", payload: pluginId })}
          />
          <VoiceLanguageSelector
            value={draft.voiceLanguage}
            onChange={(lang) => dispatch({ type: "SET_VOICE_LANGUAGE", payload: lang })}
          />
        </div>

        <CategorySelector
          pluginId={draft.pluginId}
          value={draft.categoryHint}
          onChange={(category) => dispatch({ type: "SET_CATEGORY", payload: category })}
        />

        <BackgroundMusicPicker
          value={draft.backgroundMusicPath}
          onChange={(path) => dispatch({ type: "SET_BACKGROUND_MUSIC", payload: path })}
        />

        <div className={glass.ctaRow}>
          {error && (
            <p role="alert" className={glass.helperText}>
              {error}
            </p>
          )}
          {!error && !canSubmit && (
            <p className={glass.helperText}>Nhập script, chọn plugin và danh mục để tiếp tục</p>
          )}
          <button
            type="button"
            data-testid="new-project-submit-button"
            className={glass.btnPrimary}
            disabled={!canSubmit || isSubmitting}
            onClick={handleSubmit}
          >
            Bắt đầu render
            <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M5 12h14M13 6l6 6-6 6" />
            </svg>
          </button>
        </div>
      </AppShell>
    </div>
  );
}
