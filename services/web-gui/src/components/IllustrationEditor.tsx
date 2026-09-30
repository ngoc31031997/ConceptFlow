import { useEffect, useRef, useState } from "react";
import {
  ApiError,
  createIllustration,
  deleteIllustration,
  EXEMPLAR_FOLDER_ID,
  redrawIllustration,
  tryIllustration,
  updateIllustration,
  type CodeDiagnostic,
  type Illustration,
  type IllustrationFolder,
  type IllustrationInput,
  type IllustrationTry,
} from "../api/client";
import { Button, Card, CtaRow, Dropdown, FormField, TextArea, TextInput } from "./ui";
import { StyleWarningList } from "./StyleWarnings";
import { useStyleWarnings } from "../hooks/useStyleRuleNames";
import { fixWarningsNote } from "../utils/styleRules";
import glass from "../styles/glass.module.css";
import styles from "./IllustrationEditor.module.css";

/** Khung code mặc định cho một hình mới: dùng lại Figure/Face để cùng nét với bộ có sẵn. */
export const NEW_ILLUSTRATION_TEMPLATE = `import React from 'react';
import {useCurrentFrame} from 'remotion';
import {Figure, Face, shadeOf, type FigureProps} from './conceptflow-mini/illustration';

// Hộp vẽ 240 x 200: sửa vw/vh cho đúng tỉ lệ hình của bạn.
export function MyFigure({color = '#FFC72C', ...fig}: FigureProps & {color?: string}) {
  const frame = useCurrentFrame();
  const bob = fig.still ? 0 : Math.sin(frame / 8) * 3; // chuyển động tự thân
  return (
    <Figure {...fig} size={fig.size ?? 280} vw={240} vh={200}>
      <g transform={\`translate(0 \${bob})\`}>
        <rect x={20} y={40} width={200} height={140} rx={40} fill={color} />
        <rect x={140} y={40} width={80} height={140} rx={40} fill={shadeOf(color, -0.12)} />
        <Face mood="happy" blink={false} talking={false} frame={frame} cx={120} cy={110} glasses={false} />
      </g>
    </Figure>
  );
}
`;

interface IllustrationEditorProps {
  /** null = a new drawing. */
  illustration: Illustration | null;
  /** Prefill for a new drawing (an uploaded SVG converted to code). */
  draft?: Partial<IllustrationInput>;
  folders: IllustrationFolder[];
  defaultFolderId?: string;
  onSaved: (saved: Illustration) => void;
  onDeleted?: () => void;
  onClose: () => void;
  /**
   * A note to put in the "Vẽ lại bằng AI" box (from "Nhờ AI sửa các
   * cảnh báo này"). `n` changes on every request, so the same note can be
   * filled in again after the Creator cleared it.
   */
  redrawNote?: { text: string; n: number };
}

function toInput(ill: Illustration | null, folderId: string, draft?: Partial<IllustrationInput>): IllustrationInput {
  return {
    name: ill?.name ?? draft?.name ?? "MyFigure",
    title: ill?.title ?? draft?.title ?? "",
    folder_id: ill?.folder_id ?? draft?.folder_id ?? folderId,
    tags: ill?.tags ?? draft?.tags ?? [],
    description: ill?.description ?? draft?.description ?? "",
    usage: ill?.usage ?? draft?.usage ?? "",
    code: ill?.code ?? draft?.code ?? NEW_ILLUSTRATION_TEMPLATE,
  };
}

/**
 * Edit one library drawing by hand: its details and its TSX. "Xem
 * trước" renders the code as it is in the box without saving; saving renders
 * it again and keeps the preview. Code the renderer refuses comes back with
 * line numbers, listed under the box.
 */
export function IllustrationEditor({ illustration, draft, folders, defaultFolderId, onSaved, onDeleted, onClose, redrawNote }: IllustrationEditorProps) {
  const creating = illustration === null;
  // A Hình mẫu is read-only like the kit; it changes by picking another one.
  const exemplar = illustration?.exemplar === true;
  const readOnly = illustration?.builtin === true || exemplar;
  // The Hình mẫu folder only takes drawings through "Đặt làm mẫu".
  const writable = folders.filter((f) => f.id !== EXEMPLAR_FOLDER_ID);
  const startFolder = defaultFolderId && defaultFolderId !== EXEMPLAR_FOLDER_ID ? defaultFolderId : (writable[0]?.id ?? "");
  const [form, setForm] = useState<IllustrationInput>(() => toInput(illustration, startFolder, draft));
  const [tagsText, setTagsText] = useState(form.tags.join(", "));
  const [busy, setBusy] = useState<"" | "try" | "save" | "delete" | "redraw">("");
  const [note, setNote] = useState("");
  const [diagnostics, setDiagnostics] = useState<CodeDiagnostic[]>([]);
  const [message, setMessage] = useState<string | null>(null);
  const [tried, setTried] = useState<IllustrationTry | null>(null);
  const noteBox = useRef<HTMLTextAreaElement>(null);
  const shownWarnings = useStyleWarnings(tried?.warnings ?? illustration?.warnings ?? []);

  useEffect(() => {
    const next = toInput(illustration, startFolder, draft);
    setForm(next);
    setTagsText(next.tags.join(", "));
    setDiagnostics([]);
    setMessage(null);
    setTried(null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [illustration?.id, illustration?.version, draft]);

  useEffect(() => {
    if (!redrawNote) return;
    setNote(redrawNote.text);
    noteBox.current?.scrollIntoView?.({ block: "center", behavior: "smooth" });
    noteBox.current?.focus();
  }, [redrawNote]);

  const set = <K extends keyof IllustrationInput>(key: K, value: IllustrationInput[K]) =>
    setForm((f) => ({ ...f, [key]: value }));

  const input = (): IllustrationInput => ({
    ...form,
    tags: tagsText.split(",").map((t) => t.trim()).filter(Boolean),
  });

  async function run(kind: "try" | "save" | "delete" | "redraw", action: () => Promise<void>) {
    setBusy(kind);
    setDiagnostics([]);
    setMessage(null);
    try {
      await action();
    } catch (e) {
      if (e instanceof ApiError && e.diagnostics?.length) setDiagnostics(e.diagnostics);
      setMessage(e instanceof Error && e.message ? e.message : "Thao tác không thành công.");
    } finally {
      setBusy("");
    }
  }

  const handleTry = () =>
    run("try", async () => {
      setTried(await tryIllustration(form.name, form.code));
    });

  const handleSave = () =>
    run("save", async () => {
      const saved = creating ? await createIllustration(input()) : await updateIllustration(illustration.id, input());
      onSaved(saved);
      setMessage(creating ? "Đã thêm vào thư viện — chờ duyệt." : "Đã lưu.");
    });

  const handleRedraw = () =>
    run("redraw", async () => {
      if (!illustration) return;
      const saved = await redrawIllustration(illustration.id, note);
      setNote("");
      onSaved(saved);
      setMessage("AI đã vẽ lại — phiên bản mới đang chờ duyệt.");
    });

  const handleDelete = () => {
    if (!illustration || !window.confirm(`Xoá "${illustration.title}" khỏi thư viện? Không khôi phục được.`)) return;
    return run("delete", async () => {
      await deleteIllustration(illustration.id);
      onDeleted?.();
    });
  };

  const folderOptions = (readOnly ? folders : writable).map((f) => ({ value: f.id, label: f.name, hint: f.description }));

  return (
    <Card
      title={
        creating ? "Hình mới" : exemplar ? `${illustration.title} (Hình mẫu)` : readOnly ? `${illustration.title} (có sẵn)` : `Sửa: ${illustration.title}`
      }
      hint={
        exemplar
          ? "Hình mẫu AI vẽ học theo: chỉ xem. Muốn đổi thì bỏ làm mẫu, rồi đặt làm mẫu một hình khác."
          : readOnly
            ? "Hình có sẵn của bộ minh hoạ: chỉ xem. Code nằm trong conceptflow-mini/illustration.tsx."
            : "Sửa code rồi bấm Xem trước để dựng thử — chưa lưu gì. Lưu thì hình quay về 'Chờ duyệt'."
      }
      headerAction={
        <Button variant="ghost" onClick={onClose} data-testid="illustration-editor-close">
          Đóng
        </Button>
      }
      data-testid="illustration-editor"
    >
      <div className={styles.fields}>
        <FormField label="Tên component (PascalCase)">
          <TextInput
            value={form.name}
            onChange={(e) => set("name", e.target.value)}
            readOnly={readOnly}
            data-testid="illustration-name-input"
          />
        </FormField>
        <FormField label="Tên hiển thị">
          <TextInput value={form.title} onChange={(e) => set("title", e.target.value)} readOnly={readOnly} data-testid="illustration-title-input" />
        </FormField>
        <FormField label="Thư mục">
          <Dropdown
            value={form.folder_id}
            options={folderOptions}
            onChange={(v) => set("folder_id", v)}
            disabled={readOnly}
            data-testid="illustration-folder-input"
          />
        </FormField>
        <FormField label="Thẻ (cách nhau bằng dấu phẩy)">
          <TextInput value={tagsText} onChange={(e) => setTagsText(e.target.value)} readOnly={readOnly} data-testid="illustration-tags-input" />
        </FormField>
      </div>
      <FormField label="Mô tả — trông ra sao, dùng khi nào" className={glass.mtSm}>
        <TextArea rows={2} value={form.description} onChange={(e) => set("description", e.target.value)} readOnly={readOnly} />
      </FormField>
      <FormField label="Cách gọi (tham số, tỉ lệ khung) — Kỹ sư Remotion đọc dòng này" className={glass.mtSm}>
        <TextInput value={form.usage} onChange={(e) => set("usage", e.target.value)} readOnly={readOnly} data-testid="illustration-usage-input" />
      </FormField>
      {(!readOnly || exemplar) && (
        <FormField label="Code TSX" className={glass.mtSm}>
          <TextArea
            rows={20}
            spellCheck={false}
            className={styles.code}
            value={form.code}
            onChange={(e) => set("code", e.target.value)}
            readOnly={readOnly}
            data-testid="illustration-code-input"
          />
        </FormField>
      )}
      {diagnostics.length > 0 && (
        <ul className={styles.diagnostics} data-testid="illustration-diagnostics">
          {diagnostics.map((d, i) => (
            <li key={i}>
              {d.line != null && <span className={styles.line}>dòng {d.line}</span>} {d.message}
            </li>
          ))}
        </ul>
      )}
      {shownWarnings.length > 0 && (
        <div className={styles.warnings}>
          <StyleWarningList items={shownWarnings} testId="illustration-style-warnings" />
          {!readOnly && !creating && (
            <Button
              variant="ghost"
              className={glass.mtSm}
              onClick={() => {
                setNote(fixWarningsNote(shownWarnings));
                noteBox.current?.focus();
              }}
              data-testid="illustration-fix-warnings"
            >
              Nhờ AI sửa các cảnh báo này
            </Button>
          )}
        </div>
      )}
      {tried && (
        <div className={styles.tried} data-testid="illustration-tried">
          <img src={`data:image/png;base64,${tried.png}`} alt="Ảnh tĩnh" />
          {tried.gif && <img src={`data:image/gif;base64,${tried.gif}`} alt="Chuyển động" />}
        </div>
      )}
      {!readOnly && !creating && (
        <div className={`${styles.redraw} ${glass.mtSm}`}>
          <TextArea
            ref={noteBox}
            rows={note.includes("\n") ? Math.min(8, note.split("\n").length + 1) : 1}
            placeholder="Ghi chú cho AI, vd: bánh xe to hơn, mặt vui hơn (có thể để trống)"
            value={note}
            onChange={(e) => setNote(e.target.value)}
            data-testid="illustration-redraw-note"
          />
          <Button variant="ghost" onClick={handleRedraw} disabled={busy !== ""} data-testid="illustration-redraw-button">
            {busy === "redraw" ? "AI đang vẽ lại… (1–3 phút)" : "Vẽ lại bằng AI"}
          </Button>
        </div>
      )}
      {!readOnly && (
        <CtaRow helperText={message ?? undefined}>
          {!creating && (
            <Button variant="dangerGhost" onClick={handleDelete} disabled={busy !== ""} data-testid="illustration-delete-button">
              {busy === "delete" ? "Đang xoá…" : "Xoá"}
            </Button>
          )}
          <Button variant="ghost" onClick={handleTry} disabled={busy !== ""} data-testid="illustration-try-button">
            {busy === "try" ? "Đang dựng…" : "Xem trước"}
          </Button>
          <Button onClick={handleSave} disabled={busy !== "" || !form.folder_id} data-testid="illustration-save-button">
            {busy === "save" ? "Đang lưu…" : creating ? "Thêm vào thư viện" : "Lưu"}
          </Button>
        </CtaRow>
      )}
    </Card>
  );
}
