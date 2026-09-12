import { estimateNarrationDuration, formatDuration } from "../utils/durationEstimate";
import type { UseOutlineReview } from "../hooks/useOutlineReview";
import type { Project } from "../types";
import glass from "../styles/glass.module.css";
import styles from "./OutlineReview.module.css";

interface OutlineReviewProps {
  project: Project;
  outline: UseOutlineReview;
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
 *
 * Nút Duyệt/Từ chối SỐNG Ở `OutlineActions`, không phải ở đây (bug report):
 * danh sách này có thể dài hàng chục dòng, đặt nút ở cuối nó thì nút cuộn mất
 * khỏi tầm nhìn đúng lúc cần nhất. State dùng chung qua `useOutlineReview` nên
 * bấm nút ở cột kia vẫn phản ánh đúng vào danh sách này (busy khoá cả sửa dòng
 * lẫn duyệt/từ chối cùng lúc).
 */
export function OutlineReview({ project, outline }: OutlineReviewProps) {
  const { busy, error, editing, draftText, setDraftText, startEdit, cancelEdit, saveEdit } = outline;

  const language = project.voice_language === "en" ? "en" : "vi";
  const scenes = project.scenes ?? [];
  const beatFor = new Map((project.beats ?? []).map((beat) => [beat.scene_index, beat.id]));
  const total = scenes.reduce(
    (sum, scene) => sum + estimateNarrationDuration(scene.narration_text, language),
    0,
  );

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
                        className={glass.btnPrimary}
                        disabled={busy}
                        onClick={() => saveEdit(scene.scene_index)}
                        data-testid={`outline-save-${scene.scene_index}`}
                      >
                        Lưu và kiểm lại
                      </button>
                      <button type="button" className={glass.ghostBtn} onClick={cancelEdit}>
                        Huỷ
                      </button>
                    </div>
                  </>
                ) : (
                  <button
                    type="button"
                    className={styles.text}
                    onClick={() => startEdit(scene.scene_index, scene.narration_text)}
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
    </div>
  );
}
