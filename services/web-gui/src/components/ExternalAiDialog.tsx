import { useEffect, useState } from "react";
import { ExternalPromptReady, type ExternalAi } from "../api/client";
import type { AiBridgeModes } from "../hooks/useAiBridge";
import { DEFAULT_AI_MODES, messageOf, useQueuedExternalRun } from "../hooks/useQueuedExternalRun";
import { QueuedRunButton } from "./QueuedRunButton";
import { Button, CopyBox, Dialog, TextArea } from "./ui";
import styles from "./ExternalAiDialog.module.css";

interface ExternalAiDialogBaseProps<T> {
  open: boolean;
  onClose: () => void;
  /** Names the item and the task ("Câu chuyện · AI ngoài"); also names the item in the extension's queue. */
  title: string;
  /**
   * Makes the step's own AI request in the given external mode. In "prompt" mode it rejects with
   * `ExternalPromptReady`; in "paste" mode it resolves once the step has stored the result; in "describe" mode
   * it rejects with `ExternalRequestDescribed`, which is how the queue gets the request.
   */
  run: (external: ExternalAi) => Promise<T>;
  /** Modes for the item queued on DeepSeek; reasoning only unless the step needs web sources. */
  modes?: AiBridgeModes;
  /** A reply to show in the reply box on opening, e.g. an answer the step could not read. */
  initialReply?: string;
  /** Why `initialReply` could not be read, shown under the reply box on opening. */
  initialError?: string;
  testId: string;
}

/**
 * What happens once the reply was read. By default the dialog also offers "Thêm vào hàng đợi AI", whose answer
 * is read in the extension, so `onDone` gets nothing and the screen loads what changed. A step whose result is
 * not stored but shown (`queued: false`) gets the result of the pasted reply and has no queue button.
 */
type ExternalAiDialogProps<T> = ExternalAiDialogBaseProps<T> &
  ({ queued?: true; onDone: () => void | Promise<void> } | { queued: false; onDone: (result: T) => void | Promise<void> });

interface QueuedSectionProps<T> {
  title: string;
  run: (external: ExternalAi) => Promise<T>;
  modes: AiBridgeModes;
  disabled: boolean;
  /** The item is done: the step stored the answer and the screen loads what changed. */
  onDone: () => void | Promise<void>;
  /** The item failed with an answer the step could not read: it goes in the reply box with the reason. */
  onUnreadable: (reply: string, message: string) => void;
  testId: string;
}

/**
 * "Thêm vào hàng đợi AI": the step's request goes into the AI Bridge extension's queue, which runs the prompt
 * on an outside AI (the provider chosen in the extension) and has the step read the answer. Rendered only while
 * the dialog is open; closing the dialog stops following the item but does not take it out of the queue.
 * Without the extension it says how to install it and the dialog stays copy-and-paste.
 */
function QueuedSection<T>({ title, run, modes, disabled, onDone, onUnreadable, testId }: QueuedSectionProps<T>) {
  const queued = useQueuedExternalRun({ run, label: title, onDone, onUnreadable, modes });
  if (!queued.available && !queued.outdated) {
    return (
      <p className={styles.hint} data-testid={`${testId}-extension-missing`}>
        Cài extension ConceptFlow AI Bridge (tools/ai-bridge-extension) để chạy prompt bằng DeepSeek, Gemini hoặc Grok và tự dán kết quả.
      </p>
    );
  }
  return (
    <div className={styles.section}>
      <div>
        <QueuedRunButton run={queued} disabled={disabled} testId={`${testId}-queue`} />
        {!queued.busy && queued.available && (
          <p className={styles.hint}>
            Extension chạy prompt trên AI ngoài ở một tab mới (nhà cung cấp chọn trong extension), tự đọc câu trả lời và lưu vào bước này. Đóng hộp thoại không huỷ mục.
          </p>
        )}
      </div>
    </div>
  );
}

/** How a redo form hands its note to an AI outside the system, when the step supports it. */
export interface ExternalRedo {
  /** Makes the redo request with this note in the given external mode (see `ExternalAiDialog.run`). */
  run: (note: string, external: ExternalAi) => Promise<unknown>;
  /** The reply was read and stored; the screen loads what changed. */
  onDone: () => void | Promise<void>;
}

/**
 * Runs one AI step through an AI outside the system (docs/ux-ui-design-rules.md,
 * rules 4 and 8): the prompt the step would send, in a box to copy; below it a box
 * to paste the other AI's answer into; "Đọc kết quả" then runs the step on that
 * answer exactly as it would on the model's own. A reply the step cannot use is
 * explained here and the pasted text stays, so the Creator fixes it or pastes again.
 * With the AI Bridge extension installed, "Thêm vào hàng đợi AI" has the extension
 * do the copy, the run and the paste, then read the answer the same way.
 */
export function ExternalAiDialog<T>(props: ExternalAiDialogProps<T>) {
  const { open, onClose, title, run, modes = DEFAULT_AI_MODES, initialReply = "", initialError, testId } = props;
  const [prompt, setPrompt] = useState<string | null>(null);
  const [promptError, setPromptError] = useState<string | null>(null);
  const [reply, setReply] = useState("");
  const [reading, setReading] = useState(false);
  const [readError, setReadError] = useState<string | null>(null);

  // The prompt is built each time the dialog opens, from the step as it stands now.
  useEffect(() => {
    if (!open) return;
    let stale = false;
    setPrompt(null);
    setPromptError(null);
    setReadError(initialError ?? null);
    setReply(initialReply);
    run({ mode: "prompt" }).then(
      () => {
        if (!stale) setPromptError("Mục này không trả prompt để copy.");
      },
      (err: unknown) => {
        if (stale) return;
        if (err instanceof ExternalPromptReady) setPrompt(err.prompt);
        else setPromptError(messageOf(err));
      },
    );
    return () => {
      stale = true;
    };
    // `run` is rebuilt on every render of the screen; the prompt is fetched once per opening.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  async function read(text: string) {
    setReading(true);
    setReadError(null);
    let result: T;
    try {
      result = await run({ mode: "paste", reply: text });
    } catch (err) {
      setReadError(messageOf(err));
      setReading(false);
      return;
    }
    setReading(false);
    if (props.queued === false) await props.onDone(result);
    else await props.onDone();
    onClose();
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={title}
      size="lg"
      hasInput
      testId={testId}
      footer={
        <>
          <Button variant="ghost" onClick={onClose} data-testid={`${testId}-cancel`}>
            Huỷ
          </Button>
          <Button
            variant="primary"
            onClick={() => void read(reply)}
            disabled={reading || prompt === null || reply.trim() === ""}
            data-testid={`${testId}-read`}
          >
            {reading ? "Đang đọc…" : "Đọc kết quả"}
          </Button>
        </>
      }
    >
      <div className={styles.section}>
        <p className={styles.label}>1. Chép prompt sang AI ngoài</p>
        {promptError && (
          <p className={styles.error} role="alert" data-testid={`${testId}-prompt-error`}>
            {promptError}
          </p>
        )}
        {!promptError && prompt === null && (
          <p className={styles.hint} role="status">
            Đang dựng prompt…
          </p>
        )}
        {prompt !== null && (
          <CopyBox text={prompt} what="prompt" copyTestId={`${testId}-copy`}>
            <pre className={styles.prompt} data-testid={`${testId}-prompt`}>
              {prompt}
            </pre>
          </CopyBox>
        )}
      </div>
      {open && props.queued !== false && (
        <QueuedSection
          title={title}
          run={run}
          modes={modes}
          disabled={reading}
          onDone={async () => {
            await props.onDone();
            onClose();
          }}
          onUnreadable={(reply, message) => {
            setReply(reply);
            setReadError(message);
          }}
          testId={testId}
        />
      )}
      <div className={styles.section}>
        <p className={styles.label}>2. Dán kết quả AI ngoài trả về</p>
        <TextArea
          className={styles.paste}
          value={reply}
          onChange={(e) => setReply(e.target.value)}
          placeholder="Dán toàn bộ câu trả lời vào đây"
          aria-label="Kết quả của AI ngoài"
          data-testid={`${testId}-reply`}
        />
        <p className={styles.hint}>
          Hệ thống đọc kết quả rồi chạy tiếp như khi gọi AI của hệ thống.
        </p>
        {readError && (
          <p className={styles.error} role="alert" data-testid={`${testId}-read-error`}>
            {readError}
          </p>
        )}
      </div>
    </Dialog>
  );
}
