import { useEffect, useState } from "react";
import type { SequenceTempo, SequenceUnit, VisualSequenceState } from "../../api/client";
import researchStyles from "../research/research.module.css";
import { FormField, Select, TextArea, TextInput } from "../ui";
import { FormDialog } from "../visualplan/PlanDialogs";
import styles from "./sequence.module.css";
import { TEMPOS, TEMPO_LABELS } from "./sequenceModel";

const same = (a: unknown, b: unknown) => JSON.stringify(a) === JSON.stringify(b);

interface SequenceItemDialogProps {
  open: boolean;
  onClose: () => void;
  state: VisualSequenceState;
  unit: SequenceUnit;
  onSave: (fields: Record<string, unknown>) => Promise<void>;
}

/**
 * Hand edit of one unit's sequence: its purpose in the flow, its pacing, the
 * transition into it and the motifs it carries. Only the fields the Creator
 * changed are sent, so only those become the Creator's and survive the AI
 * writing the beat again. The unit that opens the video has no transition.
 */
export function SequenceItemDialog({
  open,
  onClose,
  state,
  unit,
  onSave,
}: SequenceItemDialogProps) {
  const first = unit.previous_unit === "";
  const [purpose, setPurpose] = useState("");
  const [paceName, setPaceName] = useState("");
  const [tempo, setTempo] = useState<SequenceTempo | "">("");
  const [paceDescription, setPaceDescription] = useState("");
  const [trType, setTrType] = useState("");
  const [trDescription, setTrDescription] = useState("");
  const [via, setVia] = useState("");
  const [motifs, setMotifs] = useState<string[]>([]);

  useEffect(() => {
    if (!open) return;
    const it = unit.item;
    setPurpose(it?.purpose_in_flow ?? "");
    setPaceName(it?.pacing.name ?? "");
    setTempo(it?.pacing.tempo ?? "");
    setPaceDescription(it?.pacing.description ?? "");
    setTrType(it?.transition_from_previous?.type ?? "");
    setTrDescription(it?.transition_from_previous?.description ?? "");
    setVia(it?.transition_from_previous?.via_motif ?? "");
    setMotifs(it?.motif_refs ?? []);
    // The form starts from the item when it opens; a refresh of the screen behind must not reset typed text.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, unit.unit_id]);

  async function save() {
    const it = unit.item;
    const fields: Record<string, unknown> = {};
    const put = (key: string, next: unknown, prev: unknown) => {
      if (it === null || !same(next, prev)) fields[key] = next;
    };
    put("purpose_in_flow", purpose.trim(), it?.purpose_in_flow);
    put(
      "pacing",
      { name: paceName.trim(), tempo, description: paceDescription.trim() },
      it?.pacing,
    );
    put(
      "transition_from_previous",
      first ? null : { type: trType.trim(), description: trDescription.trim(), via_motif: via },
      it?.transition_from_previous,
    );
    put(
      "motif_refs",
      state.motifs.motifs.map((m) => m.motif_id).filter((id) => motifs.includes(id)),
      it?.motif_refs,
    );
    if (Object.keys(fields).length > 0) await onSave(fields);
  }

  const toggle = (id: string) =>
    setMotifs(motifs.includes(id) ? motifs.filter((m) => m !== id) : [...motifs, id]);

  return (
    <FormDialog
      open={open}
      onClose={onClose}
      title={`Sửa chuỗi ${unit.unit_id}`}
      confirmLabel="Lưu"
      confirmDisabled={tempo === ""}
      onConfirm={save}
      size="lg"
      testId="seq-item-dialog"
    >
      <FormField label="Vai trò trong mạch">
        <TextArea
          value={purpose}
          onChange={(e) => setPurpose(e.target.value)}
          rows={2}
          data-testid="seq-item-purpose"
        />
      </FormField>
      <div className={researchStyles.grid2}>
        <FormField label="Tên nhịp">
          <TextInput
            value={paceName}
            onChange={(e) => setPaceName(e.target.value)}
            data-testid="seq-item-pace-name"
          />
        </FormField>
        <FormField label="Tốc độ">
          <Select
            value={tempo}
            onChange={(e) => setTempo(e.target.value as SequenceTempo | "")}
            data-testid="seq-item-tempo"
          >
            <option value="">Chọn tốc độ</option>
            {TEMPOS.map((t) => (
              <option key={t} value={t}>
                {TEMPO_LABELS[t]}
              </option>
            ))}
          </Select>
        </FormField>
      </div>
      <FormField label="Mô tả nhịp">
        <TextArea
          value={paceDescription}
          onChange={(e) => setPaceDescription(e.target.value)}
          rows={2}
          data-testid="seq-item-pace-description"
        />
      </FormField>
      {first ? (
        <p className={researchStyles.hint}>Unit mở đầu video, không có chuyển cảnh vào.</p>
      ) : (
        <>
          <div className={researchStyles.grid2}>
            <FormField label={`Chuyển cảnh từ ${unit.previous_unit}`}>
              <TextInput
                value={trType}
                onChange={(e) => setTrType(e.target.value)}
                data-testid="seq-item-tr-type"
              />
            </FormField>
            <FormField label="Motif nối (không bắt buộc)">
              <Select
                value={via}
                onChange={(e) => setVia(e.target.value)}
                data-testid="seq-item-via"
              >
                <option value="">Không</option>
                {state.motifs.motifs.map((m) => (
                  <option key={m.motif_id} value={m.motif_id}>
                    {m.motif_id} · {m.name}
                  </option>
                ))}
              </Select>
            </FormField>
          </div>
          <FormField label="Mô tả chuyển cảnh">
            <TextArea
              value={trDescription}
              onChange={(e) => setTrDescription(e.target.value)}
              rows={2}
              data-testid="seq-item-tr-description"
            />
          </FormField>
        </>
      )}
      <FormField label="Motif có trong unit">
        <div className={styles.checks}>
          {state.motifs.motifs.length === 0 && (
            <span className={researchStyles.hint}>Chưa có motif.</span>
          )}
          {state.motifs.motifs.map((m) => (
            <label key={m.motif_id}>
              <input
                type="checkbox"
                checked={motifs.includes(m.motif_id)}
                onChange={() => toggle(m.motif_id)}
                data-testid={`seq-item-motif-${m.motif_id}`}
              />{" "}
              {m.motif_id} · {m.name}
            </label>
          ))}
        </div>
      </FormField>
      {via !== "" && !motifs.includes(via) && (
        <p className={researchStyles.error} role="status">
          Motif nối {via} phải có trong motif của unit này (và của {unit.previous_unit}).
        </p>
      )}
      <p className={researchStyles.hint}>
        Lưu xong, unit về “Chờ duyệt”. Trường bạn sửa được giữ khi AI viết lại beat.
      </p>
    </FormDialog>
  );
}
