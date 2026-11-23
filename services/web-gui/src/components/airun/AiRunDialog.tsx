import { useEffect, useRef, useState, type ReactNode } from "react";
import {
  applyAiRun,
  cancelAiRun,
  replyAiRun,
  type AiChannel,
  type AiRun,
  type ExternalRequest,
} from "../../api/client";
import { messageOf } from "../../hooks/useQueuedExternalRun";
import { Button, CopyBox, Dialog, FormField, TextArea } from "../ui";
import { isOpen, useAiRuns } from "./AiRunsContext";
import styles from "./AiRun.module.css";

/** How each channel is named on buttons, chips and the menu. */
export const CHANNEL_LABEL: Record<AiChannel, string> = {
  api: "API",
  extension: "Extension",
  manual: "Thủ công",
};

const CHANNEL_HINT: Record<AiChannel, string> = {
  api: "Model của hệ thống (đặt ở thẻ Chạy AI)",
  extension: "DeepSeek, Gemini hoặc Grok qua AI Bridge",
  manual: "Tự chép prompt, dán câu trả lời",
};

const CHANNELS: AiChannel[] = ["api", "extension", "manual"];

/** The server's reason, or the reason the page itself gave (no extension, queue refused). */
function reasonOf(err: unknown): string {
  return err instanceof Error && err.message && !(err instanceof TypeError) ? err.message : messageOf(err);
}

/** The run's status in a few words, with a tone for its colour. */
export function statusOf(run: AiRun): { text: string; tone: "run" | "ok" | "warn" | "bad" } {
  switch (run.status) {
    case "running":
      return { text: "Đang gọi API…", tone: "run" };
    case "waiting":
      return run.channel === "extension"
        ? { text: "Đang chờ extension trả lời…", tone: "run" }
        : { text: "Chờ bạn dán câu trả lời", tone: "warn" };
    case "ready":
      return { text: "Đã có kết quả, xem rồi bấm Áp dụng", tone: "ok" };
    case "unreadable":
      return { text: "Câu trả lời chưa dùng được", tone: "bad" };
    case "failed":
      return { text: "Lỗi, chưa có câu trả lời", tone: "bad" };
    case "applied":
      return { text: "Đã áp dụng", tone: "ok" };
    default:
      return { text: "Đã huỷ", tone: "warn" };
  }
}

export function ChannelChip({ channel }: { channel: AiChannel }) {
  return (
    <span className={styles.chip}>
      <span className={styles.dot} />
      {CHANNEL_LABEL[channel]}
    </span>
  );
}

interface ChannelButtonProps {
  channel: AiChannel;
  label: string;
  onRun: (channel: AiChannel) => void;
  disabled?: boolean;
  /** Channels that cannot run this task, with the reason shown in the menu. */
  unavailable?: Partial<Record<AiChannel, string>>;
  testId: string;
}

/**
 * "▶ <label> (<kênh>)" with ▾ beside it: the main part runs on the step's channel from Settings,
 * the menu runs this once on another channel without changing the setting.
 */
export function ChannelButton({ channel, label, onRun, disabled, unavailable = {}, testId }: ChannelButtonProps) {
  const [menu, setMenu] = useState(false);
  const wrap = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!menu) return;
    function close(e: MouseEvent) {
      if (!wrap.current?.contains(e.target as Node)) setMenu(false);
    }
    function esc(e: KeyboardEvent) {
      if (e.key === "Escape") {
        e.stopPropagation();
        setMenu(false);
      }
    }
    document.addEventListener("mousedown", close);
    document.addEventListener("keydown", esc, true);
    return () => {
      document.removeEventListener("mousedown", close);
      document.removeEventListener("keydown", esc, true);
    };
  }, [menu]);
  const main = unavailable[channel] ? CHANNELS.find((c) => !unavailable[c]) ?? channel : channel;
  return (
    <div className={styles.split} ref={wrap}>
      <Button variant="run" onClick={() => onRun(main)} disabled={disabled} data-testid={testId}>
        ▶ {label} ({CHANNEL_LABEL[main]})
      </Button>
      <Button
        variant="run"
        onClick={() => setMenu((m) => !m)}
        disabled={disabled}
        aria-haspopup="menu"
        aria-expanded={menu}
        aria-label="Chạy bằng kênh khác"
        data-testid={`${testId}-more`}
      >
        ▾
      </Button>
      {menu && (
        <div className={styles.menu} role="menu">
          <span className={styles.menuHint}>Chỉ cho lần chạy này, không đổi Cài đặt</span>
          {CHANNELS.map((c) => (
            <button
              key={c}
              type="button"
              role="menuitem"
              className={styles.menuItem}
              disabled={!!unavailable[c]}
              title={unavailable[c]}
              onClick={() => {
                setMenu(false);
                onRun(c);
              }}
              data-testid={`${testId}-${c}`}
            >
              <ChannelChip channel={c} />
              {unavailable[c] ?? CHANNEL_HINT[c]}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

interface AiRunDialogProps {
  open: boolean;
  onClose: () => void;
  /** Names the item and the task ("Viết shot cho VP-001"); also names the extension queue item. */
  title: string;
  /**
   * The path of the step request this action sends (`/v1/projects/p1/storyboard/units/VP-001/generate`):
   * the key of its run, known before the note is written. It must be the path `describe` returns.
   */
  runPath: string;
  /** Builds the step's own AI request with the note, without sending it (`describeAiRequest`). */
  describe: (note: string) => Promise<ExternalRequest>;
  /** The action takes no note: the form shows only the impact. */
  noNote?: boolean;
  /** What applying changes for the rest of the step, shown under the note. */
  impact?: ReactNode;
  /** Shows what applying would store; without it the preview is shown as JSON. */
  renderPreview?: (preview: unknown) => ReactNode;
  /** The screen loads what an applied run stored; `state` is what the step's request answered. */
  onApplied: (state: unknown) => void | Promise<void>;
  testId: string;
}

/** The preview as indented JSON, for an action with no preview of its own. */
function JsonPreview({ preview }: { preview: unknown }) {
  return (
    <pre className={`${styles.prompt} ${styles.preview}`} data-testid="ai-run-preview">
      {JSON.stringify(preview, null, 2)}
    </pre>
  );
}

/**
 * One AI action of a step in one modal (docs/ux-ui-design-rules.md, rule 4; ADR-0052). Before a
 * run: the note and the impact, and "▶ Chạy AI" on the step's channel with ▾ for another. During
 * and after it: where it stands, the prompt (to copy), the reply (pasted here on the manual channel,
 * or filled in by the system's model or the extension), what applying would store, and "Áp dụng
 * kết quả". Closing the dialog does not stop the run; the pill next to the action's button opens it
 * again. Needs an `AiRunsProvider` above it.
 */
export function AiRunDialog({
  open,
  onClose,
  title,
  runPath,
  describe,
  noNote = false,
  impact,
  renderPreview,
  onApplied,
  testId,
}: AiRunDialogProps) {
  const ai = useAiRuns();
  const current = ai.runs[runPath];
  const [composing, setComposing] = useState(true);
  const [note, setNote] = useState("");
  const [reply, setReply] = useState("");
  const [tab, setTab] = useState<"prompt" | "reply">("reply");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [lastChannel, setLastChannel] = useState<AiChannel | null>(null);

  // Opening shows the open run if there is one, else the note form.
  useEffect(() => {
    if (!open) return;
    const showRun = isOpen(current);
    setComposing(!showRun);
    setNote("");
    setError(null);
    setReply(current?.reply ?? "");
    setTab(showRun && current.channel === "manual" && current.status === "waiting" ? "prompt" : "reply");
    // Only on opening; later changes of the run are followed below.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  // A reply that arrives from the model or the extension fills the reply box.
  useEffect(() => {
    if (current?.reply && current.status !== "waiting") setReply(current.reply);
  }, [current?.reply, current?.status]);

  async function run(channel: AiChannel) {
    setBusy(true);
    setError(null);
    setLastChannel(channel);
    try {
      const r = await ai.start(await describe(note.trim()), channel, title);
      setComposing(false);
      setReply("");
      setTab(r.channel === "manual" ? "prompt" : "reply");
    } catch (err) {
      setComposing(false);
      setError(reasonOf(err));
    } finally {
      setBusy(false);
    }
  }

  async function read() {
    if (!current) return;
    setBusy(true);
    setError(null);
    try {
      ai.put(await replyAiRun(ai.service, ai.projectId, current.id, reply));
    } catch (err) {
      setError(reasonOf(err));
    } finally {
      setBusy(false);
    }
  }

  async function apply() {
    if (!current) return;
    setBusy(true);
    setError(null);
    try {
      const out = await applyAiRun(ai.service, ai.projectId, current.id);
      ai.put(out.run);
      await onApplied(out.state);
      onClose();
    } catch (err) {
      setError(reasonOf(err));
      await ai.refresh().catch(() => undefined);
    } finally {
      setBusy(false);
    }
  }

  async function drop() {
    if (!current) return;
    setBusy(true);
    try {
      ai.put(await cancelAiRun(ai.service, ai.projectId, current.id));
      setComposing(true);
    } catch (err) {
      setError(reasonOf(err));
    } finally {
      setBusy(false);
    }
  }

  const showRun = !composing && !!current && current.status !== "cancelled";
  const queueItemId = current ? ai.queueItems[current.id] : undefined;
  const queueItem = queueItemId ? ai.queue.items[queueItemId] : undefined;
  const st = current ? statusOf(current) : null;
  const canEditReply = !!current && ["waiting", "unreadable", "failed", "ready"].includes(current.status) && current.status !== "running";
  const replyChanged = !!current && reply.trim() !== "" && reply.trim() !== current.reply.trim();
  const others = CHANNELS.filter((c) => c !== (current?.channel ?? lastChannel));

  const footer = showRun ? (
    <>
      <Button variant="ghost" onClick={onClose} data-testid={`${testId}-done`}>
        Đóng
      </Button>
      {current.status !== "applied" && (
        <Button variant="ghost" onClick={() => void drop()} disabled={busy} data-testid={`${testId}-drop`}>
          Bỏ lượt này
        </Button>
      )}
      <Button
        variant="ghost"
        onClick={() => {
          setComposing(true);
          setError(null);
        }}
        disabled={busy}
        data-testid={`${testId}-again`}
      >
        Chạy lại…
      </Button>
      {(current.status === "waiting" || current.status === "unreadable" || current.status === "failed" || replyChanged) &&
      current.status !== "running" ? (
        <Button
          variant="primary"
          onClick={() => void read()}
          disabled={busy || reply.trim() === ""}
          data-testid={`${testId}-read`}
        >
          {busy ? "Đang đọc…" : "Đọc kết quả"}
        </Button>
      ) : (
        <Button
          variant="primary"
          onClick={() => void apply()}
          disabled={busy || current.status !== "ready"}
          data-testid={`${testId}-apply`}
        >
          {busy ? "Đang áp dụng…" : "Áp dụng kết quả"}
        </Button>
      )}
    </>
  ) : (
    <>
      <Button variant="ghost" onClick={onClose} data-testid={`${testId}-cancel`}>
        Huỷ
      </Button>
      <ChannelButton channel={ai.defaultChannel} label="Chạy AI" onRun={(c) => void run(c)} disabled={busy} testId={`${testId}-run`} />
    </>
  );

  return (
    <Dialog open={open} onClose={onClose} title={title} size="lg" hasInput testId={testId} footer={footer}>
      {!showRun && (
        <>
          {!noNote && (
            <FormField label="Ghi chú cho AI (không bắt buộc)">
              <TextArea
                value={note}
                onChange={(e) => setNote(e.target.value)}
                rows={4}
                placeholder="Viết điều chưa ổn và muốn đổi thế nào"
                data-testid={`${testId}-note`}
              />
            </FormField>
          )}
          {impact}
          <p className={styles.previewSub}>
            Kết quả chỉ được lưu khi bạn bấm “Áp dụng kết quả” sau khi xem. Đóng hộp thoại không huỷ lượt chạy.
          </p>
          {error && (
            <div className={styles.alert} role="alert" data-testid={`${testId}-error`}>
              <span>{error}</span>
            </div>
          )}
        </>
      )}
      {showRun && st && (
        <div className={styles.section}>
          <div className={`${styles.status} ${styles[st.tone === "run" ? "running" : st.tone]}`} role="status" data-testid={`${testId}-status`}>
            <span className={styles.dot} />
            {st.text}
            <ChannelChip channel={current.channel} />
            {queueItem && queueItem.state === "running" && queueItem.stage && (
              <span className={styles.previewSub}>
                {queueItem.provider} · {queueItem.stage}
              </span>
            )}
          </div>
          {(error || current.error || queueItem?.error) && (
            <div className={styles.alert} role="alert" data-testid={`${testId}-error`}>
              <span>{error || current.error || queueItem?.error}</span>
              {(current.status === "failed" || (queueItem?.state === "failed" && current.status === "waiting") || error) && (
                <div className={styles.row}>
                  {others.map((c) => (
                    <Button key={c} variant="ghost" onClick={() => void run(c)} disabled={busy} data-testid={`${testId}-switch-${c}`}>
                      Chuyển sang {CHANNEL_LABEL[c]}
                    </Button>
                  ))}
                </div>
              )}
            </div>
          )}
          <div className={styles.tabs} role="tablist">
            <button type="button" role="tab" className={styles.tab} aria-selected={tab === "prompt"} onClick={() => setTab("prompt")} data-testid={`${testId}-tab-prompt`}>
              Prompt
            </button>
            <button type="button" role="tab" className={styles.tab} aria-selected={tab === "reply"} onClick={() => setTab("reply")} data-testid={`${testId}-tab-reply`}>
              Kết quả
            </button>
          </div>
          {tab === "prompt" ? (
            <div className={styles.section}>
              <CopyBox text={current.prompt} what="prompt" copyTestId={`${testId}-copy`}>
                <pre className={styles.prompt} data-testid={`${testId}-prompt`}>
                  {current.prompt}
                </pre>
              </CopyBox>
              {current.channel === "manual" && (
                <p className={styles.previewSub}>
                  Chép prompt sang AI bạn dùng, rồi dán toàn bộ câu trả lời vào tab Kết quả.
                </p>
              )}
            </div>
          ) : (
            <div className={styles.section}>
              {current.status === "ready" &&
                !replyChanged &&
                (renderPreview ? renderPreview(current.preview) : <JsonPreview preview={current.preview} />)}
              <FormField label={current.status === "ready" ? "Câu trả lời của AI (sửa được, rồi bấm Đọc kết quả)" : "Câu trả lời của AI"}>
                <TextArea
                  className={styles.reply}
                  value={reply}
                  onChange={(e) => setReply(e.target.value)}
                  readOnly={!canEditReply}
                  placeholder={
                    current.status === "running" || (current.status === "waiting" && current.channel === "extension")
                      ? "Câu trả lời sẽ tự hiện ở đây…"
                      : "Dán toàn bộ câu trả lời vào đây"
                  }
                  aria-label="Câu trả lời của AI"
                  data-testid={`${testId}-reply`}
                />
              </FormField>
            </div>
          )}
        </div>
      )}
    </Dialog>
  );
}

interface AiRunPillProps {
  /** The same `runPath` as the action's `AiRunDialog`. */
  runPath: string;
  onOpen: () => void;
  testId: string;
}

/** Where the open run of an action stands, next to its button; a click opens its dialog again. */
export function AiRunPill({ runPath, onOpen, testId }: AiRunPillProps) {
  const { runs } = useAiRuns();
  const run = runs[runPath];
  if (!isOpen(run)) return null;
  const st = statusOf(run);
  return (
    <button
      type="button"
      className={`${styles.pill} ${styles[st.tone === "run" ? "running" : st.tone]}`}
      onClick={onOpen}
      title="Mở lượt chạy AI"
      data-testid={testId}
    >
      <span className={styles.dot} />
      {st.text} · {CHANNEL_LABEL[run.channel]}
    </button>
  );
}
