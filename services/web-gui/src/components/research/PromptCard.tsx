import { useEffect, useState } from "react";
import { getResearchPrompt, type ResearchQuestion } from "../../api/client";
import type { AiBridgeModes, AiBridgeStage } from "../../hooks/useAiBridge";
import { AiBridgeRunStatus } from "../AiBridgeRunStatus";
import { Button, Card, CopyBox } from "../ui";
import { errorText } from "./errorText";
import styles from "./research.module.css";

/** The "Chạy bằng DeepSeek" control, shown when the AI Bridge extension is installed. */
export interface PromptCardBridge {
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

interface PromptCardProps {
  projectId: string;
  questions: ResearchQuestion[];
  bridge: PromptCardBridge;
}


/**
 * Section 2 of the research step: the prompt the Creator takes to an outside
 * research tool. It is built from the brief and the questions, so it is read
 * again whenever a question changes. The card renders the whole text in a
 * scrollable box with a copy button. With the AI Bridge extension installed it
 * can also run the prompt on DeepSeek; the answer goes to section 3.
 */
export function PromptCard({ projectId, questions, bridge }: PromptCardProps) {
  const [prompt, setPrompt] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const questionsKey = questions.map((q) => `${q.id}:${q.text}`).join("\n");
  useEffect(() => {
    if (!projectId || questionsKey === "") {
      setPrompt(null);
      setError(null);
      return;
    }
    let cancelled = false;
    getResearchPrompt(projectId)
      .then((text) => {
        if (cancelled) return;
        setPrompt(text);
        setError(null);
      })
      .catch((err) => {
        if (cancelled) return;
        setPrompt(null);
        setError(errorText(err));
      });
    return () => {
      cancelled = true;
    };
  }, [projectId, questionsKey]);

  return (
    <Card data-testid="research-prompt-card">
      <div className={styles.body}>
        <div className={styles.head}>
          <h3>② Prompt nghiên cứu</h3>
          <span className={styles.hint}>sinh từ brief và câu hỏi ở trên, cập nhật khi bạn sửa</span>
        </div>
        {questions.length === 0 && (
          <p className={styles.hint} data-testid="research-prompt-empty">
            Thêm câu hỏi ở ① để dựng prompt.
          </p>
        )}
        {error && (
          <p className={styles.error} role="alert" data-testid="research-prompt-error">
            {error}
          </p>
        )}
        {prompt !== null && (
          <>
            <CopyBox text={prompt} what="prompt nghiên cứu" copyTestId="research-prompt-quick-copy">
              <pre className={styles.promptBox} data-testid="research-prompt-preview">
                {prompt}
              </pre>
            </CopyBox>
            <p className={styles.hint}>
              Dán vào công cụ có tra web: ChatGPT (tìm kiếm hoặc Deep Research), Perplexity, Gemini Deep Research. Chép
              toàn bộ câu trả lời rồi dán vào ô ③.
            </p>
            {bridge.available ? (
              <div data-testid="research-bridge">
                {bridge.stage === null ? (
                  <div className={styles.actions}>
                    <Button variant="run" onClick={() => bridge.onRun(prompt)} disabled={!bridge.enabled} data-testid="research-bridge-run">
                      Chạy bằng DeepSeek
                    </Button>
                  </div>
                ) : (
                  <AiBridgeRunStatus stage={bridge.stage} modes={bridge.modes} onCancel={bridge.onCancel} testId="research-bridge" />
                )}
              </div>
            ) : (
              <p className={styles.hint} data-testid="research-bridge-missing">
                Cài extension ConceptFlow AI Bridge (tools/ai-bridge-extension) để chạy prompt bằng DeepSeek và tự dán kết quả.
              </p>
            )}
            {bridge.error && (
              <p className={styles.warn} role="alert" data-testid="research-bridge-error">
                {bridge.error}
              </p>
            )}
          </>
        )}
      </div>
    </Card>
  );
}
