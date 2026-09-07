import { useEffect, useRef, useState } from "react";
import { ApiError, getMusicInfo, getProjectMusicUrl, uploadMusic } from "../api/client";
import glass from "../styles/glass.module.css";
import styles from "./BackgroundMusicPicker.module.css";

interface BackgroundMusicPickerProps {
  projectId: string;
  value: string | null;
  onChange: (path: string | null) => void;
}

type UploadState = "idle" | "uploading" | "success" | "error";

function MusicIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M9 18V5l12-2v13" />
      <circle cx="6" cy="18" r="3" />
      <circle cx="18" cy="16" r="3" />
    </svg>
  );
}

export function BackgroundMusicPicker({ projectId, value, onChange }: BackgroundMusicPickerProps) {
  const [enabled, setEnabled] = useState(value !== null);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [uploadState, setUploadState] = useState<UploadState>("idle");
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    let cancelled = false;
    getMusicInfo(projectId)
      .then((info) => {
        if (cancelled || !info.exists) return;
        setEnabled(true);
        setPreviewUrl(`${getProjectMusicUrl(projectId)}?t=${Date.now()}`);
        setUploadState("success");
        onChange(info.background_music_path);
      })
      .catch(() => {
        /* no existing music — leave state as idle */
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId]);

  function handleToggle() {
    const next = !enabled;
    setEnabled(next);
    if (!next) {
      onChange(null);
      setPreviewUrl(null);
      setUploadState("idle");
      setErrorMessage(null);
    }
  }

  async function handleFileSelected(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;

    setPreviewUrl(URL.createObjectURL(file));
    setUploadState("uploading");
    setErrorMessage(null);

    try {
      const result = await uploadMusic(projectId, file);
      setUploadState("success");
      onChange(result.background_music_path);
    } catch (err) {
      setUploadState("error");
      setErrorMessage(err instanceof ApiError ? err.message : String(err));
      onChange(null);
    }
  }

  return (
    <div className={glass.card}>
      <div className={styles.row}>
        <div className={styles.label}>
          <div className={styles.icon}>
            <MusicIcon />
          </div>
          Thêm nhạc nền (tùy chọn)
        </div>
        <button
          type="button"
          role="switch"
          aria-checked={enabled}
          className={`${styles.toggle} ${enabled ? "" : styles.off}`}
          onClick={handleToggle}
        >
          <span className={styles.knob} />
        </button>
      </div>
      {enabled && (
        <div className={styles.uploadArea}>
          <button
            type="button"
            data-testid="music-upload-button"
            className={styles.uploadBtn}
            disabled={uploadState === "uploading"}
            onClick={() => fileInputRef.current?.click()}
          >
            {uploadState === "uploading" ? "Đang tải lên..." : "Chọn file nhạc từ máy"}
          </button>
          <input
            ref={fileInputRef}
            type="file"
            accept="audio/mpeg,audio/wav,audio/ogg,audio/mp4,.mp3,.wav,.ogg,.m4a"
            data-testid="music-file-input"
            className={styles.hiddenInput}
            onChange={handleFileSelected}
          />

          {previewUrl && (
            /* eslint-disable-next-line jsx-a11y/media-has-caption */
            <audio controls src={previewUrl} className={styles.player} data-testid="music-preview-player" />
          )}

          {uploadState === "success" && (
            <span data-testid="music-upload-success" className={styles.successBadge}>
              ✓ Đã tải lên thành công
            </span>
          )}
          {uploadState === "error" && (
            <span role="alert" data-testid="music-upload-error" className={styles.errorBadge}>
              {errorMessage}
            </span>
          )}
        </div>
      )}
    </div>
  );
}
