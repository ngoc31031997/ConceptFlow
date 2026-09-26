import { useEffect, useRef, useState } from "react";
import { ACCENTS, useTheme, type Accent } from "../context/ThemeContext";
import styles from "./AccentPicker.module.css";

const LABELS: Record<Accent, string> = {
  mustard: "Vàng mù tạt",
  orange: "Cam",
  coral: "San hô",
  rose: "Hồng",
  lilac: "Tím lilac",
  indigo: "Chàm",
  sky: "Xanh trời",
  teal: "Xanh ngọc",
  mint: "Bạc hà",
  lime: "Xanh chanh",
};

/** Nút hiện màu đang dùng; bấm mở danh sách màu, chọn xong tự đóng. */
export function AccentPicker() {
  const { accent, setAccent } = useTheme();
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (!rootRef.current?.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", onDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  return (
    <div className={styles.root} ref={rootRef} data-testid="accent-picker">
      <button
        type="button"
        className={styles.trigger}
        onClick={() => setOpen((o) => !o)}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-label={`Màu nhấn: ${LABELS[accent]}`}
        title={`Màu nhấn: ${LABELS[accent]}`}
      >
        <span className={`${styles.swatch} ${styles[accent]}`} />
        <span className={styles.chevron} aria-hidden="true">{open ? "▴" : "▾"}</span>
      </button>
      {open && (
        <div className={styles.menu} role="listbox" aria-label="Chọn màu nhấn">
          {ACCENTS.map((a) => (
            <button
              key={a}
              type="button"
              role="option"
              aria-selected={accent === a}
              className={`${styles.option} ${accent === a ? styles.on : ""}`}
              onClick={() => {
                setAccent(a);
                setOpen(false);
              }}
            >
              <span className={`${styles.swatch} ${styles[a]}`} />
              {LABELS[a]}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
