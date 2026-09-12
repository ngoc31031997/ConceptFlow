import { useContext, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { getPromptTemplate, getAuthoringState, saveAuthoringReview } from "../api/client";
import { validateScript } from "../utils/scriptValidation";
import styles from "./WizardSteps.module.css";

/** Plain-text rendering of validateScript's result, for the {{lint_results}}
 * placeholder — reuses the existing client-side lint instead of calling any
 * backend lint endpoint (none is needed here). */
function renderLintResults(code: string, language: "vi" | "en"): string {
  const validation = validateScript(code, language);
  if (code.trim().length === 0) {
    return "(chưa có code Manim đã lưu ở bước 3)";
  }
  const lines = [
    validation.isValid ? "OK — không phát hiện lỗi cấu trúc." : `LỖI: ${validation.message}`,
    `Số đoạn lời thoại (self.narrate): ${validation.narrationCount}`,
    `Có class Scene: ${validation.hasSceneClass ? "có" : "KHÔNG — cần class kế thừa ConceptFlowScene"}`,
    `Tổng số từ lời thoại: ${validation.totalWords}`,
    `Ước lượng thời lượng lời thoại: ${Math.round(validation.estimatedNarrationSeconds)}s`,
  ];
  return lines.join("\n");
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

  const previousOutput = [draft.authoringStory, draft.authoringStoryboard, draft.scriptContent]
    .filter((part) => part.trim().length > 0)
    .join("\n\n---\n\n");
  const lintResults = renderLintResults(draft.scriptContent, draft.voiceLanguage);

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

  async function handleContinue() {
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

  const hint = saveError
    ? saveError
    : verdictIsEmpty
      ? "Dán kết quả đánh giá AI trả về để tiếp tục"
      : "Kết quả đánh giá đã sẵn sàng — bước tiếp theo là cấu hình video";

  return (
    <div data-testid="script-reviewer-step-page">
      <AppShell
        title="Bước 4 — Script Reviewer"
        subtitle="Rà soát lại toàn bộ dàn ý, storyboard và code trước khi render."
        wide
      >
        <div className={styles.scriptLayout}>
          <div>
            <p>
              1. Copy prompt bên dưới và dán vào ChatGPT, Claude hoặc Gemini. 2. Dán kết quả đánh giá
              (PASS/REVISE) AI trả về vào ô phía dưới. 3. Nếu REVISE, quay lại bước 3 để sửa code. Nếu PASS,
              bấm Tiếp tục để lưu và chuyển sang bước cấu hình video.
            </p>
            <div>
              <button type="button" onClick={handleCopy} data-testid="script-reviewer-copy">
                {copied ? "Đã copy!" : "Copy prompt"}
              </button>
            </div>
            <textarea
              readOnly
              value={prompt}
              rows={20}
              style={{ width: "100%", fontFamily: "monospace" }}
              data-testid="script-reviewer-prompt"
            />

            <label htmlFor="script-reviewer-verdict-input" style={{ display: "block", marginTop: "1rem" }}>
              Dán kết quả đánh giá AI trả về vào đây
            </label>
            <textarea
              id="script-reviewer-verdict-input"
              value={verdict}
              onChange={(event) => setVerdict(event.target.value)}
              rows={16}
              style={{ width: "100%", fontFamily: "monospace" }}
              placeholder={"VERDICT: PASS|REVISE\n\n### NỘI DUNG\n..."}
              data-testid="script-reviewer-verdict-input"
            />
          </div>
        </div>
      </AppShell>

      <WizardNav
        hint={hint}
        isBlocked={!!saveError}
        onBack={() => navigate("/create/manim-engineer")}
        backLabel="Quay lại Manim Engineer để sửa"
        onNext={handleContinue}
        nextLabel={saving ? "Đang lưu..." : "Tiếp tục"}
        nextDisabled={verdictIsEmpty || saving}
        nextTestId="script-reviewer-step-next"
      />
    </div>
  );
}
