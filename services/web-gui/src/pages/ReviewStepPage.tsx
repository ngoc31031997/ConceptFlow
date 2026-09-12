import { useContext, useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { WizardNav } from "../components/WizardNav";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { startRenderSaga, ApiError, listVoices } from "../api/client";
import { useRequireScript } from "../hooks/useRequireScript";
import { validateScript } from "../utils/scriptValidation";
import type { Voice } from "../types";
import glass from "../styles/glass.module.css";
import styles from "./WizardSteps.module.css";

const QUALITY_LABELS: Record<string, { label: string; hint: string }> = {
  "480p15": { label: "Test", hint: "480p15 — chỉ để kiểm nội dung, không nên dùng để đăng" },
  "720p30": { label: "Nháp", hint: "720p30 — không nên dùng để đăng" },
  "1080p60": { label: "Chuẩn", hint: "1080p60 — mức nên dùng khi đăng YouTube" },
  "4k60": { label: "Cao", hint: "4K60 — render rất nặng" },
};

const OUTPUT_MODE_LABELS: Record<string, string> = {
  long: "Chỉ video dài",
  short: "Chỉ video ngắn (Shorts/TikTok)",
  both: "Cả hai",
};

const SUBTITLE_SIZE_LABELS: Record<string, string> = { small: "Nhỏ", medium: "Vừa", large: "Lớn" };

const SUBTITLE_MODE_LABELS: Record<string, string> = {
  off: "Tắt",
  track: "Track CC (YouTube)",
  burn_in: "Ghi cứng vào hình",
  both: "Cả hai",
};

/** Edge and Azure share voice names, so the summary has to name the engine too. */
const ENGINE_NAMES: Record<string, string> = { edge: "Edge", azure: "Azure", google: "Google" };

/**
 * Step 3 of 5 (AppShell's STEP_LABELS) — everything that is about to be rendered, in one place.
 *
 * A render takes many minutes, so the last thing before committing to one is
 * a plain-language summary of what was chosen. Each row links back to the step
 * that owns it rather than making the Creator hunt for the control.
 */
export function ReviewStepPage() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();
  const hasScript = useRequireScript();
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [voices, setVoices] = useState<Voice[] | null>(null);

  useEffect(() => {
    let cancelled = false;
    listVoices()
      .then((result) => !cancelled && setVoices(Array.isArray(result) ? result : []))
      .catch(() => !cancelled && setVoices([]));
    return () => {
      cancelled = true;
    };
  }, []);

  if (!hasScript) return null;

  const validation = validateScript(draft.scriptContent, draft.voiceLanguage);
  const quality = QUALITY_LABELS[draft.renderQuality] ?? { label: draft.renderQuality, hint: "" };

  // With narration on but no voice resolved — the catalog failed to load, or
  // none matches the language — the render would go out silent. A bare "—"
  // here read as "nothing to see"; it is the one thing on this page worth
  // stopping for.
  const isLoadingVoices = voices === null;
  const selectedVoice = voices?.find((v) => v.voice_id === draft.voiceId) ?? null;
  const resolvedVoice = selectedVoice
    ? `${selectedVoice.label} (${ENGINE_NAMES[selectedVoice.engine] ?? selectedVoice.engine})`
    : null;
  const isVoiceMissing = draft.ttsEnabled && !isLoadingVoices && !resolvedVoice;
  const voiceValue = !draft.ttsEnabled
    ? "Tắt — video không có giọng đọc"
    : (resolvedVoice ?? (isLoadingVoices ? "Đang tải..." : "Chưa chọn được giọng đọc"));

  async function handleSubmit() {
    setError(null); // Clear previous errors
    setIsSubmitting(true);
    try {
      const projectId = draft.projectId;
      await startRenderSaga({
        project_id: projectId,
        script_content: draft.scriptContent,
        voice_language: draft.voiceLanguage,
        background_music_path: draft.backgroundMusicPath ?? undefined,
        tts_enabled: draft.ttsEnabled,
        voice_id: draft.ttsEnabled ? (draft.voiceId ?? undefined) : undefined,
        subtitle_mode: draft.subtitleMode,
        subtitle_style:
          draft.subtitleMode === "burn_in" || draft.subtitleMode === "both"
            ? {
                font_size: draft.subtitleStyle.fontSize,
                text_color: draft.subtitleStyle.textColor,
                background_opacity: draft.subtitleStyle.backgroundOpacity,
                position: draft.subtitleStyle.position,
              }
            : undefined,
        render_quality: draft.renderQuality,
        video_output_mode: draft.videoOutputMode,
        video_format_id: draft.videoFormatId,
        background_music_volume: draft.backgroundMusicPath ? draft.backgroundMusicVolume : undefined,
      });
      dispatch({ type: "MARK_SUBMITTED" });
      navigate(`/projects/${projectId}/render`);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setIsSubmitting(false);
    }
  }

  const rows: { label: string; value: string; hint?: string }[] = [
    {
      label: "Ngôn ngữ nội dung",
      value: draft.voiceLanguage === "vi" ? "Tiếng Việt" : "English",
      hint: "Giọng đọc, phụ đề và tiêu đề YouTube đều theo ngôn ngữ này",
    },
    {
      label: "Lời thoại",
      value: `${validation.narrationCount} đoạn`,
      hint: "Mỗi đoạn là một lần video dừng chờ giọng đọc",
    },
    {
      label: "Giọng đọc",
      value: voiceValue,
      hint: isVoiceMissing
        ? "Không tải được danh sách giọng đọc — quay lại bước 2 và chọn lại trước khi render"
        : undefined,
    },
    {
      label: "Phụ đề",
      value: SUBTITLE_MODE_LABELS[draft.subtitleMode],
      hint:
        draft.subtitleMode === "burn_in" || draft.subtitleMode === "both"
          ? `Cỡ ${SUBTITLE_SIZE_LABELS[draft.subtitleStyle.fontSize] ?? draft.subtitleStyle.fontSize}, ${draft.subtitleStyle.position === "bottom" ? "dưới" : "trên"} khung hình`
          : undefined,
    },
    { label: "Chất lượng", value: quality.label, hint: quality.hint },
    { label: "Loại video", value: OUTPUT_MODE_LABELS[draft.videoOutputMode] ?? draft.videoOutputMode },
    {
      label: "Nhạc nền",
      value: draft.backgroundMusicPath ? "Có" : "Không",
      hint: draft.backgroundMusicPath
        ? `Âm lượng ${Math.round(draft.backgroundMusicVolume * 100)}%, tự hạ khi có giọng đọc`
        : undefined,
    },
  ];

  return (
    <div data-testid="review-step-page">
      <AppShell
        wide
        currentStep={3}
        title="Bước 3 — Xem lại trước khi render"
        subtitle="Render mất vài phút và không dừng giữa chừng được. Kiểm tra nhanh những lựa chọn dưới đây."
      >
        <div className={styles.reviewLayout}>
          <div className={glass.card}>
            <div className={glass.cardTitle} style={{ marginBottom: 14 }}>
              Video sắp render
            </div>
            <div className={styles.summary} data-testid="review-summary">
              {rows.map((row) => (
                <div key={row.label} className={styles.summaryItem}>
                  <div className={styles.summaryLabel}>{row.label}</div>
                  <div className={styles.summaryValue}>{row.value}</div>
                  {row.hint && <div className={styles.summaryHint}>{row.hint}</div>}
                </div>
              ))}
            </div>
            <div className={styles.summaryActions}>
              <button type="button" className={glass.ghostBtn} onClick={() => navigate("/create/settings")}>
                Sửa cấu hình
              </button>
            </div>
          </div>

          {error && (
            <p role="alert" className={glass.helperText} style={{ marginRight: 0 }}>
              {error}
            </p>
          )}
        </div>
      </AppShell>

      <WizardNav
        hint={
          isVoiceMissing
            ? "Chưa chọn được giọng đọc — quay lại bước 2 hoặc tắt giọng đọc"
            : isSubmitting
              ? "Đang gửi yêu cầu render..."
              : "Sau khi bắt đầu, bạn sẽ theo dõi tiến trình ở bước tiếp theo."
        }
        isBlocked={isVoiceMissing}
        onBack={() => navigate("/create/settings")}
        onNext={handleSubmit}
        nextLabel={isSubmitting ? "Đang gửi..." : "Bắt đầu render"}
        nextDisabled={isSubmitting || isVoiceMissing}
        nextTestId="new-project-submit-button"
      />
    </div>
  );
}
