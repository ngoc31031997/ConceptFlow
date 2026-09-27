import { useEffect, useMemo, useRef, useState } from "react";
import { listVoices } from "../api/client";
import type { Voice } from "../types";
import { Card } from "./ui";
import styles from "./NarrationPanel.module.css";
import selectable from "../styles/selectable.module.css";

interface NarrationPanelProps {
  /** Set on the content-language picker; this panel only reads it. */
  voiceLanguage: "vi" | "en";
  ttsEnabled: boolean;
  onTtsEnabledChange: (enabled: boolean) => void;
  voiceId: string | null;
  onVoiceIdChange: (voiceId: string | null) => void;
}

const GENDER_LABEL: Record<string, string> = { female: "Nữ", male: "Nam" };

/*
  Azure and Edge serve the identical neural voices under identical names, so
  the row's label cannot distinguish them — the badge is the only thing that
  says where the audio will come from. The catalogue only lists engines this
  deployment has credentials for, so every badge here is a voice that really
  will be used, not one that would quietly fall back.
*/
const ENGINE_BADGE: Record<string, { text: string; title: string }> = {
  edge: { text: "Edge", title: "Microsoft Edge · Miễn phí" },
  azure: { text: "Azure", title: "Azure AI Speech · Dùng tài khoản của bạn" },
  google: { text: "Google", title: "Google Cloud · Dùng tài khoản của bạn, có tính phí" },
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
 * Narration: the on/off toggle followed immediately by the voice list.
 *
 * Subtitles used to share this card; they moved to SubtitleSettings, shown
 * where the merge step that burns/attaches them runs (review gate and the
 * merge failure panel), so a subtitle problem can be fixed next to its retry.
 */
export function NarrationPanel({
  voiceLanguage,
  ttsEnabled,
  onTtsEnabledChange,
  voiceId,
  onVoiceIdChange,
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
        if (!cancelled) setLoadError("Không tải được danh sách giọng đọc. Vui lòng thử lại.");
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
    void audio.play().catch(() => setLoadError("Không phát được âm thanh mẫu"));
  }

  useEffect(() => () => audioRef.current?.pause(), []);

  return (
    <Card title="Giọng đọc" data-testid="narration-panel">
      <Toggle
        label="Giọng đọc"
        hint={ttsEnabled ? "Video sẽ có giọng đọc" : "Video sẽ không có giọng đọc"}
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
                      {GENDER_LABEL[voice.gender] ?? voice.gender} · {voice.quality}
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
    </Card>
  );
}
