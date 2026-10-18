import type { ReactNode } from "react";
import styles from "../pages/WizardSteps.module.css";

interface StepSectionProps {
  /** "1. Nội dung video": the number is the order of the decisions. */
  title: string;
  hint: string;
  testId: string;
  children: ReactNode;
}

/** A numbered group on a wizard step, decided top to bottom. */
export function StepSection({ title, hint, testId, children }: StepSectionProps) {
  return (
    <section className={styles.settingsSection} data-testid={testId}>
      <header className={styles.settingsSectionHead}>
        <h2>{title}</h2>
        <p>{hint}</p>
      </header>
      {children}
    </section>
  );
}
