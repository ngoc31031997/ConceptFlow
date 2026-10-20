import styles from "./story.module.css";

/** The facts a piece of the story rests on, as small code tags (F01, F02). */
export function FactTags({ ids, testId }: { ids: string[]; testId?: string }) {
  if (ids.length === 0) return null;
  return (
    <span className={styles.tags} data-testid={testId}>
      {ids.map((id) => (
        <span key={id} className={styles.tag}>
          {id}
        </span>
      ))}
    </span>
  );
}
