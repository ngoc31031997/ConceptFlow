import { defaultModelFor, useLlmStatus } from "../hooks/useLlmStatus";
import { Dropdown } from "./ui";
import { LOCAL_CODE_MODEL_WARNING } from "./localModelWarning";
import styles from "./AiModelSelect.module.css";

interface AiModelFieldProps {
  /** What the AI does here ("Model AI vẽ hình"). */
  label: string;
  /** The chosen model id; "" is the default saved for the drawings. */
  value: string;
  onChange: (model: string) => void;
  disabled?: boolean;
  testId: string;
}

/**
 * A model chosen next to the button that starts an AI call, for a call that
 * belongs to no project (a drawing in the illustration library). The list is
 * the server's catalogue; "" runs on the default saved for the drawings. Warns when the chosen model runs on this machine, since
 * drawings are code. Draws nothing when no model is configured.
 */
export function AiModelField({ label, value, onChange, disabled, testId }: AiModelFieldProps) {
  const llm = useLlmStatus();
  const options = llm?.models ?? [];
  if (options.length === 0) return null;
  const defaultModel = defaultModelFor(llm, "illustrations") || options[0].id;
  const chosen = value || defaultModel;
  const local = options.find((o) => o.id === chosen)?.local === true;
  return (
    <div className={styles.field} data-testid={`${testId}-field`}>
      <span className={styles.label}>{label}</span>
      <Dropdown
        aria-label={label}
        value={chosen}
        onChange={onChange}
        disabled={disabled}
        data-testid={testId}
        options={options.map((o) => ({ value: o.id, label: o.label, badge: o.id === defaultModel ? "Mặc định" : undefined }))}
      />
      {local && (
        <span className={styles.warn} data-testid={`${testId}-local-warning`}>
          {LOCAL_CODE_MODEL_WARNING}
        </span>
      )}
    </div>
  );
}
