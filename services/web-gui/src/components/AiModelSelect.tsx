import { useAuthoringModels } from "../hooks/useAuthoringModels";
import { useLlmStatus } from "../hooks/useLlmStatus";
import { Dropdown } from "./ui";
import styles from "./AiModelSelect.module.css";

interface AiModelSelectProps {
  /** The project the choice belongs to. */
  projectId: string;
  /** Which model of the project this chooses. */
  step: "brief" | "publish" | "research";
  /** What the AI does here ("Model viết brief"). */
  label: string;
  testId?: string;
}

/**
 * The model an AI call runs on, chosen at the button that starts it. The
 * choice is the project's own (saved on the server the moment it changes, so
 * the run reads it back from there). Until the Creator picks one it shows, and
 * the run uses, the default set in Cài đặt. Draws nothing when no model is
 * configured.
 */
export function AiModelSelect({ projectId, step, label, testId = "ai-model-select" }: AiModelSelectProps) {
  const llm = useLlmStatus();
  const { models, setModels } = useAuthoringModels(projectId);
  const options = llm?.models ?? [];
  if (options.length === 0) return null;

  const defaultModel = llm?.default_model ?? "";
  const listed = options.some((o) => o.id === defaultModel) || !defaultModel ? options : [...options, { id: defaultModel, label: defaultModel }];
  const value = models[step] || defaultModel || listed[0].id;

  return (
    <div className={styles.field} data-testid={`${testId}-field`}>
      <span className={styles.label}>{label}</span>
      <Dropdown
        aria-label={label}
        value={value}
        onChange={(next) => setModels({ ...models, [step]: next })}
        data-testid={testId}
        options={listed.map((o) => ({
          value: o.id,
          label: o.label,
          badge: o.id === defaultModel ? "Mặc định" : undefined,
        }))}
      />
    </div>
  );
}
