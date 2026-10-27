import { useState } from "react";
import { getFollowUpPrompt, type QuestionCoverage, type ResearchQuestion } from "../../api/client";
import { Button } from "../ui";
import { errorText } from "./errorText";
import { PromptModal } from "./PromptModal";
import styles from "./research.module.css";

interface FollowUpPromptProps {
  projectId: string;
  questions: ResearchQuestion[];
  coverage: QuestionCoverage[];
  /** Ids of the questions that have a fact whose link is still being checked. */
  checking: ReadonlySet<string>;
}

/**
 * Says how many questions still lack two independent sources and opens the
 * follow-up prompt that asks an outside research tool for another round on
 * them. While a link is being checked the count is not final, so it only
 * says so.
 */
export function FollowUpPrompt({ projectId, questions, coverage, checking }: FollowUpPromptProps) {
  const [open, setOpen] = useState(false);
  const [prompt, setPrompt] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const weak = coverage.filter((c) => c.weak);

  async function showFollowUp() {
    setOpen(true);
    setPrompt(null);
    setError(null);
    try {
      setPrompt(await getFollowUpPrompt(projectId));
    } catch (err) {
      setError(errorText(err));
    }
  }

  if (questions.length === 0) return null;

  return (
    <>
      {checking.size > 0 ? (
        <span className={styles.hint} role="status" data-testid="coverage-checking">
          Đang chờ kiểm link nên độ phủ chưa chốt.
        </span>
      ) : weak.length > 0 ? (
        <>
          <span className={styles.warn} data-testid="coverage-weak">
            {weak.length} câu hỏi thiếu nguồn
          </span>
          <Button variant="view" className={styles.sm} onClick={() => void showFollowUp()} data-testid="coverage-follow-up">
            Prompt bổ sung
          </Button>
        </>
      ) : (
        <span className={styles.ok} data-testid="coverage-all-sourced">
          ✓ Mọi câu hỏi đã đủ nguồn
        </span>
      )}
      <PromptModal
        open={open}
        onClose={() => setOpen(false)}
        title="Prompt bổ sung"
        text={prompt}
        error={error}
        testId="research-follow-up-modal"
      />
    </>
  );
}
