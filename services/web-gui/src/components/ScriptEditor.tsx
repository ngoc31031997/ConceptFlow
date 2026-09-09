import { useMemo, useState, type ChangeEvent } from "react";
import { END_SCREEN_SNIPPETS, HOOK_SNIPPETS } from "./scriptTemplates";
import glass from "../styles/glass.module.css";
import styles from "./ScriptEditor.module.css";
import { validateScript } from "../utils/scriptValidation";

interface ScriptEditorProps {
  value: string;
  onChange: (value: string) => void;
  /** Picks the language of the snippets this inserts (CR-008 FR21.4). */
  contentLanguage: "vi" | "en";
}

function UploadIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 16V4M12 4L7 9M12 4L17 9" />
      <path d="M4 17V19a2 2 0 002 2h12a2 2 0 002-2v-2" />
    </svg>
  );
}

function TemplateIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <rect x="4" y="4" width="16" height="16" rx="2" />
      <path d="M4 9h16M9 9v11" />
    </svg>
  );
}

function CheckCircleIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="9" />
      <path d="M8.5 12.5l2.3 2.3L16 10" />
    </svg>
  );
}

function WarningIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 3l10 18H2L12 3z" />
      <path d="M12 10v4M12 17.5v.01" />
    </svg>
  );
}

/**
 * The editor is now only an editor.
 *
 * It used to carry a "Công cụ AI" dropdown that mixed prompts meant to be
 * copied into another tool with snippets inserted right here — the getting-a-
 * script part of the job now belongs to ScriptAssistant, which explains the
 * round trip instead of hiding it in a menu.
 */
export function ScriptEditor({ value, onChange, contentLanguage }: ScriptEditorProps) {
  const validation = useMemo(() => validateScript(value), [value]);
  const [importError, setImportError] = useState<string | null>(null);
  const hasScript = value.trim().length > 0;

  function handleFileImport(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      onChange(String(reader.result ?? ""));
      setImportError(null);
    };
    reader.onerror = () => setImportError("Không đọc được file, thử lại hoặc dán trực tiếp");
    reader.readAsText(file);
    event.target.value = "";
  }

  return (
    <div className={glass.card} id="script-editor">
      <div className={glass.cardHeader}>
        <div className={glass.cardTitle}>Script Manim (.py)</div>
        <div className={styles.headerActions}>
          {/* Snippets append to an existing script, so they only make sense
              once there is one to append to. */}
          {hasScript && (
            <>
              <button
                type="button"
                className={glass.ghostBtn}
                data-testid="script-editor-insert-hook"
                onClick={() => onChange(`${value}\n${HOOK_SNIPPETS[contentLanguage]}`)}
              >
                <TemplateIcon />
                Chèn hook mở đầu
              </button>
              <button
                type="button"
                className={glass.ghostBtn}
                data-testid="script-editor-insert-end-screen"
                onClick={() => onChange(`${value}\n${END_SCREEN_SNIPPETS[contentLanguage]}`)}
              >
                <TemplateIcon />
                Chèn end screen
              </button>
            </>
          )}
          <label className={glass.ghostBtn}>
            <UploadIcon />
            Nhập từ file
            <input type="file" accept=".py" onChange={handleFileImport} className={styles.hiddenFileInput} />
          </label>
        </div>
      </div>

      <textarea
        className={`${glass.textArea} ${styles.textarea}`}
        data-testid="new-project-script-textarea"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={
          'class DemoScene(Scene):\n    def construct(self):\n        # NARRATION: "Loi thoai cho canh nay"\n        self.wait(AUTO)'
        }
        rows={16}
      />

      {importError && (
        <div className={styles.validationError} role="alert">
          <WarningIcon />
          {importError}
        </div>
      )}

      {hasScript && (
        <div
          id="script-validation"
          className={validation.isValid ? styles.validationOk : styles.validationError}
          data-testid="script-editor-validation"
        >
          {validation.isValid ? (
            <>
              <CheckCircleIcon />
              Hợp lệ: {validation.narrationCount} đoạn NARRATION khớp {validation.autoWaitCount} self.wait(AUTO).
            </>
          ) : (
            <>
              <WarningIcon />
              {validation.message}
            </>
          )}
        </div>
      )}

      <div className={glass.cardHint}>
        Đặt <code>{'# NARRATION: "..."'}</code> ngay trước mỗi <code>self.wait(AUTO)</code> — hệ thống tạo
        giọng đọc cho từng đoạn và thay <code>AUTO</code> bằng thời lượng thật trước khi render.
      </div>
    </div>
  );
}
