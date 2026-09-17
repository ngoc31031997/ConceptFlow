import { useContext, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { getPromptTemplate, getAuthoringState, saveAuthoringReview, saveAuthoringCode } from "../api/client";
import { validateScript, stripMarkdownCodeFence } from "../utils/scriptValidation";
import { NARRATION_LANGUAGE_RULE, REMOTION_NARRATION_LANGUAGE_RULE } from "../components/scriptPrompts";
import { Card, Button, TextArea } from "../components/ui";
import { ScriptPipelineTabs } from "../components/ScriptPipelineTabs";
import styles from "./WizardSteps.module.css";

/** Plain-text rendering of validateScript's result, for the {{lint_results}}
 * placeholder — reuses the existing client-side lint instead of calling any
 * backend lint endpoint (none is needed here).
 *
 * feature/remotion-engine: validateScript only understands Manim's
 * self.narrate/ConceptFlowScene conventions — running it against Remotion
 * code would just report a confident-looking but meaningless "LỖI". Remotion
 * has no client-side lint yet (same as ManimEngineerStepPage), so this just
 * says so and asks the reviewer AI to check structure by reading the code.
 */
function renderLintResults(code: string, language: "vi" | "en", isRemotion: boolean): string {
  if (code.trim().length === 0) {
    return `(chưa có code ${isRemotion ? "Remotion" : "Manim"} đã lưu ở bước 1c)`;
  }
  if (isRemotion) {
    return "(Remotion chưa có lint tự động ở phía hệ thống — tự rà code ở trên theo đúng cấu trúc bắt buộc: export const narrations, Composition id=\"creator\" với calculateMetadata, component chính dùng đúng <Segments>.)";
  }
  const validation = validateScript(code, language);
  const lines = [
    validation.isValid ? "OK — không phát hiện lỗi cấu trúc." : `LỖI: ${validation.message}`,
    `Số đoạn lời thoại (self.narrate): ${validation.narrationCount}`,
    `Có class Scene: ${validation.hasSceneClass ? "có" : "KHÔNG — cần class kế thừa ConceptFlowScene"}`,
    `Tổng số từ lời thoại: ${validation.totalWords}`,
    `Ước lượng thời lượng lời thoại: ${Math.round(validation.estimatedNarrationSeconds)}s`,
  ];
  return lines.join("\n");
}

/** Extracts the AI's PASS/REVISE call from the free-text verdict it pasted
 * back — the prompt asks for a leading "VERDICT: PASS|REVISE" line. Only an
 * explicit REVISE blocks continuing; freeform text that doesn't match the
 * expected shape is left unblocked rather than guessing. */
function parseVerdict(verdict: string): "PASS" | "REVISE" | null {
  const match = verdict.match(/VERDICT:\s*(PASS|REVISE)/i);
  return match ? (match[1].toUpperCase() as "PASS" | "REVISE") : null;
}

/**
 * CR-025 step 4 (Script Reviewer) — the final pipeline step: fetch the
 * current template, fill it with story+storyboard+code as {{previous_output}}
 * and a plain-text lint summary as {{lint_results}}, let the Creator copy it
 * out and paste the AI's PASS/REVISE verdict back, save it server-side, then
 * advance to /create/settings (the existing next step). A "Quay lại" link
 * covers the REVISE case, where the Creator needs to go fix the code.
 */
export function ScriptReviewerStepPage() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();
  const [prompt, setPrompt] = useState("Đang tải...");
  const [copied, setCopied] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [verdict, setVerdict] = useState("");

  // Rehydrate saved story/storyboard/code from the server on mount — mirrors
  // ManimEngineerStepPage's own rehydration effect.
  useEffect(() => {
    if (!draft.projectId || (draft.authoringStory && draft.authoringStoryboard && draft.scriptContent)) return;
    let cancelled = false;
    getAuthoringState(draft.projectId)
      .then((state) => {
        if (cancelled) return;
        if (state.story && !draft.authoringStory) {
          dispatch({ type: "SET_AUTHORING_STORY", payload: state.story });
        }
        if (state.storyboard && !draft.authoringStoryboard) {
          dispatch({ type: "SET_AUTHORING_STORYBOARD", payload: state.storyboard });
        }
        if (state.code && !draft.scriptContent) {
          dispatch({ type: "SET_SCRIPT", payload: state.code });
        }
      })
      .catch(() => {
        /* best-effort — the Creator can still go back to fix earlier steps */
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [draft.projectId]);

  const isRemotion = draft.renderEngine === "remotion";
  const engineerRole = isRemotion ? "remotion_engineer" : "manim_engineer";
  const engineerLabel = isRemotion ? "Remotion Engineer" : "Manim Engineer";
  const previousOutput = [draft.authoringStory, draft.authoringStoryboard, draft.scriptContent]
    .filter((part) => part.trim().length > 0)
    .join("\n\n---\n\n");
  const lintResults = renderLintResults(draft.scriptContent, draft.voiceLanguage, isRemotion);

  useEffect(() => {
    let cancelled = false;
    getPromptTemplate("script_reviewer", draft.voiceLanguage)
      .then((template) => {
        if (cancelled) return;
        const filled = template.template_text
          .split("{{previous_output}}")
          .join(previousOutput || "(chưa có dàn ý/storyboard/code đã lưu ở các bước trước)")
          .split("{{lint_results}}")
          .join(lintResults);
        setPrompt(filled);
      })
      .catch(() => {
        if (!cancelled) setPrompt("Không tải được template script_reviewer.");
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [draft.voiceLanguage, previousOutput, lintResults]);

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(prompt);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopied(false);
    }
  }

  const verdictIsEmpty = verdict.trim().length === 0;
  const verdictWord = parseVerdict(verdict);
  const isRevise = verdictWord === "REVISE";
  // The reviewer prompt lists BẮT BUỘC SỬA/NÊN SỬA items under EVERY verdict,
  // including PASS — PASS just means nothing found was blocking. A PASS with
  // "NÊN SỬA" notes still deserves the same fix-prompt tool as REVISE; the
  // Creator should get to choose (fix now, or continue anyway), not be
  // forced either way.
  const hasSuggestions = /BẮT BUỘC SỬA|NÊN SỬA/i.test(verdict);
  const showFixTools = isRevise || (verdictWord === "PASS" && hasSuggestions);

  async function handleContinue() {
    if (isRevise) return;
    setSaving(true);
    setSaveError(null);
    try {
      await saveAuthoringReview(draft.projectId, verdict);
      navigate("/create/settings");
    } catch {
      setSaveError("Không lưu được kết quả đánh giá, thử lại.");
    } finally {
      setSaving(false);
    }
  }

  // REVISE case — instead of sending the Creator back to re-derive an
  // engineer prompt by hand, build one here that already carries the buggy
  // code plus the reviewer's own feedback, so the round trip to fix it is
  // exactly as fast as any other step: copy prompt, paste AI's fixed code
  // back, save. Saving updates draft.scriptContent, which the effects above
  // already react to — the main prompt/lint above regenerate against the
  // fixed code automatically, ready for a fresh verdict.
  const [fixPrompt, setFixPrompt] = useState("Đang tải...");
  const [copiedFix, setCopiedFix] = useState(false);
  const [fixCode, setFixCode] = useState("");
  const [fixSaving, setFixSaving] = useState(false);
  const [fixSaveError, setFixSaveError] = useState<string | null>(null);

  useEffect(() => {
    if (!showFixTools) return;
    let cancelled = false;
    getPromptTemplate(engineerRole, draft.voiceLanguage)
      .then((template) => {
        if (cancelled) return;
        const fixContext = [draft.authoringStory, draft.authoringStoryboard]
          .filter((part) => part.trim().length > 0)
          .join("\n\n---\n\n")
          .concat(
            draft.scriptContent.trim()
              ? `\n\n---\n\nCODE HIỆN TẠI (có điểm cần sửa theo góp ý bên dưới):\n\n${draft.scriptContent}`
              : "",
          )
          .concat(`\n\n---\n\nGÓP Ý CỦA REVIEWER (VERDICT: ${verdictWord}):\n\n${verdict}`);
        const filled = template.template_text
          .split("{{previous_output}}")
          .join(fixContext)
          .split("{{narration_language_rule}}")
          .join(
            isRemotion
              ? REMOTION_NARRATION_LANGUAGE_RULE[draft.voiceLanguage]
              : NARRATION_LANGUAGE_RULE[draft.voiceLanguage],
          );
        setFixPrompt(filled);
      })
      .catch(() => {
        if (!cancelled) setFixPrompt(`Không tải được template ${engineerRole}.`);
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [showFixTools, engineerRole, draft.voiceLanguage, draft.authoringStory, draft.authoringStoryboard, draft.scriptContent, verdict, verdictWord]);

  async function handleCopyFix() {
    try {
      await navigator.clipboard.writeText(fixPrompt);
      setCopiedFix(true);
      setTimeout(() => setCopiedFix(false), 2000);
    } catch {
      setCopiedFix(false);
    }
  }

  async function handleSaveFix() {
    if (fixCode.trim().length === 0) return;
    setFixSaving(true);
    setFixSaveError(null);
    try {
      await saveAuthoringCode(draft.projectId, fixCode);
      dispatch({ type: "SET_SCRIPT", payload: fixCode });
      // The old verdict was about the old code — clear it so the Creator
      // pastes a fresh AI review of the fixed code above, instead of
      // re-submitting a REVISE that no longer describes what's here now.
      setVerdict("");
      setFixCode("");
    } catch {
      setFixSaveError("Không lưu được code đã sửa, thử lại.");
    } finally {
      setFixSaving(false);
    }
  }

  const hint = saveError
    ? saveError
    : verdictIsEmpty
      ? "Dán kết quả đánh giá AI trả về để tiếp tục"
      : isRevise
        ? `AI yêu cầu REVISE — dùng prompt sửa lỗi bên dưới, hoặc quay lại bước 1c (${engineerLabel}) để sửa code trước khi tiếp tục.`
        : hasSuggestions
          ? "PASS — còn vài điểm NÊN SỬA (không bắt buộc). Bấm Tiếp tục để render luôn, hoặc dùng prompt sửa lỗi bên dưới trước rồi mới tiếp tục."
          : "Kết quả đánh giá đã sẵn sàng — bước tiếp theo là cấu hình video";

  return (
    <div data-testid="script-reviewer-step-page">
      <AppShell currentStep={1} title="Bước 1 — Script" subtitle="1d. Rà soát lại toàn bộ dàn ý, storyboard và code trước khi render." wide>
        <ScriptPipelineTabs
          active="review"
          outlineDone={draft.authoringStory.trim().length > 0}
          storyboardDone={draft.authoringStoryboard.trim().length > 0}
          codeDone={draft.scriptContent.trim().length > 0}
        />

        <div className={styles.scriptLayout}>
          <Card
            title="1. Copy prompt"
            hint="Dán vào ChatGPT, Claude hoặc Gemini — đọc lại nội dung, đúng rồi thì copy."
          >
            <TextArea
              readOnly
              value={prompt}
              rows={18}
              className={styles.promptTextarea}
              data-testid="script-reviewer-prompt"
            />
            <Button onClick={handleCopy} className={styles.copyButton} data-testid="script-reviewer-copy">
              {copied ? "Đã copy!" : "Copy prompt"}
            </Button>
          </Card>

          <Card
            title="2. Dán kết quả"
            hint="Dán kết quả đánh giá (PASS/REVISE) AI trả về. PASS thì bấm Tiếp tục; REVISE sẽ mở khối sửa lỗi bên dưới."
          >
            <TextArea
              id="script-reviewer-verdict-input"
              value={verdict}
              onChange={(event) => setVerdict(event.target.value)}
              rows={18}
              className={styles.promptTextarea}
              placeholder={"VERDICT: PASS|REVISE\n\n### NỘI DUNG\n..."}
              data-testid="script-reviewer-verdict-input"
            />
          </Card>
        </div>

        {showFixTools && (
          <div className={styles.scriptLayout} style={{ marginTop: 22 }}>
            <Card
              title="3. Copy prompt sửa lỗi"
              hint="Đã kèm sẵn code hiện tại và góp ý reviewer ở trên — dán vào ChatGPT, Claude hoặc Gemini để lấy code đã sửa."
            >
              <TextArea
                readOnly
                value={fixPrompt}
                rows={18}
                className={styles.promptTextarea}
                data-testid="script-reviewer-fix-prompt"
              />
              <Button onClick={handleCopyFix} className={styles.copyButton} data-testid="script-reviewer-fix-copy">
                {copiedFix ? "Đã copy!" : "Copy prompt sửa lỗi"}
              </Button>
            </Card>

            <Card
              title="4. Dán code đã sửa"
              hint="Lưu xong, prompt đánh giá ở bước 2 sẽ tự cập nhật theo code mới — dán verdict mới của AI để tiếp tục."
            >
              <TextArea
                id="script-reviewer-fix-code-input"
                value={fixCode}
                onChange={(event) => setFixCode(stripMarkdownCodeFence(event.target.value))}
                rows={18}
                className={styles.promptTextarea}
                data-testid="script-reviewer-fix-code-input"
              />
              {fixSaveError && (
                <p role="alert" className={styles.fixSaveError}>
                  {fixSaveError}
                </p>
              )}
              <Button
                onClick={handleSaveFix}
                disabled={fixCode.trim().length === 0 || fixSaving}
                className={styles.copyButton}
                data-testid="script-reviewer-fix-save"
              >
                {fixSaving ? "Đang lưu..." : "Lưu code đã sửa"}
              </Button>
            </Card>
          </div>
        )}
      </AppShell>

      <WizardNav
        hint={hint}
        isBlocked={!!saveError || isRevise}
        onBack={() => navigate("/create/script/code")}
        backLabel={`Quay lại ${engineerLabel} để sửa`}
        onNext={handleContinue}
        nextLabel={saving ? "Đang lưu..." : "Tiếp tục"}
        nextDisabled={verdictIsEmpty || saving || isRevise}
        nextTestId="script-reviewer-step-next"
      />
    </div>
  );
}
