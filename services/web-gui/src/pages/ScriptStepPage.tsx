import { useContext, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { ScriptAssistant } from "../components/ScriptAssistant";
import { ScriptEditor } from "../components/ScriptEditor";
import { useVoiceCalibration, wordsPerMinuteFor } from "../hooks/useVoiceCalibration";
import { useVideoFormats } from "../hooks/useVideoFormats";
import { useDebounce } from "../hooks/useDebounce";
import { ContentLanguagePicker } from "../components/ContentLanguagePicker";
import { RenderEnginePicker } from "../components/RenderEnginePicker";
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

  // feature/remotion-engine: Remotion has no 4-role pipeline (no Visual
  // Director/Manim Engineer/Script Reviewer stub for it) — its "blank"
  // prompt (remotion_engineer, see ScriptAssistant) hands back CODE directly,
  // not a story outline. draft.authoringStory is reused as the paste-back
  // staging field regardless of engine (same textarea either way); only what
  // happens to it on "Tiếp tục" differs — see handleContinue.
  const isRemotionBlank = isStoryMode && draft.renderEngine === "remotion";

  // feature/remotion-engine: the engine picker lives in the Settings step,
  // which comes AFTER this one — so at script-paste time we don't yet know
  // whether this project will render with Manim or Remotion. validateScript
  // only understands Manim's conventions (self.narrate, ConceptFlowScene);
  // running it against pasted Remotion (.tsx) code would just block a
  // Creator who hasn't done anything wrong. Detect "this doesn't look like
  // Manim" instead of asking the Creator to declare the engine twice, and
  // defer the real check to the backend's validate_script step, which
  // already skips Manim lint for engine=remotion (see
  // rendering/application/validate_script.py).
  const looksLikeManim = /from\s+conceptflow\s+import|ConceptFlowScene/.test(draft.scriptContent);
  const skipManimValidation = !isStoryMode && !isEmpty && !looksLikeManim;

  const canContinue = isStoryMode
    ? !storyIsEmpty && !savingStory
    : !isEmpty && (skipManimValidation || validation.isValid);

  const hint = isStoryMode
    ? saveError
      ? saveError
      : storyIsEmpty
        ? isRemotionBlank
          ? "Dán code Remotion AI trả về để tiếp tục"
          : "Dán dàn ý câu chuyện AI trả về để tiếp tục"
        : isRemotionBlank
          ? "Code đã sẵn sàng — bước tiếp theo là cấu hình render"
          : "Dàn ý đã sẵn sàng — bước tiếp theo sẽ dựng storyboard hình ảnh"
    : isEmpty
      ? "Dán hoặc tạo script Manim để tiếp tục"
      : skipManimValidation
        ? "Không phải script Manim — bỏ qua kiểm tra ở đây, hệ thống sẽ kiểm tra thật ở bước render"
        : validation.isValid
          ? `Script hợp lệ — ${validation.narrationCount} đoạn lời thoại`
          : validation.message;

  async function handleContinue() {
    if (!isStoryMode) {
      navigate("/create/settings");
      return;
    }
    if (isRemotionBlank) {
      // No server-side authoring pipeline for Remotion yet — the pasted
      // content IS the final script, same as the "ready" source path.
      dispatch({ type: "SET_SCRIPT", payload: draft.authoringStory });
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
        title={draft.renderEngine === "remotion" ? "Bước 1 — Script Remotion" : "Bước 1 — Script Manim"}
        subtitle={
          draft.renderEngine === "remotion"
            ? "Hệ thống render script Remotion của bạn thành video. Trước tiên cần một script có đánh dấu lời thoại."
            : "Hệ thống render script Manim của bạn thành video. Trước tiên cần một script có đánh dấu lời thoại."
        }
      >
        <div className={styles.settingsRow}>
          <ContentLanguagePicker
            value={draft.voiceLanguage}
            onChange={(lang) => dispatch({ type: "SET_VOICE_LANGUAGE", payload: lang })}
          />

          {/* feature/remotion-engine: chọn engine NGAY Ở ĐÂY, không phải chỉ
              ở bước Cấu hình — prompt bên dưới (ScriptAssistant) đổi ngay
              theo lựa chọn này, nên Creator cần thấy nó trước khi copy
              prompt, không phải sau khi quay lại từ bước 2. */}
          <RenderEnginePicker
            value={draft.renderEngine}
            onChange={(engine) => {
              dispatch({ type: "SET_RENDER_ENGINE", payload: engine });
              // "draft"/"ready" only mean anything for a Manim script
              // (self.narrate/ConceptFlowScene) — switching to Remotion
              // while one of those is selected would leave the Creator on
              // an option ScriptAssistant no longer offers.
              if (engine === "remotion" && draft.scriptSource !== "blank") {
                dispatch({ type: "SET_SCRIPT_SOURCE", payload: "blank" });
              }
            }}
          />
        </div>

        {/* CR-025: "blank" has no ScriptEditor to show (see below) — the
            2-column grid used to reserve that column's width anyway, leaving
            the assistant squeezed into ~40% of the page with the other 60%
            empty. Single column, full width, when there's nothing to put in
            the second one. */}
        <div className={isStoryMode ? styles.scriptLayoutSingle : styles.scriptLayout}>
          <div className={styles.assistantColumn}>
            <ScriptAssistant
              contentLanguage={draft.voiceLanguage}
              format={formats.find((f) => f.id === draft.videoFormatId)}
              wordsPerMinute={wordsPerMinuteFor(calibration, draft.voiceId)}
              renderEngine={draft.renderEngine}
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
        isBlocked={isStoryMode ? !!saveError : !isEmpty && !skipManimValidation && !validation.isValid}
        onNext={handleContinue}
        nextLabel={isStoryMode ? (savingStory ? "Đang lưu..." : "Tiếp tục") : "Tiếp tục"}
        nextDisabled={!canContinue}
        nextTestId="script-step-next"
      />
    </div>
  );
}
