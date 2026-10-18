import { useEffect, useRef, useState, type ReactNode } from "react";
import { Button, Dialog, FormField, TextArea } from "../ui";
import { reachesLaterSteps, type UnitImpact } from "../../hooks/useUnitImpact";
import { ImpactList } from "./ImpactList";
import styles from "./units.module.css";

/** One stored text the dialog edits. */
export interface UnitEditField {
  name: string;
  label: string;
  value: string;
  rows?: number;
}

interface UnitEditModalProps {
  open: boolean;
  onClose: () => void;
  /** "Sửa tay · Creative brief". */
  title: string;
  fields: UnitEditField[];
  impact: UnitImpact;
  /** Saves the fields whose text changed; a thrown error stays in the dialog. */
  onSave: (changed: Record<string, string>) => Promise<void>;
  /** A line under the fields saying what a hand edit does to the unit. */
  note?: ReactNode;
  testId: string;
}

function valuesOf(fields: UnitEditField[]): Record<string, string> {
  return Object.fromEntries(fields.map((f) => [f.name, f.value]));
}

/**
 * "Sửa tay" for one unit (docs/ux-ui-design-rules.md, rule 4): the unit's
 * texts as a form, then what saving sends back in later steps, then "Huỷ" and
 * "Lưu bản sửa". The save button is orange when later work depends on the
 * unit, yellow when nothing does yet. What the Creator typed lives in the
 * dialog, so a refresh of the page behind it does not reset it.
 */
export function UnitEditModal({ open, onClose, title, fields, impact, onSave, note, testId }: UnitEditModalProps) {
  const [values, setValues] = useState<Record<string, string>>(() => valuesOf(fields));
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // The texts to start from are read when the dialog opens, never again while
  // it is open: a poll redrawing the page must not overwrite what is typed.
  const latestFields = useRef(fields);
  useEffect(() => {
    latestFields.current = fields;
  });
  useEffect(() => {
    if (!open) return;
    setValues(valuesOf(latestFields.current));
    setError(null);
  }, [open]);

  async function handleSave() {
    const changed: Record<string, string> = {};
    for (const field of latestFields.current) {
      const value = values[field.name] ?? "";
      if (value.trim() !== field.value.trim()) changed[field.name] = value;
    }
    if (Object.keys(changed).length === 0) {
      onClose();
      return;
    }
    setSaving(true);
    setError(null);
    try {
      await onSave(changed);
      onClose();
    } catch (err) {
      setError(err instanceof Error && err.message ? err.message : "Không lưu được bản sửa. Thử lại.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={title}
      hasInput
      size={fields.length > 4 ? "lg" : "md"}
      testId={testId}
      footer={
        <>
          <Button variant="ghost" onClick={onClose} data-testid={`${testId}-cancel`}>
            Huỷ
          </Button>
          <Button
            variant={reachesLaterSteps(impact) ? "impact" : "primary"}
            onClick={() => void handleSave()}
            disabled={saving || impact.items === null}
            data-testid={`${testId}-save`}
          >
            {saving ? "Đang lưu…" : "Lưu bản sửa"}
          </Button>
        </>
      }
    >
      {fields.map((field) => (
        <FormField key={field.name} label={field.label}>
          <TextArea
            value={values[field.name] ?? ""}
            onChange={(e) => setValues((prev) => ({ ...prev, [field.name]: e.target.value }))}
            rows={field.rows ?? 2}
            data-testid={`${testId}-field-${field.name}`}
          />
        </FormField>
      ))}
      {note && <p className={styles.note}>{note}</p>}
      <ImpactList head="Lưu bản sửa thì phần phụ thuộc sẽ làm lại:" impact={impact} testId={`${testId}-impact`} />
      {error && (
        <p className={styles.error} role="alert">
          {error}
        </p>
      )}
    </Dialog>
  );
}
