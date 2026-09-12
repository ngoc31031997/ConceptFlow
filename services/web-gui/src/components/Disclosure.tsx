import { useState, type ReactNode } from "react";
import glass from "../styles/glass.module.css";

interface DisclosureProps {
  title: string;
  hint?: string;
  /** Mở sẵn khi lần đầu mount — dùng khi nội dung ít khả năng cần thu gọn. */
  defaultOpen?: boolean;
  children: ReactNode;
  testId?: string;
}

/**
 * Một affordance "thu gọn/mở rộng" duy nhất (UX review #5) — thay cho 3 kiểu
 * khác nhau đã mọc lên độc lập: chevron tự chế ở ProjectInputPanel, ghostBtn
 * hoán bằng cả một card ở khu render-lại/tạo bản Shorts của ResultPage, và
 * <details>/<summary> gốc ở ShortScriptAssistant.
 */
export function Disclosure({ title, hint, defaultOpen = false, children, testId }: DisclosureProps) {
  const [isOpen, setIsOpen] = useState(defaultOpen);

  return (
    <div className={glass.card} data-testid={testId}>
      <button
        type="button"
        className={glass.discloseBtn}
        onClick={() => setIsOpen((v) => !v)}
        aria-expanded={isOpen}
        data-testid={testId ? `${testId}-toggle` : undefined}
      >
        <span>
          <span className={glass.cardTitle}>{title}</span>
          {hint && <span className={glass.cardHint} style={{ display: "block" }}>{hint}</span>}
        </span>
        <span className={glass.discloseChevron} data-open={isOpen} aria-hidden="true">
          ▾
        </span>
      </button>

      {isOpen && <div className={glass.mtSm}>{children}</div>}
    </div>
  );
}
