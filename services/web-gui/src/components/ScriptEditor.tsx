import type { ChangeEvent } from "react";
import glass from "../styles/glass.module.css";
import styles from "./ScriptEditor.module.css";

interface ScriptEditorProps {
  value: string;
  onChange: (value: string) => void;
}

function UploadIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 16V4M12 4L7 9M12 4L17 9" />
      <path d="M4 17V19a2 2 0 002 2h12a2 2 0 002-2v-2" />
    </svg>
  );
}

export function ScriptEditor({ value, onChange }: ScriptEditorProps) {
  function handleFileImport(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => onChange(String(reader.result ?? ""));
    reader.readAsText(file);
    event.target.value = "";
  }

  return (
    <div className={glass.card}>
      <div className={glass.cardHeader}>
        <div className={glass.cardTitle}>Nội dung script</div>
        <label className={glass.ghostBtn}>
          <UploadIcon />
          Nhập từ file
          <input type="file" accept=".md,.txt" onChange={handleFileImport} className={styles.hiddenFileInput} />
        </label>
      </div>
      <textarea
        className={`${glass.textArea} ${styles.textarea}`}
        data-testid="new-project-script-textarea"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={"## Scene 1: Giới thiệu vòng lặp for\n\nMột vòng lặp for cho phép lặp qua từng phần tử...\n\n```python\nfor i in range(5):\n    print(i)\n```"}
        rows={12}
      />
    </div>
  );
}
