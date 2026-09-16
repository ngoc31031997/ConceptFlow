import type { SelectHTMLAttributes } from "react";
import glass from "../../styles/glass.module.css";

type SelectProps = SelectHTMLAttributes<HTMLSelectElement>;

export function Select({ className, ...rest }: SelectProps) {
  return <select className={className ? `${glass.select} ${className}` : glass.select} {...rest} />;
}
