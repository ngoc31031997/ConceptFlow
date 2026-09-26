import { useEffect, useState } from "react";
import { listProjectErrors, type ProjectErrorEntry } from "../api/client";
import { stepLabel } from "../utils/pipelineLabels";
import glass from "../styles/glass.module.css";
import styles from "./ErrorBanner.module.css";

interface ErrorBannerProps {
  errorMessage: string;
  onRetry: () => void;
  isRetrying: boolean;
  /** For the technical detail behind the "?" (the project's error log). */
  projectId?: string;
  /** The saga step that failed, to pick the matching log entry. */
  step?: string;
}

function AlertIcon() {
  return (
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 8v5M12 17h.01" />
      <circle cx="12" cy="12" r="9" />
    </svg>
  );
}

function RetryIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M21 12a9 9 0 11-3-6.7" />
      <path d="M21 4v6h-6" />
    </svg>
  );
}

/** One short line for the banner: the first line, cut at a sentence-ish length. */
export function shortError(message: string, max = 140): string {
  const first = (message.split("\n")[0] ?? "").trim();
  return first.length > max ? `${first.slice(0, max - 1).trimEnd()}…` : first;
}

/**
 * A failed step: a short message, a red "?" that opens the full detail, and one
 * action — retry. Nothing here retries by itself, and there is no "go back and
 * edit" button: the step bar above already lets the Creator open any earlier
 * step, so which one to fix is their call once they have read what went wrong.
 *
 * `projectId`/`step` let the detail include the technical trace kept in the
 * project's error log (loaded only when the "?" is opened).
 */
export function ErrorBanner({ errorMessage, onRetry, isRetrying, projectId, step }: ErrorBannerProps) {
  const [open, setOpen] = useState(false);
  const [trace, setTrace] = useState<ProjectErrorEntry | null>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!open || !projectId) return;
    let cancelled = false;
    listProjectErrors(projectId)
      .then((rows) => {
        if (cancelled || !Array.isArray(rows) || rows.length === 0) return;
        const newestFirst = [...rows].reverse();
        setTrace(newestFirst.find((r) => step && r.step === step) ?? newestFirst[0]);
      })
      .catch(() => {
        /* the message itself is already shown */
      });
    return () => {
      cancelled = true;
    };
  }, [open, projectId, step]);

  const short = shortError(errorMessage);
  const fullText = [errorMessage, trace?.detail].filter(Boolean).join("\n\n");
  // What decides "should I retry?": which step, when, how long it ran, and what
  // kind of failure it was. Only what the error log actually recorded.
  const meta = [
    step ? `bước: ${stepLabel(step)}` : "",
    trace?.at ? `lúc: ${new Date(trace.at).toLocaleTimeString("vi-VN", { hour12: false })}` : "",
    trace?.elapsed_seconds ? `đã chạy: ${trace.elapsed_seconds}s` : "",
    trace?.kind ? `loại: ${trace.kind}` : "",
  ].filter(Boolean);

  async function copy() {
    try {
      await navigator.clipboard.writeText(fullText);
      setCopied(true);
    } catch {
      /* clipboard refused (some app views): the text is selectable in the box */
    }
  }

  return (
    <div className={glass.card} role="alert">
      <div className={styles.wrap}>
        <div className={styles.icon}>
          <AlertIcon />
        </div>
        <div className={styles.body}>
          <p className={styles.message} data-testid="error-banner-message">
            {short}
            <button
              type="button"
              className={styles.helpBtn}
              onClick={() => setOpen((v) => !v)}
              aria-expanded={open}
              aria-label="Xem chi tiết lỗi"
              title="Xem chi tiết lỗi"
              data-testid="error-banner-detail-toggle"
            >
              ?
            </button>
          </p>
          {open && (
            <>
              {meta.length > 0 && (
                <p className={styles.meta} data-testid="error-banner-meta">
                  {meta.map((m) => (
                    <span key={m}>{m}</span>
                  ))}
                </p>
              )}
              <pre className={styles.detail} data-testid="error-banner-detail">
                {fullText}
              </pre>
              <button type="button" className={styles.copyBtn} onClick={copy} data-testid="error-banner-copy">
                {copied ? "Đã sao chép" : "Sao chép chi tiết"}
              </button>
            </>
          )}
          <div className={styles.actions}>
            <button
              type="button"
              data-testid="error-banner-retry-button"
              className={styles.retryBtn}
              onClick={onRetry}
              disabled={isRetrying}
            >
              <RetryIcon />
              {step ? `Thử lại bước ${stepLabel(step)}` : "Thử lại"}
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}
