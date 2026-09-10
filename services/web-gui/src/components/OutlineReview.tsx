import { useState } from "react";
import { approveOutline, editNarration, rejectOutline } from "../api/client";
import { estimateNarrationDuration, formatDuration } from "../utils/durationEstimate";
import type { Project } from "../types";
import glass from "../styles/glass.module.css";
import styles from "./OutlineReview.module.css";

interface OutlineReviewProps {
  project: Project;
  onDecided: () => void;
}

/**
 * Màn duyệt dàn ý — điểm dừng duy nhất trước khi hệ thống tiêu tiền (CR-024).
 *
 * Trước CR này, lần đầu Creator biết video nói gì là lúc xem video đã dựng
 * xong: nội dung chỉ tồn tại dưới dạng chuỗi ký tự nằm rải rác trong một file
 * Python do AI ngoài viết ra, không có cách nào đọc lướt.
 *
 * Màn này cố ý KHÔNG hiện code, tên class hay đường dẫn artifact (FR68.4). Nó
 * hiện thứ Creator cần để trả lời đúng một câu hỏi: video này nói gì, theo thứ
 * tự nào, và trên màn hình có gì.
 */
export function OutlineReview({ project, onDecided }: OutlineReviewProps) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [editing, setEditing] = useState<number | null>(null);
  const [draftText, setDraftText] = useState("");

  const language = project.voice_language === "en" ? "en" : "vi";
  const scenes = project.scenes ?? [];
  const beatFor = new Map((project.beats ?? []).map((beat) => [beat.scene_index, beat.id]));
  const total = scenes.reduce(
    (sum, scene) => sum + estimateNarrationDuration(scene.narration_text, language),
    0,
  );

  async function run(action: () => Promise<void>) {
    setBusy(true);
    setError(null);
    try {
      await action();
      onDecided();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Có lỗi xảy ra");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className={glass.card} style={{ padding: 22 }} data-testid="outline-review">
      <div className={glass.cardTitle}>Duyệt dàn ý trước khi sản xuất</div>
      <p className={glass.helperText} style={{ marginRight: 0 }}>
        Chưa tạo giọng đọc và chưa render — sửa ở đây không tốn gì. Ước tính{" "}
        <strong>{formatDuration(total)}</strong> lời thoại, {scenes.length} câu.
      </p>

      {(project.validation_warnings ?? []).length > 0 && (
        <ul className={styles.warnings} data-testid="outline-warnings">
          {(project.validation_warnings ?? []).map((warning, index) => (
            <li key={index}>{warning}</li>
          ))}
        </ul>
      )}

      <ol className={styles.lines}>
        {scenes.map((scene) => {
          const beat = beatFor.get(scene.scene_index);
          return (
            <li key={scene.scene_index} className={styles.line}>
              {beat && <div className={styles.beat}>{beat}</div>}
              <div className={styles.lineBody}>
                {editing === scene.scene_index ? (
                  <>
                    <textarea
                      className={styles.editor}
                      value={draftText}
                      onChange={(event) => setDraftText(event.target.value)}
                      data-testid={`outline-edit-${scene.scene_index}`}
                    />
                    <div className={styles.editActions}>
                      <button
                        type="button"
                        disabled={busy}
                        onClick={() =>
                          run(async () => {
                            await editNarration(project.project_id, scene.scene_index, draftText);
                            setEditing(null);
                          })
                        }
                        data-testid={`outline-save-${scene.scene_index}`}
                      >
                        Lưu và kiểm lại
                      </button>
                      <button type="button" onClick={() => setEditing(null)}>
                        Huỷ
                      </button>
                    </div>
                  </>
                ) : (
                  <button
                    type="button"
                    className={styles.text}
                    onClick={() => {
                      setEditing(scene.scene_index);
                      setDraftText(scene.narration_text);
                    }}
                    data-testid={`outline-line-${scene.scene_index}`}
                  >
                    {scene.narration_text}
                  </button>
                )}
                <div className={styles.meta}>
                  {formatDuration(estimateNarrationDuration(scene.narration_text, language))}
                  {scene.visual ? ` · trên màn hình: ${scene.visual}` : ""}
                </div>
              </div>
            </li>
          );
        })}
      </ol>

      {error && (
        <p className={styles.error} role="alert" data-testid="outline-error">
          {error}
        </p>
      )}

      <div className={styles.actions}>
        <button
          type="button"
          disabled={busy}
          onClick={() => run(() => approveOutline(project.project_id))}
          data-testid="outline-approve"
        >
          Duyệt và sản xuất
        </button>
        <button
          type="button"
          disabled={busy}
          onClick={() => run(() => rejectOutline(project.project_id))}
          data-testid="outline-reject"
        >
          Quay lại sửa script
        </button>
      </div>
    </div>
  );
}
