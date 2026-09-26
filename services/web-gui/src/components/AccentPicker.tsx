import { ACCENTS, useTheme, type Accent } from "../context/ThemeContext";
import styles from "./AccentPicker.module.css";

const LABELS: Record<Accent, string> = {
  mustard: "Vàng mù tạt",
  coral: "San hô",
  mint: "Bạc hà",
  sky: "Xanh trời",
  lilac: "Tím lilac",
};

/** Hàng ô màu để đổi màu nhấn của cả giao diện; nhớ lựa chọn trong localStorage. */
export function AccentPicker() {
  const { accent, setAccent } = useTheme();
  return (
    <div className={styles.picker} role="radiogroup" aria-label="Màu nhấn" data-testid="accent-picker">
      {ACCENTS.map((a) => (
        <button
          key={a}
          type="button"
          role="radio"
          aria-checked={accent === a}
          aria-label={LABELS[a]}
          title={LABELS[a]}
          className={`${styles.swatch} ${styles[a]} ${accent === a ? styles.on : ""}`}
          onClick={() => setAccent(a)}
        />
      ))}
    </div>
  );
}
