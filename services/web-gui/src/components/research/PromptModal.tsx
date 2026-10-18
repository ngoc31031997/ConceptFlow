import { useEffect, useState } from "react";
import { Button, Dialog } from "../ui";
import styles from "./research.module.css";

interface PromptModalProps {
  open: boolean;
  onClose: () => void;
  title: string;
  /** The prompt, or null while it is being built. */
  text: string | null;
  /** Why the prompt could not be built. */
  error: string | null;
  testId: string;
}

/**
 * A generated text the Creator copies into another tool (docs/ux-ui-design-rules.md,
 * rule 4): the whole prompt, selectable, with a button that copies it. When the
 * browser refuses to copy, the text is already selected for copying by hand.
 */
export function PromptModal({ open, onClose, title, text, error, testId }: PromptModalProps) {
  const [copied, setCopied] = useState(false);
  const [copyFailed, setCopyFailed] = useState(false);

  useEffect(() => {
    if (open) {
      setCopied(false);
      setCopyFailed(false);
    }
  }, [open]);

  async function copy() {
    if (text === null) return;
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      setCopyFailed(false);
    } catch {
      setCopied(false);
      setCopyFailed(true);
    }
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={title}
      size="lg"
      testId={testId}
      footer={
        <>
          <Button variant="ghost" onClick={onClose} data-testid={`${testId}-close-button`}>
            Đóng
          </Button>
          <Button variant="ghost" onClick={() => void copy()} disabled={text === null} data-testid={`${testId}-copy`}>
            {copied ? "✓ Đã chép" : "Sao chép prompt"}
          </Button>
        </>
      }
    >
      {error && (
        <p className={styles.error} role="alert" data-testid={`${testId}-error`}>
          {error}
        </p>
      )}
      {!error && text === null && (
        <p className={styles.hint} role="status">
          Đang dựng prompt…
        </p>
      )}
      {text !== null && (
        <pre className={`${styles.promptBox} ${styles.promptFull}`} data-testid={`${testId}-text`}>
          {text}
        </pre>
      )}
      {copyFailed && (
        <p className={styles.warn} role="alert">
          Trình duyệt không cho chép tự động. Bôi đen đoạn trên rồi chép bằng tay.
        </p>
      )}
    </Dialog>
  );
}
