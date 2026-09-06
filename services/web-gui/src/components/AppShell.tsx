import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import styles from "./AppShell.module.css";

const STEP_LABELS = ["Soạn nội dung", "Xử lý", "Xuất bản"] as const;

interface AppShellProps {
  currentStep?: 1 | 2 | 3;
  title: string;
  subtitle: string;
  children: ReactNode;
}

function CheckIcon() {
  return (
    <svg width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M5 13l4 4L19 7" />
    </svg>
  );
}

export function AppShell({ currentStep, title, subtitle, children }: AppShellProps) {
  return (
    <div className={styles.stage}>
      <div className={`${styles.blob} ${styles.blob1}`} />
      <div className={`${styles.blob} ${styles.blob2}`} />
      <div className={`${styles.blob} ${styles.blob3}`} />

      <div className={styles.content}>
        <div className={styles.topbar}>
          <div className={styles.logo}>
            <div className={styles.logoMark}>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none">
                <path d="M5 4L20 12L5 20V4Z" fill="white" />
              </svg>
            </div>
            <div className={styles.logoName}>ConceptFlow</div>
          </div>

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

          <div className={styles.spacer} />

          <Link to="/videos" className={styles.logoName} style={{ fontSize: 12.5, fontWeight: 500 }}>
            Danh sách video
          </Link>
        </div>

        <div className={styles.mainCol}>
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
