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

function TemplateIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <rect x="4" y="4" width="16" height="16" rx="2" />
      <path d="M4 9h16M9 9v11" />
    </svg>
  );
}

const SCRIPT_TEMPLATE = `## Scene 1: Giới thiệu vòng lặp for trong Java

Vòng lặp for trong Java gồm 3 phần: khởi tạo biến đếm, điều kiện lặp, và bước tăng/giảm — cả ba được viết gọn trên cùng một dòng.

> Hoạt hình hiển thị biến i tăng dần từ 0 đến 4, mỗi vòng lặp in ra một giá trị

\`\`\`java
for (int i = 0; i < 5; i++) {
    System.out.println(i);
}
\`\`\`

## Scene 2: Ứng dụng thực tế — tính tổng một mảng

Chúng ta có thể dùng vòng lặp for để duyệt qua từng phần tử của một mảng và cộng dồn giá trị.

\`\`\`java
int[] numbers = {1, 2, 3, 4, 5};
int total = 0;
for (int i = 0; i < numbers.length; i++) {
    total += numbers[i];
}
System.out.println(total);
\`\`\`
`;

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
        <div className={styles.headerActions}>
          <button type="button" className={glass.ghostBtn} onClick={() => onChange(SCRIPT_TEMPLATE)}>
            <TemplateIcon />
            Dùng mẫu
          </button>
          <label className={glass.ghostBtn}>
            <UploadIcon />
            Nhập từ file
            <input type="file" accept=".md,.txt" onChange={handleFileImport} className={styles.hiddenFileInput} />
          </label>
        </div>
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
