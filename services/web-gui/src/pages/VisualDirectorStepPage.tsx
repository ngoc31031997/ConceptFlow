import { useContext, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { getPromptTemplate, getAuthoringState, saveAuthoringStoryboard } from "../api/client";
import styles from "./WizardSteps.module.css";

/**
 * CR-025 step 2 (Visual Director) — mirrors step 1 (ScriptStepPage +
 * ScriptAssistant)'s round-trip-through-an-external-AI shape: fetch the
 * current template, fill it with the previous step's saved output, let the
 * Creator copy it out and paste the AI's storyboard back, then save it
 * server-side and advance.
 *
 * Steps 3-4 (Manim Engineer code generation, Script Reviewer verdict) stay a
 * documented TODO — this only makes step 2 itself real.
 */
export function VisualDirectorStepPage() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();
  const [prompt, setPrompt] = useState("Đang tải...");
  const [copied, setCopied] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  // Rehydrate the saved story from the server on mount — a reload or direct
  // navigation to this step loses the in-memory draft's authoringStory only
  // if localStorage was also cleared, but the server copy is the one source
  // of truth the next pipeline step reads from anyway.
  useEffect(() => {
    if (!draft.projectId || draft.authoringStory) return;
    let cancelled = false;
    getAuthoringState(draft.projectId)
      .then((state) => {
        if (!cancelled && state.story) {
          dispatch({ type: "SET_AUTHORING_STORY", payload: state.story });
        }
      })
      .catch(() => {
        /* best-effort — the Creator can still go back to step 1 */
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [draft.projectId]);

  useEffect(() => {
    let cancelled = false;
    getPromptTemplate("visual_director", draft.voiceLanguage)
      .then((template) => {
        if (cancelled) return;
        const filled = template.template_text.split("{{previous_output}}").join(
          draft.authoringStory || "(chưa có dàn ý câu chuyện đã lưu ở bước 1)",
        );
        setPrompt(filled);
      })
      .catch(() => {
        if (!cancelled) setPrompt("Không tải được template visual_director.");
      });
    return () => {
      cancelled = true;
    };
  }, [draft.voiceLanguage, draft.authoringStory]);

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(prompt);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopied(false);
    }
  }

  const storyboardIsEmpty = draft.authoringStoryboard.trim().length === 0;

  async function handleContinue() {
    setSaving(true);
    setSaveError(null);
    try {
      await saveAuthoringStoryboard(draft.projectId, draft.authoringStoryboard);
      navigate("/create/manim-engineer");
    } catch {
      setSaveError("Không lưu được storyboard, thử lại.");
    } finally {
      setSaving(false);
    }
  }

  const hint = saveError
    ? saveError
    : storyboardIsEmpty
      ? "Dán storyboard AI trả về để tiếp tục"
      : "Storyboard đã sẵn sàng — bước tiếp theo sẽ sinh code Manim";

  return (
    <div data-testid="visual-director-step-page">
      <AppShell
        title="Bước 2 — Visual Director"
        subtitle="Dựng storyboard hình ảnh từ dàn ý câu chuyện đã lưu ở bước 1."
        wide
      >
        <div className={styles.scriptLayout}>
          <div>
            <p>
              1. Copy prompt bên dưới và dán vào ChatGPT, Claude hoặc Gemini. 2. Dán storyboard AI trả về vào
              ô phía dưới. 3. Bấm Tiếp tục để lưu và chuyển sang bước 3 (Manim Engineer).
            </p>
            <div>
              <button type="button" onClick={handleCopy} data-testid="visual-director-copy">
                {copied ? "Đã copy!" : "Copy prompt"}
              </button>
            </div>
            <textarea
              readOnly
              value={prompt}
              rows={20}
              style={{ width: "100%", fontFamily: "monospace" }}
              data-testid="visual-director-prompt"
            />

            <label htmlFor="storyboard-input" style={{ display: "block", marginTop: "1rem" }}>
              Dán storyboard AI trả về vào đây
            </label>
            <textarea
              id="storyboard-input"
              value={draft.authoringStoryboard}
              onChange={(event) =>
                dispatch({ type: "SET_AUTHORING_STORYBOARD", payload: event.target.value })
              }
              rows={12}
              style={{ width: "100%" }}
              placeholder={"CẢNH 1 — ...\nCẢNH 2 — ..."}
              data-testid="visual-director-storyboard-input"
            />
          </div>
        </div>
      </AppShell>

      <WizardNav
        hint={hint}
        isBlocked={!!saveError}
        onNext={handleContinue}
        nextLabel={saving ? "Đang lưu..." : "Tiếp tục"}
        nextDisabled={storyboardIsEmpty || saving}
        nextTestId="visual-director-step-next"
      />
    </div>
  );
}
