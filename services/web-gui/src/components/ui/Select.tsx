import { Children, isValidElement, type ReactNode, type SelectHTMLAttributes } from "react";
import glass from "../../styles/glass.module.css";
import { useReadOnly } from "../../context/ReadOnlyContext";
import { ReadOnlyValue } from "./ReadOnlyValue";

type SelectProps = SelectHTMLAttributes<HTMLSelectElement>;

/** The visible label of the <option> whose value is selected. */
function selectedLabel(children: ReactNode, value: unknown): string {
  let found = "";
  Children.forEach(children, (child) => {
    if (!isValidElement<{ value?: unknown; children?: ReactNode }>(child)) return;
    if (child.type === "option" && String(child.props.value ?? "") === String(value ?? "")) {
      found = Children.toArray(child.props.children).join("");
    } else if (child.props.children) {
      const nested = selectedLabel(child.props.children, value);
      if (nested) found = nested;
    }
  });
  return found;
}

export function Select({ className, ...rest }: SelectProps) {
  const readOnly = useReadOnly();
  if (readOnly) {
    return (
      <ReadOnlyValue
        value={selectedLabel(rest.children, rest.value ?? rest.defaultValue)}
        id={rest.id}
        testId={(rest as Record<string, unknown>)["data-testid"]}
      />
    );
  }
  return <select className={className ? `${glass.select} ${className}` : glass.select} {...rest} />;
}
