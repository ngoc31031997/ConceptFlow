import { useContext, useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { WizardNav } from "../components/WizardNav";
import { AiModelSelect } from "../components/AiModelSelect";
import { IdeaCard } from "../components/IdeaCard";
import type { StepStripProps } from "../components/StepStrip";
import { Button, Card, TextInput } from "../components/ui";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { useProjectFlow } from "../context/ProjectFlowContext";
import {
  DEFAULT_TARGET_DURATION,
  createProjectDraft,
  patchWizardSettings,
  saveAuthoringModels,
  setTargetDuration,
  type SimilarProject,
} from "../api/client";
import { useIdea } from "../hooks/useIdea";
import { useLlmStatus } from "../hooks/useLlmStatus";
import { FLOW_INIT, flowStepName } from "../utils/flow";
import wizard from "./WizardSteps.module.css";
import styles from "./ScriptStepPage.module.css";

const SUBTITLE =
  "Viết ý tưởng ban đầu thành một câu hỏi rõ: vì sao đáng xem, điều đã biết và chưa biết. Chưa cần lời giải và chưa chọn cách kể.";

/**
 * Bước 1 — Ý tưởng ban đầu, trên xuống:
 *
 * ① Đề tài: chủ đề hoặc câu hỏi ban đầu, rồi "Tạo project và viết ý tưởng".
 *   Dạng video, ngôn ngữ và thời lượng chọn ở bước Creative brief; project
 *   được tạo với giá trị đang chọn trong bản nháp.
 * ② Ý tưởng (IdeaCard): AI viết từ đề tài, Creator sửa tay hoặc nhờ làm lại,
 *   rồi duyệt. Không có AI thì Creator tự viết.
 *
 * "Sang Creative brief" chỉ mở khi ý tưởng đã duyệt: brief được viết từ ý tưởng.
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
  const [createError, setCreateError] = useState<string | null>(null);
  const [similarProjects, setSimilarProjects] = useState<SimilarProject[]>([]);

  useEffect(() => {
    if (draft.hasSubmitted) dispatch({ type: "RESET" });
  }, [draft.hasSubmitted, dispatch]);

  // "Tạo video mới" → projectId mới. Xoá cờ khỏi history để reload/Quay lại
  // không reset thêm lần nữa và làm mất chủ đề vừa gõ.
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
  const idea = useIdea(draft.projectId, created);
  const unit = idea.state?.unit ?? null;
  const topic = draft.authoringTopic;
  const topicLocked = created || creating || checking;
  const canCreate = topic.trim().length > 0 && !topicLocked;

  async function handleCreate() {
    const trimmed = topic.trim();
    if (!trimmed) return;
    const projectId = draft.projectId;
    setCreateError(null);
    setCreating(true);
    try {
      const { similarProjects: similar } = await createProjectDraft(projectId, trimmed, draft.voiceLanguage);
      // The idea and the brief run on the model picked for the story step: the
      // last one the Creator used, until the AI settings change it.
      await saveAuthoringModels(projectId, draft.authoringModels);
      if (draft.videoOutputMode === "short") {
        await patchWizardSettings(projectId, { videoOutputMode: "short" });
      } else {
        await setTargetDuration(projectId, DEFAULT_TARGET_DURATION);
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
    if (aiAvailable) void idea.generate();
  }

  function strip(): StepStripProps {
    if (!created) {
      return createError
        ? { tone: "todo", label: "Chưa tạo project", message: createError }
        : { tone: "todo", label: "Chưa tạo project", message: "Nhập đề tài rồi bấm Tạo project và viết ý tưởng." };
    }
    if (idea.running) return { tone: "run", label: "AI đang viết ý tưởng", message: "Ô bạn đã sửa giữ nguyên." };
    if (!unit) return { tone: "todo", label: "Chưa có ý tưởng", message: "AI viết ý tưởng từ đề tài, hoặc bạn tự viết; duyệt rồi sang Creative brief." };
    if (unit.status === "approved") return { tone: "ok", label: "Ý tưởng đã duyệt", message: "Tiếp theo: Creative brief." };
    return {
      tone: "wait",
      label: "Chờ bạn duyệt ý tưởng",
      message: "AI đã viết ý tưởng từ đề tài. Sửa ô nào cũng được; ô bạn viết thì AI giữ nguyên khi làm lại.",
    };
  }

  return (
    <div data-testid="script-step-page">
      <AppShell currentStep={FLOW_INIT} wide title={flowStepName(FLOW_INIT)} subtitle={SUBTITLE} strip={strip()}>
        <Card className={styles.card} data-testid="idea-topic-card">
          <h3 className={styles.cardTitle}>① Đề tài</h3>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="idea-topic">
              Chủ đề hoặc câu hỏi ban đầu <span className={styles.hint}>· tên tạm</span>
            </label>
            <TextInput
              id="idea-topic"
              value={topic}
              onChange={(e) => dispatch({ type: "SET_AUTHORING_TOPIC", payload: e.target.value })}
              placeholder="Ý tưởng/chủ đề video của bạn là gì?"
              disabled={topicLocked}
              data-testid="script-step-topic"
            />
          </div>

          {!created && aiAvailable && <AiModelSelect projectId={draft.projectId} step="brief" label="Model AI viết ý tưởng và brief" testId="idea-brief-model" />}
          <div className={styles.actions}>
            {created ? (
              <p className={styles.note} data-testid="idea-created-note">
                ✓ Đã tạo project. Đổi đề tài thì dùng “Làm lại kèm ghi chú” ở ý tưởng.
              </p>
            ) : (
              <>
                {createError && (
                  <p className={styles.error} role="alert">
                    {createError}
                  </p>
                )}
                <Button
                  variant={aiAvailable ? "run" : "primary"}
                  onClick={() => void handleCreate()}
                  disabled={!canCreate}
                  data-testid="script-step-create"
                >
                  {creating ? "Đang tạo…" : aiAvailable ? "Tạo project và viết ý tưởng →" : "Tạo project →"}
                </Button>
              </>
            )}
          </div>
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

        <IdeaCard projectId={draft.projectId} created={created} idea={idea} aiAvailable={aiAvailable} />
      </AppShell>

      <WizardNav
        allowWhenLocked
        hint={
          unit?.status === "approved"
            ? "Ý tưởng đã duyệt. Sang bước Creative brief."
            : "Duyệt ý tưởng để sang bước Creative brief."
        }
        onNext={() => navigate("/create/script/brief")}
        nextDisabled={unit?.status !== "approved"}
        nextLabel="Lưu và sang Creative brief"
        nextTestId="script-step-next"
      />
    </div>
  );
}
