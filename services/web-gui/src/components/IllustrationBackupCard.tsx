import { useRef, useState, type ChangeEvent } from "react";
import { Button, Card } from "./ui";
import { usePresence } from "../hooks/usePresence";
import {
  ILLUSTRATION_BACKUP_MAX_BYTES,
  exportIllustrationLibrary,
  importIllustrationLibrary,
  type IllustrationImportItem,
  type IllustrationImportReport,
} from "../api/client";
import glass from "../styles/glass.module.css";
import styles from "./IllustrationBackupCard.module.css";

const RESULT_LABEL: Record<IllustrationImportItem["result"], string> = {
  created: "Đã thêm",
  replaced: "Đã ghi đè",
  skipped: "Bỏ qua",
  failed: "Lỗi",
};

function backupFileName(now: Date): string {
  const p = (n: number) => String(n).padStart(2, "0");
  return `conceptflow-thu-vien-hinh-${now.getFullYear()}${p(now.getMonth() + 1)}${p(now.getDate())}-${p(now.getHours())}${p(now.getMinutes())}.zip`;
}

function saveBlob(blob: Blob, name: string) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = name;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

interface Props {
  /** Gọi sau khi nhập xong, để trang tải lại thư mục và hình. */
  onImported: () => void;
}

/**
 * CR-044 — sao lưu và khôi phục thư viện hình. Xuất: một file ZIP gồm mọi thư
 * mục và mọi hình tự vẽ (code, ảnh PNG, GIF, thẻ, trạng thái duyệt). Nhập: mỗi
 * hình về đúng thư mục của nó; thư mục thiếu được tạo lại. Hình có sẵn của hệ
 * thống không nằm trong file: chúng đi kèm phần mềm.
 */
export function IllustrationBackupCard({ onImported }: Props) {
  const [exporting, setExporting] = useState(false);
  const [file, setFile] = useState<File | null>(null);
  const [replace, setReplace] = useState(false);
  const [importing, setImporting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [report, setReport] = useState<IllustrationImportReport | null>(null);
  const fileInput = useRef<HTMLInputElement>(null);

  async function doExport() {
    setExporting(true);
    setError(null);
    try {
      saveBlob(await exportIllustrationLibrary(), backupFileName(new Date()));
    } catch (e) {
      setError(e instanceof Error && e.message ? e.message : "Không xuất được thư viện hình.");
    } finally {
      setExporting(false);
    }
  }

  function onFile(e: ChangeEvent<HTMLInputElement>) {
    const picked = e.target.files?.[0] ?? null;
    e.target.value = "";
    setReport(null);
    setError(null);
    if (picked && picked.size > ILLUSTRATION_BACKUP_MAX_BYTES) {
      setFile(null);
      setError(`File lớn quá (tối đa ${ILLUSTRATION_BACKUP_MAX_BYTES / 1024 / 1024} MB).`);
      return;
    }
    setFile(picked);
  }

  async function doImport() {
    if (!file) return;
    setImporting(true);
    setError(null);
    setReport(null);
    try {
      const out = await importIllustrationLibrary(file, replace);
      setReport(out);
      setFile(null);
      onImported();
    } catch (e) {
      setError(e instanceof Error && e.message ? e.message : "Không nhập được file sao lưu.");
    } finally {
      setImporting(false);
    }
  }

  // Keep the last report and error on screen while they slide out.
  const lastReport = useRef<IllustrationImportReport | null>(null);
  if (report) lastReport.current = report;
  const shownReport = report ?? lastReport.current;
  const reportPresence = usePresence(report !== null);
  const lastError = useRef<string | null>(null);
  if (error) lastError.current = error;
  const errorPresence = usePresence(error !== null);

  const count = (r: IllustrationImportItem["result"]) => shownReport?.items.filter((i) => i.result === r).length ?? 0;
  const notes = shownReport?.items.filter((i) => i.result === "failed" || i.result === "skipped") ?? [];

  return (
    <Card
      className={glass.mtSm}
      title="Sao lưu"
      hint="Xuất mọi thư mục và hình tự vẽ (code, ảnh, thẻ, trạng thái duyệt) ra một file ZIP; nhập lại để khôi phục đúng thư mục."
      data-testid="illustration-backup"
    >
      <Button variant="ghost" onClick={doExport} disabled={exporting || importing} data-testid="illustration-export-button">
        {exporting ? "Đang xuất…" : "Xuất thư viện (.zip)"}
      </Button>

      <div className={styles.restore}>
        <span className={styles.label}>Khôi phục từ file sao lưu</span>
        <Button variant="ghost" onClick={() => fileInput.current?.click()} disabled={importing} data-testid="illustration-import-pick">
          Chọn file .zip
        </Button>
        <input
          ref={fileInput}
          type="file"
          accept=".zip,application/zip"
          className={styles.hiddenInput}
          onChange={onFile}
          data-testid="illustration-import-input"
        />
        {file && (
          <span className={styles.fileName} data-testid="illustration-import-file">
            {file.name}
          </span>
        )}
        <label className={styles.check}>
          <input
            type="checkbox"
            checked={replace}
            onChange={(e) => setReplace(e.target.checked)}
            disabled={importing}
            data-testid="illustration-import-replace"
          />
          Ghi đè hình trùng tên
        </label>
        <Button onClick={doImport} disabled={!file || importing} data-testid="illustration-import-button">
          {importing ? "Đang nhập…" : "Nhập vào thư viện"}
        </Button>
        {importing && <p className={styles.hint}>Mỗi hình được kiểm tra và dựng lại — có thể mất vài phút.</p>}
      </div>

      {errorPresence.mounted && (
        <p className={`${styles.error} ${errorPresence.closing ? glass.revealOut : glass.reveal}`} role="alert">
          {error ?? lastError.current}
        </p>
      )}

      {reportPresence.mounted && shownReport && (
        <div
          className={`${styles.report} ${reportPresence.closing ? glass.revealOut : glass.reveal}`}
          data-testid="illustration-import-report"
        >
          <p className={styles.summary}>
            Đã thêm {count("created")} · Ghi đè {count("replaced")} · Bỏ qua {count("skipped")} · Lỗi {count("failed")}
          </p>
          {shownReport.folders_created.length > 0 && <p>Thư mục tạo lại: {shownReport.folders_created.join(", ")}</p>}
          {shownReport.folder_errors.map((e) => (
            <p key={e} className={styles.error}>
              Thư mục {e}
            </p>
          ))}
          {shownReport.aborted && (
            <p className={styles.error} data-testid="illustration-import-aborted">
              Dừng giữa chừng, còn {shownReport.not_processed} hình chưa nhập: {shownReport.aborted}. Nhập lại cùng file để làm tiếp.
            </p>
          )}
          {notes.length > 0 && (
            <ul className={styles.notes}>
              {notes.map((i, n) => (
                <li key={`${i.name}-${n}`} className={i.result === "failed" ? styles.error : undefined}>
                  <strong>{i.name}</strong> — {RESULT_LABEL[i.result]}: {i.reason}
                </li>
              ))}
            </ul>
          )}
        </div>
      )}
    </Card>
  );
}
