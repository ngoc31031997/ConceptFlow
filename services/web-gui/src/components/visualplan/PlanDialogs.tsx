import { useEffect, useState } from "react";
import type { PlanMethod, PlanRelationship, VisualPlanUnit } from "../../api/client";
import { visualErrorText } from "../../hooks/useVisualConcept";
import researchStyles from "../research/research.module.css";
import { Button, Dialog, FormField, Select, TextArea, TextInput } from "../ui";
import { METHODS, METHOD_LABELS, RELATIONSHIPS, RELATIONSHIP_LABELS, refsLabel, seconds } from "./planModel";
import styles from "./plan.module.css";

interface FormDialogProps {
  open: boolean;
  onClose: () => void;
  title: string;
  confirmLabel: string;
  confirmDisabled?: boolean;
  /** Runs the change; a throw is shown in the dialog and keeps it open. */
  onConfirm: () => Promise<void>;
  testId: string;
  size?: "sm" | "md" | "lg" | "full";
  children: React.ReactNode;
}

/** The shell of every hand-edit dialog of step 9: Huỷ, the main action, and the server's reason when it refuses. */
function FormDialog({ open, onClose, title, confirmLabel, confirmDisabled, onConfirm, testId, size, children }: FormDialogProps) {
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  useEffect(() => {
    if (!open) setError(null);
  }, [open]);
  async function confirm() {
    setSaving(true);
    setError(null);
    try {
      await onConfirm();
      onClose();
    } catch (err) {
      setError(visualErrorText(err));
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
      size={size}
      testId={testId}
      footer={
        <>
          <Button variant="ghost" onClick={onClose} data-testid={`${testId}-cancel`}>
            Huỷ
          </Button>
          <Button
            variant="primary"
            onClick={() => void confirm()}
            disabled={saving || confirmDisabled}
            data-testid={`${testId}-confirm`}
          >
            {confirmLabel}
          </Button>
        </>
      }
    >
      {children}
      {error && (
        <p className={researchStyles.error} role="alert" data-testid={`${testId}-error`}>
          {error}
        </p>
      )}
    </Dialog>
  );
}

const lines = (text: string): string[] =>
  text
    .split("\n")
    .map((l) => l.trim())
    .filter(Boolean);

const same = (a: unknown, b: unknown) => JSON.stringify(a) === JSON.stringify(b);

interface EditDialogProps {
  open: boolean;
  onClose: () => void;
  unit: VisualPlanUnit;
  onSave: (fields: Record<string, unknown>) => Promise<void>;
}

/**
 * Hand edit of one unit. Only the fields the Creator changed are sent, so
 * only those become the Creator's and survive the AI planning the beat again.
 */
export function PlanEditDialog({ open, onClose, unit, onSave }: EditDialogProps) {
  const [goal, setGoal] = useState("");
  const [summary, setSummary] = useState("");
  const [info, setInfo] = useState<{ item: string; priority: "must" | "nice" }[]>([]);
  const [text, setText] = useState("");
  const [method, setMethod] = useState<PlanMethod>("diagram");
  const [description, setDescription] = useState("");
  const [refs, setRefs] = useState("");
  const [relationship, setRelationship] = useState<PlanRelationship>("direct_illustration");
  const [planned, setPlanned] = useState("");
  const [reason, setReason] = useState("");
  const [risks, setRisks] = useState<{ fact_id: string; note: string }[]>([]);

  useEffect(() => {
    if (!open) return;
    setGoal(unit.communication_goal);
    setSummary(unit.narration_summary);
    setInfo(unit.essential_visual_info.map((i) => ({ ...i })));
    setText(unit.on_screen_text.join("\n"));
    setMethod(unit.visual_approach.method);
    setDescription(unit.visual_approach.description);
    setRefs(unit.visual_approach.concept_rule_refs.join("\n"));
    setRelationship(unit.narration_relationship);
    setPlanned(unit.duration.planned_s > 0 ? String(unit.duration.planned_s) : "");
    setReason(unit.duration.extra_reason);
    setRisks(unit.risk_notes.map((r) => ({ ...r })));
    // The form starts from the unit when it opens; a refresh of the screen behind must not reset typed text.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, unit.unit_id]);

  const plannedS = planned.trim() === "" ? 0 : Number(planned);
  const plannedValid = Number.isFinite(plannedS) && plannedS >= 0;
  const longer = plannedValid && plannedS > unit.narration_min_s;

  async function save() {
    const fields: Record<string, unknown> = {};
    const put = (key: string, next: unknown, prev: unknown) => {
      if (!same(next, prev)) fields[key] = next;
    };
    put("communication_goal", goal.trim(), unit.communication_goal);
    put("narration_summary", summary.trim(), unit.narration_summary);
    put(
      "essential_visual_info",
      info.filter((i) => i.item.trim()).map((i) => ({ item: i.item.trim(), priority: i.priority })),
      unit.essential_visual_info,
    );
    put("on_screen_text", lines(text), unit.on_screen_text);
    put(
      "visual_approach",
      { method, description: description.trim(), concept_rule_refs: lines(refs) },
      unit.visual_approach,
    );
    put("narration_relationship", relationship, unit.narration_relationship);
    put("duration", { planned_s: plannedS, extra_reason: longer ? reason.trim() : "" }, unit.duration);
    put(
      "risk_notes",
      risks.filter((r) => r.note.trim()).map((r) => ({ fact_id: r.fact_id, note: r.note.trim() })),
      unit.risk_notes,
    );
    if (Object.keys(fields).length > 0) await onSave(fields);
  }

  return (
    <FormDialog
      open={open}
      onClose={onClose}
      title={`Sửa ${unit.unit_id}`}
      confirmLabel="Lưu"
      confirmDisabled={!plannedValid || (longer && reason.trim() === "")}
      onConfirm={save}
      size="full"
      testId="plan-edit-dialog"
    >
      <FormField label="Mục tiêu truyền đạt">
        <TextArea value={goal} onChange={(e) => setGoal(e.target.value)} rows={3} data-testid="plan-edit-goal" />
      </FormField>
      <FormField label="Tóm tắt lời thoại">
        <TextArea value={summary} onChange={(e) => setSummary(e.target.value)} rows={3} data-testid="plan-edit-summary" />
      </FormField>
      <FormField label="Phương pháp hiển thị">
        <Select value={method} onChange={(e) => setMethod(e.target.value as PlanMethod)} data-testid="plan-edit-method">
          {METHODS.map((m) => (
            <option key={m} value={m}>
              {METHOD_LABELS[m]}
            </option>
          ))}
        </Select>
      </FormField>
      <FormField label="Mô tả cách hiển thị">
        <TextArea value={description} onChange={(e) => setDescription(e.target.value)} rows={8} data-testid="plan-edit-description" />
      </FormField>
      <FormField label="Quy tắc ngôn ngữ hình ảnh áp dụng (mỗi dòng một mã)">
        <TextArea value={refs} onChange={(e) => setRefs(e.target.value)} rows={8} data-testid="plan-edit-refs" />
      </FormField>
      <FormField label="Quan hệ với lời">
        <Select
          value={relationship}
          onChange={(e) => setRelationship(e.target.value as PlanRelationship)}
          data-testid="plan-edit-relationship"
        >
          {RELATIONSHIPS.map((r) => (
            <option key={r} value={r}>
              {RELATIONSHIP_LABELS[r]}
            </option>
          ))}
        </Select>
      </FormField>
      <div className={styles.rows}>
        <span className={styles.lbl}>Phải có trong hình</span>
        {info.map((row, i) => (
          <div key={i} className={styles.row}>
            <TextInput
              value={row.item}
              onChange={(e) => setInfo(info.map((r, j) => (j === i ? { ...r, item: e.target.value } : r)))}
              aria-label={`Thông tin hình ${i + 1}`}
            />
            <Select
              value={row.priority}
              onChange={(e) =>
                setInfo(info.map((r, j) => (j === i ? { ...r, priority: e.target.value as "must" | "nice" } : r)))
              }
              aria-label={`Mức ưu tiên ${i + 1}`}
            >
              <option value="must">Bắt buộc</option>
              <option value="nice">Nên có</option>
            </Select>
            <Button variant="ghost" onClick={() => setInfo(info.filter((_, j) => j !== i))}>
              Xoá
            </Button>
          </div>
        ))}
        <div>
          <Button variant="ghost" onClick={() => setInfo([...info, { item: "", priority: "must" }])}>
            Thêm dòng
          </Button>
        </div>
      </div>
      <FormField label="Chữ trên hình (mỗi dòng một cụm)">
        <TextArea value={text} onChange={(e) => setText(e.target.value)} rows={2} data-testid="plan-edit-text" />
      </FormField>
      {risks.length > 0 && (
        <div className={styles.rows}>
          <span className={styles.lbl}>Ghi chú rủi ro</span>
          {risks.map((r, i) => (
            <FormField key={r.fact_id} label={`Dữ kiện ${r.fact_id}`}>
              <TextArea
                value={r.note}
                onChange={(e) => setRisks(risks.map((x, j) => (j === i ? { ...x, note: e.target.value } : x)))}
                rows={2}
              />
            </FormField>
          ))}
        </div>
      )}
      <div className={styles.dur}>
        <span className={styles.durKey}>Tối thiểu theo lời</span>
        <span data-testid="plan-edit-min">
          {seconds(unit.narration_min_s)} <span className={researchStyles.hint}>· máy tính, không sửa</span>
        </span>
        <span className={styles.durKey}>Dự kiến (s)</span>
        <TextInput
          value={planned}
          onChange={(e) => setPlanned(e.target.value)}
          inputMode="decimal"
          placeholder="Để trống = bằng mức tối thiểu"
          aria-label="Thời lượng dự kiến"
          data-testid="plan-edit-planned"
        />
      </div>
      {!plannedValid && (
        <p className={researchStyles.error} role="alert">
          Thời lượng dự kiến phải là một số không âm.
        </p>
      )}
      {longer && (
        <FormField label="Lý do giữ hình lâu hơn lời (bắt buộc)">
          <TextArea value={reason} onChange={(e) => setReason(e.target.value)} rows={2} data-testid="plan-edit-reason" />
        </FormField>
      )}
      <p className={researchStyles.hint}>
        Lưu xong, unit về “Chờ duyệt”. Trường bạn sửa được giữ khi AI lập lại beat.
      </p>
    </FormDialog>
  );
}

interface SplitDialogProps {
  open: boolean;
  onClose: () => void;
  unit: VisualPlanUnit;
  onSplit: (atLine: string) => Promise<void>;
}

/** Cuts a unit before a line the Creator picks (never before the first). */
export function PlanSplitDialog({ open, onClose, unit, onSplit }: SplitDialogProps) {
  const choices = unit.lines.slice(1);
  const [at, setAt] = useState("");
  useEffect(() => {
    if (open) setAt(choices[0]?.id ?? "");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, unit.unit_id]);
  return (
    <FormDialog
      open={open}
      onClose={onClose}
      title={`Tách ${unit.unit_id}`}
      confirmLabel="Tách"
      confirmDisabled={choices.length === 0 || at === ""}
      onConfirm={() => onSplit(at)}
      testId="plan-split-dialog"
    >
      {choices.length === 0 ? (
        <p className={researchStyles.hint}>Unit chỉ có một câu nên không tách được.</p>
      ) : (
        <>
          <FormField label="Tách trước câu">
            <Select value={at} onChange={(e) => setAt(e.target.value)} data-testid="plan-split-at">
              {choices.map((l) => (
                <option key={l.id} value={l.id}>
                  {l.id} · {l.text.length > 70 ? `${l.text.slice(0, 70)}…` : l.text}
                </option>
              ))}
            </Select>
          </FormField>
          <p className={researchStyles.hint}>
            {unit.unit_id} giữ các câu trước điểm tách; unit mới nhận phần còn lại với mã mới. Cả hai về “Chờ
            duyệt”.
          </p>
        </>
      )}
    </FormDialog>
  );
}

interface MergeDialogProps {
  open: boolean;
  onClose: () => void;
  unit: VisualPlanUnit;
  /** The unit right after this one in the same beat. */
  next: VisualPlanUnit;
  onMerge: () => Promise<void>;
}

/** Merges the next unit of the same beat into this one. */
export function PlanMergeDialog({ open, onClose, unit, next, onMerge }: MergeDialogProps) {
  return (
    <FormDialog
      open={open}
      onClose={onClose}
      title={`Gộp ${unit.unit_id} với ${next.unit_id}`}
      confirmLabel="Gộp"
      onConfirm={onMerge}
      testId="plan-merge-dialog"
    >
      <p>
        Chỉ gộp được hai unit liền nhau trong cùng beat. Unit gộp có các câu{" "}
        {refsLabel([...unit.script_refs, ...next.script_refs])} và về “Chờ duyệt”.
      </p>
      <p className={researchStyles.hint}>
        Thời lượng tối thiểu tính lại theo các câu; phần mô tả hình giữ của {unit.unit_id}, bạn sửa sau.
      </p>
    </FormDialog>
  );
}

interface AnswerDialogProps {
  open: boolean;
  onClose: () => void;
  unitId: string;
  question: { id: string; question: string; answer: string } | null;
  onAnswer: (questionId: string, answer: string) => Promise<void>;
}

/** Answers one open question of a unit. */
export function PlanAnswerDialog({ open, onClose, unitId, question, onAnswer }: AnswerDialogProps) {
  const [answer, setAnswer] = useState("");
  useEffect(() => {
    if (open) setAnswer(question?.answer ?? "");
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, question?.id]);
  return (
    <FormDialog
      open={open}
      onClose={onClose}
      title={`Trả lời ${question?.id ?? ""} · ${unitId}`}
      confirmLabel="Lưu câu trả lời"
      confirmDisabled={answer.trim() === "" || !question}
      onConfirm={() => (question ? onAnswer(question.id, answer.trim()) : Promise.resolve())}
      testId="plan-answer-dialog"
    >
      <p>
        <b>{question?.question}</b>
      </p>
      <FormField label="Câu trả lời">
        <TextArea value={answer} onChange={(e) => setAnswer(e.target.value)} rows={3} data-testid="plan-answer-text" />
      </FormField>
    </FormDialog>
  );
}
