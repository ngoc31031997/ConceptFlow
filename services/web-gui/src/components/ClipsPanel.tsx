import type { Clip } from "../types";
import { getProjectClipUrl } from "../api/client";
import glass from "../styles/glass.module.css";
import styles from "./ClipsPanel.module.css";

interface ClipsPanelProps {
  projectId: string;
  clips: Clip[];
  /** ResultPage only renders this panel for "short"/"both" — "long" never gets here. */
  videoOutputMode: "long" | "short" | "both";
}

const PRESET_LABELS: Record<string, string> = {
  short: "Shorts (30–60s)",
  long: "TikTok (60–180s)",
};

/**
 * CR-007 D7 follow-up: the API to list/download generated vertical clips
 * (`GET /v1/projects/{id}/clips`, streamed via clipHandler.js) has existed
 * since CR-007 shipped, but no screen ever called it — a Creator had no way
 * to see or download a clip generate_clips produced. Publishing a clip stays
 * a manual upload outside this app (no Shorts/TikTok auto-publish adapter),
 * so this panel's job ends at "here is the file."
 */
export function ClipsPanel({ projectId, clips, videoOutputMode }: ClipsPanelProps) {
  const okClips = clips.filter((c) => c.status === "ok");
  const errorClips = clips.filter((c) => c.status !== "ok");

  return (
    <div className={glass.card} style={{ marginTop: 20 }} data-testid="clips-panel">
      <div className={glass.cardTitle} style={{ marginBottom: 6 }}>
        Clip dọc Shorts/TikTok
      </div>

      {clips.length === 0 ? (
        <p className={glass.cardHint}>
          Chưa có clip nào. Clip chỉ cắt được từ đoạn script có đánh dấu{" "}
          <code>with self.clip(&quot;tên&quot;):</code> — nếu script không dùng, sẽ không có gì
          xuất hiện ở đây dù đã chọn{" "}
          {videoOutputMode === "short" ? "“Chỉ video ngắn”" : "“Cả hai”"}.
        </p>
      ) : (
        <>
          <p className={glass.cardHint} style={{ marginBottom: 12 }}>
            Tải clip về rồi đăng tay lên YouTube Shorts/TikTok — hệ thống chưa tự đăng thay được.
          </p>
          <ul className={styles.list}>
            {okClips.map((clip) => (
              <li key={`${clip.name}-${clip.preset}`} className={styles.item} data-testid={`clip-${clip.name}-${clip.preset}`}>
                <div>
                  <span className={styles.name}>{clip.name}</span>
                  <span className={styles.preset}>{PRESET_LABELS[clip.preset] ?? clip.preset}</span>
                  {clip.duration_seconds !== undefined && (
                    <span className={glass.cardHint}> · {Math.round(clip.duration_seconds)}s</span>
                  )}
                </div>
                <a
                  className={glass.ghostBtn}
                  href={getProjectClipUrl(projectId, clip.name, clip.preset)}
                  target="_blank"
                  rel="noreferrer"
                >
                  Tải clip
                </a>
              </li>
            ))}
          </ul>

          {errorClips.length > 0 && (
            <ul className={styles.list} style={{ marginTop: 10 }}>
              {errorClips.map((clip) => (
                <li
                  key={`${clip.name}-${clip.preset}`}
                  className={styles.item}
                  role="alert"
                  data-testid={`clip-error-${clip.name}-${clip.preset}`}
                >
                  <div>
                    <span className={styles.name}>{clip.name}</span>
                    <span className={styles.preset}>{PRESET_LABELS[clip.preset] ?? clip.preset}</span>
                  </div>
                  <span className={glass.cardHint}>{clip.error_message ?? "Cắt clip thất bại"}</span>
                </li>
              ))}
            </ul>
          )}
        </>
      )}
    </div>
  );
}
