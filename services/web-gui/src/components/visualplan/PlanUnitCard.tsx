import { useState } from "react";
import { generateVisualPlanBeat, type VisualPlanBeat, type VisualPlanUnit } from "../../api/client";
import type { VisualPlanControls } from "../../hooks/useVisualPlan";
import { ConceptAiDialog } from "../visual/ConceptAiDialog";
import researchStyles from "../research/research.module.css";
import { Button, CopyBox } from "../ui";
import { UnitStatusChip } from "../units/UnitStatusChip";
import { PlanAnswerDialog, PlanEditDialog, PlanMergeDialog, PlanSplitDialog } from "./PlanDialogs";
import { METHOD_LABELS, RELATIONSHIP_LABELS, hasOpenQuestion, refsLabel, seconds } from "./planModel";
import foldStyles from "../visual/visual.module.css";
import styles from "./plan.module.css";

interface PlanUnitCardProps {
  projectId: string;
  beat: VisualPlanBeat;
  unit: VisualPlanUnit;
  /** The unit right after this one in the same beat, if any. */
  next: VisualPlanUnit | null;
  plan: VisualPlanControls;
  readOnly: boolean;
}

/**
 * One visual unit: the lines it covers and what the viewer must take away on
 * the left, how it is shown and the minutes it holds on the right, then the
 * actions on it (edit, split, merge, redo with a note, approve).
 */
export function PlanUnitCard({ projectId, beat, unit, next, plan, readOnly }: PlanUnitCardProps) {
  const [editing, setEditing] = useState(false);
  const [splitting, setSplitting] = useState(false);
  const [merging, setMerging] = useState(false);
  const [redoing, setRedoing] = useState(false);
  const [answering, setAnswering] = useState<string | null>(null);
  // An approved unit folds to one line; the Creator opens it to read or change it.
  const [expanded, setExpanded] = useState(false);
  const approved = unit.status === "approved";
  const disabled = readOnly || plan.busy || plan.running !== null;
  const testId = `plan-unit-${unit.unit_id}`;
  const open = unit.open_questions.filter((q) => q.answer.trim() === "");
  const longer = unit.effective_planned_s > unit.narration_min_s;
  const warn = unit.risk_notes.length > 0 || unit.problems.length > 0;
  const approveBlock = beat.stale
    ? "Beat đã lỗi thời so với kịch bản, lập lại beat trước"
    : unit.problems.length > 0
      ? unit.problems.join("; ")
      : "";

  if (approved && !expanded) {
    return (
      <article className={styles.unit} id={testId} data-testid={testId}>
        <div className={foldStyles.folded}>
          <span className={styles.uid}>{unit.unit_id}</span>
          <span className={foldStyles.summary}>{unit.communication_goal}</span>
          <UnitStatusChip state="approved" testId={`${testId}-status`} />
          <Button variant="view" onClick={() => setExpanded(true)} data-testid={`${testId}-open`}>
            Mở
          </Button>
        </div>
      </article>
    );
  }

  return (
    <article className={`${styles.unit} ${warn && !approved ? styles.unitWarn : ""}`} id={testId} data-testid={testId}>
      <div className={styles.unitHead}>
        <span>
          <span className={styles.uid}>{unit.unit_id}</span>{" "}
          <span className={researchStyles.hint}>
            #{unit.number} · {refsLabel(unit.script_refs)}
          </span>
        </span>
        <span>
          {unit.risk_notes.length > 0 && (
            <span className={researchStyles.chip} data-testid={`${testId}-risk`}>
              ⚠ Rủi ro
            </span>
          )}{" "}
          <UnitStatusChip state={approved ? "approved" : "pending"} testId={`${testId}-status`} />
        </span>
      </div>
      <div className={styles.cols}>
        <div className={`${styles.col} ${styles.colScript}`}>
          <span className={styles.lbl}>Lời thoại · {refsLabel(unit.script_refs)}</span>
          {unit.lines.map((l) => (
            <p key={l.id} className={styles.quote}>
              “{l.text}”
            </p>
          ))}
          <span className={styles.lbl}>Mục tiêu truyền đạt</span>
          <span>{unit.communication_goal}</span>
          <span className={styles.lbl}>Thời lượng</span>
          <div className={styles.dur} data-testid={`${testId}-duration`}>
            <span className={styles.durKey}>Tối thiểu (theo lời)</span>
            <span>
              {seconds(unit.narration_min_s)} <span className={researchStyles.hint}>· máy tính</span>
            </span>
            <span className={styles.durKey}>Dự kiến</span>
            <span className={longer ? styles.durLong : undefined}>
              {seconds(unit.effective_planned_s)}{" "}
              <span className={researchStyles.hint}>
                {longer
                  ? `· dài hơn lời ${seconds(unit.effective_planned_s - unit.narration_min_s)}`
                  : "· bằng mức tối thiểu"}
              </span>
            </span>
            {longer && (
              <>
                <span className={styles.durKey}>Lý do giữ lâu hơn</span>
                <span>{unit.duration.extra_reason || "Chưa có lý do"}</span>
              </>
            )}
          </div>
        </div>
        <div className={styles.col}>
          <span className={styles.lbl}>
            Cách hiển thị · {METHOD_LABELS[unit.visual_approach.method] ?? unit.visual_approach.method}
          </span>
          <CopyBox text={unit.visual_approach.description} what="mô tả cách hiển thị">
            <p style={{ margin: 0, padding: 8 }}>{unit.visual_approach.description}</p>
          </CopyBox>
          <span className={styles.lbl}>Phải có trong hình</span>
          <div className={styles.tags}>
            {unit.essential_visual_info.map((i) => (
              <span key={i.item} className={`${researchStyles.chip} ${i.priority === "must" ? styles.must : ""}`}>
                {i.item}
              </span>
            ))}
          </div>
          {unit.on_screen_text.length > 0 && (
            <>
              <span className={styles.lbl}>Chữ trên hình</span>
              <div className={styles.tags}>
                {unit.on_screen_text.map((t) => (
                  <span key={t} className={researchStyles.chip}>
                    “{t}”
                  </span>
                ))}
              </div>
            </>
          )}
          <span className={styles.lbl}>Quy tắc concept · Quan hệ với lời</span>
          <div className={styles.tags}>
            {unit.visual_approach.concept_rule_refs.map((r) => (
              <span key={r} className={researchStyles.chip}>
                {r}
              </span>
            ))}
            <span className={researchStyles.chip}>
              {RELATIONSHIP_LABELS[unit.narration_relationship] ?? unit.narration_relationship}
            </span>
          </div>
          {unit.risk_notes.map((r) => (
            <div key={r.fact_id} className={styles.risk} data-testid={`${testId}-risk-${r.fact_id}`}>
              <b>Dữ kiện {r.fact_id}</b> · {r.note}
            </div>
          ))}
          {unit.open_questions.map((q) => (
            <div key={q.id} className={styles.question} data-testid={`${testId}-question-${q.id}`}>
              <span>
                <b>{q.id}</b> {q.question}
                {q.answer.trim() !== "" && (
                  <>
                    <br />
                    <span className={researchStyles.hint}>Trả lời: {q.answer}</span>
                  </>
                )}
              </span>
              <Button
                variant="ghost"
                onClick={() => setAnswering(q.id)}
                disabled={disabled}
                data-testid={`${testId}-answer-${q.id}`}
              >
                {q.answer.trim() === "" ? "Trả lời" : "Sửa câu trả lời"}
              </Button>
            </div>
          ))}
        </div>
      </div>
      {unit.problems.length > 0 && (
        <ul className={styles.problems} role="status" data-testid={`${testId}-problems`}>
          {unit.problems.map((p) => (
            <li key={p}>{p}</li>
          ))}
        </ul>
      )}
      <div className={styles.unitFoot}>
        <Button variant="ghost" onClick={() => setEditing(true)} disabled={disabled} data-testid={`${testId}-edit`}>
          Sửa
        </Button>
        <Button
          variant="ghost"
          onClick={() => setSplitting(true)}
          disabled={disabled || unit.lines.length < 2}
          data-testid={`${testId}-split`}
        >
          Tách
        </Button>
        {next && (
          <Button variant="ghost" onClick={() => setMerging(true)} disabled={disabled} data-testid={`${testId}-merge`}>
            Gộp với {next.unit_id}
          </Button>
        )}
        {approved ? (
          <>
            <Button variant="ghost" onClick={() => setExpanded(false)} data-testid={`${testId}-fold`}>
              Thu gọn
            </Button>
            <Button variant="ghost" onClick={() => void plan.unapprove(unit.unit_id)} disabled={disabled} data-testid={`${testId}-unapprove`}>
              Bỏ duyệt
            </Button>
          </>
        ) : (
          <>
            <Button variant="run" onClick={() => setRedoing(true)} disabled={disabled} data-testid={`${testId}-redo`}>
              Làm lại kèm ghi chú
            </Button>
            <Button
              variant="primary"
              onClick={() => {
                setExpanded(false);
                void plan.approve(unit.unit_id);
              }}
              disabled={disabled || approveBlock !== "" || hasOpenQuestion(unit)}
              title={approveBlock || (open.length > 0 ? "Còn câu hỏi chưa trả lời" : undefined)}
              data-testid={`${testId}-approve`}
            >
              Duyệt
            </Button>
          </>
        )}
      </div>

      <PlanEditDialog
        open={editing}
        onClose={() => setEditing(false)}
        unit={unit}
        onSave={(fields) => plan.edit(unit.unit_id, fields)}
      />
      <PlanSplitDialog
        open={splitting}
        onClose={() => setSplitting(false)}
        unit={unit}
        onSplit={(at) => plan.split(unit.unit_id, at)}
      />
      {next && (
        <PlanMergeDialog
          open={merging}
          onClose={() => setMerging(false)}
          unit={unit}
          next={next}
          onMerge={() => plan.merge(unit.unit_id, next.unit_id)}
        />
      )}
      <PlanAnswerDialog
        open={answering !== null}
        onClose={() => setAnswering(null)}
        unitId={unit.unit_id}
        question={unit.open_questions.find((q) => q.id === answering) ?? null}
        onAnswer={(qid, text) => plan.answer(unit.unit_id, qid, text)}
      />
      <ConceptAiDialog
        open={redoing}
        onClose={() => setRedoing(false)}
        title={`Làm lại ${unit.unit_id} kèm ghi chú`}
        impact={
          <p className={researchStyles.hint}>
            AI lập lại cả beat {beat.key}: các unit khác của beat giữ mã nếu phủ đúng các câu cũ, trường bạn đã sửa
            tay được giữ, mọi unit của beat về “Chờ duyệt”.
          </p>
        }
        onSystem={(note) => void plan.generateBeat(beat.key, note)}
        external={{
          run: (note, external) => generateVisualPlanBeat(projectId, beat.key, note, { external }),
          onDone: plan.reload,
        }}
        testId={`${testId}-redo-dialog`}
      />
    </article>
  );
}
