import glass from "../styles/glass.module.css";
import selectable from "../styles/selectable.module.css";
import styles from "./NarrationPanel.module.css";
import { SelectableOption } from "./SelectableOption";
import { SubtitleStyleFields } from "./SubtitleStylePanel";
import type { SubtitleMode, SubtitleStyle } from "../context/ProjectDraftContext";

interface SubtitleSettingsProps {
  mode: SubtitleMode;
  onModeChange: (mode: SubtitleMode) => void;
  style: SubtitleStyle;
  onStyleChange: (patch: Partial<SubtitleStyle>) => void;
  /** Whether the project has narration — only to warn about a silent, caption-less video. */
  ttsEnabled: boolean;
}

/**
 * Replaces the old on/off toggle. Which delivery is right
 * depends on where the video will be watched (ADR-0027) — YouTube reads a
 * caption track, a short-form platform needs burned-in text — so the choice
 * is spelled out rather than collapsed back into a boolean.
 */
const SUBTITLE_MODE_OPTIONS: { value: SubtitleMode; label: string; hint: string }[] = [
  { value: "off", label: "Tắt", hint: "Không có phụ đề" },
  {
    value: "track",
    label: "Phụ đề YouTube (khuyên dùng)",
    hint: "Người xem tự bật/tắt, YouTube hỗ trợ tìm kiếm và dịch tự động",
  },
  {
    value: "burn_in",
    label: "Ghi cứng vào hình",
    hint: "Chữ nằm cố định trên hình, phù hợp nền tảng không hỗ trợ phụ đề rời — không khuyến nghị cho video dài",
  },
  {
    value: "both",
    label: "Cả hai",
    hint: "Có cả phụ đề rời và phụ đề cố định trên hình",
  },
];

/**
 * Subtitle delivery and, when the text is burned in, its style. Only the merge
 * step (assemble_video) reads these, so they are chosen at the review gate and
 * can be changed again after a failure up to that step — not in step 2.
 */
export function SubtitleSettings({ mode, onModeChange, style, onStyleChange, ttsEnabled }: SubtitleSettingsProps) {
  return (
    <div data-testid="subtitle-settings">
      <div className={selectable.stack} role="radiogroup" aria-label="Phụ đề" data-testid="subtitle-mode">
        {SUBTITLE_MODE_OPTIONS.map((option) => (
          <SelectableOption
            key={option.value}
            selected={mode === option.value}
            onSelect={() => onModeChange(option.value)}
            label={option.label}
            hint={option.hint}
            testId={`subtitle-mode-${option.value}`}
          />
        ))}
      </div>

      {mode === "both" && (
        // "both" is a valid choice (e.g. repost target without a
        // caption-track upload path) — flagged, not blocked.
        <p className={glass.helperText} style={{ marginRight: 0, marginTop: 10 }} role="status">
          Người xem bật phụ đề sẽ thấy chữ bị trùng lặp.
        </p>
      )}

      {(mode === "burn_in" || mode === "both") && (
        <div className={styles.nested} data-testid="subtitle-style-panel">
          <SubtitleStyleFields value={style} onChange={onStyleChange} />
        </div>
      )}

      {!ttsEnabled && mode === "off" && (
        <p className={glass.helperText} style={{ marginRight: 0, marginTop: 10 }} role="status">
          Video sẽ không có lời thoại và phụ đề.
        </p>
      )}
    </div>
  );
}
