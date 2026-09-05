import { useState } from "react";
import glass from "../styles/glass.module.css";
import styles from "./BackgroundMusicPicker.module.css";

interface BackgroundMusicPickerProps {
  value: string | null;
  onChange: (path: string | null) => void;
}

function MusicIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M9 18V5l12-2v13" />
      <circle cx="6" cy="18" r="3" />
      <circle cx="18" cy="16" r="3" />
    </svg>
  );
}

export function BackgroundMusicPicker({ value, onChange }: BackgroundMusicPickerProps) {
  const [enabled, setEnabled] = useState(value !== null);

  function handleToggle() {
    const next = !enabled;
    setEnabled(next);
    if (!next) onChange(null);
  }

  return (
    <div className={glass.card}>
      <div className={styles.row}>
        <div className={styles.label}>
          <div className={styles.icon}>
            <MusicIcon />
          </div>
          Thêm nhạc nền (tùy chọn)
        </div>
        <button
          type="button"
          role="switch"
          aria-checked={enabled}
          className={`${styles.toggle} ${enabled ? "" : styles.off}`}
          onClick={handleToggle}
        >
          <span className={styles.knob} />
        </button>
      </div>
      {enabled && (
        <input
          type="text"
          data-testid="new-project-music-input"
          className={`${glass.textInput} ${styles.input}`}
          value={value ?? ""}
          onChange={(event) => onChange(event.target.value)}
          placeholder="Đường dẫn file nhạc nền, ví dụ: /music/ambient.mp3"
        />
      )}
    </div>
  );
}
