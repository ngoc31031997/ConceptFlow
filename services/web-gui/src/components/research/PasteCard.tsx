import { useState } from "react";
import type { PasteLog } from "../../api/client";
import type { PasteMessage } from "../../hooks/useResearch";
import { Button, Card, TextArea } from "../ui";
import { describePaste, pasteTime } from "./pasteText";
import styles from "./research.module.css";

interface PasteCardProps {
  pastes: PasteLog[];
  message: PasteMessage | null;
  busy: boolean;
  locked: boolean;
  onRead: (raw: string) => Promise<boolean>;
}

/**
 * Section 3 of the research step: the Creator pastes the whole answer of the
 * outside tool, as many times as they like. The server finds the JSON block,
 * merges the facts by code and logs each paste with what it read and left out.
 */
export function PasteCard({ pastes, message, busy, locked, onRead }: PasteCardProps) {
  const [text, setText] = useState("");
  const [empty, setEmpty] = useState(false);

  async function read() {
    if (text.trim() === "") {
      setEmpty(true);
      return;
    }
    setEmpty(false);
    if (await onRead(text)) setText("");
  }

  return (
    <Card data-testid="research-paste-card">
      <div className={styles.body}>
        <div className={styles.head}>
          <h3>③ Dán kết quả</h3>
          <span className={styles.hint}>dán nhiều lần được · gộp theo mã dữ kiện</span>
        </div>
        {pastes.length > 0 && (
          <div className={styles.scroll}>
            <table className={styles.table} data-testid="research-paste-log">
              <thead>
                <tr>
                  <th>Lần</th>
                  <th>Lúc</th>
                  <th>Kết quả</th>
                </tr>
              </thead>
              <tbody>
                {pastes.map((p) => (
                  <tr key={p.n} data-testid={`research-paste-log-${p.n}`}>
                    <td className={styles.num}>{p.n}</td>
                    <td className={styles.num}>{pasteTime(p.at)}</td>
                    <td>{describePaste(p)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <label className={styles.hint} htmlFor="research-paste-input">
          Câu trả lời của công cụ
        </label>
        <TextArea
          id="research-paste-input"
          className={styles.paste}
          value={text}
          onChange={(e) => {
            setText(e.target.value);
            setEmpty(false);
          }}
          rows={5}
          placeholder="Dán toàn bộ câu trả lời vào đây. Hệ thống tự tìm khối JSON, đọc từng dữ kiện, gộp theo mã rồi kiểm link."
          disabled={locked}
          data-testid="research-paste-input"
        />
        {empty && (
          <p className={styles.warn} role="alert" data-testid="research-paste-empty">
            Ô đang trống. Dán toàn bộ câu trả lời của công cụ; hệ thống tự tìm khối JSON.
          </p>
        )}
        {message && (
          <p className={message.tone === "ok" ? styles.ok : styles.warn} role={message.tone === "ok" ? "status" : "alert"} data-testid="research-paste-message">
            {message.text}
          </p>
        )}
        <div className={`${styles.actions} ${styles.actionsEnd}`}>
          <Button variant="primary" onClick={() => void read()} disabled={locked || busy} data-testid="research-paste-read">
            Đọc kết quả
          </Button>
        </div>
      </div>
    </Card>
  );
}
