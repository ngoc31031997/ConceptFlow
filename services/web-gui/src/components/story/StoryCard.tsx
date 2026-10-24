import { useState } from "react";
import type { StoryFieldKey, StoryPatch, StoryState } from "../../api/client";
import type { StoryControls } from "../../hooks/useStory";
import { AiOperationCard } from "../AiOperationCard";
import researchStyles from "../research/research.module.css";
import { Button, Card } from "../ui";
import unitStyles from "../units/units.module.css";
import { UnitStatusChip } from "../units/UnitStatusChip";
import { FactTags } from "./FactTags";
import { StoryArcEditModal } from "./StoryArcEditModal";
import { StoryTextEditModal } from "./StoryTextEditModal";
import styles from "./story.module.css";

interface StoryCardProps {
  story: StoryControls;
  state: StoryState;
  /** A language model is configured, so the AI can write the story. */
  aiAvailable: boolean;
  /** The step cannot be changed now (the project is past editing). */
  locked: boolean;
}

const LABELS: Record<StoryFieldKey, string> = {
  central_question: "Câu hỏi trung tâm",
  core_answer: "Câu trả lời cốt lõi",
  narrative_arc: "Mạch kể",
};

function Mine({ field, owned }: { field: StoryFieldKey; owned: boolean }) {
  if (!owned) return null;
  return (
    <span className={unitStyles.mine} title="AI giữ nguyên ô này khi viết lại" data-testid={`story-mine-${field}`}>
      ✎ Bạn viết
    </span>
  );
}

/**
 * Section 2 of the story step: the central question, the core answer and the
 * narrative arc. Each reads as text with its own "Sửa tay" (docs/ux-ui-design-rules.md,
 * rule 4); one the Creator wrote is tagged and kept by the AI. The AI writes
 * the whole story, and "Làm lại kèm ghi chú" sits with the approval below.
 */
export function StoryCard({ story, state, aiAvailable, locked }: StoryCardProps) {
  const [editing, setEditing] = useState<"central_question" | "core_answer" | "narrative_arc" | null>(null);
  const s = state.story;
  const open = state.briefApproved && state.researchResolved;
  const busy = locked || story.busy || story.writing;
  const owns = (key: StoryFieldKey) => s?.creatorFields.includes(key) ?? false;
  const approved = s?.status === "approved";

  const chip = story.writing ? (
    <UnitStatusChip state="running" testId="story-status" />
  ) : (
    <UnitStatusChip state={state.status} testId="story-status" />
  );

  async function save(patch: StoryPatch) {
    await story.edit(patch);
  }

  const textField = (key: "central_question" | "core_answer", value: string) => (
    <div className={styles.field} data-testid={`story-field-${key}`}>
      <span className={styles.fieldHead}>
        {LABELS[key]}
        <Mine field={key} owned={owns(key)} />
      </span>
      {value.trim() ? <p className={styles.text}>{value}</p> : <p className={`${styles.text} ${styles.empty}`}>(trống)</p>}
      <div className={styles.fieldActions}>
        <Button variant="ghost" className={researchStyles.sm} onClick={() => setEditing(key)} disabled={busy} data-testid={`story-edit-${key}`}>
          Sửa tay
        </Button>
      </div>
    </div>
  );

  return (
    <Card data-testid="story-card">
      <div className={researchStyles.body}>
        <div className={researchStyles.head}>
          <h3>② Câu chuyện</h3>
          {chip}
        </div>
        {!open && (
          <p className={researchStyles.hint} data-testid="story-not-open">
            {!state.briefApproved ? "Duyệt brief ở bước Creative brief trước." : "Duyệt bảng dữ kiện hoặc bỏ qua bước Nghiên cứu trước."}
          </p>
        )}
        {open && !s && (
          <>
            <p className={researchStyles.hint}>
              {aiAvailable
                ? "Chưa có câu chuyện. AI chọn góc kể và xây mạch từ brief và dữ kiện, hoặc bạn tự viết từng ô."
                : "Chưa có câu chuyện. Chưa cấu hình AI nên bạn tự viết từng ô rồi duyệt."}
            </p>
            {aiAvailable && (
              <div className={researchStyles.actions} style={{ justifyContent: "flex-start" }}>
                <Button variant="run" onClick={() => void story.generate("")} disabled={busy} data-testid="story-generate">
                  AI viết câu chuyện
                </Button>
              </div>
            )}
          </>
        )}
        {s && (state.briefStale || state.researchStale) && (
          <p className={researchStyles.hint} data-testid="story-stale">
            ↻ {state.briefStale && state.researchStale ? "Brief và bảng dữ kiện" : state.briefStale ? "Brief" : "Bảng dữ kiện"} đã đổi sau khi câu chuyện được viết. Làm lại kèm ghi chú nếu cần viết lại.
          </p>
        )}
        <AiOperationCard run={story.run} active={story.writing} testId="story-progress" title="AI đang viết câu chuyện" />
        {s && (
          <>
            {textField("central_question", s.centralQuestion)}
            {textField("core_answer", s.coreAnswer)}
            <div className={styles.field} data-testid="story-field-narrative_arc">
              <span className={styles.fieldHead}>
                {LABELS.narrative_arc}
                <Mine field="narrative_arc" owned={owns("narrative_arc")} />
              </span>
              {s.narrativeArc.length === 0 ? (
                <p className={`${styles.text} ${styles.empty}`}>(trống)</p>
              ) : (
                <ol className={styles.arc} data-testid="story-arc">
                  {s.narrativeArc.map((b, i) => (
                    <li key={i}>
                      <span className={styles.beatName}>{b.beat}</span>: {b.text}
                      <FactTags ids={b.facts} />
                    </li>
                  ))}
                </ol>
              )}
              <div className={styles.fieldActions}>
                <Button variant="ghost" className={researchStyles.sm} onClick={() => setEditing("narrative_arc")} disabled={busy} data-testid="story-edit-narrative_arc">
                  Sửa tay
                </Button>
              </div>
            </div>
          </>
        )}
        {open && !s && (
          <div className={researchStyles.actions} style={{ justifyContent: "flex-start" }}>
            <Button variant="ghost" onClick={() => setEditing("central_question")} disabled={busy} data-testid="story-write-by-hand">
              Tự viết từng ô
            </Button>
          </div>
        )}
        {story.actionError && (
          <p className={researchStyles.error} role="alert" data-testid="story-error">
            {story.actionError}
          </p>
        )}
      </div>

      <StoryTextEditModal
        open={editing === "central_question"}
        onClose={() => setEditing(null)}
        label={LABELS.central_question}
        value={s?.centralQuestion ?? ""}
        rows={2}
        approved={approved}
        onSave={(text) => save({ centralQuestion: text })}
        testId="story-edit-question-dialog"
      />
      <StoryTextEditModal
        open={editing === "core_answer"}
        onClose={() => setEditing(null)}
        label={LABELS.core_answer}
        value={s?.coreAnswer ?? ""}
        rows={4}
        approved={approved}
        onSave={(text) => save({ coreAnswer: text })}
        testId="story-edit-answer-dialog"
      />
      <StoryArcEditModal
        open={editing === "narrative_arc"}
        onClose={() => setEditing(null)}
        arc={s?.narrativeArc ?? []}
        approved={approved}
        onSave={(arc) => save({ narrativeArc: arc })}
      />
    </Card>
  );
}
