import { useEffect, useState } from "react";
import { Button, Dialog, FormField, TextArea, TextInput } from "../ui";
import { asList, asObj, asText, emptyOf, type FieldSpec } from "./conceptSchema";
import researchStyles from "../research/research.module.css";
import styles from "./visual.module.css";

interface EditorProps {
  spec: FieldSpec;
  value: unknown;
  onChange: (next: unknown) => void;
  testId: string;
}

/** The editor of one field, by its kind; rows and objects edit their own fields. */
function FieldEditor({ spec, value, onChange, testId }: EditorProps) {
  switch (spec.kind) {
    case "text":
      return (
        <TextInput
          value={asText(value)}
          onChange={(e) => onChange(e.target.value)}
          data-testid={testId}
        />
      );
    case "longtext":
      return (
        <TextArea
          rows={2}
          value={asText(value)}
          onChange={(e) => onChange(e.target.value)}
          data-testid={testId}
        />
      );
    case "number":
      return (
        <TextInput
          type="number"
          step="any"
          value={asText(value)}
          onChange={(e) => onChange(e.target.value === "" ? 0 : Number(e.target.value))}
          data-testid={testId}
        />
      );
    case "bool":
      return (
        <input
          type="checkbox"
          checked={value === true}
          onChange={(e) => onChange(e.target.checked)}
          data-testid={testId}
        />
      );
    case "enum":
      return (
        <select
          className={researchStyles.sm}
          value={asText(value)}
          onChange={(e) => onChange(e.target.value)}
          data-testid={testId}
        >
          {spec.options.map((o) => (
            <option key={o} value={o}>
              {o}
            </option>
          ))}
        </select>
      );
    case "lines":
      return (
        <TextArea
          rows={3}
          value={asList(value).map(asText).join("\n")}
          onChange={(e) =>
            onChange(
              e.target.value
                .split("\n")
                .map((l) => l.trim())
                .filter(Boolean),
            )
          }
          placeholder="Mỗi dòng một mục"
          data-testid={testId}
        />
      );
    case "object": {
      const obj = asObj(value);
      return (
        <div className={styles.formGrid}>
          {spec.fields.map((f) => (
            <FormField key={f.key} label={f.label}>
              <FieldEditor
                spec={f}
                value={obj[f.key]}
                onChange={(v) => onChange({ ...obj, [f.key]: v })}
                testId={`${testId}-${f.key}`}
              />
            </FormField>
          ))}
        </div>
      );
    }
    case "rows": {
      const rows = asList(value).map(asObj);
      const set = (i: number, row: Record<string, unknown>) =>
        onChange(rows.map((r, j) => (j === i ? row : r)));
      return (
        <div className={styles.formRows}>
          {rows.map((row, i) => (
            <div key={i} className={styles.formRow} data-testid={`${testId}-row-${i}`}>
              <div className={styles.formGrid}>
                {spec.columns.map((c) => (
                  <FormField key={c.key} label={c.label}>
                    <FieldEditor
                      spec={c}
                      value={row[c.key]}
                      onChange={(v) => set(i, { ...row, [c.key]: v })}
                      testId={`${testId}-${i}-${c.key}`}
                    />
                  </FormField>
                ))}
              </div>
              <div className={researchStyles.actionsEnd}>
                <Button
                  variant="dangerGhost"
                  onClick={() => onChange(rows.filter((_, j) => j !== i))}
                  data-testid={`${testId}-remove-${i}`}
                >
                  Bỏ dòng này
                </Button>
              </div>
            </div>
          ))}
          <div>
            <Button
              variant="ghost"
              onClick={() =>
                onChange([
                  ...rows,
                  Object.fromEntries(spec.columns.map((c) => [c.key, emptyOf(c)])),
                ])
              }
              data-testid={`${testId}-add`}
            >
              + Thêm dòng
            </Button>
          </div>
        </div>
      );
    }
  }
}

interface PartEditDialogProps {
  open: boolean;
  onClose: () => void;
  title: string;
  schema: readonly FieldSpec[];
  content: Record<string, unknown> | null;
  /** Saves the fields that changed; rejects with the server's reason. */
  onSave: (fields: Record<string, unknown>) => Promise<void>;
  /** The part is approved and goes back to review on save. */
  approved: boolean;
  testId: string;
}

/**
 * "Sửa tay" of one part: a form built from the part's schema. Only the fields
 * that changed are sent; they become the Creator's, so the AI keeps them when it
 * rewrites the part.
 */
export function PartEditDialog({
  open,
  onClose,
  title,
  schema,
  content,
  onSave,
  approved,
  testId,
}: PartEditDialogProps) {
  const [draft, setDraft] = useState<Record<string, unknown>>({});
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    const start: Record<string, unknown> = {};
    for (const f of schema) start[f.key] = content?.[f.key] ?? emptyOf(f);
    setDraft(start);
    setError(null);
    // Re-seeded only when the dialog opens, so a refresh behind it keeps what was typed.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  const changed = schema
    .map((f) => f.key)
    .filter(
      (k) =>
        JSON.stringify(draft[k]) !==
        JSON.stringify(content?.[k] ?? emptyOf(schema.find((f) => f.key === k)!)),
    );

  async function save() {
    setSaving(true);
    setError(null);
    try {
      await onSave(Object.fromEntries(changed.map((k) => [k, draft[k]])));
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Không lưu được.");
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
      size="full"
      testId={testId}
      footer={
        <>
          <Button variant="ghost" onClick={onClose} data-testid={`${testId}-cancel`}>
            Huỷ
          </Button>
          <Button
            variant={approved ? "impact" : "primary"}
            onClick={() => void save()}
            disabled={saving || changed.length === 0}
            data-testid={`${testId}-save`}
          >
            Lưu bản sửa
          </Button>
        </>
      }
    >
      {schema.map((f) => (
        <FormField key={f.key} label={f.label}>
          <FieldEditor
            spec={f}
            value={draft[f.key]}
            onChange={(v) => setDraft((d) => ({ ...d, [f.key]: v }))}
            testId={`${testId}-${f.key}`}
          />
        </FormField>
      ))}
      <p className={researchStyles.hint}>
        Phần bạn sửa thành của bạn: AI giữ nguyên khi làm lại, trừ khi bạn ghi chú yêu cầu đổi. Lưu
        xong phần này về “chờ duyệt”
        {approved ? " (đang đã duyệt)" : ""}.
      </p>
      {error && (
        <p className={researchStyles.error} role="alert" data-testid={`${testId}-error`}>
          {error}
        </p>
      )}
    </Dialog>
  );
}
