import type { InputHTMLAttributes } from "react";
import glass from "../../styles/glass.module.css";
import { useReadOnly } from "../../context/ReadOnlyContext";
import { ReadOnlyValue } from "./ReadOnlyValue";

type TextInputProps = InputHTMLAttributes<HTMLInputElement>;

/** Input types whose value reads fine as plain text; the rest (range, color,
 *  file...) stay controls and are locked by AppShell's fieldset. */
const TEXT_TYPES = new Set([undefined, "text", "search", "url", "email", "number", "tel"]);

export function TextInput({ className, ...rest }: TextInputProps) {
  const readOnly = useReadOnly();
  if (readOnly && TEXT_TYPES.has(rest.type)) {
    const value = rest.value ?? rest.defaultValue ?? "";
    return (
      <ReadOnlyValue
        value={String(value)}
        id={rest.id}
        testId={(rest as Record<string, unknown>)["data-testid"]}
      />
    );
  }
  return <input className={className ? `${glass.textInput} ${className}` : glass.textInput} {...rest} />;
}
