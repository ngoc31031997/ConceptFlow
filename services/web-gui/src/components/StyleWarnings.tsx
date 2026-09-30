import { useState } from "react";
import type { CodeDiagnostic } from "../api/client";
import { usePresence } from "../hooks/usePresence";
import { useStyleWarnings } from "../hooks/useStyleRuleNames";
import { fixWarningsNote, warningText, type StyleWarning } from "../utils/styleRules";
import { Button } from "./ui";
import glass from "../styles/glass.module.css";
import styles from "./StyleWarnings.module.css";

/** Danh sách cảnh báo, mỗi dòng: mã · tên luật — dòng N: chi tiết. */
export function StyleWarningList({ items, testId }: { items: StyleWarning[]; testId?: string }) {
  return (
    <ul className={styles.list} data-testid={testId}>
      {items.map((w, i) => (
        <li key={i}>
          {(w.code || w.rule) && <b className={styles.rule}>{[w.code, w.rule].filter(Boolean).join(" · ")}</b>}
          {(w.code || w.rule) && " — "}
          {w.line != null && <span className={styles.line}>dòng {w.line}</span>}
          {w.line != null && ": "}
          {w.detail}
        </li>
      ))}
    </ul>
  );
}

interface StyleWarningsProps {
  warnings: CodeDiagnostic[];
  /** Tên component của hình, cho data-testid. */
  name: string;
  /**
   * "Nhờ AI sửa các cảnh báo này": nhận ghi chú đã soạn sẵn từ danh sách cảnh
   * báo. Không truyền (hình có sẵn/hình mẫu, không vẽ lại được) thì không có nút.
   */
  onFix?: (note: string) => void;
  /** Hears the list open and close, so the tile can widen while it is open. */
  onToggle?: (open: boolean) => void;
}

/**
 * "⚠ N cảnh báo style" trên ô hình. Rê chuột (hoặc focus) hiện ngay
 * danh sách dạng tooltip; bấm thì mở danh sách ngay dưới ô (trượt ra, dùng
 * được trên màn hình cảm ứng), kèm nút sao chép và nút nhờ AI sửa.
 */
export function StyleWarnings({ warnings, name, onFix, onToggle }: StyleWarningsProps) {
  const items = useStyleWarnings(warnings);
  const [open, setOpen] = useState(false);
  const [copied, setCopied] = useState<"" | "ok" | "fail">("");
  const { mounted, closing } = usePresence(open);
  if (warnings.length === 0) return null;

  async function copy() {
    try {
      await navigator.clipboard.writeText(items.map(warningText).join("\n"));
      setCopied("ok");
    } catch {
      setCopied("fail");
    }
    window.setTimeout(() => setCopied(""), 2000);
  }

  return (
    <div className={styles.wrap}>
      <button
        type="button"
        className={styles.badge}
        aria-expanded={open}
        onClick={() => {
          setOpen(!open);
          onToggle?.(!open);
        }}
        data-testid={`illustration-warnings-${name}`}
      >
        ⚠ {warnings.length} cảnh báo style <span aria-hidden="true">{open ? "▴" : "▾"}</span>
      </button>
      {!open && (
        <div role="tooltip" className={styles.tooltip} data-testid={`illustration-warnings-tip-${name}`}>
          <StyleWarningList items={items} />
          <span className={styles.tipHint}>Bấm để mở, sao chép hoặc nhờ AI sửa.</span>
        </div>
      )}
      {mounted && (
        <div
          className={`${styles.panel} ${closing ? glass.revealOut : glass.reveal}`}
          data-testid={`illustration-warnings-panel-${name}`}
        >
          <StyleWarningList items={items} />
          <div className={styles.actions}>
            <Button variant="ghost" onClick={copy} data-testid={`illustration-warnings-copy-${name}`}>
              {copied === "ok" ? "Đã sao chép" : copied === "fail" ? "Không sao chép được" : "Sao chép cảnh báo"}
            </Button>
            {onFix && (
              <Button variant="ghost" onClick={() => onFix(fixWarningsNote(items))} data-testid={`illustration-warnings-fix-${name}`}>
                Nhờ AI sửa các cảnh báo này
              </Button>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
