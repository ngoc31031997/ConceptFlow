import type { UseOutlineReview } from "../hooks/useOutlineReview";
import { Card, Button } from "./ui";
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
    <Card data-testid="outline-actions">
      <div className={styles.actions} style={{ marginTop: 0 }}>
        <Button disabled={busy} onClick={approve} data-testid="outline-approve">
          Duyệt và sản xuất
        </Button>
        <Button variant="ghost" disabled={busy} onClick={reject} data-testid="outline-reject">
          Quay lại sửa script
        </Button>
      </div>
    </Card>
  );
}
