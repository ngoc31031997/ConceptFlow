import { useContext } from "react";
import { useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { ContentLanguagePicker } from "../components/ContentLanguagePicker";
import { RenderEnginePicker } from "../components/RenderEnginePicker";
import { AuthoringModeBar } from "../components/AuthoringModeBar";
import { AuthoringModelPicker } from "../components/AuthoringModelPicker";
import { useLlmStatus } from "../hooks/useLlmStatus";
import { useAuthoringMode } from "../hooks/useAuthoringMode";
import { useAuthoringModels } from "../hooks/useAuthoringModels";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { createProjectDraft } from "../api/client";
import styles from "./WizardSteps.module.css";

/**
 * Bước 2 — ngôn ngữ, render engine và cách làm (manual/AI), tách ra từ
 * ScriptStepPage (CR-031 nay chỉ còn tình huống "ý tưởng"). Project đã được
 * tạo ở Bước 1; đổi engine ở đây chỉ cần echo lại, không cần await.
 */
export function ScriptAuthoringSettingsStepPage() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();
  const llm = useLlmStatus();
  const { mode: authoringMode, setMode: setAuthoringMode } = useAuthoringMode(draft.projectId);
  const { models: authoringModels, setModels: setAuthoringModels } = useAuthoringModels(draft.projectId);

  const isRemotion = draft.renderEngine === "remotion";

  function handleEngineChange(engine: "manim" | "remotion") {
    dispatch({ type: "SET_RENDER_ENGINE", payload: engine });
    if (draft.projectId) {
      void createProjectDraft(draft.projectId, "", draft.voiceLanguage, engine).catch(() => {});
    }
  }

  function handleContinue() {
    navigate("/create/script/outline");
  }

  return (
    <div data-testid="script-authoring-settings-step-page">
      <AppShell
        currentStep={2}
        wide
        title={isRemotion ? "Bước 2 — Script Remotion" : "Bước 2 — Script Manim"}
        subtitle="Chốt ngôn ngữ, công cụ render và cách làm trước khi vào dàn ý."
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
      </AppShell>

      <WizardNav
        hint={
          authoringMode === "ai" && llm?.enabled
            ? "Chế độ gọi API đang bật — các tab 1a–1c sẽ có nút chạy bằng AI."
            : "Chế độ copy prompt ra ngoài — đổi sang gọi API ở đây hoặc ở bất kỳ tab nào."
        }
        onBack={() => navigate("/")}
        onNext={handleContinue}
        nextLabel="Sang 1a. Dàn ý"
        nextTestId="script-authoring-settings-next"
      />
    </div>
  );
}
