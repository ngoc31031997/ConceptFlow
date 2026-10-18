import { useContext, useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { AiModelSelect } from "../components/AiModelSelect";
import { CreativeBriefCard } from "../components/CreativeBriefCard";
import { SelectableOption } from "../components/SelectableOption";
import type { StepStripProps } from "../components/StepStrip";
import { Button, Card, Select, TextInput } from "../components/ui";
import {
  ProjectDraftContext,
  ProjectDraftDispatchContext,
  type VideoOutputMode,
} from "../context/ProjectDraftContext";
import { useProjectFlow } from "../context/ProjectFlowContext";
import {
  ApiError,
  DEFAULT_TARGET_DURATION,
  GENERIC_CONNECTION_ERROR,
  TARGET_DURATIONS,
  createProjectDraft,
  patchWizardSettings,
  saveAuthoringModels,
  setTargetDuration,
  type SimilarProject,
  type TargetDuration,
} from "../api/client";
import { useCreativeBrief } from "../hooks/useCreativeBrief";
import { useLlmStatus } from "../hooks/useLlmStatus";
import { useVideoFormats } from "../hooks/useVideoFormats";
import { DEFAULT_LONG_FORMAT_ID, SHORT_FORMAT_ID, isShortFormat } from "../utils/outputMode";
import wizard from "./WizardSteps.module.css";
import styles from "./ScriptStepPage.module.css";

const TITLE = "Ý tưởng · creative brief";
const SUBTITLE =
  "Chuyển ý tưởng ban đầu thành một đề tài đủ cụ thể để phát triển. Brief là điểm tựa cho nghiên cứu, câu chuyện và kịch bản phía sau.";

const OUTPUT_MODES: { value: VideoOutputMode; label: string; hint: string }[] = [
  { value: "long", label: "Video dài 16:9", hint: "3–10 phút, có chương" },
  { value: "short", label: "Shorts 9:16", hint: "dưới 60 giây" },
];
const LANGUAGES: { value: "vi" | "en"; label: string; hint?: string }[] = [
  { value: "vi", label: "Tiếng Việt", hint: "giọng, phụ đề, mô tả" },
  { value: "en", label: "English" },
];
const SHORT_DURATION_LABEL = "dưới 60 giây";

function durationLabel(d: TargetDuration): string {
  return `${d.replace("-", "–")} phút`;
}

/**
 * Bước 1 — Ý tưởng, trên xuống:
 *
 * ① Đề tài: chủ đề, dạng video, ngôn ngữ nội dung, thời lượng mục tiêu, rồi
 *   "Tạo project và viết brief". Tạo xong thì các lựa chọn khoá lại, vì brief
 *   viết theo đúng chúng; muốn đổi hướng thì làm lại brief kèm ghi chú. Thời
 *   lượng còn đổi được tới khi có brief.
 * ② Creative brief (CreativeBriefCard): AI viết từ đề tài, Creator sửa tay
 *   hoặc nhờ làm lại, rồi duyệt. Không có AI thì Creator tự viết.
 *
 * "Sang cấu hình" chỉ mở khi brief đã duyệt: các bước sau dựng từ brief.
 */
export function ScriptStepPage() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();
  const location = useLocation();
  const flow = useProjectFlow();
  const llm = useLlmStatus();
  const formats = useVideoFormats();
  const aiAvailable = llm?.enabled === true;

  const [createdFor, setCreatedFor] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);
  const [similarProjects, setSimilarProjects] = useState<SimilarProject[]>([]);
  const [duration, setDuration] = useState<TargetDuration>(DEFAULT_TARGET_DURATION);
  const [durationError, setDurationError] = useState<string | null>(null);

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
  const brief = useCreativeBrief(draft.projectId, created);
  const unit = brief.state?.unit ?? null;
  const isShort = draft.videoOutputMode === "short";
  const topic = draft.authoringTopic;
  const choicesLocked = created || creating || checking;
  const canCreate = topic.trim().length > 0 && !choicesLocked;

  // A short is built vertically from the start: the short format and captions
  // burned in at the top. The server applies the same defaults when the
  // project is created as a short; the draft follows so later screens show them.
  function handleOutputModeChange(mode: VideoOutputMode) {
    dispatch({ type: "SET_VIDEO_OUTPUT_MODE", payload: mode });
    if (mode === "short") {
      dispatch({ type: "SET_VIDEO_FORMAT", payload: SHORT_FORMAT_ID });
      dispatch({ type: "SET_SUBTITLE_MODE", payload: "burn_in" });
      dispatch({ type: "SET_SUBTITLE_STYLE", payload: { position: "top" } });
      return;
    }
    const current = formats.find((f) => f.id === draft.videoFormatId);
    if (current && isShortFormat(current)) dispatch({ type: "SET_VIDEO_FORMAT", payload: DEFAULT_LONG_FORMAT_ID });
  }

  async function handleCreate() {
    const trimmed = topic.trim();
    if (!trimmed) return;
    const projectId = draft.projectId;
    setCreateError(null);
    setCreating(true);
    try {
      const { similarProjects: similar } = await createProjectDraft(projectId, trimmed, draft.voiceLanguage);
      // The brief runs on the model picked for the story step: the last one
      // the Creator used, until step 2 changes it.
      await saveAuthoringModels(projectId, draft.authoringModels);
      if (isShort) {
        await patchWizardSettings(projectId, { videoOutputMode: "short" });
      } else {
        await setTargetDuration(projectId, duration);
      }
      setSimilarProjects(similar);
      setCreatedFor(projectId);
    } catch {
      setCreateError("Không tạo được dự án. Vui lòng kiểm tra kết nối và thử lại.");
      return;
    } finally {
      setCreating(false);
    }
    if (aiAvailable) void brief.generate();
  }

  async function handleDurationChange(next: TargetDuration) {
    if (!created) {
      setDuration(next);
      return;
    }
    setDurationError(null);
    try {
      await setTargetDuration(draft.projectId, next);
      brief.reload();
    } catch (err) {
      setDurationError(err instanceof ApiError && err.message ? err.message : GENERIC_CONNECTION_ERROR);
    }
  }

  // Before a brief exists the target length can still change (an older
  // project, or a first AI run that failed); after, the brief was written for it.
  const durationLocked = created ? !!unit || brief.running || brief.state === null : creating || checking;
  const shownDuration = created ? (brief.state?.targetDuration ?? DEFAULT_TARGET_DURATION) : duration;

  function strip(): StepStripProps {
    if (!created) {
      return createError
        ? { tone: "todo", label: "Chưa tạo project", message: createError }
        : { tone: "todo", label: "Chưa tạo project", message: "Nhập đề tài rồi bấm Tạo project và viết brief." };
    }
    if (brief.running) return { tone: "run", label: "AI đang viết brief", message: "Ô bạn đã sửa giữ nguyên." };
    if (!unit) return { tone: "todo", label: "Chưa có brief", message: "AI viết brief từ đề tài, hoặc bạn tự viết; duyệt rồi sang Cấu hình." };
    if (unit.status === "approved") return { tone: "ok", label: "Brief đã duyệt", message: "Tiếp theo: Cấu hình." };
    return {
      tone: "wait",
      label: "Chờ bạn duyệt brief",
      message: "AI đã viết brief từ đề tài. Sửa ô nào cũng được; ô bạn viết thì AI giữ nguyên khi làm lại.",
    };
  }

  return (
    <div data-testid="script-step-page">
      <AppShell currentStep={1} wide title={TITLE} subtitle={SUBTITLE} strip={strip()}>
        <Card className={styles.card} data-testid="idea-topic-card">
          <h3 className={styles.cardTitle}>① Đề tài</h3>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="idea-topic">
              1. Chủ đề hoặc câu hỏi ban đầu
            </label>
            <TextInput
              id="idea-topic"
              value={topic}
              onChange={(e) => dispatch({ type: "SET_AUTHORING_TOPIC", payload: e.target.value })}
              placeholder="Ý tưởng/chủ đề video của bạn là gì?"
              disabled={choicesLocked}
              data-testid="script-step-topic"
            />
          </div>

          <div className={styles.field}>
            <span className={styles.label} id="idea-output-mode">
              2. Dạng video
            </span>
            <div className={styles.options} role="radiogroup" aria-labelledby="idea-output-mode">
              {OUTPUT_MODES.map((o) => (
                <SelectableOption
                  key={o.value}
                  selected={draft.videoOutputMode === o.value}
                  onSelect={() => handleOutputModeChange(o.value)}
                  label={o.label}
                  hint={o.hint}
                  disabled={choicesLocked}
                  testId={`video-output-mode-${o.value}`}
                />
              ))}
            </div>
          </div>

          <div className={styles.grid2}>
            <div className={styles.field}>
              <span className={styles.label} id="idea-language">
                3. Ngôn ngữ nội dung
              </span>
              <div className={styles.options} role="radiogroup" aria-labelledby="idea-language">
                {LANGUAGES.map((o) => (
                  <SelectableOption
                    key={o.value}
                    selected={draft.voiceLanguage === o.value}
                    onSelect={() => dispatch({ type: "SET_VOICE_LANGUAGE", payload: o.value })}
                    label={o.label}
                    hint={o.hint}
                    disabled={choicesLocked}
                    testId={`content-language-${o.value}`}
                  />
                ))}
              </div>
            </div>
            <div className={styles.field}>
              <label className={styles.label} htmlFor="idea-duration">
                4. Thời lượng mục tiêu
              </label>
              {isShort ? (
                <Select id="idea-duration" value="short" disabled data-testid="idea-duration">
                  <option value="short">{SHORT_DURATION_LABEL}</option>
                </Select>
              ) : (
                <Select
                  id="idea-duration"
                  value={shownDuration}
                  onChange={(e) => void handleDurationChange(e.target.value as TargetDuration)}
                  disabled={durationLocked}
                  data-testid="idea-duration"
                >
                  {TARGET_DURATIONS.map((d) => (
                    <option key={d} value={d}>
                      {durationLabel(d)}
                    </option>
                  ))}
                </Select>
              )}
              <span className={styles.hint}>Beat sheet và kịch bản bám theo khung này.</span>
            </div>
          </div>

          {durationError && (
            <p className={styles.error} role="alert">
              {durationError}
            </p>
          )}
          {!created && aiAvailable && <AiModelSelect projectId={draft.projectId} step="brief" label="Model AI viết brief" testId="idea-brief-model" />}
          <div className={styles.actions}>
            {created ? (
              <p className={styles.note} data-testid="idea-created-note">
                ✓ Đã tạo project. Đổi đề tài thì dùng “Làm lại kèm ghi chú” ở brief.
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
                  {creating ? "Đang tạo…" : aiAvailable ? "Tạo project và viết brief →" : "Tạo project →"}
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

        <CreativeBriefCard projectId={draft.projectId} created={created} brief={brief} aiAvailable={aiAvailable} />

        <div className={styles.actions}>
          <Button
            onClick={() => navigate("/create/script/settings")}
            disabled={unit?.status !== "approved"}
            data-testid="script-step-next"
          >
            Sang cấu hình →
          </Button>
        </div>
      </AppShell>
    </div>
  );
}
