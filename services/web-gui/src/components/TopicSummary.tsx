import { Button } from "./ui";
import styles from "./TopicSummary.module.css";

interface TopicSummaryProps {
  topic: string;
  /** Mở lại bước 1 (Ý tưởng), nơi có đề tài và creative brief. */
  onOpen: () => void;
}

/**
 * Chủ đề của video dạng chỉ đọc, ở đầu các màn sau bước 1: nhắc Creator đang
 * làm video về cái gì, và mở lại bước Ý tưởng để xem đề tài cùng brief. Đề
 * tài chốt lúc tạo project; muốn đổi hướng thì làm lại brief kèm ghi chú.
 */
export function TopicSummary({ topic, onOpen }: TopicSummaryProps) {
  const text = topic.trim();
  return (
    <div className={styles.summary} data-testid="topic-summary">
      <span className={styles.label}>Chủ đề</span>
      <span className={styles.topic} title={text}>
        {text || "Chưa có chủ đề"}
      </span>
      <Button variant="view" onClick={onOpen} data-testid="topic-summary-open">
        Xem ở Ý tưởng
      </Button>
    </div>
  );
}
