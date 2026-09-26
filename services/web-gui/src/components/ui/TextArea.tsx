import type { TextareaHTMLAttributes } from "react";
import glass from "../../styles/glass.module.css";
import { useReadOnly } from "../../context/ReadOnlyContext";
import { ReadOnlyValue } from "./ReadOnlyValue";

type TextAreaProps = TextareaHTMLAttributes<HTMLTextAreaElement>;

export function TextArea({ className, ...rest }: TextAreaProps) {
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
  return <textarea className={className ? `${glass.textArea} ${className}` : glass.textArea} {...rest} />;
}
