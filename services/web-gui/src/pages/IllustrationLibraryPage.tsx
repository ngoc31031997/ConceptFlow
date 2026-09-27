import { useCallback, useEffect, useMemo, useRef, useState, type ChangeEvent } from "react";
import { AppShell } from "../components/AppShell";
import { Button, Card, CtaRow, Dropdown, FormField, TextArea, TextInput } from "../components/ui";
import { Disclosure } from "../components/Disclosure";
import { IllustrationTile } from "../components/IllustrationTile";
import { IllustrationEditor } from "../components/IllustrationEditor";
import { IllustrationBackupCard } from "../components/IllustrationBackupCard";
import {
  createIllustrationFolder,
  deleteIllustrationFolder,
  drawIllustration,
  getIllustrationStyle,
  listIllustrationFolders,
  listIllustrations,
  rerenderIllustration,
  setIllustrationStatus,
  type Illustration,
  type IllustrationFolder,
  type IllustrationInput,
} from "../api/client";
import { componentNameFromFile, svgToComponent } from "../utils/svgToComponent";
import glass from "../styles/glass.module.css";
import styles from "./IllustrationLibraryPage.module.css";

const ALL = "";
const NEW = "new";

/**
 * CR-044 — thư viện hình minh hoạ. Mỗi hình nằm trong đúng một thư mục; Kỹ sư
 * Remotion chỉ được dùng hình "Có sẵn" hoặc "Đã duyệt". Mỗi ô là một hình: rê
 * chuột để xem nó chuyển động, bấm để mở trình sửa code.
 */
export function IllustrationLibraryPage() {
  const [folders, setFolders] = useState<IllustrationFolder[]>([]);
  const [items, setItems] = useState<Illustration[]>([]);
  const [folder, setFolder] = useState(ALL);
  const [query, setQuery] = useState("");
  const [openId, setOpenId] = useState<string | null>(null);
  // CR-045 — "Nhờ AI sửa các cảnh báo này" on a tile: open that drawing with the redraw note filled in.
  const [fixNote, setFixNote] = useState<{ id: string; text: string; n: number } | null>(null);
  const [busyIds, setBusyIds] = useState<Record<string, boolean>>({});
  const [bust, setBust] = useState<Record<string, number>>({});
  const [status, setStatus] = useState<string | null>(null);
  const [newFolderId, setNewFolderId] = useState("");
  const [newFolderName, setNewFolderName] = useState("");
  const [rules, setRules] = useState("");
  const [exemplarIds, setExemplarIds] = useState<string[]>([]);
  const [draft, setDraft] = useState<Partial<IllustrationInput> | undefined>(undefined);
  const [drawOpen, setDrawOpen] = useState(false);
  const [drawText, setDrawText] = useState("");
  const [drawFolder, setDrawFolder] = useState("");
  const [drawing, setDrawing] = useState(false);
  const fileInput = useRef<HTMLInputElement>(null);

  const reload = useCallback(async () => {
    try {
      const [f, i] = await Promise.all([listIllustrationFolders(), listIllustrations()]);
      setFolders(f);
      setItems(i);
    } catch (e) {
      setStatus(e instanceof Error && e.message ? e.message : "Không tải được thư viện hình.");
    }
  }, []);

  useEffect(() => {
    void reload();
    getIllustrationStyle()
      .then((st) => {
        setRules(st.rules);
        setExemplarIds(st.exemplar_ids);
      })
      .catch(() => setRules(""));
  }, [reload]);

  // 2.1 — viết code: mở trình sửa với khung code mẫu.
  const startCode = () => {
    setDraft(undefined);
    setDrawOpen(false);
    setOpenId(NEW);
  };

  // 2.2 — tải SVG lên: chuyển thành code ngay trong trình duyệt, mở trình sửa để xem trước rồi lưu.
  async function onSvgFile(e: ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    e.target.value = "";
    if (!file) return;
    setStatus(null);
    try {
      const name = componentNameFromFile(file.name);
      const converted = svgToComponent(await file.text(), name);
      setDraft({
        name,
        title: file.name.replace(/\.svg$/i, ""),
        code: converted.code,
        usage: `<${name} /> — ${converted.width}×${converted.height}`,
        folder_id: folder || undefined,
      });
      setDrawOpen(false);
      setOpenId(NEW);
    } catch (err) {
      setStatus(err instanceof Error ? err.message : "Không đọc được file SVG.");
    }
  }

  // 2.3 — AI vẽ theo luật style.
  async function draw() {
    setDrawing(true);
    setStatus(null);
    try {
      const made = await drawIllustration({ description: drawText.trim(), folder_id: drawFolder || folder || folders[0]?.id || "" });
      setItems((list) => [...list, made]);
      setDrawText("");
      setDrawOpen(false);
      setDraft(undefined);
      setOpenId(made.id);
    } catch (err) {
      setStatus(err instanceof Error && err.message ? err.message : "AI chưa vẽ được hình này.");
    } finally {
      setDrawing(false);
    }
  }

  const folderName = useMemo(() => Object.fromEntries(folders.map((f) => [f.id, f.name])), [folders]);
  const counts = useMemo(() => {
    const c: Record<string, number> = {};
    for (const i of items) c[i.folder_id] = (c[i.folder_id] ?? 0) + 1;
    return c;
  }, [items]);

  const q = query.trim().toLowerCase();
  const visible = items.filter(
    (i) =>
      (folder === ALL || i.folder_id === folder) &&
      (q === "" || [i.name, i.title, i.description, ...i.tags].some((s) => s.toLowerCase().includes(q))),
  );
  const pending = items.filter((i) => !i.builtin && i.status === "draft").length;
  const opened = openId === NEW ? null : items.find((i) => i.id === openId) ?? null;

  const replace = (saved: Illustration) =>
    setItems((list) => (list.some((i) => i.id === saved.id) ? list.map((i) => (i.id === saved.id ? saved : i)) : [...list, saved]));

  async function onTile(id: string, action: () => Promise<void>) {
    setBusyIds((b) => ({ ...b, [id]: true }));
    setStatus(null);
    try {
      await action();
    } catch (e) {
      setStatus(e instanceof Error && e.message ? e.message : "Thao tác không thành công.");
    } finally {
      setBusyIds((b) => ({ ...b, [id]: false }));
    }
  }

  const approve = (ill: Illustration, next: "approved" | "draft") =>
    onTile(ill.id, async () => replace(await setIllustrationStatus(ill.id, next)));

  const rerender = (ill: Illustration) =>
    onTile(ill.id, async () => {
      await rerenderIllustration(ill.id);
      setBust((b) => ({ ...b, [ill.id]: (b[ill.id] ?? 0) + 1 }));
    });

  async function addFolder() {
    setStatus(null);
    try {
      const f = await createIllustrationFolder({ id: newFolderId.trim(), name: newFolderName.trim() });
      setFolders((list) => [...list, f]);
      setFolder(f.id);
      setNewFolderId("");
      setNewFolderName("");
    } catch (e) {
      setStatus(e instanceof Error ? e.message : "Không thêm được thư mục.");
    }
  }

  async function removeFolder(f: IllustrationFolder) {
    if (!window.confirm(`Xoá thư mục "${f.name}"?`)) return;
    try {
      await deleteIllustrationFolder(f.id);
      setFolders((list) => list.filter((x) => x.id !== f.id));
      setFolder(ALL);
    } catch (e) {
      setStatus(e instanceof Error ? e.message : "Không xoá được thư mục.");
    }
  }

  const current = folders.find((f) => f.id === folder);

  return (
    <div data-testid="illustration-library-page">
      <AppShell
        title="Thư viện hình"
        subtitle="Mọi hình Kỹ sư Remotion được dùng. Hình mới nằm ở 'Chờ duyệt' cho tới khi bạn duyệt. Rê chuột lên ô để xem hình chuyển động."
        wide
      >
        <div className={styles.layout}>
          <aside className={styles.side}>
            <Card title="Thư mục">
              <TextInput
                type="search"
                placeholder="Tìm theo tên, thẻ, mô tả…"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                data-testid="illustration-search"
              />
              <ul className={styles.folders} data-testid="illustration-folders">
                <li>
                  <button type="button" className={`${styles.folder} ${folder === ALL ? styles.active : ""}`} onClick={() => setFolder(ALL)}>
                    <span>Tất cả</span>
                    <span className={styles.count}>{items.length}</span>
                  </button>
                </li>
                {folders.map((f) => (
                  <li key={f.id}>
                    <button
                      type="button"
                      className={`${styles.folder} ${folder === f.id ? styles.active : ""}`}
                      onClick={() => setFolder(f.id)}
                      title={f.description}
                      data-testid={`illustration-folder-${f.id}`}
                    >
                      <span>{f.name}</span>
                      <span className={styles.count}>{counts[f.id] ?? 0}</span>
                    </button>
                  </li>
                ))}
              </ul>
              <div className={`${styles.newFolder} ${glass.mtSm}`}>
                <TextInput placeholder="ma-thu-muc" value={newFolderId} onChange={(e) => setNewFolderId(e.target.value)} data-testid="new-folder-id" />
                <TextInput placeholder="Tên thư mục" value={newFolderName} onChange={(e) => setNewFolderName(e.target.value)} data-testid="new-folder-name" />
                <Button variant="ghost" onClick={addFolder} disabled={!newFolderId.trim() || !newFolderName.trim()} data-testid="new-folder-button">
                  Thêm thư mục
                </Button>
              </div>
              {current && !current.is_system && (counts[current.id] ?? 0) === 0 && (
                <Button variant="dangerGhost" className={glass.mtXs} onClick={() => removeFolder(current)}>
                  Xoá thư mục này
                </Button>
              )}
            </Card>
            <IllustrationBackupCard onImported={() => void reload()} />
          </aside>

          <section className={styles.main}>
            <div className={styles.toolbar}>
              <span className={styles.summary}>
                {visible.length} hình{pending > 0 ? ` · ${pending} chờ duyệt` : ""}
              </span>
              <div className={styles.addRow}>
                <Button variant="ghost" onClick={startCode} data-testid="illustration-new-button">
                  Viết code
                </Button>
                <Button variant="ghost" onClick={() => fileInput.current?.click()} data-testid="illustration-upload-button">
                  Tải SVG lên
                </Button>
                <input
                  ref={fileInput}
                  type="file"
                  accept=".svg,image/svg+xml"
                  className={styles.hiddenInput}
                  onChange={onSvgFile}
                  data-testid="illustration-upload-input"
                />
                <Button onClick={() => setDrawOpen((v) => !v)} data-testid="illustration-draw-open">
                  AI vẽ
                </Button>
              </div>
            </div>
            <div className={glass.mtSm}>
              <Disclosure
                title="Luật style của kênh"
                hint="AI vẽ phải theo đúng luật này; hình tải lên hay tự viết code được kiểm tra theo nó. Vi phạm nặng chặn lưu, vi phạm nhẹ chỉ cảnh báo."
                testId="illustration-style"
              >
                <ul className={styles.exemplars}>
                  {items
                    .filter((i) => exemplarIds.includes(i.id))
                    .map((ill) => (
                      <IllustrationTile key={ill.id} illustration={ill} onOpen={() => setOpenId(ill.id)} />
                    ))}
                </ul>
                <pre className={styles.rules} data-testid="illustration-style-rules">{rules}</pre>
              </Disclosure>
            </div>
            {drawOpen && (
              <Card
                className={glass.mtSm}
                title="AI vẽ hình mới"
                hint="Mô tả vật cần vẽ. AI theo luật style và học theo hình mẫu cùng hình bạn đã duyệt trong thư mục; hình vẽ xong nằm ở 'Chờ duyệt'."
                data-testid="illustration-draw-card"
              >
                <FormField label="Vẽ gì">
                  <TextArea
                    rows={2}
                    placeholder="vd: xe máy màu đỏ nhìn ngang, có mặt cười"
                    value={drawText}
                    onChange={(e) => setDrawText(e.target.value)}
                    data-testid="illustration-draw-text"
                  />
                </FormField>
                <FormField label="Thư mục" className={glass.mtSm}>
                  <Dropdown
                    value={drawFolder || folder || folders[0]?.id || ""}
                    options={folders.map((f) => ({ value: f.id, label: f.name, hint: f.description }))}
                    onChange={setDrawFolder}
                    data-testid="illustration-draw-folder"
                  />
                </FormField>
                <CtaRow helperText={drawing ? "AI đang vẽ và tự kiểm tra — thường mất 1–3 phút." : undefined}>
                  <Button onClick={draw} disabled={drawing || drawText.trim() === ""} data-testid="illustration-draw-button">
                    {drawing ? "Đang vẽ…" : "Vẽ"}
                  </Button>
                </CtaRow>
              </Card>
            )}
            {status && (
              <p className={styles.status} role="status">
                {status}
              </p>
            )}
            {openId !== null && (
              <div className={glass.mtSm}>
                <IllustrationEditor
                  illustration={opened}
                  draft={openId === NEW ? draft : undefined}
                  folders={folders}
                  defaultFolderId={folder || undefined}
                  onSaved={(saved) => {
                    replace(saved);
                    setOpenId(saved.id);
                    setBust((b) => ({ ...b, [saved.id]: (b[saved.id] ?? 0) + 1 }));
                  }}
                  onDeleted={() => {
                    setItems((list) => list.filter((i) => i.id !== openId));
                    setOpenId(null);
                  }}
                  onClose={() => setOpenId(null)}
                  redrawNote={fixNote && fixNote.id === openId ? fixNote : undefined}
                />
              </div>
            )}
            <ul className={`${styles.grid} ${glass.mtSm}`} data-testid="illustration-grid">
              {visible.map((ill) => (
                <IllustrationTile
                  key={ill.id}
                  illustration={ill}
                  folderName={folder === ALL ? folderName[ill.folder_id] : undefined}
                  busy={busyIds[ill.id]}
                  bust={bust[ill.id]}
                  selected={ill.id === openId}
                  onOpen={() => setOpenId(ill.id)}
                  onFixWarnings={(text) => {
                    setOpenId(ill.id);
                    setFixNote({ id: ill.id, text, n: Date.now() });
                  }}
                  actions={
                    <>
                      {!ill.builtin && ill.status === "draft" && (
                        <Button onClick={() => approve(ill, "approved")} disabled={busyIds[ill.id]} data-testid={`approve-${ill.name}`}>
                          Duyệt
                        </Button>
                      )}
                      {!ill.builtin && ill.status === "approved" && (
                        <Button variant="ghost" onClick={() => approve(ill, "draft")} disabled={busyIds[ill.id]}>
                          Bỏ duyệt
                        </Button>
                      )}
                      <Button variant="ghost" onClick={() => rerender(ill)} disabled={busyIds[ill.id]} data-testid={`rerender-${ill.name}`}>
                        Dựng lại
                      </Button>
                      {!ill.builtin && (
                        <Button variant="ghost" onClick={() => setOpenId(ill.id)} data-testid={`edit-${ill.name}`}>
                          Sửa code / Vẽ lại
                        </Button>
                      )}
                    </>
                  }
                />
              ))}
            </ul>
            {visible.length === 0 && <p className={styles.empty}>Không có hình nào khớp.</p>}
          </section>
        </div>
      </AppShell>
    </div>
  );
}
