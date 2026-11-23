import styles from "../airun/AiRun.module.css";

/** What applying a run of the research-question writer would store: the new questions. */
export function QuestionsRunPreview({ preview }: { preview: unknown }) {
  const raw = (preview as { questions?: unknown } | null)?.questions;
  const questions = Array.isArray(raw) ? raw.filter((q): q is string => typeof q === "string") : [];
  return (
    <div className={styles.preview} data-testid="ai-run-preview">
      <p className={styles.previewHead}>{questions.length} câu hỏi mới sẽ được lưu.</p>
      <ol className={styles.previewList}>
        {questions.map((q, i) => (
          <li key={i}>{q}</li>
        ))}
      </ol>
    </div>
  );
}
