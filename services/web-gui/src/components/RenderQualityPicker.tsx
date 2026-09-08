import type { RenderQuality } from "../context/ProjectDraftContext";
import glass from "../styles/glass.module.css";
import styles from "./NarrationPanel.module.css";

interface RenderQualityPickerProps {
  value: RenderQuality;
  onChange: (quality: RenderQuality) => void;
}

/**
 * CR-004 FR12.6. Rendering was pinned to 720p30, which is below what a
 * monetized channel should publish and throws away Manim's smooth motion.
 *
 * The presets are framed around what the Creator is doing rather than the
 * numbers: a draft pass exists to check that the content works, and only the
 * upload pass needs to be worth publishing.
 */
const OPTIONS: { value: RenderQuality; label: string; hint: string }[] = [
  { value: "720p30", label: "Nháp nhanh", hint: "720p30 — render nhanh nhất, để duyệt nội dung" },
  { value: "1080p60", label: "Chuẩn", hint: "1080p60 — mức nên dùng khi đăng YouTube" },
  { value: "4k60", label: "Cao", hint: "4K60 — rất nặng, chỉ dùng khi thật sự cần" },
];

export function RenderQualityPicker({ value, onChange }: RenderQualityPickerProps) {
  const selected = OPTIONS.find((option) => option.value === value) ?? OPTIONS[1];

  return (
    <div className={glass.card} style={{ padding: 22 }} data-testid="render-quality-picker">
      <div className={glass.cardTitle} style={{ marginBottom: 8 }}>
        Chất lượng video
      </div>

      <div style={{ display: "flex", gap: 8, width: "100%" }}>
        {OPTIONS.map((option) => (
          <button
            key={option.value}
            type="button"
            className={`${styles.voiceCard} ${value === option.value ? styles.selected : ""}`}
            aria-pressed={value === option.value}
            style={{ justifyContent: "center", flex: 1 }}
            onClick={() => onChange(option.value)}
            data-testid={`render-quality-${option.value}`}
          >
            <span className={styles.voiceName}>{option.label}</span>
          </button>
        ))}
      </div>

      <p className={glass.helperText} style={{ marginRight: 0, marginTop: 10 }}>
        {selected.hint}
      </p>
      {value === "720p30" && (
        <p className={glass.helperText} style={{ marginRight: 0, marginTop: 6 }} role="status">
          Bản nháp không nên dùng để đăng — YouTube sẽ nén lại một lần nữa, nên nguồn cần dư chất lượng.
        </p>
      )}
      {value === "4k60" && (
        <p className={glass.helperText} style={{ marginRight: 0, marginTop: 6 }} role="status">
          Render 4K nặng hơn 1080p nhiều lần và hiếm khi tăng lượt xem — cân nhắc kỹ trước khi dùng.
        </p>
      )}
    </div>
  );
}
