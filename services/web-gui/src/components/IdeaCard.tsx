import { useState } from "react";
import { IDEA_FIELDS, IDEA_UNIT, type Idea, type IdeaField } from "../api/client";
import type { IdeaControls } from "../hooks/useIdea";
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

/** Each field's label on the card and in the edit dialog. */
const FIELD_LABELS: Record<IdeaField, { label: string; rows: number }> = {
  question: { label: "Câu hỏi trung tâm", rows: 2 },
  why: { label: "Vì sao câu hỏi này thú vị", rows: 3 },
  known: { label: "Điều đã biết", rows: 3 },
  unknown: { label: "Điều chưa biết", rows: 3 },
};

const EMPTY_IDEA = Object.fromEntries(IDEA_FIELDS.map((f) => [f, ""])) as Idea;

interface IdeaCardProps {
  projectId: string;
  /** The project exists on the server; before that there is no idea to load. */
  created: boolean;
  idea: IdeaControls;
  /** A language model is configured, so the AI can write and redo the idea. */
  aiAvailable: boolean;
}

/**
 * Section 2 of the idea step: the idea as one review unit. The fields read as
 * text; "Sửa tay" and "Làm lại kèm ghi chú" open their dialogs
 * (docs/ux-ui-design-rules.md, rule 4), each with the later work it sends back.
 * Without a language model the Creator writes the idea and approves it.
 */
export function IdeaCard({ projectId, created, idea, aiAvailable }: IdeaCardProps) {
  const [editOpen, setEditOpen] = useState(false);
  const [redoOpen, setRedoOpen] = useState(false);
  const editImpact = useUnitImpact(projectId, IDEA_UNIT, editOpen);
  const redoImpact = useUnitImpact(projectId, IDEA_UNIT, redoOpen);

  const unit = idea.state?.unit ?? null;
  const content = unit?.content ?? EMPTY_IDEA;
  const missing = IDEA_FIELDS.filter((f) => !content[f].trim());
  const editFields: UnitEditField[] = IDEA_FIELDS.map((f) => ({
    name: f,
    label: FIELD_LABELS[f].label,
    value: content[f],
    rows: FIELD_LABELS[f].rows,
  }));

  const chip = idea.running ? (
    <UnitStatusChip state="running" testId="idea-status" />
  ) : unit ? (
    <UnitStatusChip state={unit.status} testId="idea-status" />
  ) : null;

  function field(name: IdeaField) {
    const value = content[name].trim();
    return (
      <div className={styles.field} data-testid={`idea-field-${name}`}>
        <span className={styles.label}>
          {FIELD_LABELS[name].label}
          {unit?.creatorFields.includes(name) && (
            <span className={unitStyles.mine} title="AI giữ nguyên ô này khi làm lại" data-testid={`idea-mine-${name}`}>
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
            ? "Chưa có ý tưởng. Bấm “Tạo project và viết ý tưởng”: AI viết từ đề tài, bạn sửa rồi duyệt."
            : "Chưa có ý tưởng. Bấm “Tạo project”, rồi tự viết ý tưởng và duyệt."}
        </p>
      );
    }
    if (!idea.state) {
      if (idea.loadError) {
        return (
          <>
            <p className={styles.error} role="alert">
              Không tải được ý tưởng: {idea.loadError}
            </p>
            <div className={styles.actions}>
              <Button variant="ghost" onClick={idea.reload} data-testid="idea-reload">
                Tải lại
              </Button>
            </div>
          </>
        );
      }
      return (
        <p className={styles.hint} role="status">
          Đang tải ý tưởng…
        </p>
      );
    }
    if (!unit) {
      if (idea.running) return null;
      return (
        <>
          <p className={styles.hint}>
            {aiAvailable
              ? "Chưa có ý tưởng. AI viết từ đề tài, hoặc bạn tự viết."
              : "Chưa có ý tưởng. Chưa cấu hình AI nên bạn tự viết ý tưởng rồi duyệt."}
          </p>
          {aiAvailable && <AiModelSelect projectId={projectId} step="brief" label="Model AI viết ý tưởng và brief" testId="idea-model" />}
          <div className={styles.actions}>
            <Button variant="ghost" onClick={() => setEditOpen(true)} data-testid="idea-write-own">
              Tự viết ý tưởng
            </Button>
            {aiAvailable && (
              <Button variant="run" onClick={() => void idea.generate()} data-testid="idea-generate">
                AI viết ý tưởng
              </Button>
            )}
          </div>
        </>
      );
    }
    return (
      <>
        <p className={styles.hint}>
          AI viết từ đề tài. Ô nào cũng sửa được bằng “Sửa tay”; ô bạn đã sửa thì AI giữ nguyên khi làm lại. Không cần có
          sẵn lời giải hay cấu trúc video.
        </p>
        {field("question")}
        {field("why")}
        <div className={styles.grid2}>
          {field("known")}
          {field("unknown")}
        </div>
        {unit.status === "pending" && missing.length > 0 && (
          <p className={styles.hint} data-testid="idea-missing">
            Còn trống: {missing.map((f) => FIELD_LABELS[f].label).join("; ")}. Điền đủ các ô rồi mới duyệt được.
          </p>
        )}
        <UnitReviewBar
          status={unit.status}
          approveLabel="Duyệt ý tưởng"
          approveDisabled={missing.length > 0}
          busy={idea.busy || idea.running}
          canRedo={aiAvailable}
          onEdit={() => setEditOpen(true)}
          onRedo={() => setRedoOpen(true)}
          onApprove={() => void idea.approve()}
          onUnapprove={() => void idea.unapprove()}
          testId="idea-review"
        />
      </>
    );
  }

  const editNote = !unit
    ? "Ô bạn tự viết, AI giữ nguyên nếu sau này bạn nhờ AI làm lại."
    : `${unit.status === "approved" ? "Ý tưởng đã duyệt vẫn giữ duyệt sau khi bạn sửa, trừ khi có ô bị xoá trắng. " : ""}Ô bạn sửa thành “Bạn viết”: AI giữ nguyên khi làm lại. Xoá trắng một ô thì AI viết lại ô đó.`;

  return (
    <Card data-testid="idea-card">
      <div className={styles.body}>
        <div className={styles.head}>
          <h3>② Ý tưởng</h3>
          {chip}
        </div>
        {body()}
        <AiOperationCard run={idea.run} active={idea.running} testId="idea-progress" title="AI đang viết ý tưởng" />
        {idea.actionError && (
          <p className={styles.error} role="alert" data-testid="idea-error">
            {idea.actionError}
          </p>
        )}
      </div>

      <UnitEditModal
        open={editOpen}
        onClose={() => setEditOpen(false)}
        title={unit ? "Sửa tay · Ý tưởng" : "Tự viết · Ý tưởng"}
        fields={editFields}
        impact={editImpact}
        onSave={idea.edit}
        note={editNote}
        size="xl"
        testId="idea-edit"
      />
      <UnitRedoModal
        open={redoOpen}
        onClose={() => setRedoOpen(false)}
        title="Làm lại kèm ghi chú · Ý tưởng"
        impact={redoImpact}
        initialNote={idea.failedNote}
        modelSelect={aiAvailable ? <AiModelSelect projectId={projectId} step="brief" label="Model AI viết ý tưởng và brief" testId="idea-redo-model" /> : undefined}
        onSend={(note) => void idea.redo(note)}
        testId="idea-redo"
      />
    </Card>
  );
}
