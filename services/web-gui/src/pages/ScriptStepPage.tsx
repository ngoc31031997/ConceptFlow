import { useContext, useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { WizardNav } from "../components/WizardNav";
import { AiModelSelect } from "../components/AiModelSelect";
import { DeepSeekRunButton } from "../components/DeepSeekRunButton";
import { ExternalAiDialog } from "../components/ExternalAiDialog";
import { ContentLanguagePicker } from "../components/ContentLanguagePicker";
import { ProjectIdeaCard } from "../components/idea/ProjectIdeaCard";
import type { StepStripProps } from "../components/StepStrip";
import { Button, Card, TextArea } from "../components/ui";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { useProjectFlow } from "../context/ProjectFlowContext";
import {
  createProjectDraft,
  generateProjectIdea,
  patchWizardSettings,
  saveAuthoringModels,
  updateProjectTopic,
  type SimilarProject,
} from "../api/client";
import { useDeepSeekRun } from "../hooks/useDeepSeekRun";
import { useProjectIdea } from "../hooks/useProjectIdea";
import { useLlmStatus } from "../hooks/useLlmStatus";
import { FLOW_INIT, flowStepName } from "../utils/flow";
import { sanitizeIdeaText } from "../utils/sanitizeIdeaText";
import wizard from "./WizardSteps.module.css";
import styles from "./ScriptStepPage.module.css";

const SUBTITLE =
  "Gõ ý tưởng của bạn bằng lời thường. AI biến nó thành câu hỏi trung tâm, các điểm gây tò mò và phạm vi; bạn sửa rồi duyệt. Chưa cần lời giải và chưa chọn cách kể.";

/**
 * Bước 1 — Ý tưởng, trên xuống:
 *
 * ① Ý tưởng của bạn: một ô lớn để gõ ý tưởng, ngôn ngữ nội dung, rồi "Tạo
 *   project và viết ý tưởng". Dạng video, thời lượng và nền tảng không chọn ở
 *   đây; project nhận giá trị mặc định.
 * ② Ý tưởng (ProjectIdeaCard): AI viết ba trường từ nội dung đã gõ, đánh giá,
 *   Creator sửa tay hoặc nhờ làm lại, rồi duyệt. Không có AI thì Creator tự viết.
 *
 * "Sang bước tiếp theo" chỉ mở khi ý tưởng đã duyệt.
 */
export function ScriptStepPage() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();
  const location = useLocation();
  const flow = useProjectFlow();
  const llm = useLlmStatus();
  const aiAvailable = llm?.enabled === true;

  const [createdFor, setCreatedFor] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  // The project is made first, then the idea is written through an AI outside the system.
  const [outsideOpen, setOutsideOpen] = useState(false);
  const [savingText, setSavingText] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const [similarProjects, setSimilarProjects] = useState<SimilarProject[]>([]);

  useEffect(() => {
    if (draft.hasSubmitted) dispatch({ type: "RESET" });
  }, [draft.hasSubmitted, dispatch]);

  // "Tạo video mới" → projectId mới. Xoá cờ khỏi history để reload/Quay lại
  // không reset thêm lần nữa và làm mất nội dung vừa gõ.
  useEffect(() => {
    const st = location.state as { newVideo?: boolean } | null;
    if (!st?.newVideo) return;
    dispatch({ type: "RESET" });
    navigate(location.pathname, { replace: true, state: null });
  }, [location.key]); // eslint-disable-line react-hooks/exhaustive-deps

  // The project exists when this screen made it, or when the server has it
  // (a reload, or a project reopened from the list). Until the server has
  // answered, the form stays shut rather than offer to create it twice.
  const created = createdFor === draft.projectId || flow.project?.project_id === draft.projectId;
  const checking = !created && !flow.loaded;
  const idea = useProjectIdea(draft.projectId, created);
  // DeepSeek's answer the step could not read, shown in the outside-AI dialog for the Creator to fix.
  const [unreadable, setUnreadable] = useState<{ reply: string; message: string } | null>(null);
  const deepSeek = useDeepSeekRun({
    run: (external) => generateProjectIdea(draft.projectId, "", { external }),
    onDone: idea.reload,
    onUnreadable: (reply, message) => {
      setUnreadable({ reply, message });
      setOutsideOpen(true);
    },
  });
  const stored = idea.screen?.idea ?? null;
  // A draft reopened from storage may hold text typed before the cleaning existed.
  const text = sanitizeIdeaText(draft.authoringTopic);
  const textLocked = creating || checking || savingText;
  const canCreate = text.trim().length > 0 && !created && !textLocked;
  // The text the server keeps; once it differs from the box, "Lưu nội dung" is offered.
  const savedText = idea.screen?.subject ?? "";
  const textEdited = created && idea.screen !== null && text.trim() !== savedText.trim() && text.trim().length > 0;

  async function handleCreate(via: "system" | "outside" | "deepseek") {
    const trimmed = text.trim();
    if (!trimmed) return;
    const projectId = draft.projectId;
    setCreateError(null);
    setCreating(true);
    try {
      const { similarProjects: similar } = await createProjectDraft(projectId, trimmed, draft.voiceLanguage);
      // The idea and the brief run on the model picked for the story step: the
      // last one the Creator used, until the AI settings change it.
      await saveAuthoringModels(projectId, draft.authoringModels);
      // The video form is picked on the brief step; a short is the one choice
      // made before it.
      if (draft.videoOutputMode === "short") {
        await patchWizardSettings(projectId, { videoOutputMode: "short" });
      }
      setSimilarProjects(similar);
      setCreatedFor(projectId);
      flow.refetch();
    } catch {
      setCreateError("Không tạo được dự án. Vui lòng kiểm tra kết nối và thử lại.");
      return;
    } finally {
      setCreating(false);
    }
    if (via === "outside") {
      setUnreadable(null);
      setOutsideOpen(true);
    } else if (via === "deepseek") await deepSeek.start();
    else if (aiAvailable) void idea.generate();
  }

  async function handleSaveText() {
    setCreateError(null);
    setSavingText(true);
    try {
      const { similarProjects: similar } = await updateProjectTopic(draft.projectId, text.trim());
      setSimilarProjects(similar);
      idea.reload();
    } catch {
      setCreateError("Không lưu được nội dung. Vui lòng kiểm tra kết nối và thử lại.");
    } finally {
      setSavingText(false);
    }
  }

  function strip(): StepStripProps {
    if (!created) {
      return createError
        ? { tone: "todo", label: "Chưa tạo project", message: createError }
        : { tone: "todo", label: "Chưa tạo project", message: "Gõ ý tưởng rồi bấm Tạo project và viết ý tưởng." };
    }
    if (idea.writing) return { tone: "run", label: "AI đang viết ý tưởng", message: "Ô bạn đã sửa giữ nguyên." };
    if (!stored) return { tone: "todo", label: "Chưa có ý tưởng", message: "AI viết ý tưởng từ nội dung bạn gõ, hoặc bạn tự viết; duyệt rồi sang bước tiếp theo." };
    if (stored.status === "approved") return { tone: "ok", label: "Ý tưởng đã duyệt", message: "Tiếp theo: Creative brief." };
    return {
      tone: "wait",
      label: "Chờ bạn duyệt ý tưởng",
      message: "Xem đánh giá của AI rồi duyệt. Sửa ô nào cũng được; ô bạn viết thì AI giữ nguyên khi làm lại.",
    };
  }

  return (
    <div data-testid="script-step-page">
      <AppShell currentStep={FLOW_INIT} wide title={flowStepName(FLOW_INIT)} subtitle={SUBTITLE} strip={strip()}>
        <Card className={styles.card} data-testid="idea-topic-card">
          <h3 className={styles.cardTitle}>① Ý tưởng của bạn</h3>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="idea-input">
              Bạn muốn làm video về điều gì? <span className={styles.hint}>· viết tự nhiên, vài chữ hay vài câu đều được</span>
            </label>
            <TextArea
              id="idea-input"
              value={text}
              onChange={(e) => dispatch({ type: "SET_AUTHORING_TOPIC", payload: sanitizeIdeaText(e.target.value) })}
              placeholder="Ví dụ: Chuyện gì xảy ra nếu ta rơi vào hố đen? Ta có bị kéo dài như sợi mì không? Người bên ngoài có thấy ta biến mất không?"
              rows={8}
              disabled={textLocked}
              data-testid="script-step-topic"
            />
          </div>

          {!created && <ContentLanguagePicker value={draft.voiceLanguage} onChange={(language) => dispatch({ type: "SET_VOICE_LANGUAGE", payload: language })} />}
          {created && (
            <p className={styles.note} data-testid="idea-language-note">
              Ngôn ngữ nội dung: {(idea.screen?.language || draft.voiceLanguage) === "en" ? "English" : "Tiếng Việt"}. Ý tưởng được viết bằng ngôn ngữ này.
            </p>
          )}

          {!created && aiAvailable && <AiModelSelect projectId={draft.projectId} step="brief" label="Model AI viết ý tưởng và brief" testId="idea-brief-model" />}
          <div className={styles.actions}>
            {created ? (
              <>
                <p className={styles.note} data-testid="idea-created-note">
                  ✓ Đã tạo project. Sửa nội dung thì bấm “Lưu nội dung”, rồi dùng “Làm lại kèm ghi chú” ở ý tưởng nếu muốn AI viết lại.
                </p>
                {textEdited && (
                  <Button variant="primary" onClick={() => void handleSaveText()} disabled={savingText} data-testid="script-step-save-text">
                    {savingText ? "Đang lưu…" : "Lưu nội dung"}
                  </Button>
                )}
              </>
            ) : (
              <>
                {createError && (
                  <p className={styles.error} role="alert">
                    {createError}
                  </p>
                )}
                <Button variant="ghost" onClick={() => void handleCreate("outside")} disabled={!canCreate} data-testid="script-step-create-external">
                  Tạo project và viết bằng AI ngoài
                </Button>
                <DeepSeekRunButton
                  run={deepSeek}
                  disabled={!canCreate || creating}
                  onStart={() => void handleCreate("deepseek")}
                  label="Tạo project và chạy bằng DeepSeek"
                  testId="script-step-create-deepseek"
                />
                <Button
                  variant={aiAvailable ? "run" : "primary"}
                  onClick={() => void handleCreate("system")}
                  disabled={!canCreate}
                  data-testid="script-step-create"
                >
                  {creating ? "Đang tạo…" : aiAvailable ? "Tạo project và viết ý tưởng →" : "Tạo project →"}
                </Button>
              </>
            )}
          </div>
          {created && createError && (
            <p className={styles.error} role="alert">
              {createError}
            </p>
          )}
          {similarProjects.length > 0 && (
            <div className={wizard.topicCollisionBanner} data-testid="topic-collision-banner">
              Chủ đề này giống {similarProjects.length} dự án khác:{" "}
              {similarProjects.map((p, i) => (
                <span key={p.projectId}>
                  {i > 0 && ", "}
                  <a href="/videos" target="_blank" rel="noreferrer">
                    {p.topic || p.projectId} ({p.status})
                  </a>
                </span>
              ))}
              . Bạn vẫn có thể tiếp tục.
            </div>
          )}
        </Card>

        <ProjectIdeaCard projectId={draft.projectId} created={created} idea={idea} aiAvailable={aiAvailable} />
      </AppShell>

      <ExternalAiDialog
        open={outsideOpen}
        onClose={() => setOutsideOpen(false)}
        title="Ý tưởng · AI ngoài"
        run={(external) => generateProjectIdea(draft.projectId, "", { external })}
        onDone={idea.reload}
        initialReply={unreadable?.reply}
        initialError={unreadable?.message}
        testId="idea-create-external-dialog"
      />

      <WizardNav
        allowWhenLocked
        hint={
          stored?.status === "approved"
            ? "Ý tưởng đã duyệt. Sang bước Creative brief."
            : "Duyệt ý tưởng để sang bước Creative brief."
        }
        onNext={() => navigate("/create/script/brief")}
        nextDisabled={stored?.status !== "approved"}
        nextLabel="Lưu và sang Creative brief"
        nextTestId="script-step-next"
      />
    </div>
  );
}
