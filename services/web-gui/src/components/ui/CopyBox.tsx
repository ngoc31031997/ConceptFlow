import type { ReactNode } from "react";
import { CopyButton } from "./CopyButton";
import styles from "./CopyBox.module.css";

interface CopyBoxProps {
  /** The text the quick-copy button copies. */
  text: string;
  /** What the box holds ("prompt nghiên cứu"); names the button for a screen reader. */
  what?: string;
  /** Disables the button, e.g. while the text is being built or is out of date. */
  disabled?: boolean;
  /** Test id of the copy button. */
  copyTestId?: string;
  /** The text itself: a `<pre>`, a `<TextArea>` or any block. Its own border and fill are dropped, the box draws them. */
  children: ReactNode;
}

/**
 * One bordered box for a block of text the Creator may paste elsewhere (a
 * prompt, a script, code, an error trace), with the quick-copy button inside it
 * at the top-left (docs/ux-ui-design-rules.md, rule 8). Wrap the `<pre>` or
 * `<TextArea>` in it; the box owns the outline, the button and the divider.
 */
export function CopyBox({ text, what, disabled, copyTestId, children }: CopyBoxProps) {
  return (
    <div className={styles.box}>
      <div className={styles.bar}>
        <CopyButton text={text} what={what} disabled={disabled} testId={copyTestId} />
      </div>
      <div className={styles.body}>{children}</div>
    </div>
  );
}
