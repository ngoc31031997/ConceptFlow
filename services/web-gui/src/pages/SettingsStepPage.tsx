import { useContext } from "react";
import { useNavigate } from "react-router-dom";
import { AppShell } from "../components/AppShell";
import { NarrationPanel } from "../components/NarrationPanel";
import { RenderQualityPicker } from "../components/RenderQualityPicker";
import { VideoOutputModePicker } from "../components/VideoOutputModePicker";
import { VideoFormatPicker } from "../components/VideoFormatPicker";
import { useVideoFormats } from "../hooks/useVideoFormats";
import { BackgroundMusicPicker } from "../components/BackgroundMusicPicker";
import { WizardNav } from "../components/WizardNav";
import { Disclosure } from "../components/Disclosure";
import { ProjectDraftContext, ProjectDraftDispatchContext } from "../context/ProjectDraftContext";
import { useRequireScript } from "../hooks/useRequireScript";
import styles from "./WizardSteps.module.css";

const RENDER_QUALITY_LABELS: Record<string, string> = {
  "480p15": "Test (480p15)",
  "720p30": "Nháp (720p30)",
  "1080p60": "Chuẩn (1080p60)",
  "4k60": "Cao (4K60)",
};

/**
 * Step 2 of 5 (AppShell's STEP_LABELS) — how the video should sound and look. Every setting here has a
 * working default, so this step can be walked straight through.
 */
export function SettingsStepPage() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  const formats = useVideoFormats();
  const navigate = useNavigate();
  const hasScript = useRequireScript();

  if (!hasScript) return null;

  return (
    <div data-testid="settings-step-page">
      <AppShell
        wide
        currentStep={2}
        title="Bước 2 — Giọng đọc & hình ảnh"
        subtitle="Mọi mục ở đây đều đã có sẵn lựa chọn hợp lý. Bạn có thể bấm Tiếp tục ngay nếu không cần đổi gì."
      >
        <div className={styles.settingsLayout}>
          <NarrationPanel
            voiceLanguage={draft.voiceLanguage}
            ttsEnabled={draft.ttsEnabled}
            onTtsEnabledChange={(enabled) => dispatch({ type: "SET_TTS_ENABLED", payload: enabled })}
            voiceId={draft.voiceId}
            onVoiceIdChange={(voiceId) => dispatch({ type: "SET_VOICE_ID", payload: voiceId })}
            subtitleMode={draft.subtitleMode}
            onSubtitleModeChange={(mode) => dispatch({ type: "SET_SUBTITLE_MODE", payload: mode })}
            subtitleStyle={draft.subtitleStyle}
            onSubtitleStyleChange={(patch) => dispatch({ type: "SET_SUBTITLE_STYLE", payload: patch })}
          />

          <div className={styles.settingsRow}>
            <VideoFormatPicker
              formats={formats}
              value={draft.videoFormatId}
              onChange={(formatId) => dispatch({ type: "SET_VIDEO_FORMAT", payload: formatId })}
            />

            {/*
              UX review #7: RenderQualityPicker/BackgroundMusicPicker have no
              consequence for how the script gets written, unlike
              VideoFormatPicker (beats the script must follow — stays fully
              visible, per its own docstring) and VideoOutputModePicker
              (changes what step 5 looks like later — review kept this one
              expanded too). Collapsed by default with the current value
              shown, so the subtitle's "bấm Tiếp tục ngay cũng được" promise
              is visually true instead of contradicted by 4 full option-cards.
            */}
            <Disclosure
              title="Chất lượng video"
              hint={`Hiện tại: ${RENDER_QUALITY_LABELS[draft.renderQuality] ?? draft.renderQuality}`}
              testId="settings-render-quality"
            >
              <RenderQualityPicker
                value={draft.renderQuality}
                onChange={(quality) => dispatch({ type: "SET_RENDER_QUALITY", payload: quality })}
              />
            </Disclosure>

            <VideoOutputModePicker
              value={draft.videoOutputMode}
              onChange={(mode) => dispatch({ type: "SET_VIDEO_OUTPUT_MODE", payload: mode })}
            />

            <Disclosure
              title="Nhạc nền"
              hint={draft.backgroundMusicPath ? "Hiện tại: đã chọn nhạc nền" : "Hiện tại: không có"}
              testId="settings-background-music"
            >
              <BackgroundMusicPicker
                projectId={draft.projectId}
                value={draft.backgroundMusicPath}
                volume={draft.backgroundMusicVolume}
                onVolumeChange={(v) => dispatch({ type: "SET_BACKGROUND_MUSIC_VOLUME", payload: v })}
                onChange={(path) => dispatch({ type: "SET_BACKGROUND_MUSIC", payload: path })}
              />
            </Disclosure>
          </div>
        </div>
      </AppShell>

      <WizardNav
        hint="Không đổi gì cũng được — mặc định là 1080p60, có giọng đọc, không phụ đề."
        onBack={() => navigate("/")}
        backLabel="Quay lại script"
        onNext={() => navigate("/create/review")}
        nextLabel="Tiếp tục"
        nextTestId="settings-step-next"
      />
    </div>
  );
}
