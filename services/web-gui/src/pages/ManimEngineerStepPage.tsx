import { useContext, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { getPromptTemplate, getAuthoringState, saveAuthoringCode } from "../api/client";
import { validateScript, validateRemotionScript, stripMarkdownCodeFence } from "../utils/scriptValidation";
import { NARRATION_LANGUAGE_RULE, REMOTION_NARRATION_LANGUAGE_RULE } from "../components/scriptPrompts";
import { Card, Button, TextArea } from "../components/ui";
import { ScriptPipelineTabs } from "../components/ScriptPipelineTabs";
import { RenderEnginePicker } from "../components/RenderEnginePicker";
import { AuthoringModeBar } from "../components/AuthoringModeBar";
import { useLlmStatus } from "../hooks/useLlmStatus";
import { useAuthoringMode } from "../hooks/useAuthoringMode";
import styles from "./WizardSteps.module.css";

const TOPIC_PLACEHOLDER = "[DÁN CHỦ ĐỀ CỦA BẠN VÀO ĐÂY]";

/**
 * Bước 1c (Engineer) — third tab of the "Bước 1 — Script" sub-wizard (see
 * ScriptPipelineTabs): fetch the current template, fill it with the
 * previous tabs' saved output (story + storyboard), let the Creator copy it
 * out and paste the AI's code back, then save it server-side, store it as
 * the draft's scriptContent, and advance to tab 1d (Duyệt).
 *
 * feature/remotion-engine: the render engine picker lives HERE, not on the
 * situation-chooser page — tabs 1a/1b (story/storyboard) are identical
 * either way; this is the only tab whose prompt role (manim_engineer vs
 * remotion_engineer) and lint behavior (validateScript only understands
 * Manim's self.narrate/ConceptFlowScene conventions; Remotion has no
 * client-side lint yet) actually depend on which engine renders the video.
 */
export function ManimEngineerStepPage() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();
  const isRemotion = draft.renderEngine === "remotion";
  const engineerRole = isRemotion ? "remotion_engineer" : "manim_engineer";
  const engineerLabel = isRemotion ? "Remotion Engineer" : "Manim Engineer";
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
    getPromptTemplate(engineerRole, draft.voiceLanguage)
      .then((template) => {
        if (cancelled) return;
        const filled = template.template_text
          // feature/remotion-engine: the remotion_engineer prompt is a flat
          // topic -> code prompt, so it carries {{topic}} instead of only
          // {{previous_output}}. Without this substitution the Creator copies
          // out a prompt that still literally says "paste your topic here"
          // and the AI writes a video about nothing in particular.
          .split("{{topic}}")
          .join(draft.authoringTopic.trim() || TOPIC_PLACEHOLDER)
          .split("{{previous_output}}")
          .join(previousOutput || "(chưa có dàn ý/storyboard đã lưu ở các bước trước)")
          .split("{{narration_language_rule}}")
          .join(
            isRemotion
              ? REMOTION_NARRATION_LANGUAGE_RULE[draft.voiceLanguage]
              : NARRATION_LANGUAGE_RULE[draft.voiceLanguage],
          );
        setPrompt(filled);
      })
      .catch(() => {
        if (!cancelled) setPrompt(`Không tải được template ${engineerRole}.`);
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [draft.voiceLanguage, draft.authoringTopic, previousOutput, engineerRole]);

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(prompt);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopied(false);
    }
  }

  // Bound directly to draft.scriptContent (not a local buffer) so switching
  // to another tab and back — now that all 4 tabs are freely reachable —
  // never loses code that hasn't been through "Tiếp tục" yet.
  const code = draft.scriptContent;
  // The engineer prompt asks the AI to wrap its answer in a ```python/```tsx
  // fence — pasting that whole block (fence included) is the single most
  // common way this round trip fails: the fence markers are not valid
  // Python/TSX, so esbuild/ast.parse chokes on line 1 with a syntax error
  // that says nothing about the real cause. Strip it the same way
  // ScriptEditor already does for the "draft"/"ready" situations.
  const setCode = (value: string) => dispatch({ type: "SET_SCRIPT", payload: stripMarkdownCodeFence(value) });
  // validateScript only understands Manim's self.narrate/ConceptFlowScene
  // conventions; validateRemotionScript checks the structural rules the
  // remotion_engineer prompt requires (narrations export, Composition
  // id="creator", calculateMetadata, <Segments>, balanced braces) — same
  // idea as validateScript, different syntax. Neither catches everything
  // the prompt's self-check asks for (e.g. overlapping full-frame JSX), but
  // both catch the recurring failure modes actually hit in production
  // before wasting a render cycle on them.
  const validation = isRemotion ? validateRemotionScript(code) : validateScript(code, draft.voiceLanguage);
  const isEmpty = code.trim().length === 0;
  const isValid = !isEmpty && validation.isValid;

  async function handleContinue() {
    if (!isValid) return;
    setSaving(true);
    setSaveError(null);
    try {
      await saveAuthoringCode(draft.projectId, code);
      navigate("/create/script/review");
    } catch {
      setSaveError("Không lưu được code, thử lại.");
    } finally {
      setSaving(false);
    }
  }

  const hint = saveError
    ? saveError
    : isEmpty
      ? `Dán code ${engineerLabel} AI trả về để tiếp tục`
      : validation.isValid
        ? `Code hợp lệ — ${validation.narrationCount} đoạn lời thoại`
        : validation.message;

  // CR-027 FR79 — cùng lựa chọn chế độ với các tab khác của bước 1.
  const llm = useLlmStatus();
  // CR-027 FR79 — chế độ lấy từ project ở server (qua draft), nên mở lại dự án
  // ở bất cứ tab nào, trình duyệt nào, sau restart nào cũng đúng chế độ đã chọn.
  const { mode: authoringMode, setMode: setAuthoringMode } = useAuthoringMode(draft.projectId);
  // Chế độ AI chỉ "thật" khi máy chủ có provider: một draft chọn AI trên máy
  // chưa cấu hình key phải quay về đường copy tay, chứ không mất cả hai.
  const aiMode = authoringMode === "ai" && llm?.enabled === true;

  return (
    <div data-testid="manim-engineer-step-page">
      <AppShell currentStep={1} title="Bước 1 — Script" subtitle={`1c. Sinh code ${isRemotion ? "Remotion" : "Manim"} từ storyboard.`} wide>
        <ScriptPipelineTabs
          active="code"
          outlineDone={draft.authoringStory.trim().length > 0}
          storyboardDone={draft.authoringStoryboard.trim().length > 0}
          codeDone={!isEmpty}
        />

        <div className={styles.settingsRow} style={{ marginBottom: 16 }}>
          <RenderEnginePicker
            value={draft.renderEngine}
            onChange={(engine) => dispatch({ type: "SET_RENDER_ENGINE", payload: engine })}
          />
        </div>

        <div className={styles.settingsRow}>
          <AuthoringModeBar
            llm={llm}
            mode={authoringMode}
            onModeChange={setAuthoringMode}
            projectId={draft.projectId}
            step="code"
            what={`code ${isRemotion ? "Remotion" : "Manim"}`}
            runDisabled={draft.authoringStoryboard.trim().length === 0}
            runDisabledReason="Cần storyboard ở tab 1b trước — server đọc dàn ý + storyboard làm {{previous_output}}."
            onGenerated={(content) => setCode(content)}
          />
        </div>

        <div className={aiMode ? styles.scriptLayoutSingle : styles.scriptLayout}>
          {/* Ở chế độ AI, thẻ prompt không còn việc gì: server render đúng văn
              bản này rồi tự gọi. Đổi lại chế độ là nó quay lại nguyên vẹn. */}
          {!aiMode && (
            <Card
              title="1. Copy prompt"
              hint="Dán vào ChatGPT, Claude hoặc Gemini — đọc lại nội dung, đúng rồi thì copy."
            >
              <TextArea
                readOnly
                value={prompt}
                rows={18}
                className={styles.promptTextarea}
                data-testid="manim-engineer-prompt"
              />
              <Button onClick={handleCopy} className={styles.copyButton} data-testid="manim-engineer-copy">
                {copied ? "Đã copy!" : "Copy prompt"}
              </Button>
            </Card>
          )}

          <Card
            title={aiMode ? `Code ${isRemotion ? "Remotion" : "Manim"}` : "2. Dán kết quả"}
            hint={
              aiMode
                ? `Kết quả AI sinh ra hiện ở đây để bạn sửa — lint bên dưới vẫn chạy như khi dán tay. Bấm Tiếp tục để chuyển sang bước 1d (Duyệt).`
                : `Dán code ${isRemotion ? "Remotion" : "Manim"} AI trả về, rồi bấm Tiếp tục để chuyển sang bước 1d (Duyệt).`
            }
          >
            <TextArea
              id="manim-engineer-code-input"
              value={code}
              onChange={(event) => setCode(event.target.value)}
              rows={18}
              className={styles.promptTextarea}
              placeholder={
                isRemotion
                  ? "import {registerRoot, Composition} from 'remotion';\n..."
                  : "from conceptflow import *\n\nclass ...Scene(ConceptFlowScene):\n    def construct(self):\n        ..."
              }
              data-testid="manim-engineer-code-input"
            />
          </Card>
        </div>
      </AppShell>

      <WizardNav
        hint={hint}
        isBlocked={!!saveError || (!isEmpty && !validation.isValid)}
        onBack={() => navigate("/create/script/storyboard")}
        backLabel="Quay lại Storyboard"
        onNext={handleContinue}
        nextLabel={saving ? "Đang lưu..." : "Tiếp tục"}
        nextDisabled={!isValid || saving}
        nextTestId="manim-engineer-step-next"
      />
    </div>
  );
}
