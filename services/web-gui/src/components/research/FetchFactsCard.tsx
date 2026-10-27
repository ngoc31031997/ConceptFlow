import { useEffect, useState } from "react";
import { getResearchPrompt, type PasteLog, type ResearchQuestion } from "../../api/client";
import type { AiBridgeModes, AiBridgeStage } from "../../hooks/useAiBridge";
import type { PasteMessage } from "../../hooks/useResearch";
import { usePresence } from "../../hooks/usePresence";
import glass from "../../styles/glass.module.css";
import { AiBridgeRunStatus } from "../AiBridgeRunStatus";
import { Button, Card, CopyButton, TextArea } from "../ui";
import { errorText } from "./errorText";
import { describePaste, pasteTime } from "./pasteText";
import { PromptModal } from "./PromptModal";
import styles from "./research.module.css";

/** The "Chạy bằng DeepSeek" control, shown when the AI Bridge extension is installed. */
export interface FetchFactsBridge {
  available: boolean;
  stage: AiBridgeStage | null;
  /** The DeepSeek modes the run turns on. */
  modes: AiBridgeModes;
  /** Why the last run gave no answer, or null. */
  error: string | null;
  /** False while the step is locked or a paste is being read. */
  enabled: boolean;
  onRun: (prompt: string) => void;
  onCancel: () => void;
}

interface FetchFactsCardProps {
  projectId: string;
  questions: ResearchQuestion[];
  pastes: PasteLog[];
  message: PasteMessage | null;
  busy: boolean;
  locked: boolean;
  /** The text in the box; the page holds it so a DeepSeek run can fill it. */
  text: string;
  onTextChange: (text: string) => void;
  /** Reads `text`; resolves true when it was read, so the box can be emptied. */
  onRead: () => Promise<boolean>;
  bridge: FetchFactsBridge;
}

/**
 * Section 2 of the research step: getting facts into the table. The prompt is
 * built from the brief and the questions and read again whenever a question
 * changes. The Creator either has DeepSeek run it (the answer lands in the box
 * and is read by itself) or copies it to an outside research tool and pastes
 * the whole answer back, as many times as they like. The server finds the JSON
 * block, merges the facts by code and logs each paste with what it read and
 * left out.
 */
export function FetchFactsCard({ projectId, questions, pastes, message, busy, locked, text, onTextChange, onRead, bridge }: FetchFactsCardProps) {
  const [prompt, setPrompt] = useState<string | null>(null);
  const [promptError, setPromptError] = useState<string | null>(null);
  const [promptOpen, setPromptOpen] = useState(false);
  const [logOpen, setLogOpen] = useState(false);
  const [empty, setEmpty] = useState(false);
  const log = usePresence(logOpen);

  const questionsKey = questions.map((q) => `${q.id}:${q.text}`).join("\n");
  useEffect(() => {
    if (!projectId || questionsKey === "") {
      setPrompt(null);
      setPromptError(null);
      return;
    }
    let cancelled = false;
    getResearchPrompt(projectId)
      .then((built) => {
        if (cancelled) return;
        setPrompt(built);
        setPromptError(null);
      })
      .catch((err) => {
        if (cancelled) return;
        setPrompt(null);
        setPromptError(errorText(err));
      });
    return () => {
      cancelled = true;
    };
  }, [projectId, questionsKey]);

  async function read() {
    if (text.trim() === "") {
      setEmpty(true);
      return;
    }
    setEmpty(false);
    await onRead();
  }

  return (
    <Card data-testid="research-fetch-card">
      <div className={styles.body}>
        <div className={styles.head}>
          <h3>② Lấy dữ kiện</h3>
        </div>
        {questions.length === 0 && (
          <p className={styles.hint} data-testid="research-prompt-empty">
            Thêm câu hỏi ở ① để dựng prompt.
          </p>
        )}
        {promptError && (
          <p className={styles.error} role="alert" data-testid="research-prompt-error">
            {promptError}
          </p>
        )}
        <div className={styles.split}>
          <div className={styles.option} data-testid="research-bridge">
            <b>Tự động</b>
            {bridge.available ? (
              <>
                <span className={styles.hint}>DeepSeek chạy prompt và tự điền câu trả lời vào ô bên dưới.</span>
                <div className={styles.actions}>
                  {bridge.stage === null ? (
                    <Button
                      variant="run"
                      onClick={() => prompt !== null && bridge.onRun(prompt)}
                      disabled={!bridge.enabled || prompt === null}
                      data-testid="research-bridge-run"
                    >
                      Chạy bằng DeepSeek
                    </Button>
                  ) : (
                    <AiBridgeRunStatus stage={bridge.stage} modes={bridge.modes} onCancel={bridge.onCancel} testId="research-bridge" />
                  )}
                </div>
              </>
            ) : (
              <p className={styles.hint} data-testid="research-bridge-missing">
                Cài extension ConceptFlow AI Bridge (tools/ai-bridge-extension) để chạy prompt bằng DeepSeek và tự dán kết quả.
              </p>
            )}
          </div>
          <span className={styles.or}>hoặc</span>
          <div className={styles.option}>
            <b>Tự làm</b>
            <span className={styles.hint}>
              Dán prompt vào công cụ có tra web (ChatGPT, Perplexity, Gemini Deep Research), rồi dán toàn bộ câu trả lời xuống ô dưới.
            </span>
            <div className={styles.actions}>
              <CopyButton text={prompt ?? ""} label="Sao chép prompt" what="prompt nghiên cứu" disabled={prompt === null} testId="research-prompt-quick-copy" />
              <Button variant="view" className={styles.sm} onClick={() => setPromptOpen(true)} disabled={prompt === null} data-testid="research-prompt-open">
                Xem toàn bộ
              </Button>
            </div>
          </div>
        </div>
        {bridge.error && (
          <p className={styles.warn} role="alert" data-testid="research-bridge-error">
            {bridge.error}
          </p>
        )}
        <label className={styles.hint} htmlFor="research-paste-input">
          Câu trả lời của công cụ
        </label>
        <TextArea
          id="research-paste-input"
          className={styles.paste}
          value={text}
          onChange={(e) => {
            onTextChange(e.target.value);
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
        <div className={styles.actions}>
          {pastes.length > 0 && (
            <Button
              variant="ghost"
              className={styles.sm}
              onClick={() => setLogOpen((open) => !open)}
              aria-expanded={logOpen}
              data-testid="research-paste-log-toggle"
            >
              {logOpen ? "▾" : "▸"} Đã dán {pastes.length} lần
            </Button>
          )}
          <span className={styles.grow} />
          <Button variant="primary" onClick={() => void read()} disabled={locked || busy} data-testid="research-paste-read">
            Đọc kết quả
          </Button>
        </div>
        {log.mounted && pastes.length > 0 && (
          <div className={`${styles.scroll} ${log.closing ? glass.revealOut : glass.reveal}`}>
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
      </div>
      <PromptModal
        open={promptOpen}
        onClose={() => setPromptOpen(false)}
        title="Prompt nghiên cứu"
        text={prompt}
        error={promptError}
        testId="research-prompt-modal"
      />
    </Card>
  );
}
