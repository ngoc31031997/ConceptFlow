import { useContext, useState, type HTMLAttributes } from "react";
import { useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { ProjectBriefCard } from "../components/brief/ProjectBriefCard";
import { SelectableOption } from "../components/SelectableOption";
import { TopicSummary } from "../components/TopicSummary";
import type { StepStripProps } from "../components/StepStrip";
import { WizardNav } from "../components/WizardNav";
import { Button, Card } from "../components/ui";
import {
  ProjectDraftContext,
  ProjectDraftDispatchContext,
  saveLastUsedSettings,
  type VideoOutputMode,
} from "../context/ProjectDraftContext";
import { ApiError, GENERIC_CONNECTION_ERROR, patchWizardSettings } from "../api/client";
import { useProjectBrief } from "../hooks/useProjectBrief";
import { useLlmStatus } from "../hooks/useLlmStatus";
import { useVideoFormats } from "../hooks/useVideoFormats";
import { DEFAULT_LONG_FORMAT_ID, SHORT_FORMAT_ID, isShortFormat } from "../utils/outputMode";
import { FLOW_CONFIG, flowStepName } from "../utils/flow";
import wizard from "./WizardSteps.module.css";
import styles from "./ScriptStepPage.module.css";

const SUBTITLE =
  "Thống nhất video làm cho ai, dài bao lâu, nói với giọng nào, người xem hiểu được gì và điều gì phải tránh. AI viết từ ý tưởng đã duyệt ở bước 1.";

const OUTPUT_MODES: { value: VideoOutputMode; label: string; hint: string }[] = [
  { value: "long", label: "Video dài 16:9", hint: "từ 2 đến 30 phút, có chương" },
  { value: "short", label: "Shorts 9:16", hint: "1–2 phút" },
];

/** React 18's typings have no `inert`; the attribute still reaches the DOM. */
const INERT = { inert: "" } as HTMLAttributes<HTMLDivElement>;

/** Props of the step 2 screen. */
interface CreativeBriefStepPageProps {
  /**
   * Show the screen as a preview of a step the video has not reached: every
   * control is inert and the way forward is replaced by `onBack`.
   */
  readOnly?: boolean;
  /** Label of the button that leaves a read-only screen. */
  backLabel?: string;
  /** Leaves a read-only screen, back to the step the video is at. */
  onBack?: () => void;
}

/**
 * Bước 2 — Creative brief (định hướng sáng tạo), trên xuống:
 *
 * ① Dạng video: dài hay ngắn. Mỗi lần đổi là một lần lưu lên project và quyết
 *   định khoảng thời lượng brief được đặt. Ngôn ngữ nội dung đã chốt ở bước 1.
 * ② Định hướng sáng tạo (ProjectBriefCard): AI viết khán giả, thời lượng, giọng
 *   điệu, mục tiêu học tập và điều cần tránh từ ý tưởng; Creator sửa tay hoặc
 *   nhờ làm lại, xem đánh giá của AI, rồi duyệt. Không có AI thì Creator tự viết.
 *
 * "Lưu và sang nghiên cứu" chỉ mở khi brief đã duyệt.
 */
export function CreativeBriefStepPage({
  readOnly = false,
  backLabel = "Về bước Ý tưởng",
  onBack,
}: CreativeBriefStepPageProps = {}) {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();
  const llm = useLlmStatus();
  const formats = useVideoFormats();
  const aiAvailable = llm?.enabled === true;

  // This step opens only after the idea step made the project.
  const created = !readOnly && !!draft.projectId;
  const brief = useProjectBrief(draft.projectId, created);
  const stored = brief.screen?.brief ?? null;
  const approvedIdea = brief.screen?.idea ?? null;

  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  const choicesLocked = readOnly || !created || brief.writing || brief.screen === null;

  async function save(action: () => Promise<void>) {
    setSaveError(null);
    try {
      await action();
    } catch (err) {
      setSaveError(err instanceof ApiError && err.message ? err.message : GENERIC_CONNECTION_ERROR);
    }
  }

  // A short is built vertically from the start: the short format and captions
  // burned in at the top. The server applies the same defaults when the
  // project is patched to a short; the draft follows so later screens show them.
  function handleOutputModeChange(mode: VideoOutputMode) {
    dispatch({ type: "SET_VIDEO_OUTPUT_MODE", payload: mode });
    if (mode === "short") {
      dispatch({ type: "SET_VIDEO_FORMAT", payload: SHORT_FORMAT_ID });
      dispatch({ type: "SET_SUBTITLE_MODE", payload: "burn_in" });
      dispatch({ type: "SET_SUBTITLE_STYLE", payload: { position: "top" } });
    } else {
      const current = formats.find((f) => f.id === draft.videoFormatId);
      if (current && isShortFormat(current)) dispatch({ type: "SET_VIDEO_FORMAT", payload: DEFAULT_LONG_FORMAT_ID });
    }
    void save(async () => {
      await patchWizardSettings(draft.projectId, { videoOutputMode: mode });
      // The range of lengths a brief may set follows the video form.
      brief.reload();
    });
  }

  async function handleContinue() {
    setSaving(true);
    setSaveError(null);
    try {
      // The choices were saved one by one; this only confirms the step.
      await patchWizardSettings(draft.projectId, { confirm: true });
      // These choices become the defaults of the next project.
      saveLastUsedSettings(draft);
      navigate(`/projects/${draft.projectId}/research`);
    } catch (err) {
      setSaveError(err instanceof ApiError && err.message ? err.message : GENERIC_CONNECTION_ERROR);
    } finally {
      setSaving(false);
    }
  }

  function strip(): StepStripProps | undefined {
    if (readOnly) return undefined;
    if (brief.writing) return { tone: "run", label: "AI đang viết brief", message: "Ô bạn đã sửa giữ nguyên." };
    if (!stored && !approvedIdea) {
      return { tone: "todo", label: "Chưa có brief", message: "Duyệt ý tưởng ở bước 1 trước: brief được viết từ ý tưởng đã duyệt." };
    }
    if (!stored) return { tone: "todo", label: "Chưa có brief", message: "AI viết brief từ ý tưởng, hoặc bạn tự viết; duyệt rồi sang Nghiên cứu." };
    if (stored.status === "approved") return { tone: "ok", label: "Brief đã duyệt", message: "Tiếp theo: Nghiên cứu (tuỳ chọn)." };
    return {
      tone: "wait",
      label: "Chờ bạn duyệt brief",
      message: "AI đã viết brief từ ý tưởng. Sửa ô nào cũng được; ô bạn viết thì AI giữ nguyên khi làm lại.",
    };
  }

  return (
    <div data-testid="creative-brief-step-page">
      <AppShell
        currentStep={FLOW_CONFIG}
        preview={readOnly}
        wide
        title={flowStepName(FLOW_CONFIG)}
        subtitle={SUBTITLE}
        strip={strip()}
      >
        {readOnly && (
          <p className={wizard.readOnlyNotice} role="status" data-testid="brief-read-only-notice">
            <b>Chưa tới bước này.</b> Đây là xem trước; bước này mở khi video đi tới đây.
          </p>
        )}
        <TopicSummary topic={draft.authoringTopic} onOpen={() => navigate("/")} />
        <div className={readOnly ? wizard.readOnlyBody : undefined} {...(readOnly ? INERT : {})}>
          <Card className={styles.card} data-testid="brief-form-card">
            <h3 className={styles.cardTitle}>① Dạng video</h3>

            <div className={styles.field}>
              <span className={styles.label} id="brief-output-mode">
                Dạng video
              </span>
              <div className={styles.options} role="radiogroup" aria-labelledby="brief-output-mode">
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
            {stored && (
              <p className={styles.hint} data-testid="brief-form-note">
                Brief đã viết theo lựa chọn lúc trước. Đổi dạng video ở đây xong, bấm “Làm lại kèm ghi chú” ở brief để AI viết theo lựa chọn mới.
              </p>
            )}
            {saveError && (
              <p className={styles.error} role="alert" data-testid="brief-save-error">
                {saveError}
              </p>
            )}
          </Card>

          <ProjectBriefCard projectId={draft.projectId} created={created} brief={brief} aiAvailable={aiAvailable} />
        </div>
        {readOnly && (
          <Button variant="ghost" onClick={onBack} data-testid="brief-read-only-back">
            {backLabel}
          </Button>
        )}
      </AppShell>

      {!readOnly && (
        <WizardNav
          isBlocked={!!saveError}
          nextDisabled={saving || stored?.status !== "approved"}
          hint={
            saveError ??
            (stored?.status === "approved"
              ? "Brief đã duyệt. Sang bước Nghiên cứu (tuỳ chọn)."
              : "Duyệt brief để sang bước Nghiên cứu.")
          }
          onBack={() => navigate("/")}
          onNext={handleContinue}
          nextLabel={saving ? "Đang lưu..." : "Lưu và sang nghiên cứu"}
          nextTestId="creative-brief-next"
        />
      )}
    </div>
  );
}
