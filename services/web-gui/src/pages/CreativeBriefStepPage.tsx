import { useContext, useState, type HTMLAttributes } from "react";
import { useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { CreativeBriefCard } from "../components/CreativeBriefCard";
import { SelectableOption } from "../components/SelectableOption";
import { TopicSummary } from "../components/TopicSummary";
import type { StepStripProps } from "../components/StepStrip";
import { WizardNav } from "../components/WizardNav";
import { Button, Card, Select } from "../components/ui";
import {
  ProjectDraftContext,
  ProjectDraftDispatchContext,
  saveLastUsedSettings,
  type VideoOutputMode,
} from "../context/ProjectDraftContext";
import {
  ApiError,
  DEFAULT_TARGET_DURATION,
  GENERIC_CONNECTION_ERROR,
  PLATFORMS,
  TARGET_DURATIONS,
  patchWizardSettings,
  setPlatform,
  setTargetDuration,
  type TargetDuration,
} from "../api/client";
import { useCreativeBrief } from "../hooks/useCreativeBrief";
import { useIdea } from "../hooks/useIdea";
import { useLlmStatus } from "../hooks/useLlmStatus";
import { useVideoFormats } from "../hooks/useVideoFormats";
import { DEFAULT_LONG_FORMAT_ID, SHORT_FORMAT_ID, isShortFormat } from "../utils/outputMode";
import { FLOW_CONFIG, flowStepName } from "../utils/flow";
import wizard from "./WizardSteps.module.css";
import styles from "./ScriptStepPage.module.css";

const SUBTITLE =
  "Thống nhất video làm cho ai, nhằm đạt mục tiêu gì và có giới hạn nào. Brief là điểm tựa cho nghiên cứu, câu chuyện và kịch bản phía sau.";

const OUTPUT_MODES: { value: VideoOutputMode; label: string; hint: string }[] = [
  { value: "long", label: "Video dài 16:9", hint: "3–10 phút, có chương" },
  { value: "short", label: "Shorts 9:16", hint: "dưới 60 giây" },
];
const LANGUAGES: { value: "vi" | "en"; label: string; hint?: string }[] = [
  { value: "vi", label: "Tiếng Việt", hint: "giọng, phụ đề, mô tả" },
  { value: "en", label: "English" },
];
const SHORT_DURATION_LABEL = "dưới 60 giây";

/** React 18's typings have no `inert`; the attribute still reaches the DOM. */
const INERT = { inert: "" } as HTMLAttributes<HTMLDivElement>;

function durationLabel(d: TargetDuration): string {
  return `${d.replace("-", "–")} phút`;
}

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
 * ① Hình thức: dạng video, ngôn ngữ nội dung, thời lượng dự kiến, nền tảng
 *   phát hành. Mỗi lần đổi là một lần lưu lên project. Brief đã viết giữ
 *   nguyên cho tới khi Creator nhờ AI viết lại theo lựa chọn mới.
 * ② Định hướng sáng tạo (CreativeBriefCard): AI viết từ ý tưởng đã duyệt,
 *   Creator sửa tay hoặc nhờ làm lại, rồi duyệt. Không có AI thì Creator tự viết.
 *
 * "Lưu và sang nghiên cứu" chỉ mở khi brief đã duyệt: các bước sau dựng từ brief.
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
  const idea = useIdea(draft.projectId, created);
  const brief = useCreativeBrief(draft.projectId, created);
  const unit = brief.state?.unit ?? null;
  const ideaApproved = idea.state?.unit?.status === "approved";
  const isShort = draft.videoOutputMode === "short";

  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  const choicesLocked = readOnly || !created || brief.running || brief.state === null;
  const shownDuration = brief.state?.targetDuration ?? DEFAULT_TARGET_DURATION;
  const shownPlatform = brief.state?.platform ?? PLATFORMS[0];

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
    void save(() => patchWizardSettings(draft.projectId, { videoOutputMode: mode }));
  }

  function handleLanguageChange(language: "vi" | "en") {
    dispatch({ type: "SET_VOICE_LANGUAGE", payload: language });
    void save(() => patchWizardSettings(draft.projectId, { voiceLanguage: language }));
  }

  function handlePlatformChange(next: string) {
    void save(async () => {
      await setPlatform(draft.projectId, next);
      brief.reload();
    });
  }

  function handleDurationChange(next: TargetDuration) {
    void save(async () => {
      await setTargetDuration(draft.projectId, next);
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
    if (brief.running) return { tone: "run", label: "AI đang viết brief", message: "Ô bạn đã sửa giữ nguyên." };
    if (!unit && !ideaApproved) {
      return { tone: "todo", label: "Chưa có brief", message: "Duyệt ý tưởng ở bước 1 trước: brief được viết từ ý tưởng đã duyệt." };
    }
    if (!unit) return { tone: "todo", label: "Chưa có brief", message: "AI viết brief từ ý tưởng, hoặc bạn tự viết; duyệt rồi sang Nghiên cứu." };
    if (unit.status === "approved") return { tone: "ok", label: "Brief đã duyệt", message: "Tiếp theo: Nghiên cứu (tuỳ chọn)." };
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
            <h3 className={styles.cardTitle}>① Hình thức</h3>

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

            <div className={styles.grid3}>
              <div className={styles.field}>
                <span className={styles.label} id="brief-language">
                  Ngôn ngữ nội dung
                </span>
                <div className={styles.options} role="radiogroup" aria-labelledby="brief-language">
                  {LANGUAGES.map((o) => (
                    <SelectableOption
                      key={o.value}
                      selected={draft.voiceLanguage === o.value}
                      onSelect={() => handleLanguageChange(o.value)}
                      label={o.label}
                      hint={o.hint}
                      disabled={choicesLocked}
                      testId={`content-language-${o.value}`}
                    />
                  ))}
                </div>
              </div>
              <div className={styles.field}>
                <label className={styles.label} htmlFor="brief-duration">
                  Thời lượng dự kiến
                </label>
                {isShort ? (
                  <Select id="brief-duration" value="short" disabled data-testid="brief-duration">
                    <option value="short">{SHORT_DURATION_LABEL}</option>
                  </Select>
                ) : (
                  <Select
                    id="brief-duration"
                    value={shownDuration}
                    onChange={(e) => handleDurationChange(e.target.value as TargetDuration)}
                    disabled={choicesLocked}
                    data-testid="brief-duration"
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
              <div className={styles.field}>
                <label className={styles.label} htmlFor="brief-platform">
                  Nền tảng phát hành
                </label>
                <Select
                  id="brief-platform"
                  value={shownPlatform}
                  onChange={(e) => handlePlatformChange(e.target.value)}
                  disabled={choicesLocked}
                  data-testid="brief-platform"
                >
                  {PLATFORMS.map((p) => (
                    <option key={p} value={p}>
                      {p}
                    </option>
                  ))}
                </Select>
              </div>
            </div>
            {unit && (
              <p className={styles.hint} data-testid="brief-form-note">
                Brief đã viết theo lựa chọn lúc trước. Đổi ở đây xong, bấm “Làm lại kèm ghi chú” ở brief để AI viết theo lựa chọn mới.
              </p>
            )}
            {saveError && (
              <p className={styles.error} role="alert" data-testid="brief-save-error">
                {saveError}
              </p>
            )}
          </Card>

          <CreativeBriefCard
            projectId={draft.projectId}
            created={created}
            brief={brief}
            aiAvailable={aiAvailable}
            ideaApproved={ideaApproved}
          />
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
          nextDisabled={saving || unit?.status !== "approved"}
          hint={
            saveError ??
            (unit?.status === "approved"
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
