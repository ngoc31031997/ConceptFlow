import { useState } from "react";
import { storyFieldUnit, type StoryField, type StoryFieldKey, type StoryState } from "../../api/client";
import type { StoryControls } from "../../hooks/useStory";
import { useUnitImpact } from "../../hooks/useUnitImpact";
import { AiModelSelect } from "../AiModelSelect";
import { AiOperationCard } from "../AiOperationCard";
import researchStyles from "../research/research.module.css";
import { Button, Card } from "../ui";
import { UnitEditModal } from "../units/UnitEditModal";
import { UnitRedoModal } from "../units/UnitRedoModal";
import unitStyles from "../units/units.module.css";
import { UnitStatusChip } from "../units/UnitStatusChip";
import { splitChainLine } from "./chainLine";
import { FactTags } from "./FactTags";
import styles from "./story.module.css";

/** Rows of the edit box for each field. */
const FIELD_ROWS: Record<StoryFieldKey, number> = { question: 2, promise: 3, chain: 8, payoff: 3, logline: 2, summary: 5 };

const FACT_LINE_NOTE = "Mỗi dòng cuối có thể ghi mã dữ kiện dựa vào, ví dụ [F01, F02]; mã phải có trong bảng dữ kiện.";

interface FrameCardProps {
  projectId: string;
  story: StoryControls;
  state: StoryState;
  /** A language model is configured, so the AI can write the fields. */
  aiAvailable: boolean;
  /** The step cannot be changed now (the project is past editing). */
  locked: boolean;
}

function FieldBody({ field }: { field: StoryField }) {
  const text = field.text.trim();
  if (!text) return <p className={`${styles.text} ${styles.empty}`}>(trống)</p>;
  if (field.key === "chain") {
    return (
      <ol className={styles.chain} data-testid="story-chain">
        {text.split("\n").map((line, i) => {
          const step = splitChainLine(line);
          return (
            <li key={i}>
              {step.text}
              <FactTags ids={step.ids} />
            </li>
          );
        })}
      </ol>
    );
  }
  if (field.key === "payoff") {
    const part = splitChainLine(text);
    return (
      <p className={styles.text}>
        {part.text}
        <FactTags ids={part.ids} testId="story-payoff-facts" />
      </p>
    );
  }
  return <p className={styles.text}>{text}</p>;
}

/**
 * Section 2 of the story step: the six fields of the frame. Each field reads
 * as text with its own "Sửa tay" and "Làm lại kèm ghi chú" (docs/ux-ui-design-rules.md,
 * rule 4); one the Creator wrote is tagged and kept by the AI. The AI writes all
 * the fields at once only when none exists yet.
 */
export function FrameCard({ projectId, story, state, aiAvailable, locked }: FrameCardProps) {
  const [editing, setEditing] = useState<StoryField | null>(null);
  const [redoing, setRedoing] = useState<StoryField | null>(null);
  const editImpact = useUnitImpact(projectId, storyFieldUnit(editing?.key ?? "question"), editing !== null);
  const redoImpact = useUnitImpact(projectId, storyFieldUnit(redoing?.key ?? "question"), redoing !== null);

  const hasFrame = state.fields.some((f) => f.text.trim() !== "");
  const busy = locked || story.busy || story.running !== null;
  const writingAll = story.running === "story";
  const writingField = typeof story.running === "object" && story.running !== null ? story.running.field : null;
  const open = state.briefApproved && state.researchResolved;

  const chip = story.running !== null ? <UnitStatusChip state="running" testId="story-status" /> : <UnitStatusChip state={state.status} testId="story-status" />;

  return (
    <Card data-testid="story-frame-card">
      <div className={researchStyles.body}>
        <div className={researchStyles.head}>
          <h3>② Khung câu chuyện</h3>
          {chip}
        </div>
        {!open && (
          <p className={researchStyles.hint} data-testid="story-not-open">
            {!state.briefApproved
              ? "Duyệt brief ở bước Ý tưởng trước."
              : "Duyệt bảng dữ kiện hoặc bỏ qua bước Nghiên cứu trước."}
          </p>
        )}
        {open && !hasFrame && (
          <>
            <p className={researchStyles.hint}>
              {aiAvailable
                ? "Chưa có khung. AI viết sáu ô từ brief, bảng dữ kiện và góc đã chọn, hoặc bạn tự viết từng ô."
                : "Chưa có khung. Chưa cấu hình AI nên bạn tự viết từng ô rồi duyệt."}
            </p>
            {aiAvailable && (
              <div className={researchStyles.actions} style={{ justifyContent: "flex-start" }}>
                <Button variant="run" onClick={() => void story.generateStory("")} disabled={busy} data-testid="story-generate">
                  AI viết khung câu chuyện
                </Button>
              </div>
            )}
          </>
        )}
        <AiOperationCard
          run={story.run}
          active={writingAll || writingField !== null}
          testId="story-progress"
          title={writingField ? "AI đang viết lại một ô" : "AI đang viết khung câu chuyện"}
        />
        {state.fields.map((field) => (
          <div key={field.key} className={styles.field} data-testid={`story-field-${field.key}`}>
            <span className={styles.fieldHead}>
              {field.label}
              {field.mine && (
                <span className={unitStyles.mine} title="AI giữ nguyên ô này khi viết lại" data-testid={`story-mine-${field.key}`}>
                  ✎ Bạn viết
                </span>
              )}
              {field.stale && (
                <span className={styles.stale} data-testid={`story-stale-${field.key}`}>
                  ↻ brief đã đổi
                </span>
              )}
            </span>
            <FieldBody field={field} />
            <div className={styles.fieldActions}>
              <Button variant="ghost" className={researchStyles.sm} onClick={() => setEditing(field)} disabled={busy} data-testid={`story-edit-${field.key}`}>
                Sửa tay
              </Button>
              {aiAvailable && field.text.trim() !== "" && (
                <Button variant="ghost" className={researchStyles.sm} onClick={() => setRedoing(field)} disabled={busy} data-testid={`story-redo-${field.key}`}>
                  Làm lại kèm ghi chú
                </Button>
              )}
              {writingField === field.key && <span className={researchStyles.hint}>● AI đang làm lại</span>}
            </div>
          </div>
        ))}
        {story.actionError && (
          <p className={researchStyles.error} role="alert" data-testid="story-error">
            {story.actionError}
          </p>
        )}
      </div>

      <UnitEditModal
        open={editing !== null}
        onClose={() => setEditing(null)}
        title={`Sửa tay · ${editing?.label ?? ""}`}
        fields={editing ? [{ name: "text", label: editing.label, value: editing.text, rows: FIELD_ROWS[editing.key] }] : []}
        impact={editImpact}
        onSave={(changed) => story.editField(editing?.key ?? "question", changed.text ?? "")}
        note={
          <>
            {editing && (editing.key === "chain" || editing.key === "payoff") ? `${FACT_LINE_NOTE} ` : ""}
            Ô bạn sửa thành “Bạn viết”: AI giữ nguyên khi viết lại. Xoá trắng thì AI viết lại ô này. Sửa xong câu chuyện về chờ duyệt.
          </>
        }
        size={editing && (editing.key === "chain" || editing.key === "summary") ? "lg" : "md"}
        testId="story-edit"
      />
      <UnitRedoModal
        open={redoing !== null}
        onClose={() => setRedoing(null)}
        title={`Làm lại kèm ghi chú · ${redoing?.label ?? ""}`}
        impact={redoImpact}
        initialNote={story.failedNote}
        modelSelect={aiAvailable ? <AiModelSelect projectId={projectId} step="story_frame" label="Model AI cho bước Câu chuyện" testId="story-redo-model" /> : undefined}
        onSend={(note) => {
          if (redoing) void story.redoField(redoing.key, note);
        }}
        testId="story-redo"
      />
    </Card>
  );
}
