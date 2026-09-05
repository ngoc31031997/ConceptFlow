import { useContext, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ScriptEditor } from "../components/ScriptEditor";
import { PluginSelector } from "../components/PluginSelector";
import { VoiceLanguageSelector } from "../components/VoiceLanguageSelector";
import { BackgroundMusicPicker } from "../components/BackgroundMusicPicker";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { startRenderSaga, ApiError } from "../api/client";

export function NewProjectPage() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const canSubmit = draft.scriptContent.trim().length > 0 && draft.pluginId !== null;

  async function handleSubmit() {
    if (!canSubmit || draft.pluginId === null) return;
    setIsSubmitting(true);
    setError(null);
    try {
      const projectId = crypto.randomUUID();
      await startRenderSaga({
        project_id: projectId,
        script_content: draft.scriptContent,
        plugin_id: draft.pluginId,
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
      <h1>Tạo Video Mới</h1>
      <ScriptEditor
        value={draft.scriptContent}
        onChange={(value) => dispatch({ type: "SET_SCRIPT", payload: value })}
      />
      <PluginSelector
        value={draft.pluginId}
        onChange={(pluginId) => dispatch({ type: "SET_PLUGIN", payload: pluginId })}
      />
      <VoiceLanguageSelector
        value={draft.voiceLanguage}
        onChange={(lang) => dispatch({ type: "SET_VOICE_LANGUAGE", payload: lang })}
      />
      <BackgroundMusicPicker
        value={draft.backgroundMusicPath}
        onChange={(path) => dispatch({ type: "SET_BACKGROUND_MUSIC", payload: path })}
      />
      {error && <p role="alert">{error}</p>}
      <button
        type="button"
        data-testid="new-project-submit-button"
        disabled={!canSubmit || isSubmitting}
        onClick={handleSubmit}
      >
        Bắt đầu render
      </button>
    </div>
  );
}
