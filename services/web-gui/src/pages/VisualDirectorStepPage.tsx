import { useContext, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AUTHORING_STEP_PATHS } from "../components/AuthoringModeBar";
import { AppShell } from "../components/AppShell";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { getAuthoringState, saveAuthoringStoryboard } from "../api/client";
import { Card, Button, TextArea } from "../components/ui";
import { PipelineSettingsBar } from "../components/PipelineSettingsBar";
import { useLlmStatus } from "../hooks/useLlmStatus";
import { useAuthoringMode } from "../hooks/useAuthoringMode";
import { useRenderedPrompt } from "../hooks/useRenderedPrompt";
import styles from "./WizardSteps.module.css";

/**
 * Bước 1b (Visual Director) — second tab of the "Bước 3 — Script"
 * sub-wizard (see ScriptPipelineTabs): fetch the current template, fill it
 * with the previous tab's saved output, let the Creator copy it out and
 * paste the AI's storyboard back, then save it server-side and advance.
 */
export function VisualDirectorStepPage() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();
  const [copied, setCopied] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  // Rehydrate the saved story from the server on mount — a reload or direct
  // navigation to this step loses the in-memory draft's authoringStory only
  // if localStorage was also cleared, but the server copy is the one source
  // of truth the next pipeline step reads from anyway.
  useEffect(() => {
    if (!draft.projectId || (draft.authoringStory && draft.authoringTopic)) return;
    let cancelled = false;
    getAuthoringState(draft.projectId)
      .then((state) => {
        if (cancelled) return;
        if (state.story && !draft.authoringStory) {
          dispatch({ type: "SET_AUTHORING_STORY", payload: state.story });
        }
        // Chủ đề cũng phải nạp lại để AppShell hiện tên project sau reload.
        if (state.topic && !draft.authoringTopic) {
          dispatch({ type: "SET_AUTHORING_TOPIC", payload: state.topic });
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

  // One director for every render engine: the shooting script is engine
  // agnostic, and only the code step forks (manim_engineer / remotion_engineer).
  const directorRole = "visual_director";

  // CR-040 FR113: rendered by the server, not assembled here.
  const rendered = useRenderedPrompt({
    role: directorRole,
    language: draft.voiceLanguage,
    previous_output: draft.authoringStory,
  });
  const prompt = rendered.prompt ?? (rendered.failed ? `Không tải được template ${directorRole}.` : "Đang tải...");

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
  // CR-031 — "Đã có storyboard" vào thẳng tab này để dán, không để sinh. Dàn ý
  // ở 1a có thể trống hẳn trong trường hợp đó, và đấy là hợp lệ: storyboard là
  // thứ duy nhất bước 1c cần đọc.
  const hasOwnStoryboard = draft.scriptSource === "storyboard";

  async function handleContinue() {
    setSaving(true);
    setSaveError(null);
    try {
      // Đổi storyboard thì code dựng từ bản cũ bị xóa ở server: hỏi trước.
      const saved = await getAuthoringState(draft.projectId).catch(() => null);
      const changed = saved !== null && saved.storyboard !== "" && saved.storyboard !== draft.authoringStoryboard;
      if (changed && saved.code) {
        const ok = window.confirm("Hình ảnh đã thay đổi. Code đã dựng từ bản cũ sẽ bị xóa để làm lại. Tiếp tục?");
        if (!ok) return;
      }
      await saveAuthoringStoryboard(draft.projectId, draft.authoringStoryboard);
      if (changed) {
        dispatch({
          type: "SYNC_AUTHORING",
          payload: { story: draft.authoringStory, storyboard: draft.authoringStoryboard, code: "" },
        });
      }
      navigate("/create/script/code");
    } catch {
      setSaveError("Không lưu được. Vui lòng thử lại.");
    } finally {
      setSaving(false);
    }
  }

  // CR-027 FR79 — cùng một lựa chọn chế độ với tab 1a; nó nằm trong draft nên
  // không phải chọn lại ở đây.
  const llm = useLlmStatus();
  // CR-027 FR79 — chế độ lấy từ project ở server (qua draft), nên mở lại dự án
  // ở bất cứ tab nào, trình duyệt nào, sau restart nào cũng đúng chế độ đã chọn.
  const { mode: authoringMode, setMode: setAuthoringMode } = useAuthoringMode(draft.projectId);
  // Chế độ AI chỉ "thật" khi máy chủ có provider: một draft chọn AI trên máy
  // chưa cấu hình key phải quay về đường copy tay, chứ không mất cả hai.
  const aiMode = authoringMode === "ai" && llm?.enabled === true;

  const hint = saveError
    ? saveError
    : storyboardIsEmpty
      ? hasOwnStoryboard
        ? "Dán storyboard của bạn để tiếp tục"
        : "Dán storyboard từ AI để tiếp tục"
      : `Storyboard đã sẵn sàng — bước tiếp theo sẽ sinh code ${draft.renderEngine === "remotion" ? "Remotion" : "Manim"}`;

  return (
    <div data-testid="visual-director-step-page">
      <AppShell
        currentStep={3}
        title="Bước 3 — Script"
        subtitle={
          hasOwnStoryboard
            ? "Dán storyboard của bạn vào ô bên phải."
            : "Dựng storyboard hình ảnh từ dàn ý."
        }
        wide
      >
        <div className={styles.settingsRow}>
          {/* Không truyền onEngineChange: storyboard đọc dàn ý, không đọc
              engine, nên đây không phải chỗ đổi nó — chỉ tóm tắt để Creator
              biết đang dựng cho engine nào. */}
          <PipelineSettingsBar
            renderEngine={draft.renderEngine}
            llm={llm}
            mode={authoringMode}
            onModeChange={setAuthoringMode}
            projectId={draft.projectId}
            steps={["storyboard"]}
            what="storyboard"
            runDisabled={draft.authoringStory.trim().length === 0}
            runDisabledReason="Cần hoàn thành bước Kịch bản trước."
            onGenerated={(step, content) => {
              if (step === "storyboard") dispatch({ type: "SET_AUTHORING_STORYBOARD", payload: content });
            }}
            onFollow={(step) => navigate(step === "done" ? AUTHORING_STEP_PATHS.code : AUTHORING_STEP_PATHS[step])}
          />
        </div>

        <div className={aiMode ? styles.scriptLayoutSingle : styles.scriptLayout}>
          {/* Ở chế độ AI, cả thẻ prompt không còn việc gì: server render đúng
              văn bản này rồi tự gọi. Đổi lại chế độ là nó quay lại nguyên vẹn. */}
          {!aiMode && (
            <Card
              title="1. Sao chÃ©p prompt"
              hint="Xem lại nội dung, sao chép rồi dán vào ChatGPT, Claude hoặc Gemini."
            >
              <TextArea
                readOnly
                value={prompt}
                rows={18}
                className={styles.promptTextarea}
                data-testid="visual-director-prompt"
              />
              <Button onClick={handleCopy} disabled={rendered.prompt === null || rendered.stale} className={styles.copyButton} data-testid="visual-director-copy">
                {copied ? "Đã sao chép" : "Sao chÃ©p prompt"}
              </Button>
            </Card>
          )}

          <Card
            title={hasOwnStoryboard ? "Storyboard của bạn" : aiMode ? "Storyboard" : "2. Dán kết quả"}
            hint={
              hasOwnStoryboard
                ? "Dán storyboard vào đây, rồi bấm Tiếp tục."
                : aiMode
                  ? "Kết quả của AI hiện ở đây để bạn chỉnh sửa, rồi bấm Tiếp tục."
                  : "Dán storyboard từ AI vào đây, rồi bấm Tiếp tục."
            }
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
        onBack={() => navigate(hasOwnStoryboard ? "/create/script/settings" : "/create/script/outline")}
        backLabel={hasOwnStoryboard ? "Quay lại cấu hình" : "Quay lại Dàn ý"}
        onNext={handleContinue}
        nextLabel={saving ? "Đang lưu..." : "Tiếp tục"}
        nextDisabled={storyboardIsEmpty || saving}
        nextTestId="visual-director-step-next"
      />
    </div>
  );
}
