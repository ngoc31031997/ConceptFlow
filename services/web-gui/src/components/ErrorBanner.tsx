import glass from "../styles/glass.module.css";
import styles from "./ErrorBanner.module.css";

interface ErrorBannerProps {
  errorMessage: string;
  onRetry: () => void;
  isRetrying: boolean;
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

export function ErrorBanner({ errorMessage, onRetry, isRetrying }: ErrorBannerProps) {
  return (
    <div className={glass.card} role="alert">
      <div className={styles.wrap}>
        <div className={styles.icon}>
          <AlertIcon />
        </div>
        <div className={styles.body}>
          <p className={styles.message} data-testid="error-banner-message">
            {errorMessage}
          </p>
          <button
            type="button"
            data-testid="error-banner-retry-button"
            className={styles.retryBtn}
            onClick={onRetry}
            disabled={isRetrying}
          >
            <RetryIcon />
            Thử lại
          </button>
        </div>
      </div>
    </div>
  );
}
