import { useContext, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { getPromptTemplate, getAuthoringState, saveAuthoringStoryboard } from "../api/client";
import { Card, Button, TextArea } from "../components/ui";
import { ScriptPipelineTabs } from "../components/ScriptPipelineTabs";
import styles from "./WizardSteps.module.css";

/**
 * Bước 1b (Visual Director) — second tab of the "Bước 1 — Script"
 * sub-wizard (see ScriptPipelineTabs): fetch the current template, fill it
 * with the previous tab's saved output, let the Creator copy it out and
 * paste the AI's storyboard back, then save it server-side and advance.
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
  const engineerLabel = draft.renderEngine === "remotion" ? "Remotion Engineer" : "Manim Engineer";

  async function handleContinue() {
    setSaving(true);
    setSaveError(null);
    try {
      await saveAuthoringStoryboard(draft.projectId, draft.authoringStoryboard);
      navigate("/create/script/code");
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
      : `Storyboard đã sẵn sàng — bước tiếp theo sẽ sinh code ${draft.renderEngine === "remotion" ? "Remotion" : "Manim"}`;

  return (
    <div data-testid="visual-director-step-page">
      <AppShell currentStep={1} title="Bước 1 — Script" subtitle="1b. Dựng storyboard hình ảnh từ dàn ý câu chuyện." wide>
        <ScriptPipelineTabs
          active="storyboard"
          outlineDone={draft.authoringStory.trim().length > 0}
          storyboardDone={!storyboardIsEmpty}
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
              data-testid="visual-director-prompt"
            />
            <Button onClick={handleCopy} className={styles.copyButton} data-testid="visual-director-copy">
              {copied ? "Đã copy!" : "Copy prompt"}
            </Button>
          </Card>

          <Card
            title="2. Dán kết quả"
            hint={`Dán storyboard AI trả về, rồi bấm Tiếp tục để chuyển sang bước 1c (${engineerLabel}).`}
          >
            <TextArea
              id="storyboard-input"
              value={draft.authoringStoryboard}
              onChange={(event) =>
                dispatch({ type: "SET_AUTHORING_STORYBOARD", payload: event.target.value })
              }
              rows={18}
              placeholder={"CẢNH 1 — ...\nCẢNH 2 — ..."}
              data-testid="visual-director-storyboard-input"
            />
          </Card>
        </div>
      </AppShell>

      <WizardNav
        hint={hint}
        isBlocked={!!saveError}
        onBack={() => navigate("/create/script/outline")}
        backLabel="Quay lại Dàn ý"
        onNext={handleContinue}
        nextLabel={saving ? "Đang lưu..." : "Tiếp tục"}
        nextDisabled={storyboardIsEmpty || saving}
        nextTestId="visual-director-step-next"
      />
    </div>
  );
}
