import { useCallback, useEffect, useMemo, useRef, useState, type ChangeEvent } from "react";
import { AppShell } from "../components/AppShell";
import { Button, Card, CtaRow, Dropdown, FormField, TextArea, TextInput } from "../components/ui";
import { Disclosure } from "../components/Disclosure";
import { IllustrationTile } from "../components/IllustrationTile";
import { IllustrationEditor } from "../components/IllustrationEditor";
import { IllustrationBackupCard } from "../components/IllustrationBackupCard";
import {
  createIllustrationFolder,
  deleteIllustration,
  deleteIllustrationFolder,
  drawIllustration,
  EXEMPLAR_FOLDER_ID,
  getIllustrationStyle,
  listIllustrationFolders,
  listIllustrations,
  makeExemplar,
  rerenderIllustration,
  setIllustrationStatus,
  unmakeExemplar,
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
 * Thư viện hình minh hoạ. Mỗi hình nằm trong đúng một thư mục; Kỹ sư
 * Remotion chỉ được dùng hình "Có sẵn" hoặc "Đã duyệt". Mỗi ô là một hình: rê
 * chuột để xem nó chuyển động, bấm để mở trình sửa code.
 *
 * Mỗi ô có nút Xoá (server từ chối khi một dự án chưa tới bước Kết quả
 * còn dùng hình), và Hình mẫu AI vẽ học theo do Creator chọn: "Đặt làm mẫu" chép
 * một hình đã duyệt vào thư mục Hình mẫu, "Bỏ làm mẫu" gỡ nó ra.
 */
export function IllustrationLibraryPage() {
  const [folders, setFolders] = useState<IllustrationFolder[]>([]);
  const [items, setItems] = useState<Illustration[]>([]);
  const [folder, setFolder] = useState(ALL);
  const [query, setQuery] = useState("");
  const [openId, setOpenId] = useState<string | null>(null);
  // "Nhờ AI sửa các cảnh báo này" on a tile: open that drawing with the redraw note filled in.
  const [fixNote, setFixNote] = useState<{ id: string; text: string; n: number } | null>(null);
  const [busyIds, setBusyIds] = useState<Record<string, boolean>>({});
  const [bust, setBust] = useState<Record<string, number>>({});
  const [status, setStatus] = useState<string | null>(null);
  const [newFolderId, setNewFolderId] = useState("");
  const [newFolderName, setNewFolderName] = useState("");
  const [rules, setRules] = useState("");
  const [exemplarIds, setExemplarIds] = useState<string[]>([]);
  const [maxExemplars, setMaxExemplars] = useState(5);
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
        setMaxExemplars(st.max_exemplars);
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
        folder_id: writableFolder,
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
      const made = await drawIllustration({ description: drawText.trim(), folder_id: drawTarget });
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

  // The Hình mẫu folder only takes drawings through "Đặt làm mẫu".
  const writableFolders = folders.filter((f) => f.id !== EXEMPLAR_FOLDER_ID);
  const writableFolder = folder && folder !== EXEMPLAR_FOLDER_ID ? folder : undefined;
  const drawTarget = drawFolder || writableFolder || writableFolders[0]?.id || "";
  const exemplars = exemplarIds.map((id) => items.find((i) => i.id === id)).filter((i): i is Illustration => !!i);
  const copied = new Set(items.map((i) => i.source_id).filter(Boolean));

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

  const remove = (ill: Illustration) => {
    if (!window.confirm(`Xoá "${ill.title}" khỏi thư viện? Không khôi phục được.`)) return;
    return onTile(ill.id, async () => {
      await deleteIllustration(ill.id);
      setItems((list) => list.filter((i) => i.id !== ill.id));
      if (openId === ill.id) setOpenId(null);
    });
  };

  const makeSample = (ill: Illustration) =>
    onTile(ill.id, async () => {
      const copy = await makeExemplar(ill.id);
      setItems((list) => [...list, copy]);
      setExemplarIds((ids) => [...ids, copy.id]);
    });

  const unmakeSample = (ill: Illustration) => {
    if (!window.confirm(`Bỏ "${ill.title}" khỏi Hình mẫu?`)) return;
    return onTile(ill.id, async () => {
      const back = await unmakeExemplar(ill.id);
      setItems((list) => (back ? list.map((i) => (i.id === ill.id ? back : i)) : list.filter((i) => i.id !== ill.id)));
      setExemplarIds((ids) => ids.filter((id) => id !== ill.id));
      if (!back && openId === ill.id) setOpenId(null);
    });
  };

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
                <p className={styles.exemplarHead} data-testid="illustration-exemplar-count">
                  Hình mẫu ({exemplars.length}/{maxExemplars}) — AI vẽ học theo các hình này
                </p>
                {exemplars.length === 0 && (
                  <p className={styles.empty}>Chưa có Hình mẫu. Bấm "Đặt làm mẫu" trên một hình đã duyệt.</p>
                )}
                <ul className={styles.exemplars}>
                  {exemplars.map((ill) => (
                    <IllustrationTile
                      key={ill.id}
                      illustration={ill}
                      busy={busyIds[ill.id]}
                      onOpen={() => setOpenId(ill.id)}
                      actions={
                        <Button variant="ghost" onClick={() => unmakeSample(ill)} disabled={busyIds[ill.id]} data-testid={`unmake-exemplar-${ill.name}`}>
                          Bỏ làm mẫu
                        </Button>
                      }
                    />
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
                    value={drawTarget}
                    options={writableFolders.map((f) => ({ value: f.id, label: f.name, hint: f.description }))}
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
                  defaultFolderId={writableFolder}
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
                      {!ill.builtin && !ill.exemplar && ill.status === "draft" && (
                        <Button onClick={() => approve(ill, "approved")} disabled={busyIds[ill.id]} data-testid={`approve-${ill.name}`}>
                          Duyệt
                        </Button>
                      )}
                      {!ill.builtin && !ill.exemplar && ill.status === "approved" && (
                        <Button variant="ghost" onClick={() => approve(ill, "draft")} disabled={busyIds[ill.id]}>
                          Bỏ duyệt
                        </Button>
                      )}
                      {!ill.builtin && !ill.exemplar && ill.status === "approved" && !copied.has(ill.id) && (
                        <Button
                          variant="ghost"
                          onClick={() => makeSample(ill)}
                          disabled={busyIds[ill.id] || exemplars.length >= maxExemplars}
                          title={exemplars.length >= maxExemplars ? `Đã đủ ${maxExemplars} Hình mẫu — bỏ bớt một hình trước` : undefined}
                          data-testid={`make-exemplar-${ill.name}`}
                        >
                          Đặt làm mẫu
                        </Button>
                      )}
                      {ill.exemplar && (
                        <Button variant="ghost" onClick={() => unmakeSample(ill)} disabled={busyIds[ill.id]} data-testid={`unmake-exemplar-tile-${ill.name}`}>
                          Bỏ làm mẫu
                        </Button>
                      )}
                      <Button variant="ghost" onClick={() => rerender(ill)} disabled={busyIds[ill.id]} data-testid={`rerender-${ill.name}`}>
                        Dựng lại
                      </Button>
                      {!ill.builtin && !ill.exemplar && (
                        <Button variant="ghost" onClick={() => setOpenId(ill.id)} data-testid={`edit-${ill.name}`}>
                          Sửa code / Vẽ lại
                        </Button>
                      )}
                      {!ill.builtin && !ill.exemplar && (
                        <Button variant="dangerGhost" onClick={() => remove(ill)} disabled={busyIds[ill.id]} data-testid={`delete-${ill.name}`}>
                          Xoá
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
