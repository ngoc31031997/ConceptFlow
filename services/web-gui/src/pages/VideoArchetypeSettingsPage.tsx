import { useCallback, useEffect, useState } from "react";
import { AppShell } from "../components/AppShell";
import { Card, Button, FormField, TextArea, TextInput, CtaRow } from "../components/ui";
import {
  listVideoArchetypes,
  createVideoArchetype,
  copyVideoArchetype,
  updateVideoArchetype,
  deleteVideoArchetype,
  type VideoArchetype,
} from "../api/client";
import glass from "../styles/glass.module.css";
import styles from "./PromptSettingsPage.module.css";

const NEW_ROW = "new";

/**
 * CR-041 — bảng kiểu video. Prompt Biên kịch đọc bảng này (biến
 * `{{video_archetypes}}`): mỗi dòng là một kiểu để model chọn, kèm playbook nói
 * cách gán kiểu đó vào các beat của format.
 *
 * Cùng quy ước với thư viện prompt: dòng "Hệ thống" chỉ xem và copy, dòng "Của
 * bạn" sửa/xóa tự do. Thêm dòng nào là model thấy ngay ở lượt Biên kịch kế tiếp,
 * không cần build lại.
 */
export function VideoArchetypeSettingsPage() {
  const [rows, setRows] = useState<VideoArchetype[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [code, setCode] = useState("");
  const [name, setName] = useState("");
  const [whenToUse, setWhenToUse] = useState("");
  const [playbook, setPlaybook] = useState("");
  const [loading, setLoading] = useState(false);
  const [busy, setBusy] = useState(false);
  const [status, setStatus] = useState<string | null>(null);

  const reload = useCallback(async () => {
    setLoading(true);
    try {
      setRows(await listVideoArchetypes());
    } catch {
      setStatus("Không tải được danh sách kiểu video. Vui lòng thử lại.");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  const creating = selectedId === NEW_ROW;
  const selected = rows.find((r) => r.id === selectedId) ?? null;
  const readOnly = selected?.is_system === true;

  useEffect(() => {
    if (creating) return;
    if (selected === null) {
      setSelectedId(rows[0]?.id ?? null);
      return;
    }
    setCode(selected.code);
    setName(selected.name);
    setWhenToUse(selected.when_to_use);
    setPlaybook(selected.playbook);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rows, selectedId]);

  /** `action` trả về id dòng cần chọn sau khi tải lại (dòng mới chưa có trong danh sách trước đó). */
  async function run(action: () => Promise<string | void>, ok: string) {
    setBusy(true);
    setStatus(null);
    try {
      const nextId = await action();
      await reload();
      if (nextId) setSelectedId(nextId);
      setStatus(ok);
    } catch (e) {
      // Lý do từ server (mã trùng, thiếu trường...) hữu ích hơn "thất bại".
      setStatus(e instanceof Error && e.message ? e.message : "Thao tác không thành công. Vui lòng thử lại.");
    } finally {
      setBusy(false);
    }
  }

  const startNew = () => {
    setSelectedId(NEW_ROW);
    setCode("");
    setName("");
    setWhenToUse("");
    setPlaybook("");
    setStatus(null);
  };

  const input = { code, name, when_to_use: whenToUse, playbook };

  const handleSave = () =>
    run(async () => {
      if (creating) return (await createVideoArchetype(input)).id;
      if (selected) await updateVideoArchetype(selected.id, input);
    }, creating ? "Đã thêm kiểu video." : "Đã lưu.");

  const handleCopy = (id: string) =>
    run(async () => (await copyVideoArchetype(id)).id, "Đã sao chép thành một kiểu của bạn.");

  const handleDelete = () => {
    if (!selected) return;
    if (!window.confirm("Xóa kiểu video này? Không khôi phục lại được.")) return;
    return run(async () => {
      await deleteVideoArchetype(selected.id);
      setSelectedId(null);
    }, "Đã xóa.");
  };

  const incomplete = name.trim() === "" || whenToUse.trim() === "" || playbook.trim() === "";
  const editorDisabled = loading || (!creating && selected === null);

  return (
    <div data-testid="video-archetype-page">
      <AppShell
        title="Kiểu video"
        subtitle="Mỗi dòng là một kiểu video prompt Biên kịch có thể chọn. Thêm kiểu của bạn: model thấy ngay ở lượt sau. Ghi 'kiểu: <mã>' vào chủ đề để ép một kiểu."
        wide
      >
        <div className={styles.layout}>
          <div className={styles.controls}>
            <Card title="Bảng kiểu video">
              <ul className={styles.list} data-testid="archetype-list">
                {rows.map((r) => (
                  <li key={r.id}>
                    <button
                      type="button"
                      className={`${styles.row} ${r.id === selectedId ? styles.rowSelected : ""}`}
                      onClick={() => {
                        setSelectedId(r.id);
                        setStatus(null);
                      }}
                      data-testid={`archetype-row-${r.id}`}
                    >
                      <span className={styles.rowName}>
                        {r.code} — {r.name}
                      </span>
                      <span className={styles.badges}>
                        <span className={styles.badge}>{r.is_system ? "Hệ thống" : "Của bạn"}</span>
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
              <CtaRow>
                <Button variant="ghost" onClick={startNew} disabled={busy || loading} data-testid="archetype-new-button">
                  + Thêm kiểu video
                </Button>
              </CtaRow>
              {status && (
                <p className={`${glass.cardHint} ${glass.mtXs}`} data-testid="archetype-status">
                  {status}
                </p>
              )}
            </Card>
          </div>

          <Card
            title={creating ? "Kiểu video mới" : selected ? `${selected.code} — ${selected.name}` : "Chưa chọn kiểu"}
            hint={
              readOnly
                ? "Kiểu hệ thống — chỉ xem. Bấm 'Copy' để tạo bản của bạn rồi sửa."
                : "Playbook nên nói rõ cách gán vào từng id beat (hook, concrete, pattern, variation, modern, recap...)."
            }
            headerAction={
              selected !== null || creating ? (
                <div className={styles.actions} role="toolbar" aria-label="Thao tác kiểu video">
                  {selected !== null && !creating && (
                    <button
                      type="button"
                      className={styles.actionBtn}
                      onClick={() => handleCopy(selected.id)}
                      disabled={busy || loading}
                      data-testid="archetype-copy-button"
                    >
                      <span aria-hidden="true">⧉</span> Copy
                    </button>
                  )}
                  {!readOnly && selected !== null && !creating && (
                    <button
                      type="button"
                      className={`${styles.actionBtn} ${styles.actionDanger}`}
                      onClick={handleDelete}
                      disabled={busy || loading}
                      data-testid="archetype-delete-button"
                    >
                      <span aria-hidden="true">🗑</span> Xóa
                    </button>
                  )}
                  {!readOnly && (
                    <button
                      type="button"
                      className={`${styles.actionBtn} ${styles.actionSave}`}
                      onClick={handleSave}
                      disabled={busy || loading || incomplete}
                      data-testid="archetype-save-button"
                    >
                      <span aria-hidden="true">💾</span> {busy ? "Đang lưu..." : "Lưu"}
                    </button>
                  )}
                </div>
              ) : undefined
            }
          >
            <FormField label="Mã (để trống = chữ cái kế tiếp còn trống)">
              <TextInput
                data-testid="archetype-code-input"
                value={code}
                onChange={(e) => setCode(e.target.value)}
                readOnly={readOnly}
                disabled={editorDisabled}
                maxLength={8}
              />
            </FormField>
            <FormField label="Tên" className={glass.mtSm}>
              <TextInput
                data-testid="archetype-name-input"
                value={name}
                onChange={(e) => setName(e.target.value)}
                readOnly={readOnly}
                disabled={editorDisabled}
              />
            </FormField>
            <FormField label="Hợp với chủ đề nào (model chọn kiểu dựa vào đây)" className={glass.mtSm}>
              <TextArea
                data-testid="archetype-when-input"
                value={whenToUse}
                onChange={(e) => setWhenToUse(e.target.value)}
                readOnly={readOnly}
                disabled={editorDisabled}
                rows={3}
              />
            </FormField>
            <FormField label="Playbook — cách gán vào các beat của format" className={glass.mtSm}>
              <TextArea
                data-testid="archetype-playbook-input"
                value={playbook}
                onChange={(e) => setPlaybook(e.target.value)}
                readOnly={readOnly}
                disabled={editorDisabled}
                rows={14}
              />
            </FormField>
          </Card>
        </div>
      </AppShell>
    </div>
  );
}
