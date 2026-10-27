import type { AiBridgeModes, AiBridgeStage } from "../hooks/useAiBridge";
import { Button } from "./ui";
import styles from "./research/research.module.css";

interface AiBridgeRunStatusProps {
  stage: AiBridgeStage;
  /** The modes the run turned on, so the line says what DeepSeek is doing. */
  modes: AiBridgeModes;
  onCancel: () => void;
  /** Prefix of the test ids: `<testId>-stage`, `<testId>-cancel`. */
  testId: string;
}

function stageText(stage: AiBridgeStage, modes: AiBridgeModes): string {
  switch (stage) {
    case "opening":
      return "Đang mở DeepSeek trong một tab mới…";
    case "sending": {
      const on = [modes.deepThink && "DeepThink", modes.search && "Search"]
        .filter(Boolean)
        .join(" + ");
      return on ? `Đang bật ${on} và gửi prompt…` : "Đang gửi prompt…";
    }
    case "answering":
      return `DeepSeek đang ${modes.search ? "suy nghĩ và tra web" : modes.deepThink ? "suy nghĩ" : "trả lời"}… (có thể mất vài phút)`;
    case "reading":
      return "Đang lấy câu trả lời…";
  }
}

/**
 * The line shown while the AI Bridge extension runs a prompt on DeepSeek:
 * where the run stands, and "Huỷ" to stop waiting for it (the DeepSeek tab
 * stays open).
 */
export function AiBridgeRunStatus({ stage, modes, onCancel, testId }: AiBridgeRunStatusProps) {
  return (
    <div className={styles.actions}>
      <span className={styles.hint} role="status" data-testid={`${testId}-stage`}>
        {stageText(stage, modes)}
      </span>
      <Button variant="ghost" onClick={onCancel} data-testid={`${testId}-cancel`}>
        Huỷ
      </Button>
    </div>
  );
}
