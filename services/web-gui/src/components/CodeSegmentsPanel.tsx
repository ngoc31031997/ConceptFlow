import { useCallback, useEffect, useRef, useState } from "react";
import {
  getCodeSegmentPrompt,
  getCodeSegments,
  putCodeChunkShots,
  putCodeSegment,
  startAuthoringChain,
  type CodeRunOptions,
  type CodeSegment,
  type CodeSegmentsView,
} from "../api/client";
import { Button, Card, TextArea, TextInput } from "./ui";
import { useAuthoringRun, useAuthoringRunDispatch } from "../context/AuthoringRunContext";
import { usePresence } from "../hooks/usePresence";
import glass from "../styles/glass.module.css";
import styles from "./CodeSegmentsPanel.module.css";

const STATUS_LABEL: Record<CodeSegment["status"], string> = {
  pending: "Chưa chạy",
  running: "Đang chạy",
  done: "Xong",
  failed: "Lỗi",
};

const SOURCE_LABEL: Record<CodeSegment["source"], string> = {
  "": "",
  ai: "AI",
  external: "AI ngoài",
  manual: "Sửa tay",
  storyboard: "Từ storyboard",
};

const MAX_CHUNK_SHOTS = 10;

function segmentTitle(s: CodeSegment, isRemotion: boolean): string {
  if (s.kind === "frame") return isRemotion ? "Khung (LAYOUT)" : "Khung (cast)";
  const first = s.shots[0];
  const last = s.shots[s.shots.length - 1];
  return first === last ? `Shot ${first}` : `Shot ${first}–${last}`;
}

/** The segment's code as one text: what a paste of it would look like. */
function segmentText(s: CodeSegment): string {
  if (s.content?.code !== undefined) return s.content.code;
  const shots = s.content?.shots ?? {};
  return s.shots.map((id) => shots[id] ?? "").filter(Boolean).join("\n\n");
}

/** The shots a failed chunk did write and keeps for its next run. */
function savedShots(s: CodeSegment): string[] {
  if (s.status !== "failed") return [];
  const shots = s.content?.shots ?? {};
  return s.shots.filter((id) => Boolean(shots[id]));
}

function runLabel(s: CodeSegment, saved: string[]): string {
  if (s.status === "done") return "Chạy lại đoạn này";
  const failedShots = s.failed_shots ?? [];
  if (saved.length > 0 && failedShots.length > 0) return `Chạy lại shot ${failedShots.join(", ")}`;
  return "Chạy đoạn này";
}

function seconds(ms: number): string {
  if (ms <= 0) return "";
  const s = Math.round(ms / 1000);
  return s < 60 ? `${s} giây` : `${Math.floor(s / 60)} phút ${s % 60} giây`;
}

function errorText(e: unknown, fallback: string): string {
  return e instanceof Error && e.message ? e.message : fallback;
}

/**
 * How many shots one code segment holds. It sits before the run
 * button (docs/ux-ui-design-rules.md §1): the next run is cut with it.
 */
export function CodeChunkShotsField({ projectId, onSaved }: { projectId: string; onSaved: () => void }) {
  const [view, setView] = useState<CodeSegmentsView | null>(null);
  const [value, setValue] = useState("");
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const run = useAuthoringRun();
  const running = run.running && run.steps.includes("code");

  useEffect(() => {
    getCodeSegments(projectId)
      .then((v) => {
        setView(v);
        setValue(String(v.chunk_shots));
      })
      .catch(() => setView(null));
  }, [projectId]);

  if (!view) return null;
  const n = Number(value);
  const valid = Number.isInteger(n) && n >= 1 && n <= MAX_CHUNK_SHOTS;
  const changed = valid && n !== view.chunk_shots;

  async function save() {
    if (!view || !changed) return;
    const done = view.segments.filter((s) => s.kind === "shots" && s.status === "done").length;
    if (done > 0 && !window.confirm(
      `Đổi từ ${view.chunk_shots} sang ${n} shot mỗi đoạn sẽ chia lại các đoạn: ${done} đoạn đã xong không còn khớp cách chia mới sẽ phải chạy lại. Tiếp tục?`,
    )) return;
    setSaving(true);
    setMessage(null);
    try {
      await putCodeChunkShots(projectId, n);
      setView({ ...view, chunk_shots: n });
      onSaved();
    } catch (e) {
      setMessage(errorText(e, "Không lưu được số shot mỗi đoạn."));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className={styles.chunkField} data-testid="code-chunk-shots">
      <label htmlFor="code-chunk-shots-input" className={styles.chunkLabel}>
        Số shot mỗi đoạn code
      </label>
      <TextInput
        id="code-chunk-shots-input"
        type="number"
        min={1}
        max={MAX_CHUNK_SHOTS}
        value={value}
        onChange={(e) => setValue(e.target.value)}
        className={styles.chunkInput}
        disabled={running || saving}
        data-testid="code-chunk-shots-input"
      />
      <Button variant="ghost" onClick={save} disabled={!changed || running || saving} data-testid="code-chunk-shots-save">
        {saving ? "Đang lưu…" : "Lưu"}
      </Button>
      <span className={styles.chunkHint}>
        {valid ? "Đoạn nhỏ thì mỗi lượt AI ngắn hơn, ít bị ngắt hơn." : `Nhập từ 1 đến ${MAX_CHUNK_SHOTS}.`}
      </span>
      {message && <span className={styles.error} role="status">{message}</span>}
    </div>
  );
}

/** A textarea to paste an outside AI's reply, or edit a segment by hand. */
function SegmentEditor({
  initial,
  source,
  busy,
  onSave,
  onClose,
  testId,
}: {
  initial: string;
  source: "external" | "manual";
  busy: boolean;
  onSave: (text: string) => void;
  onClose: () => void;
  testId: string;
}) {
  const [text, setText] = useState(initial);
  return (
    <div className={styles.editor}>
      <TextArea
        value={text}
        onChange={(e) => setText(e.target.value)}
        rows={12}
        className={styles.code}
        placeholder={source === "external" ? "Dán nguyên câu trả lời của AI ngoài vào đây." : undefined}
        data-testid={`${testId}-text`}
      />
      <div className={styles.actions}>
        <Button onClick={() => onSave(text)} disabled={busy || text.trim() === ""} data-testid={`${testId}-save`}>
          {busy ? "Đang kiểm…" : "Kiểm và lưu"}
        </Button>
        <Button variant="ghost" onClick={onClose} disabled={busy}>
          Huỷ
        </Button>
      </div>
    </div>
  );
}

function Reveal({ open, children }: { open: boolean; children: React.ReactNode }) {
  const { mounted, closing } = usePresence(open);
  if (!mounted) return null;
  return <div className={closing ? glass.revealOut : glass.reveal}>{children}</div>;
}

/**
 * The code step's segments: the frame (LAYOUT/cast) and every
 * group of shots, each with its state, why it failed, how long it took, and
 * its own actions — re-run it with AI, copy its prompt for an outside AI,
 * paste the outside AI's reply, or edit it by hand. Runs go through the
 * server chain, so they carry on if this page closes; the list follows the
 * server while one is going.
 */
export function CodeSegmentsPanel({
  projectId,
  isRemotion,
  reloadKey,
}: {
  projectId: string;
  isRemotion: boolean;
  reloadKey: number;
}) {
  const [view, setView] = useState<CodeSegmentsView | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [editing, setEditing] = useState<{ key: string; source: "external" | "manual" } | null>(null);
  const [openKey, setOpenKey] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const [starting, setStarting] = useState(false);
  const run = useAuthoringRun();
  const dispatchRun = useAuthoringRunDispatch();
  const codeRunning = run.running && run.steps.includes("code");

  const reload = useCallback(async () => {
    try {
      setView(await getCodeSegments(projectId));
      setMessage(null);
    } catch (e) {
      setMessage(errorText(e, "Không tải được danh sách đoạn code."));
    }
  }, [projectId]);

  useEffect(() => {
    void reload();
  }, [reload, reloadKey]);

  // Follow the server while the code step runs, then read the final states once.
  const serverRunning = codeRunning || (view?.running ?? false);
  useEffect(() => {
    if (!serverRunning) return;
    const id = window.setInterval(() => void reload(), 1500);
    return () => window.clearInterval(id);
  }, [serverRunning, reload]);
  const wasRunning = useRef(false);
  useEffect(() => {
    if (wasRunning.current && !serverRunning) void reload();
    wasRunning.current = serverRunning;
  }, [serverRunning, reload]);

  async function start(options?: CodeRunOptions) {
    setStarting(true);
    setMessage(null);
    setNotice(null);
    try {
      await startAuthoringChain(projectId, ["code"], options);
      dispatchRun({ type: "START", steps: ["code"] });
      await reload();
    } catch (e) {
      setMessage(errorText(e, "Không bắt đầu chạy được."));
    } finally {
      setStarting(false);
    }
  }

  function regenerateAll() {
    if (!window.confirm("Bỏ mọi đoạn code đã có và để AI sinh lại toàn bộ? Mọi đoạn sẽ tốn token lại từ đầu.")) return;
    void start({ fresh: true });
  }

  async function copyPrompt(s: CodeSegment) {
    setBusy(s.key);
    setMessage(null);
    setNotice(null);
    try {
      const p = await getCodeSegmentPrompt(projectId, s.key);
      await navigator.clipboard.writeText(`${p.system}\n\n---\n\n${p.user}`);
      setNotice(`Đã sao chép prompt của ${segmentTitle(s, isRemotion)} — dán vào ChatGPT, Claude hoặc Gemini, rồi dán câu trả lời về đây.`);
    } catch (e) {
      setMessage(errorText(e, "Không sao chép được prompt của đoạn này."));
    } finally {
      setBusy(null);
    }
  }

  async function save(s: CodeSegment, text: string, source: "external" | "manual") {
    setBusy(s.key);
    setMessage(null);
    setNotice(null);
    try {
      await putCodeSegment(projectId, s.key, text, source);
      setEditing(null);
      setNotice(`Đã lưu ${segmentTitle(s, isRemotion)}.`);
      await reload();
    } catch (e) {
      setMessage(errorText(e, "Không lưu được đoạn này."));
    } finally {
      setBusy(null);
    }
  }

  if (!view) {
    return message ? (
      <Card title="Các đoạn code" data-testid="code-segments">
        <p className={styles.error} role="status">{message}</p>
      </Card>
    ) : null;
  }
  if (view.segments.length === 0) return null;

  const done = view.segments.filter((s) => s.status === "done").length;
  const failed = view.segments.filter((s) => s.status === "failed").length;
  const missing = view.segments.length - done;
  const locked = codeRunning || view.running || starting;

  return (
    <Card
      title="Các đoạn code"
      hint="Mỗi đoạn được lưu ngay khi xong. Một đoạn lỗi không làm mất các đoạn khác: chạy lại đúng đoạn đó, hoặc dán kết quả của AI ngoài. Khi đủ đoạn, hệ thống tự ghép, kiểm và sửa thành code của cả video."
      data-testid="code-segments"
    >
      <p className={failed > 0 ? styles.waiting : missing === 0 ? styles.ok : styles.summary} data-testid="code-segments-summary">
        {`${done}/${view.segments.length} đoạn xong`}
        {failed > 0 && ` · ${failed} lỗi`}
        {missing - failed > 0 && ` · ${missing - failed} chưa chạy`}
      </p>
      {message && <p className={styles.error} role="status">{message}</p>}
      {notice && <p className={styles.notice} role="status">{notice}</p>}
      <ul className={styles.list} data-testid="code-segments-list">
        {view.segments.map((s) => {
          const title = segmentTitle(s, isRemotion);
          const fromStoryboard = s.source === "storyboard";
          const hasCode = s.status === "done" && segmentText(s) !== "";
          const isEditing = editing?.key === s.key;
          const rowBusy = busy === s.key;
          const saved = savedShots(s);
          const namesShots = (s.failed_shots ?? []).length > 0;
          return (
            <li key={s.key} className={styles.row} data-testid={`code-segment-${s.key}`}>
              <div className={styles.head}>
                <span className={styles.title}>{title}</span>
                <span className={`${styles.badge} ${styles[s.status]}`} data-testid={`code-segment-${s.key}-status`}>
                  {STATUS_LABEL[s.status]}
                </span>
                {s.source && <span className={styles.source}>{SOURCE_LABEL[s.source]}</span>}
                {s.duration_ms > 0 && <span className={styles.meta}>{seconds(s.duration_ms)}</span>}
              </div>
              {s.status === "failed" && (
                <p className={styles.error} data-testid={`code-segment-${s.key}-error`}>
                  {s.error_kind && !namesShots ? `${s.error_kind}: ` : ""}
                  {s.error_message || "lỗi không rõ"}
                </p>
              )}
              {saved.length > 0 && (
                <p className={styles.saved} data-testid={`code-segment-${s.key}-saved`}>
                  Đã lưu: shot {saved.join(", ")}
                </p>
              )}
              {!fromStoryboard && (
                <div className={styles.actions}>
                  <Button
                    variant={s.status === "failed" ? "primary" : "ghost"}
                    onClick={() => void start({ segment: s.key })}
                    disabled={locked || rowBusy}
                    data-testid={`code-segment-${s.key}-run`}
                  >
                    {runLabel(s, saved)}
                  </Button>
                  <Button variant="ghost" onClick={() => void copyPrompt(s)} disabled={rowBusy} data-testid={`code-segment-${s.key}-copy`}>
                    Sao chép prompt
                  </Button>
                  <Button
                    variant="ghost"
                    onClick={() => setEditing(isEditing && editing?.source === "external" ? null : { key: s.key, source: "external" })}
                    disabled={locked}
                    data-testid={`code-segment-${s.key}-paste`}
                  >
                    Dán kết quả AI ngoài
                  </Button>
                  {hasCode && (
                    <>
                      <Button
                        variant="ghost"
                        onClick={() => setEditing(isEditing && editing?.source === "manual" ? null : { key: s.key, source: "manual" })}
                        disabled={locked}
                        data-testid={`code-segment-${s.key}-edit`}
                      >
                        Sửa tay
                      </Button>
                      <Button variant="ghost" onClick={() => setOpenKey(openKey === s.key ? null : s.key)} data-testid={`code-segment-${s.key}-view`}>
                        {openKey === s.key ? "Ẩn code" : "Xem code"}
                      </Button>
                    </>
                  )}
                </div>
              )}
              <Reveal open={isEditing}>
                {editing && isEditing && (
                  <SegmentEditor
                    key={`${s.key}-${editing.source}`}
                    initial={editing.source === "manual" ? segmentText(s) : ""}
                    source={editing.source}
                    busy={rowBusy}
                    onSave={(text) => void save(s, text, editing.source)}
                    onClose={() => setEditing(null)}
                    testId={`code-segment-${s.key}-editor`}
                  />
                )}
              </Reveal>
              <Reveal open={openKey === s.key && hasCode && !isEditing}>
                <pre className={styles.code} data-testid={`code-segment-${s.key}-code`}>
                  {segmentText(s)}
                </pre>
              </Reveal>
            </li>
          );
        })}
      </ul>
      <div className={styles.footer}>
        {/* With nothing missing (every segment pasted, say) the same run only
            merges, checks and repairs. */}
        <Button onClick={() => void start()} disabled={locked} data-testid="code-segments-run-missing">
          {starting ? "Đang bắt đầu…" : missing > 0 ? `Chạy các đoạn còn thiếu (${missing})` : "Ghép và kiểm lại code"}
        </Button>
        <Button variant="dangerGhost" onClick={regenerateAll} disabled={locked} data-testid="code-segments-regenerate">
          Sinh lại toàn bộ
        </Button>
        <span className={styles.meta}>Chạy bằng AI xong thì code của cả video được ghép lại từ các đoạn, thay cho bản đang có ở ô bên dưới.</span>
      </div>
    </Card>
  );
}
