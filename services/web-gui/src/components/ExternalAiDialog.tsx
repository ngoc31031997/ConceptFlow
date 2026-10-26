import { useEffect, useState } from "react";
import {
  ApiError,
  ExternalPromptReady,
  GENERIC_CONNECTION_ERROR,
  type ExternalAi,
} from "../api/client";
import { useAiBridge, type AiBridgeModes } from "../hooks/useAiBridge";
import { AiBridgeRunStatus } from "./AiBridgeRunStatus";
import { Button, CopyBox, Dialog, TextArea } from "./ui";
import styles from "./ExternalAiDialog.module.css";

interface ExternalAiDialogProps<T> {
  open: boolean;
  onClose: () => void;
  /** Names the item and the task ("Câu chuyện · AI ngoài"). */
  title: string;
  /**
   * Makes the step's own AI request in the given external mode. In "prompt" mode it rejects with
   * `ExternalPromptReady`; in "paste" mode it resolves once the step has stored the result.
   */
  run: (external: ExternalAi) => Promise<T>;
  /** The reply was read and the step ran on it; gets what the request returned. The screen loads what changed. */
  onDone: (result: T) => void | Promise<void>;
  /** DeepSeek modes for "Chạy bằng DeepSeek"; reasoning only unless the step needs web sources. */
  deepSeekModes?: AiBridgeModes;
  testId: string;
}

/** Most steps reason over what the project already holds, so DeepSeek does not search the web. */
const DEFAULT_DEEPSEEK_MODES: AiBridgeModes = { deepThink: true, search: false };

interface DeepSeekRunProps {
  prompt: string;
  modes: AiBridgeModes;
  disabled: boolean;
  /** DeepSeek's whole answer, to put in the reply box and read. */
  onAnswer: (text: string) => void;
  testId: string;
}

/**
 * "Chạy bằng DeepSeek": the AI Bridge extension runs the prompt on DeepSeek in
 * a background tab and hands the answer back. Rendered only while the dialog
 * is open, so closing the dialog stops waiting for a run. Without the
 * extension it shows nothing and the dialog stays copy-and-paste.
 */
function DeepSeekRun({ prompt, modes, disabled, onAnswer, testId }: DeepSeekRunProps) {
  const bridge = useAiBridge();
  const [error, setError] = useState<string | null>(null);
  if (!bridge.providers.includes("deepseek")) return null;

  async function start() {
    setError(null);
    const outcome = await bridge.run("deepseek", prompt, modes);
    if (outcome.ok) onAnswer(outcome.text);
    else if (!outcome.cancelled) setError(outcome.message);
  }

  return (
    <div className={styles.section}>
      {bridge.stage === null ? (
        <div>
          <Button variant="run" onClick={() => void start()} disabled={disabled} data-testid={`${testId}-deepseek`}>
            Chạy bằng DeepSeek
          </Button>
          <p className={styles.hint}>Extension chạy prompt trên DeepSeek ở tab nền, dán câu trả lời vào ô bên dưới rồi tự đọc kết quả.</p>
        </div>
      ) : (
        <AiBridgeRunStatus stage={bridge.stage} modes={modes} onCancel={bridge.cancel} testId={`${testId}-deepseek`} />
      )}
      {error && (
        <p className={styles.error} role="alert" data-testid={`${testId}-deepseek-error`}>
          {error}
        </p>
      )}
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

function messageOf(err: unknown): string {
  return err instanceof ApiError && err.message ? err.message : GENERIC_CONNECTION_ERROR;
}

/**
 * Runs one AI step through an AI outside the system (docs/ux-ui-design-rules.md,
 * rules 4 and 8): the prompt the step would send, in a box to copy; below it a box
 * to paste the other AI's answer into; "Đọc kết quả" then runs the step on that
 * answer exactly as it would on the model's own. A reply the step cannot use is
 * explained here and the pasted text stays, so the Creator fixes it or pastes again.
 * With the AI Bridge extension installed, "Chạy bằng DeepSeek" does the copy,
 * the run and the paste, then reads the answer the same way.
 */
export function ExternalAiDialog<T>({
  open,
  onClose,
  title,
  run,
  onDone,
  deepSeekModes = DEFAULT_DEEPSEEK_MODES,
  testId,
}: ExternalAiDialogProps<T>) {
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
    setReadError(null);
    setReply("");
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
    await onDone(result);
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
      {open && prompt !== null && (
        <DeepSeekRun
          prompt={prompt}
          modes={deepSeekModes}
          disabled={reading}
          onAnswer={(text) => {
            setReply(text);
            void read(text);
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
