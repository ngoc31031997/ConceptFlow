import { useEffect, useMemo, useRef, useState } from "react";
import { listVoices } from "../api/client";
import type { Voice } from "../types";
import glass from "../styles/glass.module.css";
import styles from "./NarrationPanel.module.css";
import selectable from "../styles/selectable.module.css";
import { SelectableOption } from "./SelectableOption";
import { SubtitleStyleFields } from "./SubtitleStylePanel";
import type { SubtitleMode, SubtitleStyle } from "../context/ProjectDraftContext";

interface NarrationPanelProps {
  /** Set on the content-language picker; this panel only reads it. */
  voiceLanguage: "vi" | "en";
  ttsEnabled: boolean;
  onTtsEnabledChange: (enabled: boolean) => void;
  voiceId: string | null;
  onVoiceIdChange: (voiceId: string | null) => void;
  subtitleMode: SubtitleMode;
  onSubtitleModeChange: (mode: SubtitleMode) => void;
  subtitleStyle: SubtitleStyle;
  onSubtitleStyleChange: (patch: Partial<SubtitleStyle>) => void;
}

/**
 * CR-015 FR41.1: replaces the old on/off toggle. Which delivery is right
 * depends on where the video will be watched (ADR-0027) — YouTube reads a
 * caption track, a short-form platform needs burned-in text — so the choice
 * is spelled out rather than collapsed back into a boolean.
 */
const SUBTITLE_MODE_OPTIONS: { value: SubtitleMode; label: string; hint: string }[] = [
  { value: "off", label: "Tắt", hint: "Không có phụ đề" },
  {
    value: "track",
    label: "Phụ đề YouTube (khuyên dùng)",
    hint: "Track CC riêng — người xem tự bật/tắt, YouTube lập chỉ mục và tự dịch được, không che hình",
  },
  {
    value: "burn_in",
    label: "Ghi cứng vào hình",
    hint: "Chữ vẽ thẳng lên khung hình — hợp khi đăng lại lên nền tảng không nhận track CC",
  },
  {
    value: "both",
    label: "Cả hai",
    hint: "Vừa có track CC vừa ghi cứng — người bật CC sẽ thấy chữ trùng hai lớp",
  },
];

const GENDER_LABEL: Record<string, string> = { female: "Nữ", male: "Nam" };

/*
  Azure and Edge serve the identical neural voices under identical names, so
  the row's label cannot distinguish them — the badge is the only thing that
  says where the audio will come from. The catalogue only lists engines this
  deployment has credentials for, so every badge here is a voice that really
  will be used, not one that would quietly fall back.
*/
const ENGINE_BADGE: Record<string, { text: string; title: string }> = {
  edge: { text: "Edge", title: "Microsoft Edge Read Aloud — miễn phí, không cần tài khoản" },
  azure: { text: "Azure", title: "Azure AI Speech — dùng key của bạn, có SLA và quyền thương mại" },
  google: { text: "Google", title: "Google Cloud WaveNet — dùng credential của bạn, tính phí" },
};

function Toggle({
  checked,
  onChange,
  label,
  hint,
  testId,
}: {
  checked: boolean;
  onChange: (next: boolean) => void;
  label: string;
  hint: string;
  testId: string;
}) {
  return (
    <div className={styles.toggleRow}>
      <div>
        <div className={styles.toggleLabel}>{label}</div>
        <div className={styles.toggleHint}>{hint}</div>
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        aria-label={label}
        data-testid={testId}
        className={styles.switch}
        onClick={() => onChange(!checked)}
      >
        <span className={styles.knob} />
      </button>
    </div>
  );
}

/**
 * Narration and subtitles, each toggle followed immediately by its own
 * settings.
 *
 * Both used to live elsewhere: the subtitle style was a sibling card that
 * appeared and disappeared from the sidebar — shifting everything below it,
 * including the submit button — and the content language sat between the two
 * toggles even though it drives the whole project (it is now its own card at
 * the top of the page).
 */
export function NarrationPanel({
  voiceLanguage,
  ttsEnabled,
  onTtsEnabledChange,
  voiceId,
  onVoiceIdChange,
  subtitleMode,
  onSubtitleModeChange,
  subtitleStyle,
  onSubtitleStyleChange,
}: NarrationPanelProps) {
  const [voices, setVoices] = useState<Voice[] | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const audioRef = useRef<HTMLAudioElement | null>(null);

  useEffect(() => {
    let cancelled = false;
    listVoices()
      .then((result) => {
        // The catalog crosses a service boundary; a malformed body must not
        // take the whole page down with it.
        if (!cancelled) setVoices(Array.isArray(result) ? result : []);
      })
      .catch(() => {
        if (!cancelled) setLoadError("Không tải được danh sách giọng đọc");
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const available = useMemo(
    () => (voices ?? []).filter((voice) => voice.language === voiceLanguage),
    [voices, voiceLanguage],
  );

  // Keep the chosen voice consistent with the chosen language: switching to
  // English while a Vietnamese voice is selected would otherwise submit a
  // voice the TTS Service cannot use for this script.
  useEffect(() => {
    if (!ttsEnabled || available.length === 0) return;
    if (!available.some((voice) => voice.voice_id === voiceId)) {
      onVoiceIdChange(available[0].voice_id);
    }
  }, [ttsEnabled, available, voiceId, onVoiceIdChange]);

  function playSample(voice: Voice) {
    if (audioRef.current) audioRef.current.pause();
    const audio = new Audio(voice.sample_audio_url);
    audioRef.current = audio;
    void audio.play().catch(() => setLoadError("Không phát được audio mẫu"));
  }

  useEffect(() => () => audioRef.current?.pause(), []);

  return (
    <div className={glass.card} style={{ padding: 22 }} data-testid="narration-panel">
      <div className={glass.cardTitle} style={{ marginBottom: 8 }}>
        Giọng đọc &amp; phụ đề
      </div>

      <Toggle
        label="Giọng đọc TTS"
        hint={ttsEnabled ? "Video sẽ có giọng đọc tự động" : "Video sẽ không có giọng đọc"}
        checked={ttsEnabled}
        onChange={onTtsEnabledChange}
        testId="narration-tts-toggle"
      />

      {ttsEnabled && (
        <div className={styles.nested}>
          <div className={styles.voiceList} data-testid="narration-voice-list">
            {available.map((voice) => {
              const isSelected = voice.voice_id === voiceId;
              return (
                <div
                  key={voice.voice_id}
                  className={`${selectable.option} ${isSelected ? selectable.selected : ""}`}
                  role="radio"
                  aria-checked={isSelected}
                  tabIndex={0}
                  onClick={() => onVoiceIdChange(voice.voice_id)}
                  onKeyDown={(event) => {
                    if (event.key === "Enter" || event.key === " ") {
                      event.preventDefault();
                      onVoiceIdChange(voice.voice_id);
                    }
                  }}
                >
                  <span className={selectable.check} aria-hidden="true">
                    {isSelected && (
                      <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="4" strokeLinecap="round" strokeLinejoin="round">
                        <path d="M5 13l4 4L19 7" />
                      </svg>
                    )}
                  </span>
                  <span className={selectable.body}>
                    <span className={styles.voiceLabelRow}>
                      <span className={selectable.label}>{voice.label}</span>
                      <span
                        className={styles.engineBadge}
                        data-engine={voice.engine}
                        title={ENGINE_BADGE[voice.engine]?.title}
                      >
                        {ENGINE_BADGE[voice.engine]?.text ?? voice.engine}
                      </span>
                    </span>
                    <span className={selectable.hint}>
                      {GENDER_LABEL[voice.gender] ?? voice.gender} · chất lượng {voice.quality}
                    </span>
                  </span>
                  <button
                    type="button"
                    className={styles.previewBtn}
                    aria-label={`Nghe thử ${voice.label}`}
                    onClick={(event) => {
                      event.stopPropagation();
                      playSample(voice);
                    }}
                  >
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
                      <path d="M8 5v14l11-7z" />
                    </svg>
                  </button>
                </div>
              );
            })}
          </div>

          {voices === null && !loadError && <p className={styles.status}>Đang tải giọng đọc...</p>}
          {loadError && (
            <p className={styles.status} role="alert">
              {loadError}
            </p>
          )}
          {voices !== null && available.length === 0 && (
            <p className={styles.status}>Chưa có giọng đọc nào cho ngôn ngữ này.</p>
          )}
        </div>
      )}

      <div className={styles.toggleLabel} style={{ marginTop: 18, marginBottom: 4 }}>
        Phụ đề
      </div>
      <div className={selectable.stack} role="radiogroup" aria-label="Phụ đề" data-testid="narration-subtitle-mode">
        {SUBTITLE_MODE_OPTIONS.map((option) => (
          <SelectableOption
            key={option.value}
            selected={subtitleMode === option.value}
            onSelect={() => onSubtitleModeChange(option.value)}
            label={option.label}
            hint={option.hint}
            testId={`narration-subtitle-mode-${option.value}`}
          />
        ))}
      </div>

      {subtitleMode === "both" && (
        // FR41.3: "both" is a valid choice (e.g. repost target without a
        // caption-track upload path) — flagged, not blocked.
        <p className={glass.helperText} style={{ marginRight: 0, marginTop: 10 }} role="status">
          Người xem bật CC sẽ thấy chữ phụ đề trùng lên chữ ghi cứng trong hình.
        </p>
      )}

      {(subtitleMode === "burn_in" || subtitleMode === "both") && (
        <div className={styles.nested} data-testid="subtitle-style-panel">
          <SubtitleStyleFields value={subtitleStyle} onChange={onSubtitleStyleChange} />
        </div>
      )}

      {!ttsEnabled && subtitleMode === "off" && (
        <p className={glass.helperText} style={{ marginRight: 0, marginTop: 10 }} role="status">
          Video sẽ không có lời thoại lẫn phụ đề — người xem chỉ thấy hình ảnh.
        </p>
      )}
    </div>
  );
}
