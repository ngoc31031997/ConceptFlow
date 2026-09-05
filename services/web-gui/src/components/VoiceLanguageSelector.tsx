import glass from "../styles/glass.module.css";
import styles from "./VoiceLanguageSelector.module.css";

interface VoiceLanguageSelectorProps {
  value: "vi" | "en";
  onChange: (lang: "vi" | "en") => void;
}

const OPTIONS: { value: "vi" | "en"; label: string }[] = [
  { value: "vi", label: "Tiếng Việt" },
  { value: "en", label: "English" },
];

export function VoiceLanguageSelector({ value, onChange }: VoiceLanguageSelectorProps) {
  return (
    <div className={glass.card} style={{ padding: 22 }}>
      <div className={glass.cardTitle} style={{ marginBottom: 14 }}>
        Ngôn ngữ giọng đọc
      </div>
      <div className={styles.switch} data-testid="new-project-voice-language-select">
        {OPTIONS.map((option) => (
          <button
            key={option.value}
            type="button"
            className={`${styles.option} ${value === option.value ? styles.active : ""}`}
            aria-pressed={value === option.value}
            onClick={() => onChange(option.value)}
          >
            {option.label}
          </button>
        ))}
      </div>
      <div className={glass.cardHint}>Piper TTS sẽ đọc nội dung script theo ngôn ngữ này</div>
    </div>
  );
}
