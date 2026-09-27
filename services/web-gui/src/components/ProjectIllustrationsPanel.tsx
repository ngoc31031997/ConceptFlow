import { useCallback, useEffect, useRef, useState } from "react";
import {
  deleteProjectIllustrationDrawing,
  drawProjectIllustration,
  listIllustrationFolders,
  listProjectIllustrations,
  planProjectIllustrations,
  setIllustrationStatus,
  skipProjectIllustration,
  startAuthoringChain,
  type IllustrationFolder,
  type ProjectIllustration,
} from "../api/client";
import { Button, Card } from "./ui";
import { IllustrationTile } from "./IllustrationTile";
import { IllustrationEditor } from "./IllustrationEditor";
import { OperationProgressCard } from "./OperationProgressCard";
import { useAuthoringRun } from "../context/AuthoringRunContext";
import { drawProgressText, isProjectIllustrationReady as isReady } from "../utils/illustrationLabels";
import glass from "../styles/glass.module.css";
import tileStyles from "./IllustrationTile.module.css";
import styles from "./ProjectIllustrationsPanel.module.css";

const STATE_LABEL: Record<ProjectIllustration["state"], string> = {
  planned: "Chưa vẽ",
  drawing: "AI đang vẽ…",
  drawn: "Đã vẽ",
  reused: "Dùng lại",
  failed: "Vẽ lỗi",
  skipped: "Bỏ qua",
};

/** Tóm tắt cho trang: bao nhiêu hình, bao nhiêu đã sẵn sàng, đã lập danh sách chưa. */
export interface IllustrationsSummary {
  total: number;
  ready: number;
}

/**
 * CR-044/045 — the drawings this Remotion video needs, on the illustrations
 * step. Running the step (here or from the chain) plans the list from the
 * storyboard and draws what is missing, several at once on the server; each
 * drawing in flight shows its own progress. The code step waits until every
 * drawing is approved or skipped.
 */
export function ProjectIllustrationsPanel({
  projectId,
  onSummary,
}: {
  projectId: string;
  onSummary?: (s: IllustrationsSummary) => void;
}) {
  const [rows, setRows] = useState<ProjectIllustration[] | null>(null);
  const [folders, setFolders] = useState<IllustrationFolder[]>([]);
  const [busy, setBusy] = useState<Record<string, boolean>>({});
  const [planning, setPlanning] = useState(false);
  const [starting, setStarting] = useState(false);
  const [openId, setOpenId] = useState<string | null>(null);
  const [fixNote, setFixNote] = useState<{ id: string; text: string; n: number } | null>(null);
  const [bust, setBust] = useState<Record<string, number>>({});
  const [message, setMessage] = useState<string | null>(null);
  // A chain (this step, or the whole pipeline) runs on the server and fills this list.
  const run = useAuthoringRun();
  const chainDrawing = run.running && run.steps.includes("illustrations");

  const reload = useCallback(async () => {
    try {
      setRows((await listProjectIllustrations(projectId)).illustrations);
    } catch (e) {
      setMessage(e instanceof Error ? e.message : "Không tải được danh sách hình của video.");
    }
  }, [projectId]);

  useEffect(() => {
    void reload();
    listIllustrationFolders().then(setFolders).catch(() => setFolders([]));
  }, [reload]);

  const summaryRef = useRef(onSummary);
  summaryRef.current = onSummary;
  useEffect(() => {
    if (rows) summaryRef.current?.({ total: rows.length, ready: rows.filter(isReady).length });
  }, [rows]);

  // Follow the server while it draws: each tile's own progress bar moves.
  const drawingOnServer = rows?.some((r) => r.state === "drawing") ?? false;
  useEffect(() => {
    if (!drawingOnServer && !chainDrawing) return;
    const id = window.setInterval(() => void reload(), 1500);
    return () => window.clearInterval(id);
  }, [drawingOnServer, chainDrawing, reload]);
  // One last read when a chain ends, so the final states show without waiting.
  const wasDrawing = useRef(false);
  useEffect(() => {
    if (wasDrawing.current && !chainDrawing) void reload();
    wasDrawing.current = chainDrawing;
  }, [chainDrawing, reload]);

  async function act(id: string, action: () => Promise<unknown>) {
    setBusy((b) => ({ ...b, [id]: true }));
    setMessage(null);
    try {
      await action();
    } catch (e) {
      setMessage(e instanceof Error && e.message ? e.message : "Thao tác không thành công.");
    } finally {
      setBusy((b) => ({ ...b, [id]: false }));
      await reload();
    }
  }

  async function plan() {
    if (rows && rows.length > 0 && !window.confirm("Lập lại danh sách từ storyboard? Danh sách hiện tại sẽ bị thay (hình đã vẽ vẫn còn trong thư viện).")) return;
    setPlanning(true);
    setMessage(null);
    try {
      setRows((await planProjectIllustrations(projectId)).illustrations);
    } catch (e) {
      setMessage(e instanceof Error && e.message ? e.message : "Không lập được danh sách hình.");
    } finally {
      setPlanning(false);
    }
  }

  // The server draws them, several at once, whether or not this page stays open.
  async function drawAllMissing() {
    setStarting(true);
    setMessage(null);
    try {
      await startAuthoringChain(projectId, ["illustrations"]);
      await reload();
    } catch (e) {
      setMessage(e instanceof Error && e.message ? e.message : "Không bắt đầu vẽ được.");
    } finally {
      setStarting(false);
    }
  }

  function deleteDrawing(r: ProjectIllustration) {
    if (!window.confirm(`Xoá hình "${r.illustration?.title ?? r.name}" khỏi thư viện? Hình này sẽ được bỏ qua cho video; bấm "Dùng lại" để AI vẽ lại từ đầu.`)) return;
    if (openId === r.id) setOpenId(null);
    void act(r.id, () => deleteProjectIllustrationDrawing(projectId, r.id));
  }

  if (rows === null) return null;
  const ready = rows.filter(isReady).length;
  const missing = rows.filter((r) => r.state === "planned" || r.state === "failed").length;
  const opened = rows.find((r) => r.id === openId)?.illustration ?? null;
  const locked = planning || starting || chainDrawing;

  return (
    <Card
      title="Hình minh hoạ của video"
      hint="Lập từ storyboard: hình thư viện đã có thì dùng lại, còn thiếu thì AI vẽ theo luật style. Bước Code chỉ chạy khi mọi hình đã duyệt hoặc bỏ qua."
      headerAction={
        <div className={styles.headerActions}>
          {missing > 0 && (
            <Button variant="ghost" onClick={drawAllMissing} disabled={locked} data-testid="pi-draw-all">
              {starting ? "Đang bắt đầu…" : `Vẽ ${missing} hình còn thiếu`}
            </Button>
          )}
          <Button variant="ghost" onClick={plan} disabled={locked} data-testid="pi-plan">
            {planning ? "Đang đọc storyboard…" : rows.length ? "Lập lại danh sách" : "Lập danh sách từ storyboard"}
          </Button>
        </div>
      }
      data-testid="project-illustrations"
    >
      <p className={rows.length > 0 && ready === rows.length ? styles.ok : styles.waiting} data-testid="pi-summary">
        {rows.length === 0
          ? chainDrawing
            ? "AI đang lập danh sách hình từ storyboard…"
            : "Chưa có danh sách — chạy bước Hình minh hoạ bằng AI, hoặc bấm “Lập danh sách từ storyboard”."
          : ready === rows.length
            ? `Đủ ${rows.length} hình — bước Code chạy được.`
            : `${ready}/${rows.length} hình sẵn sàng — bước Code đang chờ bạn duyệt hoặc bỏ qua các hình còn lại.`}
      </p>
      {message && <p className={styles.error} role="status">{message}</p>}
      {openId && (
        <div className={glass.mtSm}>
          <IllustrationEditor
            illustration={opened}
            folders={folders}
            onSaved={(saved) => {
              setBust((b) => ({ ...b, [saved.id]: (b[saved.id] ?? 0) + 1 }));
              void reload();
            }}
            onDeleted={() => {
              setOpenId(null);
              void reload();
            }}
            onClose={() => setOpenId(null)}
            redrawNote={fixNote && fixNote.id === openId ? fixNote : undefined}
          />
        </div>
      )}
      <ul className={`${styles.grid} ${glass.mtSm}`} data-testid="pi-grid">
        {rows.map((r) => {
          const ill = r.illustration;
          const skip = (
            <Button variant="ghost" onClick={() => act(r.id, () => skipProjectIllustration(projectId, r.id, r.state !== "skipped"))} disabled={busy[r.id]} data-testid={`pi-skip-${r.name}`}>
              {r.state === "skipped" ? "Dùng lại" : "Bỏ qua"}
            </Button>
          );
          if (ill && r.state !== "skipped") {
            const ownDraft = r.state === "drawn" && !ill.builtin && ill.status === "draft";
            return (
              <IllustrationTile
                key={r.id}
                illustration={ill}
                busy={busy[r.id]}
                bust={bust[ill.id]}
                selected={openId === r.id}
                onOpen={() => setOpenId(r.id)}
                onFixWarnings={(text) => {
                  setOpenId(r.id);
                  setFixNote({ id: r.id, text, n: Date.now() });
                }}
                actions={
                  <>
                    {!ill.builtin && ill.status === "draft" && (
                      <Button onClick={() => act(r.id, () => setIllustrationStatus(ill.id, "approved"))} disabled={busy[r.id]} data-testid={`pi-approve-${r.name}`}>
                        Duyệt
                      </Button>
                    )}
                    {!ill.builtin && (
                      <Button variant="ghost" onClick={() => setOpenId(r.id)} data-testid={`pi-edit-${r.name}`}>
                        Sửa / Vẽ lại
                      </Button>
                    )}
                    {skip}
                    {ownDraft && (
                      <Button variant="dangerGhost" onClick={() => deleteDrawing(r)} disabled={busy[r.id]} data-testid={`pi-delete-${r.name}`}>
                        Xoá khỏi thư viện
                      </Button>
                    )}
                  </>
                }
              />
            );
          }
          const drawing = busy[r.id] || r.state === "drawing";
          return (
            <li key={r.id} className={`${tileStyles.tile} ${r.state === "skipped" ? styles.skipped : ""}`} data-testid={`pi-row-${r.name}`}>
              <div className={styles.placeholder}>
                {drawing
                  ? "AI đang vẽ…"
                  : chainDrawing && (r.state === "planned" || r.state === "failed")
                    ? "Đang chờ tới lượt vẽ"
                    : STATE_LABEL[r.state]}
              </div>
              {drawing && (
                <OperationProgressCard
                  variant="step"
                  subtitle={r.progress ? drawProgressText(r.progress) : "Đang bắt đầu…"}
                  testId={`pi-progress-${r.name}`}
                />
              )}
              <div className={tileStyles.meta}>
                <span className={tileStyles.title}>{r.name}</span>
                <span className={styles.desc}>{r.description}</span>
                {r.shots.length > 0 && <span className={tileStyles.name}>shot {r.shots.join(", ")}</span>}
                {r.error && <span className={styles.error}>{r.error}</span>}
              </div>
              <div className={tileStyles.actions}>
                {(r.state === "planned" || r.state === "failed") && (
                  <Button onClick={() => act(r.id, () => drawProjectIllustration(projectId, r.id))} disabled={busy[r.id] || chainDrawing} data-testid={`pi-draw-${r.name}`}>
                    {r.state === "failed" ? "Vẽ lại" : "Vẽ"}
                  </Button>
                )}
                {!drawing && skip}
              </div>
            </li>
          );
        })}
      </ul>
    </Card>
  );
}
