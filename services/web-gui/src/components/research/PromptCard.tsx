import { useEffect, useState } from "react";
import { getResearchPrompt, type ResearchQuestion } from "../../api/client";
import { Button, Card } from "../ui";
import { errorText } from "./errorText";
import { PromptModal } from "./PromptModal";
import styles from "./research.module.css";

interface PromptCardProps {
  projectId: string;
  questions: ResearchQuestion[];
}

/**
 * Section 2 of the research step: the prompt the Creator takes to an outside
 * research tool. It is built from the brief and the questions, so it is read
 * again whenever a question changes. The card shows its start; the whole text
 * opens in a dialog that copies it.
 */
export function PromptCard({ projectId, questions }: PromptCardProps) {
  const [prompt, setPrompt] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [open, setOpen] = useState(false);

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
            <pre className={styles.promptBox} data-testid="research-prompt-preview">
              {prompt}
            </pre>
            <div className={styles.actions}>
              <Button variant="view" onClick={() => setOpen(true)} data-testid="research-prompt-open">
                Xem toàn bộ và sao chép
              </Button>
            </div>
            <p className={styles.hint}>
              Dán vào công cụ có tra web: ChatGPT (tìm kiếm hoặc Deep Research), Perplexity, Gemini Deep Research. Chép
              toàn bộ câu trả lời rồi dán vào ô ③.
            </p>
          </>
        )}
      </div>
      <PromptModal
        open={open}
        onClose={() => setOpen(false)}
        title="Prompt nghiên cứu"
        text={prompt}
        error={error}
        testId="research-prompt-modal"
      />
    </Card>
  );
}
