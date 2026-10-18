import { useState } from "react";
import { BRIEF_FIELDS, BRIEF_UNIT, type BriefField, type CreativeBrief } from "../api/client";
import type { CreativeBriefControls } from "../hooks/useCreativeBrief";
import { useUnitImpact } from "../hooks/useUnitImpact";
import { AiModelSelect } from "./AiModelSelect";
import { AiOperationCard } from "./AiOperationCard";
import { Button, Card } from "./ui";
import { UnitEditModal, type UnitEditField } from "./units/UnitEditModal";
import { UnitRedoModal } from "./units/UnitRedoModal";
import { UnitReviewBar } from "./units/UnitReviewBar";
import { UnitStatusChip } from "./units/UnitStatusChip";
import unitStyles from "./units/units.module.css";
import styles from "./CreativeBriefCard.module.css";

/** Each field's label on the card (short, under its group) and in the edit dialog (whole). */
const FIELD_LABELS: Record<BriefField, { card: string; full: string; rows: number }> = {
  question: { card: "Câu hỏi trung tâm cần trả lời", full: "Câu hỏi trung tâm cần trả lời", rows: 2 },
  audience: { card: "Khán giả", full: "Khán giả", rows: 2 },
  know: { card: "biết", full: "Sau video, người xem sẽ biết", rows: 3 },
  understand: { card: "hiểu", full: "Sau video, người xem sẽ hiểu", rows: 3 },
  feel: { card: "cảm thấy", full: "Sau video, người xem sẽ cảm thấy", rows: 3 },
  tone: { card: "Giọng điệu", full: "Giọng điệu", rows: 2 },
  out_of_scope: { card: "Ngoài phạm vi · video không nói tới", full: "Ngoài phạm vi (video không nói tới)", rows: 2 },
};

const EMPTY_BRIEF = Object.fromEntries(BRIEF_FIELDS.map((f) => [f, ""])) as CreativeBrief;

interface CreativeBriefCardProps {
  projectId: string;
  /** The project exists on the server; before that there is no brief to load. */
  created: boolean;
  brief: CreativeBriefControls;
  /** A language model is configured, so the AI can write and redo the brief. */
  aiAvailable: boolean;
}

/**
 * Section 2 of the idea step: the creative brief as one review unit. The
 * fields read as text; "Sửa tay" and "Làm lại kèm ghi chú" open their dialogs
 * (docs/ux-ui-design-rules.md, rule 4), each with the later work it sends back.
 * Without a language model the Creator writes the brief and approves it.
 */
export function CreativeBriefCard({ projectId, created, brief, aiAvailable }: CreativeBriefCardProps) {
  const [editOpen, setEditOpen] = useState(false);
  const [redoOpen, setRedoOpen] = useState(false);
  const editImpact = useUnitImpact(projectId, BRIEF_UNIT, editOpen);
  const redoImpact = useUnitImpact(projectId, BRIEF_UNIT, redoOpen);

  const unit = brief.state?.unit ?? null;
  const content = unit?.content ?? EMPTY_BRIEF;
  const missing = BRIEF_FIELDS.filter((f) => !content[f].trim());
  const editFields: UnitEditField[] = BRIEF_FIELDS.map((f) => ({
    name: f,
    label: FIELD_LABELS[f].full,
    value: content[f],
    rows: FIELD_LABELS[f].rows,
  }));

  const chip = brief.running ? (
    <UnitStatusChip state="running" testId="brief-status" />
  ) : unit ? (
    <UnitStatusChip state={unit.status} testId="brief-status" />
  ) : null;

  function field(name: BriefField) {
    const value = content[name].trim();
    return (
      <div className={styles.field} data-testid={`brief-field-${name}`}>
        <span className={styles.label}>
          {FIELD_LABELS[name].card}
          {unit?.creatorFields.includes(name) && (
            <span className={unitStyles.mine} title="AI giữ nguyên ô này khi làm lại" data-testid={`brief-mine-${name}`}>
              ✎ Bạn viết
            </span>
          )}
        </span>
        <p className={styles.value}>{value || <span className={styles.empty}>(trống)</span>}</p>
      </div>
    );
  }

  function body() {
    if (!created) {
      return (
        <p className={styles.hint}>
          {aiAvailable
            ? "Chưa có brief. Bấm “Tạo project và viết brief”: AI viết brief từ đề tài, bạn sửa rồi duyệt."
            : "Chưa có brief. Bấm “Tạo project”, rồi tự viết brief và duyệt."}
        </p>
      );
    }
    if (!brief.state) {
      if (brief.loadError) {
        return (
          <>
            <p className={styles.error} role="alert">
              Không tải được brief: {brief.loadError}
            </p>
            <div className={styles.actions}>
              <Button variant="ghost" onClick={brief.reload} data-testid="brief-reload">
                Tải lại
              </Button>
            </div>
          </>
        );
      }
      return (
        <p className={styles.hint} role="status">
          Đang tải brief…
        </p>
      );
    }
    if (!unit) {
      if (brief.running) return null;
      return (
        <>
          <p className={styles.hint}>
            {aiAvailable
              ? "Chưa có brief. AI viết từ đề tài, hoặc bạn tự viết."
              : "Chưa có brief. Chưa cấu hình AI nên bạn tự viết brief rồi duyệt."}
          </p>
          {aiAvailable && <AiModelSelect projectId={projectId} step="brief" label="Model AI viết brief" testId="brief-model" />}
          <div className={styles.actions}>
            <Button variant="ghost" onClick={() => setEditOpen(true)} data-testid="brief-write-own">
              Tự viết brief
            </Button>
            {aiAvailable && (
              <Button variant="run" onClick={() => void brief.generate()} data-testid="brief-generate">
                AI viết brief
              </Button>
            )}
          </div>
        </>
      );
    }
    return (
      <>
        <p className={styles.hint}>
          AI viết từ đề tài. Ô nào cũng sửa được bằng “Sửa tay”; ô bạn đã sửa thì AI giữ nguyên khi làm lại.
        </p>
        {field("question")}
        {field("audience")}
        <div className={styles.group}>
          <span className={styles.label}>Sau video, người xem sẽ…</span>
          <div className={styles.grid3}>
            {field("know")}
            {field("understand")}
            {field("feel")}
          </div>
        </div>
        <div className={styles.grid2}>
          {field("tone")}
          {field("out_of_scope")}
        </div>
        {unit.status === "pending" && missing.length > 0 && (
          <p className={styles.hint} data-testid="brief-missing">
            Còn trống: {missing.map((f) => FIELD_LABELS[f].full).join("; ")}. Điền đủ các ô rồi mới duyệt được.
          </p>
        )}
        <UnitReviewBar
          status={unit.status}
          approveLabel="Duyệt brief"
          approveDisabled={missing.length > 0}
          busy={brief.busy || brief.running}
          canRedo={aiAvailable}
          onEdit={() => setEditOpen(true)}
          onRedo={() => setRedoOpen(true)}
          onApprove={() => void brief.approve()}
          onUnapprove={() => void brief.unapprove()}
          testId="brief-review"
        />
      </>
    );
  }

  const editNote = !unit
    ? "Ô bạn tự viết, AI giữ nguyên nếu sau này bạn nhờ AI làm lại."
    : `${unit.status === "approved" ? "Brief đã duyệt vẫn giữ duyệt sau khi bạn sửa, trừ khi có ô bị xoá trắng. " : ""}Ô bạn sửa thành “Bạn viết”: AI giữ nguyên khi làm lại. Xoá trắng một ô thì AI viết lại ô đó.`;

  return (
    <Card data-testid="brief-card">
      <div className={styles.body}>
        <div className={styles.head}>
          <h3>② Creative brief</h3>
          {chip}
        </div>
        {body()}
        <AiOperationCard run={brief.run} active={brief.running} testId="brief-progress" title="AI đang viết brief" />
        {brief.actionError && (
          <p className={styles.error} role="alert" data-testid="brief-error">
            {brief.actionError}
          </p>
        )}
      </div>

      <UnitEditModal
        open={editOpen}
        onClose={() => setEditOpen(false)}
        title={unit ? "Sửa tay · Creative brief" : "Tự viết · Creative brief"}
        fields={editFields}
        impact={editImpact}
        onSave={brief.edit}
        note={editNote}
        testId="brief-edit"
      />
      <UnitRedoModal
        open={redoOpen}
        onClose={() => setRedoOpen(false)}
        title="Làm lại kèm ghi chú · Creative brief"
        impact={redoImpact}
        initialNote={brief.failedNote}
        modelSelect={aiAvailable ? <AiModelSelect projectId={projectId} step="brief" label="Model AI viết brief" testId="brief-redo-model" /> : undefined}
        onSend={(note) => void brief.redo(note)}
        testId="brief-redo"
      />
    </Card>
  );
}
