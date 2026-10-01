import { Button } from "./ui";
import styles from "./TopicSummary.module.css";

interface TopicSummaryProps {
  topic: string;
  /** Mở lại bước 1, chỗ duy nhất sửa chủ đề. */
  onEdit: () => void;
}

/**
 * Chủ đề của video dạng chỉ đọc, ở đầu các màn sau bước 1: nhắc Creator đang
 * làm video về cái gì, và đưa về bước 1 nếu muốn sửa. Chủ đề chỉ sửa ở một
 * chỗ để không có hai ô cùng giữ một giá trị.
 */
export function TopicSummary({ topic, onEdit }: TopicSummaryProps) {
  const text = topic.trim();
  return (
    <div className={styles.summary} data-testid="topic-summary">
      <span className={styles.label}>Chủ đề</span>
      <span className={styles.topic} title={text}>
        {text || "Chưa có chủ đề"}
      </span>
      <Button variant="ghost" onClick={onEdit} data-testid="topic-summary-edit">
        Sửa
      </Button>
    </div>
  );
}
