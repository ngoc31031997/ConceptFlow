import { forwardRef, type TextareaHTMLAttributes } from "react";
import glass from "../../styles/glass.module.css";
import { useReadOnly } from "../../context/ReadOnlyContext";
import { ReadOnlyValue } from "./ReadOnlyValue";

type TextAreaProps = TextareaHTMLAttributes<HTMLTextAreaElement>;

// forwardRef: a caller may need to focus the box (CR-045 — the redraw note).
export const TextArea = forwardRef<HTMLTextAreaElement, TextAreaProps>(function TextArea({ className, ...rest }, ref) {
  const readOnly = useReadOnly();
  if (readOnly) {
    const value = rest.value ?? rest.defaultValue ?? "";
    return (
      <ReadOnlyValue
        block
        value={String(value)}
        id={rest.id}
        testId={(rest as Record<string, unknown>)["data-testid"]}
      />
    );
  }
  return <textarea ref={ref} className={className ? `${glass.textArea} ${className}` : glass.textArea} {...rest} />;
});
