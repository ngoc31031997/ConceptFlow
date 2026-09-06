import { useEffect, useState } from "react";
import { getYoutubeAuthStartUrl, getYoutubeConnectionStatus } from "../api/client";
import glass from "../styles/glass.module.css";
import styles from "./YoutubeConnectButton.module.css";

function YoutubeIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <rect x="3" y="6" width="18" height="12" rx="3" />
      <path d="M11 10l4 2-4 2z" fill="currentColor" stroke="none" />
    </svg>
  );
}

type ConnectionState = "checking" | "connected" | "disconnected";

export function YoutubeConnectButton({ projectId }: { projectId: string }) {
  const [state, setState] = useState<ConnectionState>("checking");

  useEffect(() => {
    let cancelled = false;
    getYoutubeConnectionStatus()
      .then((connected) => {
        if (!cancelled) setState(connected ? "connected" : "disconnected");
      })
      .catch(() => {
        if (!cancelled) setState("disconnected");
      });
    return () => {
      cancelled = true;
    };
  }, []);

  function handleClick() {
    window.location.href = getYoutubeAuthStartUrl(projectId);
  }

  return (
    <div className={glass.card} style={{ padding: 0 }}>
      <div className={styles.row}>
        <div className={styles.left}>
          <div className={styles.icon}>
            <YoutubeIcon />
          </div>
          <div className={styles.title}>Kênh YouTube</div>
          {state !== "checking" && (
            <span
              data-testid="youtube-connection-status"
              className={styles.statusBadge}
              data-connected={state === "connected"}
            >
              {state === "connected" ? "Đã kết nối" : "Chưa kết nối"}
            </span>
          )}
        </div>
        <button type="button" data-testid="youtube-connect-button" className={styles.connectBtn} onClick={handleClick}>
          {state === "connected" ? "Kết nối lại" : "Kết nối YouTube"}
        </button>
      </div>
    </div>
  );
}
