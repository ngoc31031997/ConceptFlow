import { useEffect, useMemo, useRef, useState } from "react";
import { listVoices } from "../api/client";
import type { Voice } from "../types";
import glass from "../styles/glass.module.css";
import styles from "./NarrationPanel.module.css";

interface NarrationPanelProps {
  voiceLanguage: "vi" | "en";
  onVoiceLanguageChange: (lang: "vi" | "en") => void;
  ttsEnabled: boolean;
  onTtsEnabledChange: (enabled: boolean) => void;
  voiceId: string | null;
  onVoiceIdChange: (voiceId: string | null) => void;
  subtitlesEnabled: boolean;
  onSubtitlesEnabledChange: (enabled: boolean) => void;
}

const LANGUAGES: { value: "vi" | "en"; label: string }[] = [
  { value: "vi", label: "Tiếng Việt" },
  { value: "en", label: "English" },
];

const GENDER_LABEL: Record<string, string> = { female: "Nữ", male: "Nam" };

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

export function NarrationPanel({
  voiceLanguage,
  onVoiceLanguageChange,
  ttsEnabled,
  onTtsEnabledChange,
  voiceId,
  onVoiceIdChange,
  subtitlesEnabled,
  onSubtitlesEnabledChange,
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
      <Toggle
        label="Phụ đề"
        hint={subtitlesEnabled ? "Hiển thị lời thoại trên khung hình" : "Không hiển thị phụ đề"}
        checked={subtitlesEnabled}
        onChange={onSubtitlesEnabledChange}
        testId="narration-subtitles-toggle"
      />

      {!ttsEnabled && !subtitlesEnabled && (
        <p className={glass.helperText} style={{ marginRight: 0, marginTop: 10 }} role="status">
          Video sẽ không có lời thoại lẫn phụ đề — người xem chỉ thấy hình ảnh.
        </p>
      )}

      {/*
        The language picker sits OUTSIDE the TTS block (CR-008). It used to be
        nested inside it, so a Creator with narration turned off could not pick
        a language at all — and this is no longer just the voice's language: it
        decides the subtitles, the YouTube title/description/tags and the
        thumbnail prompt too.
      */}
      <div className={glass.cardHint} style={{ marginTop: 18, marginBottom: 8 }}>
        Ngôn ngữ nội dung
      </div>
      <div className={styles.toggleRow} style={{ padding: 0, borderTop: "none" }}>
        <div style={{ display: "flex", gap: 8, width: "100%" }}>
          {LANGUAGES.map((option) => (
            <button
              key={option.value}
              type="button"
              className={`${styles.voiceCard} ${voiceLanguage === option.value ? styles.selected : ""}`}
              aria-pressed={voiceLanguage === option.value}
              style={{ justifyContent: "center", flex: 1 }}
              onClick={() => onVoiceLanguageChange(option.value)}
              data-testid={`content-language-${option.value}`}
            >
              <span className={styles.voiceName}>{option.label}</span>
            </button>
          ))}
        </div>
      </div>
      <p className={glass.helperText} style={{ marginRight: 0, marginTop: 8 }}>
        Quyết định giọng đọc, phụ đề, và ngôn ngữ tiêu đề/mô tả/tag khi đăng YouTube. Giao diện vẫn giữ tiếng Việt.
      </p>

      {ttsEnabled && (
        <>
          <div className={styles.voiceList} data-testid="narration-voice-list">
            {available.map((voice) => (
              <div
                key={voice.voice_id}
                className={`${styles.voiceCard} ${voice.voice_id === voiceId ? styles.selected : ""}`}
                role="radio"
                aria-checked={voice.voice_id === voiceId}
                tabIndex={0}
                onClick={() => onVoiceIdChange(voice.voice_id)}
                onKeyDown={(event) => {
                  if (event.key === "Enter" || event.key === " ") {
                    event.preventDefault();
                    onVoiceIdChange(voice.voice_id);
                  }
                }}
              >
                <div className={styles.voiceInfo}>
                  <div className={styles.voiceName}>{voice.label}</div>
                  <div className={styles.voiceMeta}>
                    {GENDER_LABEL[voice.gender] ?? voice.gender} · chất lượng {voice.quality}
                  </div>
                </div>
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
            ))}
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
        </>
      )}
    </div>
  );
}
