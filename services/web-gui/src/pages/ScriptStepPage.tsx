import { useContext, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { ScriptAssistant } from "../components/ScriptAssistant";
import { ScriptEditor } from "../components/ScriptEditor";
import { useVoiceCalibration, wordsPerMinuteFor } from "../hooks/useVoiceCalibration";
import { useVideoFormats } from "../hooks/useVideoFormats";
import { ContentLanguagePicker } from "../components/ContentLanguagePicker";
import { WizardNav } from "../components/WizardNav";
import { SCRIPT_TEMPLATES } from "../components/scriptTemplates";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { validateScript } from "../utils/scriptValidation";
import styles from "./WizardSteps.module.css";

/**
 * Step 1 of 3 — get a valid script.
 *
 * Content language leads the step because it decides what language the AI is
 * asked to write the narration in; picking it later would mean regenerating
 * the script.
 */
export function ScriptStepPage() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();

  // A draft that already started a saga belongs to an existing project;
  // reusing it would overwrite that project's video.
  useEffect(() => {
    if (draft.hasSubmitted) dispatch({ type: "RESET" });
  }, [draft.hasSubmitted, dispatch]);

  const calibration = useVoiceCalibration();
  const formats = useVideoFormats();
  const validation = validateScript(
    draft.scriptContent,
    draft.voiceLanguage,
    wordsPerMinuteFor(calibration, draft.voiceId),
  );
  const isEmpty = draft.scriptContent.trim().length === 0;
  const canContinue = !isEmpty && validation.isValid;

  const hint = isEmpty
    ? "Dán hoặc tạo script Manim để tiếp tục"
    : validation.isValid
      ? `Script hợp lệ — ${validation.narrationCount} đoạn lời thoại`
      : validation.message;

  return (
    <div data-testid="script-step-page">
      <AppShell
        currentStep={1}
        wide
        title="Bước 1 — Script Manim"
        subtitle="Hệ thống render script Manim của bạn thành video. Trước tiên cần một script có đánh dấu lời thoại."
      >
        <ContentLanguagePicker
          value={draft.voiceLanguage}
          onChange={(lang) => dispatch({ type: "SET_VOICE_LANGUAGE", payload: lang })}
        />

        <div className={styles.scriptLayout}>
          <div className={styles.assistantColumn}>
            <ScriptAssistant
              contentLanguage={draft.voiceLanguage}
              format={formats.find((f) => f.id === draft.videoFormatId)}
              wordsPerMinute={wordsPerMinuteFor(calibration, draft.voiceId)}
              source={draft.scriptSource}
              onSourceChange={(source) => dispatch({ type: "SET_SCRIPT_SOURCE", payload: source })}
              onUseTemplate={() =>
                dispatch({ type: "SET_SCRIPT", payload: SCRIPT_TEMPLATES[draft.voiceLanguage] })
              }
            />
          </div>

          <ScriptEditor
            value={draft.scriptContent}
            onChange={(value) => dispatch({ type: "SET_SCRIPT", payload: value })}
            contentLanguage={draft.voiceLanguage}
            wordsPerMinute={wordsPerMinuteFor(calibration, draft.voiceId)}
          />
        </div>
      </AppShell>

      <WizardNav
        hint={hint}
        isBlocked={!isEmpty && !validation.isValid}
        onNext={() => navigate("/create/settings")}
        nextLabel="Tiếp tục"
        nextDisabled={!canContinue}
        nextTestId="script-step-next"
      />
    </div>
  );
}
