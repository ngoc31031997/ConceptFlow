import type { UseOutlineReview } from "../hooks/useOutlineReview";
import glass from "../styles/glass.module.css";
import styles from "./OutlineReview.module.css";

interface OutlineActionsProps {
  outline: UseOutlineReview;
}

/**
 * Duyệt/Từ chối dàn ý (CR-024), tách khỏi `OutlineReview` để sống trong cột
 * bên phải cạnh ProgressTracker — bug report: đặt hai nút này ở cuối danh
 * sách dàn ý (có thể dài hàng chục dòng) làm chúng cuộn mất khỏi tầm nhìn.
 * `outline` là cùng một instance `useOutlineReview` mà `OutlineReview` dùng,
 * nên `busy` khoá cả hai phía cùng lúc (sửa một dòng thì không bấm được Duyệt).
 */
export function OutlineActions({ outline }: OutlineActionsProps) {
  const { busy, approve, reject } = outline;

  return (
    <div className={glass.card} style={{ padding: 22 }} data-testid="outline-actions">
      <div className={styles.actions} style={{ marginTop: 0 }}>
        <button
          type="button"
          className={glass.btnPrimary}
          disabled={busy}
          onClick={approve}
          data-testid="outline-approve"
        >
          Duyệt và sản xuất
        </button>
        <button
          type="button"
          className={glass.ghostBtn}
          disabled={busy}
          onClick={reject}
          data-testid="outline-reject"
        >
          Quay lại sửa script
        </button>
      </div>
    </div>
  );
}
