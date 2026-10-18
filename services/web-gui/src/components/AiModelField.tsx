import { useLlmStatus } from "../hooks/useLlmStatus";
import { Dropdown } from "./ui";
import styles from "./AiModelSelect.module.css";

interface AiModelFieldProps {
  /** What the AI does here ("Model AI vẽ hình"). */
  label: string;
  /** The chosen model id; "" is the server default. */
  value: string;
  onChange: (model: string) => void;
  disabled?: boolean;
  testId: string;
}

/**
 * A model chosen next to the button that starts an AI call, for a call that
 * belongs to no project (a drawing in the illustration library). The list is
 * the server's catalogue; "" runs on its default. Draws nothing when no model
 * is configured.
 */
export function AiModelField({ label, value, onChange, disabled, testId }: AiModelFieldProps) {
  const llm = useLlmStatus();
  const options = llm?.models ?? [];
  if (options.length === 0) return null;
  const defaultModel = llm?.default_model ?? "";
  const listed = options.some((o) => o.id === defaultModel) || !defaultModel ? options : [...options, { id: defaultModel, label: defaultModel }];
  return (
    <div className={styles.field} data-testid={`${testId}-field`}>
      <span className={styles.label}>{label}</span>
      <Dropdown
        aria-label={label}
        value={value || defaultModel || listed[0].id}
        onChange={onChange}
        disabled={disabled}
        data-testid={testId}
        options={listed.map((o) => ({ value: o.id, label: o.label, badge: o.id === defaultModel ? "Mặc định" : undefined }))}
      />
    </div>
  );
}
