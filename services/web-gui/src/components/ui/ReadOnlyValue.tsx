import glass from "../../styles/glass.module.css";

interface ReadOnlyValueProps {
  value: string;
  className?: string;
  /** Tall text (a textarea) vs one line (an input/select). */
  block?: boolean;
  testId?: unknown;
  id?: string;
}

/** A form field's value drawn as read-only text. */
export function ReadOnlyValue({ value, className, block, testId, id }: ReadOnlyValueProps) {
  const cls = [glass.readOnlyValue, block ? glass.readOnlyBlock : "", className].filter(Boolean).join(" ");
  return (
    <div
      id={id}
      className={cls}
      data-testid={typeof testId === "string" ? testId : undefined}
      data-readonly="true"
      role="textbox"
      aria-readonly="true"
      aria-multiline={block || undefined}
      tabIndex={0}
    >
      {value.trim() ? value : <span className={glass.readOnlyEmpty}>(trống)</span>}
    </div>
  );
}
