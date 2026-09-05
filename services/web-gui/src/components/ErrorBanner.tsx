interface ErrorBannerProps {
  errorMessage: string;
  onRetry: () => void;
  isRetrying: boolean;
}

export function ErrorBanner({ errorMessage, onRetry, isRetrying }: ErrorBannerProps) {
  return (
    <div role="alert">
      <p data-testid="error-banner-message">{errorMessage}</p>
      <button
        type="button"
        data-testid="error-banner-retry-button"
        onClick={onRetry}
        disabled={isRetrying}
      >
        Thử lại
      </button>
    </div>
  );
}
