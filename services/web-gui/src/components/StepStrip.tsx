import styles from "./StepStrip.module.css";

/** Where the step stands: nothing done yet, waiting for the Creator, AI running, or done. */
export type StepStripTone = "todo" | "wait" | "run" | "ok";

export interface StepStripProps {
  tone: StepStripTone;
  /** The state in a few words ("Chờ bạn duyệt brief"). */
  label: string;
  /** What to do next, or what it means. */
  message: string;
}

const TONE_CLASS: Record<StepStripTone, string> = {
  todo: styles.todo,
  wait: styles.wait,
  run: styles.run,
  ok: styles.ok,
};

/** The one line under a step's heading that says where the step stands and what comes next. */
export function StepStrip({ tone, label, message }: StepStripProps) {
  return (
    <div className={styles.strip} data-testid="step-strip" data-tone={tone} role="status">
      <span className={`${styles.chip} ${TONE_CLASS[tone]}`}>{label}</span>
      <span className={styles.message}>{message}</span>
    </div>
  );
}
