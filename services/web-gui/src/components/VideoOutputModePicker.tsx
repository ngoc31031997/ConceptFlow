import type { VideoOutputMode } from "../context/ProjectDraftContext";
import { SelectableOption } from "./SelectableOption";
import glass from "../styles/glass.module.css";
import selectable from "../styles/selectable.module.css";

interface VideoOutputModePickerProps {
  value: VideoOutputMode;
  onChange: (mode: VideoOutputMode) => void;
  /**
   * Skip the outer `glass.card`/title (UX review #2 nesting fix) — used when
   * this already renders inside a `Disclosure` (ResultPage's rerender
   * section), which provides its own card. SettingsStepPage renders this one
   * standalone (kept fully expanded, not behind a Disclosure — review #7),
   * so it still needs its own card there.
   */
  bare?: boolean;
}

/**
 * CR-007 follow-up. A clip is always cut from the rendered 16:9 video (CR-007
 * D1 — no standalone vertical production), so "short" still runs the exact
 * same long-form pipeline as source; the choice here only decides whether
 * generate_clips runs at all, and which output step 5 puts front and center.
 * Publishing a clip is a manual upload either way — this app never auto-
 * publishes to Shorts/TikTok.
 */
const OPTIONS: { value: VideoOutputMode; label: string; hint: string }[] = [
  { value: "long", label: "Chỉ video dài", hint: "Video 16:9 chuẩn, không cắt thêm bản dọc nào" },
  {
    value: "short",
    label: "Chỉ video ngắn (Shorts/TikTok)",
    hint: "Vẫn render đủ video dài làm nguồn, nhưng bước Đăng chỉ nổi bật khu tải clip dọc",
  },
  { value: "both", label: "Cả hai", hint: "Video dài để đăng YouTube, cộng thêm clip dọc để đăng Shorts/TikTok" },
];

export function VideoOutputModePicker({ value, onChange, bare = false }: VideoOutputModePickerProps) {
  return (
    <div
      className={bare ? undefined : glass.card}
      style={bare ? undefined : { padding: 22 }}
      data-testid="video-output-mode-picker"
    >
      {!bare && (
        <div className={glass.cardTitle} style={{ marginBottom: 10 }}>
          Loại video
        </div>
      )}

      <div className={selectable.stack} role="radiogroup" aria-label="Loại video">
        {OPTIONS.map((option) => (
          <SelectableOption
            key={option.value}
            selected={value === option.value}
            onSelect={() => onChange(option.value)}
            label={option.label}
            hint={option.hint}
            testId={`video-output-mode-${option.value}`}
          />
        ))}
      </div>

      {(value === "short" || value === "both") && (
        <p className={glass.helperText} style={{ marginRight: 0, marginTop: 10 }} role="status">
          Clip dọc chỉ cắt được từ đoạn script có đánh dấu <code>with self.clip(&quot;tên&quot;):</code> — không
          đánh dấu thì không có gì để cắt.
        </p>
      )}
    </div>
  );
}
