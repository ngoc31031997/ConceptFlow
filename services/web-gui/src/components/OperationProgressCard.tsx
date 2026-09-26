import styles from "./OperationProgressCard.module.css";

interface OperationProgressCardProps {
  title?: string;
  /** Dòng phụ dạng `pha · số lượng · thời gian`. */
  subtitle?: string | null;
  /** Có `total` thì hiện phần trăm; thiếu thì thanh chạy không xác định (FR116.4). */
  done?: number | null;
  total?: number | null;
  /** Lỗi đã phân loại; thẻ vẫn giữ dòng phụ (số ký tự, thời gian) (FR116.5). */
  error?: string | null;
  /** "step": kiểu thẻ bước trong stepper. */
  variant?: "default" | "step";
  testId?: string;
}

export function OperationProgressCard({
  title,
  subtitle,
  done,
  total,
  error,
  variant = "default",
  testId,
}: OperationProgressCardProps) {
  const determinate = typeof total === "number" && total > 0;
  const pct = determinate ? Math.min(100, Math.max(0, Math.round(((done ?? 0) / total!) * 100))) : null;
  return (
    <div className={`${styles.card} ${variant === "step" ? styles.step : ""}`} data-testid={testId}>
      {title && <p className={styles.title}>{title}</p>}
      {subtitle && <p className={styles.subtitle}>{subtitle}</p>}
      <div
        className={styles.track}
        role="progressbar"
        aria-label={title ?? "Tiến độ AI"}
        aria-busy={!error}
        aria-valuemin={determinate ? 0 : undefined}
        aria-valuemax={determinate ? 100 : undefined}
        aria-valuenow={pct ?? undefined}
      >
        <div
          className={`${styles.bar} ${determinate ? styles.determinate : ""}`}
          style={determinate ? { width: `${pct}%` } : undefined}
        />
      </div>
      {error && <p className={styles.error} role="alert">{error}</p>}
    </div>
  );
}
