import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import styles from "./AppShell.module.css";

/*
  The whole journey, one pill per screen the Creator actually passes through.
  The old three-step version compressed all of the work into "Soạn nội dung"
  and gave the two automatic phases equal billing, so it described the plumbing
  rather than the path.
*/
const STEP_LABELS = ["Script", "Cấu hình", "Xem lại", "Xử lý", "Đăng"] as const;

interface AppShellProps {
  currentStep?: 1 | 2 | 3 | 4 | 5;
  title: string;
  subtitle: string;
  wide?: boolean;
  /** Page-specific action rendered at the right of the top bar. */
  headerAction?: ReactNode;
  children: ReactNode;
}

function CheckIcon() {
  return (
    <svg width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M5 13l4 4L19 7" />
    </svg>
  );
}

export function AppShell({ currentStep, title, subtitle, wide, headerAction, children }: AppShellProps) {
  return (
    <div className={styles.stage}>
      <div className={`${styles.blob} ${styles.blob1}`} />
      <div className={`${styles.blob} ${styles.blob2}`} />
      <div className={`${styles.blob} ${styles.blob3}`} />

      <div className={styles.content}>
        <div className={styles.topbar}>
          <Link to="/" className={styles.logo} style={{ textDecoration: "none" }}>
            <img
              className={styles.logoMark}
              src="/icon-192.png"
              alt=""
              width={34}
              height={34}
              /* Trang trí thuần tuý: tên kênh đã nằm ngay cạnh ở logoName,
                 nên alt rỗng để trình đọc màn hình không đọc lặp hai lần. */
            />
            <div className={styles.logoName}>ConceptFlow</div>
          </Link>

          {currentStep && (
            <div className={styles.stepsPill}>
              {STEP_LABELS.map((label, index) => {
                const stepNumber = index + 1;
                const isActive = stepNumber === currentStep;
                const isDone = stepNumber < currentStep;
                const className = [styles.stepItem, isActive ? styles.active : "", isDone ? styles.done : ""]
                  .filter(Boolean)
                  .join(" ");
                return (
                  <div key={label} className={className}>
                    <span className={styles.stepNum}>{isDone ? <CheckIcon /> : stepNumber}</span>
                    {label}
                  </div>
                );
              })}
            </div>
          )}

          <div className={styles.headerActions}>
            {headerAction}
            <Link to="/videos" className={styles.headerLink}>
              Danh sách video
            </Link>
          </div>
        </div>

        <div className={`${styles.mainCol} ${wide ? styles.mainColWide : ""}`}>
          <div className={styles.heading}>
            <h1>{title}</h1>
            <p>{subtitle}</p>
          </div>
          {children}
        </div>
      </div>
    </div>
  );
}
