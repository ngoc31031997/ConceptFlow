import glass from "../styles/glass.module.css";
import styles from "./ContentLanguagePicker.module.css";

interface ContentLanguagePickerProps {
  value: "vi" | "en";
  onChange: (language: "vi" | "en") => void;
}

const LANGUAGES: { value: "vi" | "en"; label: string; flag: string }[] = [
  { value: "vi", label: "Tiếng Việt", flag: "🇻🇳" },
  { value: "en", label: "English", flag: "🇺🇸" },
];

/**
 * Content language is the project's first decision, not a narration setting:
 * it drives the starter script, the AI prompts, the voice, the subtitles, the
 * thumbnail prompt and the YouTube metadata (CR-008). It used to sit buried
 * halfway down the narration card, below two unrelated toggles, which read as
 * if it only picked a voice.
 */
export function ContentLanguagePicker({ value, onChange }: ContentLanguagePickerProps) {
  return (
    <div className={`${glass.card} ${styles.card}`} data-testid="content-language-picker">
      <div className={styles.text}>
        <div className={glass.cardTitle}>Ngôn ngữ nội dung</div>
        <p className={styles.hint}>
          Quyết định giọng đọc, phụ đề, script mẫu và tiêu đề/mô tả khi đăng YouTube. Giao diện vẫn giữ
          tiếng Việt.
        </p>
      </div>

      <div className={styles.options} role="radiogroup" aria-label="Ngôn ngữ nội dung">
        {LANGUAGES.map((option) => (
          <button
            key={option.value}
            type="button"
            role="radio"
            aria-checked={value === option.value}
            aria-pressed={value === option.value}
            className={`${styles.option} ${value === option.value ? styles.selected : ""}`}
            onClick={() => onChange(option.value)}
            data-testid={`content-language-${option.value}`}
          >
            <span aria-hidden="true">{option.flag}</span>
            {option.label}
          </button>
        ))}
      </div>
    </div>
  );
}
