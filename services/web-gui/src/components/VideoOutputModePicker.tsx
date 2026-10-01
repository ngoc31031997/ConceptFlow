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
 * The two shapes a new video is made in. A short is built vertically at
 * 1080x1920 from the start, with its own script and storyboard (ADR-0031); a
 * short from an existing long video starts from that video's Result screen.
 * Publishing a short is a manual upload — this app never auto-publishes to
 * Shorts/TikTok.
 */
const OPTIONS: { value: VideoOutputMode; label: string; hint: string }[] = [
  { value: "long", label: "Video dài 16:9", hint: "Video ngang cho YouTube" },
  {
    value: "short",
    label: "Short dọc 9:16",
    hint: "Dựng thẳng khung dọc 30–60 giây cho Shorts/TikTok, kịch bản riêng",
  },
];

/** Picks whether the video is a long 16:9 video or a vertical short. */
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

      {value === "short" && (
        <p className={glass.helperText} style={{ marginRight: 0, marginTop: 10 }} role="status">
          Short dọc dùng format ngắn, dựng bằng Remotion và không đốt phụ đề — chữ từ khoá nằm ngay trong hình.
        </p>
      )}
    </div>
  );
}
