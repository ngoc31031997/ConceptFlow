import type { ReactNode } from "react";
import styles from "./FormField.module.css";

interface FormFieldProps {
  label: ReactNode;
  children: ReactNode;
  className?: string;
}

/**
 * Label + control wrapper shared by every form field in the app, so field
 * spacing/label styling doesn't drift page to page (see DESIGN_SYSTEM.md).
 */
export function FormField({ label, children, className }: FormFieldProps) {
  return (
    <label className={className ? `${styles.field} ${className}` : styles.field}>
      <span className={styles.label}>{label}</span>
      {children}
    </label>
  );
}
