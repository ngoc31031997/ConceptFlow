import type { ChangeEvent } from "react";
import styles from "./ScriptEditor.module.css";

interface ScriptEditorProps {
  value: string;
  onChange: (value: string) => void;
}

export function ScriptEditor({ value, onChange }: ScriptEditorProps) {
  function handleFileImport(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => onChange(String(reader.result ?? ""));
    reader.readAsText(file);
  }

  return (
    <div className={styles.container}>
      <textarea
        className={styles.textarea}
        data-testid="new-project-script-textarea"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder="Nhập nội dung script tại đây..."
        rows={12}
      />
      <input type="file" accept=".md,.txt" onChange={handleFileImport} />
    </div>
  );
}
