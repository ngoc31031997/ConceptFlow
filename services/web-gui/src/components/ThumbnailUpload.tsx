import { useEffect, useRef, useState } from "react";
import {
  ApiError,
  getProjectThumbnailUrl,
  getThumbnailInfo,
  uploadThumbnail,
} from "../api/client";
import glass from "../styles/glass.module.css";
import styles from "./ThumbnailUpload.module.css";

interface ThumbnailUploadProps {
  projectId: string;
  onThumbnailPathChange: (thumbnailPath: string | null) => void;
}

type UploadState = "idle" | "uploading" | "success" | "error";

export function ThumbnailUpload({ projectId, onThumbnailPathChange }: ThumbnailUploadProps) {
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [uploadState, setUploadState] = useState<UploadState>("idle");
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    let cancelled = false;
    getThumbnailInfo(projectId)
      .then((info) => {
        if (cancelled || !info.exists) return;
        setPreviewUrl(`${getProjectThumbnailUrl(projectId)}?t=${Date.now()}`);
        onThumbnailPathChange(info.thumbnail_path);
      })
      .catch(() => {
        /* no existing thumbnail — leave state as idle */
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId]);

  async function handleFileSelected(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;

    setPreviewUrl(URL.createObjectURL(file));
    setUploadState("uploading");
    setErrorMessage(null);

    try {
      const result = await uploadThumbnail(projectId, file);
      setUploadState("success");
      onThumbnailPathChange(result.thumbnail_path);
    } catch (err) {
      setUploadState("error");
      setErrorMessage(err instanceof ApiError ? err.message : String(err));
      onThumbnailPathChange(null);
    }
  }

  return (
    <div className={`${glass.card} ${styles.wrapper}`}>
      <div className={styles.label}>Thumbnail</div>

      <div className={styles.row}>
        {previewUrl ? (
          <img src={previewUrl} alt="Thumbnail preview" className={styles.preview} />
        ) : (
          <div className={styles.placeholder}>Chưa có thumbnail</div>
        )}

        <button
          type="button"
          data-testid="thumbnail-upload-button"
          className={styles.uploadBtn}
          disabled={uploadState === "uploading"}
          onClick={() => fileInputRef.current?.click()}
        >
          {uploadState === "uploading" ? "Đang tải lên..." : "Tải ảnh lên"}
        </button>
        <input
          ref={fileInputRef}
          type="file"
          accept="image/jpeg,image/png"
          data-testid="thumbnail-file-input"
          className={styles.hiddenInput}
          onChange={handleFileSelected}
        />
      </div>

      {/* Never replaces the preview/button above — success or error is
          appended as its own element at the end, so the existing UI stays
          intact after an upload attempt. */}
      {uploadState === "success" && (
        <span data-testid="thumbnail-upload-success" className={styles.successBadge}>
          ✓ Đã tải lên thành công
        </span>
      )}
      {uploadState === "error" && (
        <span role="alert" data-testid="thumbnail-upload-error" className={styles.errorBadge}>
          {errorMessage}
        </span>
      )}
    </div>
  );
}
