import { useContext, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AUTHORING_STEP_PATHS } from "../components/AuthoringModeBar";
import { AppShell } from "../components/AppShell";
import { IllustrationsGateNote } from "../components/IllustrationsGateNote";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { getAuthoringState, saveAuthoringCode, startRenderSaga, ApiError } from "../api/client";
import { validateRemotionScript, stripMarkdownCodeFence } from "../utils/scriptValidation";
import { useRenderedPrompt } from "../hooks/useRenderedPrompt";
import { authoringHint } from "../utils/authoringHint";
import { Card, Button, CopyBox, TextArea } from "../components/ui";
import { PipelineSettingsBar } from "../components/PipelineSettingsBar";
import { CodeChunkShotsField, CodeSegmentsPanel } from "../components/CodeSegmentsPanel";
import { useLlmStatus } from "../hooks/useLlmStatus";
import { useAuthoringMode } from "../hooks/useAuthoringMode";
import styles from "./WizardSteps.module.css";
import { FLOW_CODE, flowStepName } from "../utils/flow";


/**
 * Bước 6 — Code (Engineer), the last authoring step: the server fills the
 * template with the saved outline and storyboard; the Creator runs the AI or
 * copies the prompt out and pastes the code back. "Kiểm tra kịch bản" saves it
 * server-side, stores it as the draft's scriptContent and starts the validate
 * saga (step 7).
 */
export function CodeStepPage() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();
  const engineerRole = "remotion_engineer";
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
    video_output_mode: draft.videoOutputMode,
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
  // The engineer prompt asks the AI to wrap its answer in a ```tsx
  // fence — pasting that whole block (fence included) is the single most
  // common way this round trip fails: the fence markers are not valid
  // TSX, so esbuild chokes on line 1 with a syntax error
  // that says nothing about the real cause. Strip it before storing.
  const setCode = (value: string) => dispatch({ type: "SET_SCRIPT", payload: stripMarkdownCodeFence(value) });
  // validateRemotionScript checks the structural rules the
  // remotion_engineer prompt requires (narrations export, Composition
  // id="creator", calculateMetadata, <Segments>, balanced braces). It does not
  // catch everything the prompt's self-check asks for (e.g. overlapping
  // full-frame JSX), but it catches the recurring failure modes actually hit
  // in production before wasting a render cycle on them.
  const validation = validateRemotionScript(code);
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
      ? authoringHint({ aiMode, isEmpty, what: "code Remotion", ready: "" })
      : validation.isValid
        ? `Code hợp lệ — ${validation.narrationCount} đoạn lời thoại`
        : validation.message);
  const [segmentsVersion, setSegmentsVersion] = useState(0);

  return (
    <div data-testid="code-step-page">
      <AppShell
        currentStep={6}
        title={flowStepName(FLOW_CODE)}
        subtitle="Tạo code Remotion từ storyboard."
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
            llm={llm}
            mode={authoringMode}
            onModeChange={setAuthoringMode}
            projectId={draft.projectId}
            steps={["code"]}
            what="code Remotion"
            runDisabled={draft.authoringStoryboard.trim().length === 0}
            runDisabledReason="Cần hoàn thành bước Hình ảnh trước."
            onGenerated={(step, content) => {
              if (step === "code") setCode(content);
            }}
            onFollow={(step) => navigate(step === "done" ? AUTHORING_STEP_PATHS.code : AUTHORING_STEP_PATHS[step])}
          />
        </div>

        {/* The drawings have their own step; this only says where they stand. */}
        {draft.projectId && (
          <div className={styles.settingsRow}>
            <IllustrationsGateNote projectId={draft.projectId} onOpen={() => navigate(AUTHORING_STEP_PATHS.illustrations)} />
          </div>
        )}

        {showSegments && (
          <div className={styles.settingsRow}>
            <CodeSegmentsPanel projectId={draft.projectId} reloadKey={segmentsVersion} />
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
              <CopyBox text={prompt} what="prompt" disabled={rendered.prompt === null || rendered.stale} copyTestId="code-step-prompt-quick-copy">
                <TextArea
                  readOnly
                  value={prompt}
                  rows={18}
                  className={styles.promptTextarea}
                  data-testid="code-step-prompt"
                />
              </CopyBox>
              <Button onClick={handleCopy} disabled={rendered.prompt === null || rendered.stale} className={styles.copyButton} data-testid="code-step-copy">
                {copied ? "Đã sao chép" : "Sao chép prompt"}
              </Button>
            </Card>
          )}

          <Card
            title={aiMode ? "Code Remotion" : "2. Dán kết quả"}
            hint={
              aiMode
                ? `Kết quả của AI hiện ở đây để bạn chỉnh sửa. Hệ thống tự kiểm tra code bên dưới.`
                : "Dán code Remotion từ AI vào đây, rồi bấm Tiếp tục."
            }
          >
            <TextArea
              id="code-step-code-input"
              value={code}
              onChange={(event) => setCode(event.target.value)}
              rows={18}
              className={styles.promptTextarea}
              placeholder={"import {registerRoot, Composition} from 'remotion';\n..."}
              data-testid="code-step-code-input"
            />
          </Card>
        </div>
      </AppShell>

      <WizardNav
        hint={hint}
        isBlocked={!!saveError || (!isEmpty && !validation.isValid)}
        onBack={() => navigate(AUTHORING_STEP_PATHS.illustrations)}
        backLabel="Quay lại Hình minh hoạ"
        onNext={handleContinue}
        nextLabel={saving ? "Đang gửi..." : "Kiểm tra kịch bản"}
        nextDisabled={!isValid || saving}
        nextTestId="code-step-next"
      />
    </div>
  );
}
