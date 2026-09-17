import { useContext, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { Card } from "../components/ui";
import { SelectableOption } from "../components/SelectableOption";
import { ScriptAssistant } from "../components/ScriptAssistant";
import type { ScriptSource } from "../context/ProjectDraftContext";
import { ScriptEditor } from "../components/ScriptEditor";
import { useVoiceCalibration, wordsPerMinuteFor } from "../hooks/useVoiceCalibration";
import { useDebounce } from "../hooks/useDebounce";
import { ContentLanguagePicker } from "../components/ContentLanguagePicker";
import { RenderEnginePicker } from "../components/RenderEnginePicker";
import { WizardNav } from "../components/WizardNav";
import { SCRIPT_TEMPLATES } from "../components/scriptTemplates";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { validateScript } from "../utils/scriptValidation";
import selectable from "../styles/selectable.module.css";
import styles from "./WizardSteps.module.css";

/**
 * Same three situations for both engines — only the wording differs
 * (self.narrate/ConceptFlowScene for Manim vs narrations/Composition
 * id="creator" for Remotion).
 */
function situationOptions(
  renderEngine: "manim" | "remotion",
): { value: "blank" | "draft" | "ready"; label: string; hint: string }[] {
  if (renderEngine === "remotion") {
    return [
      {
        value: "blank",
        label: "Chưa có gì, chỉ có ý tưởng",
        hint: "Dựng dàn ý → storyboard → code cùng AI, từng bước một (chuyển sang màn dựng script)",
      },
      {
        value: "draft",
        label: "Đã có code Remotion",
        hint: "Nhưng chưa đúng chuẩn hệ thống (thiếu narrations/Composition id=\"creator\") — AI sẽ chỉnh lại giúp bạn",
      },
      {
        value: "ready",
        label: "Code đã đúng chuẩn",
        hint: "Đã có export const narrations, Composition id=\"creator\" dùng <Segments> — dán thẳng vào là chạy",
      },
    ];
  }
  return [
    {
      value: "blank",
      label: "Chưa có gì, chỉ có ý tưởng",
      hint: "Dựng dàn ý → storyboard → code cùng AI, từng bước một (chuyển sang màn dựng script)",
    },
    {
      value: "draft",
      label: "Đã có script Manim",
      hint: "Nhưng chưa có lời thoại self.narrate(...) — AI sẽ thêm giúp bạn",
    },
    {
      value: "ready",
      label: "Script đã đúng chuẩn",
      hint: "Đã dùng self.narrate(\"...\"), kế thừa ConceptFlowScene — dán thẳng vào là chạy",
    },
  ];
}

/**
 * Step 1 of 5 (AppShell's STEP_LABELS) — get a valid script.
 *
 * Only picks the situation now. "Chưa có gì, chỉ có ý tưởng" (blank) hands
 * off immediately to the 4-tab sub-wizard (dàn ý → storyboard → code →
 * duyệt, see ScriptPipelineTabs / ScriptOutlineStepPage) instead of being
 * handled inline here — that pipeline's own tabs are each reachable at any
 * time regardless of progress, which a radio button on this page can't
 * offer. "draft"/"ready" stay here: both are a single step (adjust an
 * existing script, or just paste one that's already correct), with no
 * pipeline to walk through.
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

  // Debounce script content for validation to avoid re-validating on every keystroke
  const debouncedScriptContent = useDebounce(draft.scriptContent, 500);
  const validation = validateScript(
    debouncedScriptContent,
    draft.voiceLanguage,
    wordsPerMinuteFor(calibration, draft.voiceId),
  );
  const isEmpty = draft.scriptContent.trim().length === 0;
  const isDraftOrReady = draft.scriptSource !== "blank";

  // feature/remotion-engine: validateScript only understands Manim's
  // conventions (self.narrate, ConceptFlowScene); running it against pasted
  // Remotion (.tsx) code would just block a Creator who hasn't done anything
  // wrong. Remotion has no client-side lint yet — the real check happens at
  // render time either way (rendering/application/validate_script.py
  // already skips Manim lint for engine=remotion).
  const skipManimValidation = isDraftOrReady && !isEmpty && draft.renderEngine === "remotion";
  const canContinue = isDraftOrReady && !isEmpty && (skipManimValidation || validation.isValid);

  function handleSituationSelect(source: ScriptSource) {
    dispatch({ type: "SET_SCRIPT_SOURCE", payload: source });
    if (source === "blank") navigate("/create/script/outline");
  }

  function handleContinue() {
    navigate("/create/settings");
  }

  const hint = !isDraftOrReady
    ? "Chọn 'Chưa có gì, chỉ có ý tưởng' sẽ tự chuyển sang màn dựng script."
    : isEmpty
      ? `Dán hoặc tạo script ${draft.renderEngine === "remotion" ? "Remotion" : "Manim"} để tiếp tục`
      : skipManimValidation
        ? "Không phải script Manim — bỏ qua kiểm tra ở đây, hệ thống sẽ kiểm tra thật ở bước render"
        : validation.isValid
          ? `Script hợp lệ — ${validation.narrationCount} đoạn lời thoại`
          : validation.message;

  return (
    <div data-testid="script-step-page">
      <AppShell
        currentStep={1}
        wide
        title={draft.renderEngine === "remotion" ? "Bước 1 — Script Remotion" : "Bước 1 — Script Manim"}
        subtitle="Chọn tình huống của bạn để bắt đầu."
      >
        <div className={styles.settingsRow}>
          <ContentLanguagePicker
            value={draft.voiceLanguage}
            onChange={(lang) => dispatch({ type: "SET_VOICE_LANGUAGE", payload: lang })}
          />
        </div>

        <Card title="Bạn đang ở tình huống nào?">
          <div className={selectable.stack} role="radiogroup" aria-label="Tình huống script">
            {situationOptions(draft.renderEngine).map((option) => (
              <SelectableOption
                key={option.value}
                selected={draft.scriptSource === option.value}
                onSelect={() => handleSituationSelect(option.value)}
                label={option.label}
                hint={option.hint}
                testId={`script-source-${option.value}`}
              />
            ))}
          </div>
        </Card>

        {isDraftOrReady && (
          <>
            <div className={styles.settingsRow}>
              <RenderEnginePicker
                value={draft.renderEngine}
                onChange={(engine) => dispatch({ type: "SET_RENDER_ENGINE", payload: engine })}
              />
            </div>

            <div className={styles.scriptLayout}>
              <div className={styles.assistantColumn}>
                <ScriptAssistant
                  contentLanguage={draft.voiceLanguage}
                  renderEngine={draft.renderEngine}
                  source={draft.scriptSource as "draft" | "ready"}
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
                renderEngine={draft.renderEngine}
              />
            </div>
          </>
        )}
      </AppShell>

      <WizardNav
        hint={hint}
        isBlocked={isDraftOrReady && !isEmpty && !skipManimValidation && !validation.isValid}
        onNext={handleContinue}
        nextLabel="Tiếp tục"
        nextDisabled={!canContinue}
        nextTestId="script-step-next"
      />
    </div>
  );
}
