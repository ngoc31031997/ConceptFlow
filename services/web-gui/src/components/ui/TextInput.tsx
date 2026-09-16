import type { InputHTMLAttributes } from "react";
import glass from "../../styles/glass.module.css";

type TextInputProps = InputHTMLAttributes<HTMLInputElement>;

export function TextInput({ className, ...rest }: TextInputProps) {
  return <input className={className ? `${glass.textInput} ${className}` : glass.textInput} {...rest} />;
}
