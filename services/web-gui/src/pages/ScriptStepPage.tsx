import { useContext, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { ScriptAssistant } from "../components/ScriptAssistant";
import { ScriptEditor } from "../components/ScriptEditor";
import { useVoiceCalibration, wordsPerMinuteFor } from "../hooks/useVoiceCalibration";
import { useVideoFormats } from "../hooks/useVideoFormats";
import { useDebounce } from "../hooks/useDebounce";
import { ContentLanguagePicker } from "../components/ContentLanguagePicker";
import { WizardNav } from "../components/WizardNav";
import { SCRIPT_TEMPLATES } from "../components/scriptTemplates";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { validateScript } from "../utils/scriptValidation";
import { saveAuthoringStory } from "../api/client";
import styles from "./WizardSteps.module.css";

/**
 * Step 1 of 5 (AppShell's STEP_LABELS) — get a valid script. Steps 4/5
 * (render/publish) only became visible screens after CR-024/CR-021; this
 * docstring used to still say "of 3" from before that.
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
  
  // Debounce script content for validation to avoid re-validating on every keystroke
  const debouncedScriptContent = useDebounce(draft.scriptContent, 500);
  
  const validation = validateScript(
    debouncedScriptContent,
    draft.voiceLanguage,
    wordsPerMinuteFor(calibration, draft.voiceId),
  );
  const isEmpty = draft.scriptContent.trim().length === 0;

  // CR-025: source "blank" now goes through the Story Architect pipeline —
  // "Tiếp tục" saves the pasted story outline server-side and advances to
  // the (stub) Visual Director step, instead of validating Manim code.
  const isStoryMode = draft.scriptSource === "blank";
  const [savingStory, setSavingStory] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const storyIsEmpty = draft.authoringStory.trim().length === 0;

  const canContinue = isStoryMode ? !storyIsEmpty && !savingStory : !isEmpty && validation.isValid;

  const hint = isStoryMode
    ? saveError
      ? saveError
      : storyIsEmpty
        ? "Dán dàn ý câu chuyện AI trả về để tiếp tục"
        : "Dàn ý đã sẵn sàng — bước tiếp theo sẽ dựng storyboard hình ảnh"
    : isEmpty
      ? "Dán hoặc tạo script Manim để tiếp tục"
      : validation.isValid
        ? `Script hợp lệ — ${validation.narrationCount} đoạn lời thoại`
        : validation.message;

  async function handleContinue() {
    if (!isStoryMode) {
      navigate("/create/settings");
      return;
    }
    setSavingStory(true);
    setSaveError(null);
    try {
      await saveAuthoringStory(draft.projectId, draft.authoringStory);
      navigate("/create/visual-director");
    } catch {
      setSaveError("Không lưu được dàn ý, thử lại.");
    } finally {
      setSavingStory(false);
    }
  }

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
              storyOutline={draft.authoringStory}
              onStoryOutlineChange={(value) => dispatch({ type: "SET_AUTHORING_STORY", payload: value })}
            />
          </div>

          {/* CR-025: "blank" no longer pastes Manim code directly here — the
              story outline textarea inside ScriptAssistant replaces this
              step until the pipeline (Visual Director → Manim Engineer)
              produces actual code. */}
          {!isStoryMode && (
            <ScriptEditor
              value={draft.scriptContent}
              onChange={(value) => dispatch({ type: "SET_SCRIPT", payload: value })}
              contentLanguage={draft.voiceLanguage}
              wordsPerMinute={wordsPerMinuteFor(calibration, draft.voiceId)}
            />
          )}
        </div>
      </AppShell>

      <WizardNav
        hint={hint}
        isBlocked={isStoryMode ? !!saveError : !isEmpty && !validation.isValid}
        onNext={handleContinue}
        nextLabel={isStoryMode ? (savingStory ? "Đang lưu..." : "Tiếp tục") : "Tiếp tục"}
        nextDisabled={!canContinue}
        nextTestId="script-step-next"
      />
    </div>
  );
}
