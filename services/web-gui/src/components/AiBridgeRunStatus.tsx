import { providerName } from "../hooks/useAiBridge";
import type { QueueItemStatus } from "../hooks/useAiQueue";
import { Button } from "./ui";
import styles from "./research/research.module.css";

interface AiBridgeRunStatusProps {
  /** Where the queued item stands; null while the request is still being built or sent to the extension. */
  item: QueueItemStatus | null;
  /** How many items run or wait before this one, when known. */
  ahead?: number | null;
  onCancel: () => void;
  /** Prefix of the test ids: `<testId>-stage`, `<testId>-cancel`. */
  testId: string;
}

function statusText(item: QueueItemStatus | null, ahead: number | null | undefined): string {
  if (!item) return "Đang thêm vào hàng đợi AI…";
  const name = providerName(item.provider);
  if (item.state === "waiting") {
    const before = ahead && ahead > 0 ? ` · ${ahead} mục trước` : "";
    return `Đang chờ trong hàng đợi AI${before} · sẽ chạy trên ${name}`;
  }
  switch (item.stage) {
    case "preparing":
      return `Đang lấy prompt cho ${name}…`;
    case "opening":
      return `Đang mở ${name} trong một tab mới…`;
    case "sending":
      return `Đang gửi prompt cho ${name}…`;
    case "answering":
      return `Đang chạy trên ${name} · đang trả lời… (có thể mất vài phút)`;
    case "reading":
      return `Đang lấy câu trả lời của ${name}…`;
    case "saving":
      return "Đang lưu vào ConceptFlow…";
    default:
      return `Đang chạy trên ${name}…`;
  }
}

/**
 * The line shown while a queued item is on its way: waiting, or running on a provider and
 * at which step, with "Huỷ" to take it out of the queue (a running one is stopped).
 */
export function AiBridgeRunStatus({ item, ahead, onCancel, testId }: AiBridgeRunStatusProps) {
  return (
    <div className={styles.actions}>
      <span className={styles.hint} role="status" data-testid={`${testId}-stage`}>
        {statusText(item, ahead)}
      </span>
      <Button variant="ghost" onClick={onCancel} disabled={!item} data-testid={`${testId}-cancel`}>
        Huỷ
      </Button>
    </div>
  );
}
