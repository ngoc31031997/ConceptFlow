import { useContext, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { getPromptTemplate, getAuthoringState, saveAuthoringCode } from "../api/client";
import { validateScript } from "../utils/scriptValidation";
import styles from "./WizardSteps.module.css";

/**
 * CR-025 step 3 (Manim Engineer) — mirrors step 2 (Visual Director)'s
 * round-trip-through-an-external-AI shape: fetch the current template, fill
 * it with the previous steps' saved output (story + storyboard), let the
 * Creator copy it out and paste the AI's Manim code back, validate it with
 * the same client-side lint ScriptStepPage uses for "draft"/"ready" scripts,
 * then save it server-side, store it as the draft's scriptContent, and
 * advance to step 4 (Script Reviewer).
 */
export function ManimEngineerStepPage() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();
  const [prompt, setPrompt] = useState("Đang tải...");
  const [copied, setCopied] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  // Rehydrate the saved story/storyboard from the server on mount — mirrors
  // VisualDirectorStepPage's own rehydration effect.
  useEffect(() => {
    if (!draft.projectId || (draft.authoringStory && draft.authoringStoryboard)) return;
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
      })
      .catch(() => {
        /* best-effort — the Creator can still go back to fix earlier steps */
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [draft.projectId]);

  const previousOutput = [draft.authoringStory, draft.authoringStoryboard]
    .filter((part) => part.trim().length > 0)
    .join("\n\n---\n\n");

  useEffect(() => {
    let cancelled = false;
    getPromptTemplate("manim_engineer", draft.voiceLanguage)
      .then((template) => {
        if (cancelled) return;
        const filled = template.template_text.split("{{previous_output}}").join(
          previousOutput || "(chưa có dàn ý/storyboard đã lưu ở các bước trước)",
        );
        setPrompt(filled);
      })
      .catch(() => {
        if (!cancelled) setPrompt("Không tải được template manim_engineer.");
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [draft.voiceLanguage, previousOutput]);

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(prompt);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopied(false);
    }
  }

  const [code, setCode] = useState(draft.scriptContent);
  const validation = validateScript(code, draft.voiceLanguage);
  const isEmpty = code.trim().length === 0;
  const isValid = !isEmpty && validation.isValid;

  async function handleContinue() {
    if (!isValid) return;
    setSaving(true);
    setSaveError(null);
    try {
      await saveAuthoringCode(draft.projectId, code);
      dispatch({ type: "SET_SCRIPT", payload: code });
      navigate("/create/script-reviewer");
    } catch {
      setSaveError("Không lưu được code, thử lại.");
    } finally {
      setSaving(false);
    }
  }

  const hint = saveError
    ? saveError
    : isEmpty
      ? "Dán code Manim AI trả về để tiếp tục"
      : validation.isValid
        ? `Code hợp lệ — ${validation.narrationCount} đoạn lời thoại`
        : validation.message;

  return (
    <div data-testid="manim-engineer-step-page">
      <AppShell
        title="Bước 3 — Manim Engineer"
        subtitle="Sinh code Manim từ storyboard đã lưu ở bước 2."
        wide
      >
        <div className={styles.scriptLayout}>
          <div>
            <p>
              1. Copy prompt bên dưới và dán vào ChatGPT, Claude hoặc Gemini. 2. Dán code Manim AI trả về vào
              ô phía dưới. 3. Bấm Tiếp tục để lưu và chuyển sang bước 4 (Script Reviewer).
            </p>
            <div>
              <button type="button" onClick={handleCopy} data-testid="manim-engineer-copy">
                {copied ? "Đã copy!" : "Copy prompt"}
              </button>
            </div>
            <textarea
              readOnly
              value={prompt}
              rows={20}
              style={{ width: "100%", fontFamily: "monospace" }}
              data-testid="manim-engineer-prompt"
            />

            <label htmlFor="manim-engineer-code-input" style={{ display: "block", marginTop: "1rem" }}>
              Dán code Manim AI trả về vào đây
            </label>
            <textarea
              id="manim-engineer-code-input"
              value={code}
              onChange={(event) => setCode(event.target.value)}
              rows={20}
              style={{ width: "100%", fontFamily: "monospace" }}
              placeholder={"from conceptflow import *\n\nclass ...Scene(ConceptFlowScene):\n    def construct(self):\n        ..."}
              data-testid="manim-engineer-code-input"
            />
          </div>
        </div>
      </AppShell>

      <WizardNav
        hint={hint}
        isBlocked={!!saveError || (!isEmpty && !validation.isValid)}
        onNext={handleContinue}
        nextLabel={saving ? "Đang lưu..." : "Tiếp tục"}
        nextDisabled={!isValid || saving}
        nextTestId="manim-engineer-step-next"
      />
    </div>
  );
}
