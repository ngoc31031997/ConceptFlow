import { useContext, useRef, useState, type HTMLAttributes } from "react";
import { useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { StepSection } from "../components/StepSection";
import { TopicSummary } from "../components/TopicSummary";
import { NarrationPanel } from "../components/NarrationPanel";
import { VideoFontPicker } from "../components/VideoFontPicker";
import { BackgroundMusicPicker } from "../components/BackgroundMusicPicker";
import { RenderQualityPicker } from "../components/RenderQualityPicker";
import { PinnedLanguageCard } from "../components/PinnedLanguageCard";
import { WizardNav } from "../components/WizardNav";
import { Button } from "../components/ui";
import {
  ProjectDraftContext,
  ProjectDraftDispatchContext,
  saveLastUsedSettings,
} from "../context/ProjectDraftContext";
import { patchWizardSettings, type WizardSettingsPatch } from "../api/client";
import styles from "./WizardSteps.module.css";
import { FLOW_CONFIG, FLOW_VIEW, flowStepName } from "../utils/flow";

/** How long a drag of the music-volume slider rests before it is saved. */
const VOLUME_SAVE_DELAY_MS = 400;

/** The steps where the Creator approves something, named as the step rail names them. */
const REVIEW_GATES = FLOW_VIEW.filter((step) => step.gate);

/** React 18's typings have no `inert`; the attribute still reaches the DOM. */
const INERT = { inert: "" } as HTMLAttributes<HTMLDivElement>;

/** Props of the step 2 screen. */
interface ScriptAuthoringSettingsStepPageProps {
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
 * Bước 2 — cấu hình video này, năm mục làm từ trên xuống: giọng kể, chữ trên
 * video, nhạc nền, chất lượng render, ngôn ngữ hình (đã ghim lúc tạo project,
 * chỉ xem). Giọng chọn ở đây được bước Viết kịch bản dùng để nghe thử và ước
 * tính thời lượng; chữ, nhạc và chất lượng được bước Render và Merge đọc.
 * Ngôn ngữ nội dung và dạng video đã chốt ở bước Ý tưởng. Mọi mục đều có mặc
 * định nên đi thẳng qua được. Mỗi lần đổi một mục là một lần lưu riêng lên
 * project (đã tạo ở bước 1); "Lưu và sang nghiên cứu" chỉ chốt bước.
 */
export function ScriptAuthoringSettingsStepPage({
  readOnly = false,
  backLabel = "Về bước Ý tưởng",
  onBack,
}: ScriptAuthoringSettingsStepPageProps = {}) {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const navigate = useNavigate();

  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  // PATCH nối đuôi nhau để server nhận đúng thứ tự Creator đổi. Field nào lưu
  // lỗi được giữ ở `failedRef` và gửi lại cùng lần PATCH kế tiếp.
  const queueRef = useRef<Promise<void>>(Promise.resolve());
  const failedRef = useRef<WizardSettingsPatch>({});
  const volumeTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  function send(fields: WizardSettingsPatch): Promise<boolean> {
    const projectId = draft.projectId;
    if (!projectId) return Promise.resolve(true);
    const run = queueRef.current.then(async () => {
      const body = { ...failedRef.current, ...fields };
      try {
        await patchWizardSettings(projectId, body);
        failedRef.current = {};
        setSaveError(null);
        return true;
      } catch {
        failedRef.current = body;
        setSaveError("Không lưu được cấu hình — kiểm tra kết nối rồi thử lại.");
        return false;
      }
    });
    queueRef.current = run.then(() => undefined);
    return run;
  }

  function handleMusicVolumeChange(volume: number) {
    dispatch({ type: "SET_BACKGROUND_MUSIC_VOLUME", payload: volume });
    if (volumeTimerRef.current) clearTimeout(volumeTimerRef.current);
    volumeTimerRef.current = setTimeout(() => {
      volumeTimerRef.current = null;
      void send({ backgroundMusicVolume: volume });
    }, VOLUME_SAVE_DELAY_MS);
  }

  async function handleContinue() {
    setSaving(true);
    try {
      // Các field đã lưu từng cái một; bấm tiếp chỉ chốt bước (và gửi lại
      // field nào trước đó lưu lỗi) rồi mới đi tiếp.
      if (await send({ confirm: true })) {
        // Cấu hình này thành mặc định cho project kế tiếp.
        saveLastUsedSettings(draft);
        navigate(`/projects/${draft.projectId}/research`);
      }
    } finally {
      setSaving(false);
    }
  }

  return (
    <div data-testid="script-authoring-settings-step-page">
      <AppShell
        currentStep={2}
        preview={readOnly}
        wide
        title={flowStepName(FLOW_CONFIG)}
        subtitle="Giọng kể, chữ, nhạc và chất lượng. Ngôn ngữ hình đã ghim lúc tạo project. Các mục đã có sẵn giá trị phù hợp, bạn có thể bấm Lưu ngay."
      >
        {readOnly && (
          <p
            className={styles.readOnlyNotice}
            role="status"
            data-testid="settings-read-only-notice"
          >
            <b>Chưa tới bước này.</b> Đây là xem trước; bước này mở khi video đi tới đây.
          </p>
        )}
        <TopicSummary topic={draft.authoringTopic} onOpen={() => navigate("/")} />
        <div className={readOnly ? styles.readOnlyBody : undefined} {...(readOnly ? INERT : {})}>
          <StepSection
            title="1. Giọng kể"
            hint="Một người kể cả video. Bước Viết kịch bản dùng giọng này để nghe thử từng câu và ước tính thời lượng."
            testId="settings-section-voice"
          >
            <NarrationPanel
              voiceLanguage={draft.voiceLanguage}
              ttsEnabled={draft.ttsEnabled}
              onTtsEnabledChange={(enabled) => {
                dispatch({ type: "SET_TTS_ENABLED", payload: enabled });
                void send({ ttsEnabled: enabled });
              }}
              voiceId={draft.voiceId}
              onVoiceIdChange={(voiceId) => {
                dispatch({ type: "SET_VOICE_ID", payload: voiceId });
                void send({ voiceId: voiceId ?? "" });
              }}
            />
          </StepSection>

          <StepSection
            title="2. Chữ trên video"
            hint="Font của nhãn, con số và tiêu đề hiện trên hình."
            testId="settings-section-font"
          >
            <VideoFontPicker
              value={draft.videoFont}
              onChange={(font) => {
                dispatch({ type: "SET_VIDEO_FONT", payload: font });
                void send({ videoFont: font });
              }}
            />
          </StepSection>

          <StepSection
            title="3. Nhạc nền"
            hint="Tự lặp theo độ dài video, tự hạ nhỏ khi có giọng kể."
            testId="settings-section-music"
          >
            <BackgroundMusicPicker
              projectId={draft.projectId}
              value={draft.backgroundMusicPath}
              volume={draft.backgroundMusicVolume}
              onVolumeChange={handleMusicVolumeChange}
              onChange={(path) => {
                dispatch({ type: "SET_BACKGROUND_MUSIC", payload: path });
                void send({ backgroundMusicPath: path ?? "" });
              }}
            />
          </StepSection>

          <StepSection
            title="4. Chất lượng render"
            hint="Đổi được đến khi bước Render từng cảnh bắt đầu chạy."
            testId="settings-section-quality"
          >
            <RenderQualityPicker
              value={draft.renderQuality}
              onChange={(quality) => {
                dispatch({ type: "SET_RENDER_QUALITY", payload: quality });
                void send({ renderQuality: quality });
              }}
            />
          </StepSection>

          <StepSection
            title="5. Ngôn ngữ hình"
            hint="Cách vẽ video này, ghim lúc tạo project. Đổi ở Cài đặt chỉ áp cho video mới."
            testId="settings-section-language"
          >
            <PinnedLanguageCard projectId={draft.projectId} />
          </StepSection>

          <StepSection
            title="Các chỗ bạn sẽ duyệt"
            hint="Ở mọi bước, sửa được đúng một mục, một cảnh, một shot hay một câu; chỉ phần phụ thuộc vào nó phải làm lại."
            testId="settings-section-gates"
          >
            <ul className={styles.gateList} data-testid="settings-gates">
              {REVIEW_GATES.map((step) => (
                <li key={step.n} className={styles.gatePill}>
                  ◆ {step.n}. {step.title ?? step.label}
                  {step.optional ? " · tuỳ chọn" : ""}
                </li>
              ))}
            </ul>
          </StepSection>
        </div>
        {readOnly && (
          <Button variant="ghost" onClick={onBack} data-testid="settings-read-only-back">
            {backLabel}
          </Button>
        )}
      </AppShell>

      {!readOnly && (
        <WizardNav
          isBlocked={!!saveError}
          nextDisabled={saving}
          hint={
            saveError ??
            "Các mục đã lưu ngay khi bạn đổi. Bấm Lưu để sang bước Nghiên cứu (tuỳ chọn)."
          }
          onBack={() => navigate("/")}
          onNext={handleContinue}
          nextLabel={saving ? "Đang lưu..." : "Lưu và sang nghiên cứu"}
          nextTestId="script-authoring-settings-next"
        />
      )}
    </div>
  );
}
