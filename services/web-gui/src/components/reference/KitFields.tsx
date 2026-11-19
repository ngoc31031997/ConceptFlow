import { FormField, Select, TextInput } from "../ui";
import {
  KIT_FIELD_GROUPS,
  KIT_FOCUS_LAYERS,
  kitFieldProblem,
  type KitDraft,
  type KitFieldDef,
} from "./kitForm";
import styles from "../../pages/ReferenceLibraryPage.module.css";

interface KitFieldsProps {
  draft: KitDraft;
  onChange: (path: string, value: string) => void;
}

function KitField({
  field,
  value,
  onChange,
}: {
  field: KitFieldDef;
  value: string;
  onChange: (value: string) => void;
}) {
  const testId = `reference-kit-${field.path}`;
  const problem = kitFieldProblem(field, value);
  return (
    <div>
      <FormField label={field.label}>
        {field.kind === "focus" ? (
          <Select
            value={value}
            onChange={(e) => onChange(e.target.value)}
            data-testid={testId}
          >
            {KIT_FOCUS_LAYERS.map((l) => (
              <option key={l.value} value={l.value}>
                {l.label}
              </option>
            ))}
          </Select>
        ) : (
          <TextInput
            value={value}
            onChange={(e) => onChange(e.target.value)}
            inputMode={field.kind === "color" ? "text" : "decimal"}
            aria-invalid={problem !== null}
            data-testid={testId}
          />
        )}
      </FormField>
      {problem && (
        <p className={styles.error} role="alert" data-testid={`${testId}-problem`}>
          {problem}
        </p>
      )}
    </div>
  );
}

/** Every number of a kit as an editable field, grouped by what it controls. */
export function KitFields({ draft, onChange }: KitFieldsProps) {
  return (
    <>
      {KIT_FIELD_GROUPS.map((group) => (
        <div key={group.title}>
          <h3 className={styles.detailHead}>{group.title}</h3>
          <div className={styles.grid}>
            {group.fields.map((f) => (
              <KitField
                key={f.path}
                field={f}
                value={draft[f.path] ?? ""}
                onChange={(v) => onChange(f.path, v)}
              />
            ))}
          </div>
        </div>
      ))}
    </>
  );
}
