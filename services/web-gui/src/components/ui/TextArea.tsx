import type { TextareaHTMLAttributes } from "react";
import glass from "../../styles/glass.module.css";

type TextAreaProps = TextareaHTMLAttributes<HTMLTextAreaElement>;

export function TextArea({ className, ...rest }: TextAreaProps) {
  return <textarea className={className ? `${glass.textArea} ${className}` : glass.textArea} {...rest} />;
}
