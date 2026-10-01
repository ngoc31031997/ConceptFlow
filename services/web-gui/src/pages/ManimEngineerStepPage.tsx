import { useContext, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AUTHORING_STEP_PATHS } from "../components/AuthoringModeBar";
import { AppShell } from "../components/AppShell";
import { IllustrationsGateNote } from "../components/IllustrationsGateNote";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { getAuthoringState, saveAuthoringCode, createProjectDraft, startRenderSaga, ApiError } from "../api/client";
import { validateScript, validateRemotionScript, stripMarkdownCodeFence } from "../utils/scriptValidation";
import { useRenderedPrompt } from "../hooks/useRenderedPrompt";
import { authoringHint } from "../utils/authoringHint";
import { Card, Button, TextArea } from "../components/ui";
import { PipelineSettingsBar } from "../components/PipelineSettingsBar";
import { CodeChunkShotsField, CodeSegmentsPanel } from "../components/CodeSegmentsPanel";
import { useLlmStatus } from "../hooks/useLlmStatus";
import { useAuthoringMode } from "../hooks/useAuthoringMode";
import styles from "./WizardSteps.module.css";
import { FLOW_CODE, flowTitle } from "../utils/flow";


/**
 * Bước 6 — Code (Engineer), the last authoring step: the server fills the
 * template with the saved outline and storyboard; the Creator runs the AI or
 * copies the prompt out and pastes the code back. "Kiểm tra kịch bản" saves it
 * server-side, stores it as the draft's scriptContent and starts the validate
 * saga (step 7).
 *
 * The engine can still be changed here: this step's prompt role
 * (manim_engineer vs remotion_engineer) and its client-side lint
 * (validateScript vs validateRemotionScript) both depend on it.
 */
export function ManimEngineerStepPage() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();
  const isRemotion = draft.renderEngine === "remotion";
  const engineerRole = isRemotion ? "remotion_engineer" : "manim_engineer";
  const engineerLabel = isRemotion ? "Remotion Engineer" : "Manim Engineer";
  const [copied, setCopied] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  // Rehydrate the saved story/storyboard from the server on mount — mirrors
  // VisualDirectorStepPage's own rehydration effect.
  useEffect(() => {
    if (!draft.projectId || (draft.authoringStory && draft.authoringStoryboard && draft.authoringTopic)) return;
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
        // Chủ đề cũng phải nạp lại để AppShell hiện tên project sau reload.
        if (state.topic && !draft.authoringTopic) {
          dispatch({ type: "SET_AUTHORING_TOPIC", payload: state.topic });
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

  // The server fills the template from the draft — no prompt text
  // is assembled in the browser.
  const rendered = useRenderedPrompt({
    role: engineerRole,
    language: draft.voiceLanguage,
    topic: draft.authoringTopic,
    previous_output: previousOutput,
    subtitle_mode: draft.subtitleMode,
    subtitle_font_size: draft.subtitleStyle.fontSize,
    subtitle_position: draft.subtitleStyle.position,
  });
  const prompt = rendered.prompt ?? (rendered.failed ? `Không tải được template ${engineerRole}.` : "Đang tải...");

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(prompt);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopied(false);
    }
  }

  // Bound directly to draft.scriptContent (not a local buffer) so moving to
  // another step and back never loses code that has not been submitted yet.
  const code = draft.scriptContent;
  // The engineer prompt asks the AI to wrap its answer in a ```python/```tsx
  // fence — pasting that whole block (fence included) is the single most
  // common way this round trip fails: the fence markers are not valid
  // Python/TSX, so esbuild/ast.parse chokes on line 1 with a syntax error
  // that says nothing about the real cause. Strip it before storing.
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
      const projectId = draft.projectId;
      await startRenderSaga({
        project_id: projectId,
        script_content: code,
        voice_language: draft.voiceLanguage,
        background_music_path: draft.backgroundMusicPath ?? undefined,
        tts_enabled: draft.ttsEnabled,
        voice_id: draft.ttsEnabled ? (draft.voiceId ?? undefined) : undefined,
        subtitle_mode: draft.subtitleMode,
        subtitle_style:
          draft.subtitleMode === "burn_in" || draft.subtitleMode === "both"
            ? {
                font_family: draft.subtitleStyle.fontFamily,
                font_size: draft.subtitleStyle.fontSize,
                text_color: draft.subtitleStyle.textColor,
                background_opacity: draft.subtitleStyle.backgroundOpacity,
                position: draft.subtitleStyle.position,
              }
            : undefined,
        render_quality: draft.renderQuality,
        render_engine: draft.renderEngine,
        video_font: draft.videoFont,
        video_output_mode: draft.videoOutputMode,
        video_format_id: draft.videoFormatId,
        background_music_volume: draft.backgroundMusicPath ? draft.backgroundMusicVolume : undefined,
        // Bước Validate tồn tại để dừng ở cổng duyệt dàn ý.
        review_enabled: true,
      });
      dispatch({ type: "MARK_SUBMITTED" });
      navigate(`/projects/${projectId}/validate`);
    } catch (err) {
      setSaveError(err instanceof ApiError ? err.message : "Không lưu được code. Vui lòng thử lại.");
    } finally {
      setSaving(false);
    }
  }

  // Cùng lựa chọn chế độ với các bước soạn 3–5.
  const llm = useLlmStatus();
  // Chế độ lấy từ project ở server (qua draft), nên mở lại dự án
  // ở bất cứ tab nào, trình duyệt nào, sau restart nào cũng đúng chế độ đã chọn.
  const { mode: authoringMode, setMode: setAuthoringMode } = useAuthoringMode(draft.projectId);
  // Chế độ AI chỉ "thật" khi máy chủ có provider: một draft chọn AI trên máy
  // chưa cấu hình key phải quay về đường copy tay, chứ không mất cả hai.
  const aiMode = authoringMode === "ai" && llm?.enabled === true;
  // The segments panel and the shots-per-segment setting, AI mode only.
  const showSegments = aiMode && !!draft.projectId;

  const hint =
    saveError ??
    (isEmpty
      ? authoringHint({ aiMode, isEmpty, what: `code ${engineerLabel}`, ready: "" })
      : validation.isValid
        ? `Code hợp lệ — ${validation.narrationCount} đoạn lời thoại`
        : validation.message);
  const [segmentsVersion, setSegmentsVersion] = useState(0);

  return (
    <div data-testid="manim-engineer-step-page">
      <AppShell
        currentStep={6}
        title={flowTitle(FLOW_CODE)}
        subtitle={`Tạo code ${isRemotion ? "Remotion" : "Manim"} từ storyboard.`}
        wide
      >
        {/* Read by the next run, so it sits before the run button. */}
        {showSegments && (
          <div className={styles.settingsRow}>
            <CodeChunkShotsField projectId={draft.projectId} onSaved={() => setSegmentsVersion((v) => v + 1)} />
          </div>
        )}
        <div className={styles.settingsRow}>
          <PipelineSettingsBar
            renderEngine={draft.renderEngine}
            onEngineChange={(engine) => {
              dispatch({ type: "SET_RENDER_ENGINE", payload: engine });
              // Server phải biết engine trước khi render prompt cho
              // storyboard/code (RoleFor đọc project.RenderEngine), không chỉ
              // ở lúc nộp render. Best-effort như useAuthoringMode: lỗi mạng ở
              // đây không được chặn Creator đổi lựa chọn trên màn hình.
              if (draft.projectId) {
                void createProjectDraft(draft.projectId, "", draft.voiceLanguage, engine).catch(() => {});
              }
            }}
            llm={llm}
            mode={authoringMode}
            onModeChange={setAuthoringMode}
            projectId={draft.projectId}
            steps={["code"]}
            what={`code ${isRemotion ? "Remotion" : "Manim"}`}
            runDisabled={draft.authoringStoryboard.trim().length === 0}
            runDisabledReason="Cần hoàn thành bước Hình ảnh trước."
            onGenerated={(step, content) => {
              if (step === "code") setCode(content);
            }}
            onFollow={(step) => navigate(step === "done" ? AUTHORING_STEP_PATHS.code : AUTHORING_STEP_PATHS[step])}
          />
        </div>

        {/* The drawings have their own step; this only says where they stand. */}
        {isRemotion && draft.projectId && (
          <div className={styles.settingsRow}>
            <IllustrationsGateNote projectId={draft.projectId} onOpen={() => navigate(AUTHORING_STEP_PATHS.illustrations)} />
          </div>
        )}

        {showSegments && (
          <div className={styles.settingsRow}>
            <CodeSegmentsPanel projectId={draft.projectId} isRemotion={isRemotion} reloadKey={segmentsVersion} />
          </div>
        )}

        <div className={aiMode ? styles.scriptLayoutSingle : styles.scriptLayout}>
          {/* Ở chế độ AI, thẻ prompt không còn việc gì: server render đúng văn
              bản này rồi tự gọi. Đổi lại chế độ là nó quay lại nguyên vẹn. */}
          {!aiMode && (
            <Card
              title="1. Sao chép prompt"
              hint="Xem lại nội dung, sao chép rồi dán vào ChatGPT, Claude hoặc Gemini."
            >
              <TextArea
                readOnly
                value={prompt}
                rows={18}
                className={styles.promptTextarea}
                data-testid="manim-engineer-prompt"
              />
              <Button onClick={handleCopy} disabled={rendered.prompt === null || rendered.stale} className={styles.copyButton} data-testid="manim-engineer-copy">
                {copied ? "Đã sao chép" : "Sao chép prompt"}
              </Button>
            </Card>
          )}

          <Card
            title={aiMode ? `Code ${isRemotion ? "Remotion" : "Manim"}` : "2. Dán kết quả"}
            hint={
              aiMode
                ? `Kết quả của AI hiện ở đây để bạn chỉnh sửa. Hệ thống tự kiểm tra code bên dưới.`
                : `Dán code ${isRemotion ? "Remotion" : "Manim"} từ AI vào đây, rồi bấm Tiếp tục.`
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
        onBack={() => navigate(isRemotion ? AUTHORING_STEP_PATHS.illustrations : AUTHORING_STEP_PATHS.storyboard)}
        backLabel={isRemotion ? "Quay lại Hình minh hoạ" : "Quay lại Hình ảnh"}
        onNext={handleContinue}
        nextLabel={saving ? "Đang gửi..." : "Kiểm tra kịch bản"}
        nextDisabled={!isValid || saving}
        nextTestId="manim-engineer-step-next"
      />
    </div>
  );
}
