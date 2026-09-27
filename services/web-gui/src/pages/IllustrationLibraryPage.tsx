import { useCallback, useEffect, useMemo, useState } from "react";
import { AppShell } from "../components/AppShell";
import { Button, Card, TextInput } from "../components/ui";
import { IllustrationTile } from "../components/IllustrationTile";
import { IllustrationEditor } from "../components/IllustrationEditor";
import {
  createIllustrationFolder,
  deleteIllustrationFolder,
  listIllustrationFolders,
  listIllustrations,
  rerenderIllustration,
  setIllustrationStatus,
  type Illustration,
  type IllustrationFolder,
} from "../api/client";
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
  const [busyIds, setBusyIds] = useState<Record<string, boolean>>({});
  const [bust, setBust] = useState<Record<string, number>>({});
  const [status, setStatus] = useState<string | null>(null);
  const [newFolderId, setNewFolderId] = useState("");
  const [newFolderName, setNewFolderName] = useState("");

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
  }, [reload]);

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
          </aside>

          <section className={styles.main}>
            <div className={styles.toolbar}>
              <span className={styles.summary}>
                {visible.length} hình{pending > 0 ? ` · ${pending} chờ duyệt` : ""}
              </span>
              <Button onClick={() => setOpenId(NEW)} data-testid="illustration-new-button">
                Thêm hình
              </Button>
            </div>
            {status && (
              <p className={styles.status} role="status">
                {status}
              </p>
            )}
            {openId !== null && (
              <div className={glass.mtSm}>
                <IllustrationEditor
                  illustration={opened}
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
                        <Button variant="ghost" onClick={() => setOpenId(ill.id)}>
                          Sửa code
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
