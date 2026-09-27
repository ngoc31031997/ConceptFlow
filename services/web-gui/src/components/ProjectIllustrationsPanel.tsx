import { useCallback, useEffect, useState } from "react";
import {
  drawProjectIllustration,
  listIllustrationFolders,
  listProjectIllustrations,
  planProjectIllustrations,
  setIllustrationStatus,
  skipProjectIllustration,
  type IllustrationFolder,
  type ProjectIllustration,
} from "../api/client";
import { Button, Card } from "./ui";
import { IllustrationTile } from "./IllustrationTile";
import { IllustrationEditor } from "./IllustrationEditor";
import { isProjectIllustrationReady as isReady } from "../utils/illustrationLabels";
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

/**
 * CR-044 — the drawings this Remotion video needs, on the code tab. The code
 * step plans and draws them itself when it runs, then waits here until every
 * one is approved or skipped; the Creator can also plan and draw ahead.
 */
export function ProjectIllustrationsPanel({ projectId }: { projectId: string }) {
  const [rows, setRows] = useState<ProjectIllustration[] | null>(null);
  const [folders, setFolders] = useState<IllustrationFolder[]>([]);
  const [busy, setBusy] = useState<Record<string, boolean>>({});
  const [planning, setPlanning] = useState(false);
  const [openId, setOpenId] = useState<string | null>(null);
  const [bust, setBust] = useState<Record<string, number>>({});
  const [message, setMessage] = useState<string | null>(null);

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

  // The code step draws on the server: follow it while anything is being drawn.
  const drawingOnServer = rows?.some((r) => r.state === "drawing" && !busy[r.id]) ?? false;
  useEffect(() => {
    if (!drawingOnServer) return;
    const id = window.setInterval(() => void reload(), 4000);
    return () => window.clearInterval(id);
  }, [drawingOnServer, reload]);

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

  async function drawAllMissing() {
    const todo = (rows ?? []).filter((r) => r.state === "planned" || r.state === "failed");
    // Two at a time: each is a model call plus a render, and llm-service rate-limits anyway.
    for (let i = 0; i < todo.length; i += 2) {
      await Promise.all(todo.slice(i, i + 2).map((r) => act(r.id, () => drawProjectIllustration(projectId, r.id))));
    }
  }

  if (rows === null) return null;
  const ready = rows.filter(isReady).length;
  const missing = rows.filter((r) => r.state === "planned" || r.state === "failed").length;
  const opened = rows.find((r) => r.id === openId)?.illustration ?? null;

  return (
    <Card
      title="Hình minh hoạ của video"
      hint="Lập từ storyboard: hình thư viện đã có thì dùng lại, còn thiếu thì AI vẽ theo luật style. Bước Code chỉ chạy khi mọi hình đã duyệt hoặc bỏ qua."
      headerAction={
        <div className={styles.headerActions}>
          {missing > 0 && (
            <Button variant="ghost" onClick={drawAllMissing} disabled={planning} data-testid="pi-draw-all">
              Vẽ {missing} hình còn thiếu
            </Button>
          )}
          <Button variant="ghost" onClick={plan} disabled={planning} data-testid="pi-plan">
            {planning ? "Đang đọc storyboard…" : rows.length ? "Lập lại danh sách" : "Lập danh sách từ storyboard"}
          </Button>
        </div>
      }
      data-testid="project-illustrations"
    >
      <p className={ready === rows.length ? styles.ok : styles.waiting} data-testid="pi-summary">
        {rows.length === 0
          ? "Chưa có danh sách — chạy bước Code sẽ tự lập, hoặc bấm “Lập danh sách từ storyboard” để duyệt trước."
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
            return (
              <IllustrationTile
                key={r.id}
                illustration={ill}
                busy={busy[r.id]}
                bust={bust[ill.id]}
                selected={openId === r.id}
                onOpen={() => setOpenId(r.id)}
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
                  </>
                }
              />
            );
          }
          return (
            <li key={r.id} className={`${tileStyles.tile} ${r.state === "skipped" ? styles.skipped : ""}`} data-testid={`pi-row-${r.name}`}>
              <div className={styles.placeholder}>
                {busy[r.id] || r.state === "drawing" ? "AI đang vẽ… (1–3 phút)" : STATE_LABEL[r.state]}
              </div>
              <div className={tileStyles.meta}>
                <span className={tileStyles.title}>{r.name}</span>
                <span className={styles.desc}>{r.description}</span>
                {r.shots.length > 0 && <span className={tileStyles.name}>shot {r.shots.join(", ")}</span>}
                {r.error && <span className={styles.error}>{r.error}</span>}
              </div>
              <div className={tileStyles.actions}>
                {(r.state === "planned" || r.state === "failed") && (
                  <Button onClick={() => act(r.id, () => drawProjectIllustration(projectId, r.id))} disabled={busy[r.id]} data-testid={`pi-draw-${r.name}`}>
                    {r.state === "failed" ? "Vẽ lại" : "Vẽ"}
                  </Button>
                )}
                {skip}
              </div>
            </li>
          );
        })}
      </ul>
    </Card>
  );
}
