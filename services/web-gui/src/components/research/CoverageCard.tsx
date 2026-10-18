import { useState } from "react";
import { getFollowUpPrompt, type QuestionCoverage, type ResearchQuestion } from "../../api/client";
import { Button, Card } from "../ui";
import { CoverageChip } from "./CoverageChip";
import { errorText } from "./errorText";
import { PromptModal } from "./PromptModal";
import styles from "./research.module.css";

interface CoverageCardProps {
  projectId: string;
  questions: ResearchQuestion[];
  coverage: QuestionCoverage[];
}

/**
 * Section 5 of the research step: how well each question is sourced. Every
 * question should have two independent sites whose quote was found; the ones
 * that do not, or that hold a fact whose quote was not found, go into a
 * follow-up prompt for another round.
 */
export function CoverageCard({ projectId, questions, coverage }: CoverageCardProps) {
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

  return (
    <Card data-testid="research-coverage-card">
      <div className={styles.body}>
        <div className={styles.head}>
          <h3>⑤ Độ phủ câu hỏi</h3>
          <span className={styles.hint}>mỗi câu hỏi nên có ít nhất 2 nguồn độc lập khớp</span>
        </div>
        {questions.length === 0 ? (
          <p className={styles.hint}>Chưa có câu hỏi nào.</p>
        ) : (
          <div className={styles.scroll}>
            <table className={styles.table}>
              <tbody>
                {questions.map((q) => {
                  const c = coverage.find((x) => x.question === q.id);
                  return (
                    <tr key={q.id} data-testid={`coverage-row-${q.id}`}>
                      <td className={styles.code}>{q.id}</td>
                      <td>{q.text}</td>
                      <td>
                        <CoverageChip coverage={c} testId={`coverage-chip-${q.id}`} />
                        {c && c.badFacts.length > 0 && (
                          <span className={`${styles.warn} ${styles.sub}`} data-testid={`coverage-bad-${q.id}`}>
                            {c.badFacts.join(", ")} không khớp
                          </span>
                        )}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
        {questions.length > 0 &&
          (weak.length > 0 ? (
            <div className={styles.actions}>
              <Button variant="view" className={styles.sm} onClick={() => void showFollowUp()} data-testid="coverage-follow-up">
                Xem prompt bổ sung cho {weak.length} câu hỏi
              </Button>
            </div>
          ) : (
            <p className={styles.ok} data-testid="coverage-all-sourced">
              ✓ Mọi câu hỏi đã đủ nguồn
            </p>
          ))}
      </div>
      <PromptModal
        open={open}
        onClose={() => setOpen(false)}
        title="Prompt bổ sung"
        text={prompt}
        error={error}
        testId="research-follow-up-modal"
      />
    </Card>
  );
}
