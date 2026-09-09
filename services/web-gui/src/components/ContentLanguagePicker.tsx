import { SelectableOption } from "./SelectableOption";
import glass from "../styles/glass.module.css";
import selectable from "../styles/selectable.module.css";
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
 * it drives the AI prompt, the starter script, the voice, the subtitles, the
 * thumbnail prompt and the YouTube metadata (CR-008).
 */
export function ContentLanguagePicker({ value, onChange }: ContentLanguagePickerProps) {
  return (
    <div className={`${glass.card} ${styles.card}`} data-testid="content-language-picker">
      <div className={styles.text}>
        <div className={glass.cardTitle}>Ngôn ngữ nội dung</div>
        <p className={styles.hint}>
          Quyết định ngôn ngữ AI viết lời thoại, giọng đọc, phụ đề và tiêu đề/mô tả khi đăng YouTube.
          Giao diện vẫn giữ tiếng Việt.
        </p>
      </div>

      <div className={selectable.row} role="radiogroup" aria-label="Ngôn ngữ nội dung">
        {LANGUAGES.map((option) => (
          <SelectableOption
            key={option.value}
            selected={value === option.value}
            onSelect={() => onChange(option.value)}
            label={option.label}
            leading={<span aria-hidden="true">{option.flag}</span>}
            inline
            testId={`content-language-${option.value}`}
          />
        ))}
      </div>
    </div>
  );
}
