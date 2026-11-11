import { useState } from "react";
import { generateVisualConceptUnit, type VisualConceptUnit } from "../../api/client";
import type { VisualConceptControls } from "../../hooks/useVisualConcept";
import researchStyles from "../research/research.module.css";
import { Button, Card } from "../ui";
import { UnitStatusChip } from "../units/UnitStatusChip";
import { ConceptAiDialog } from "./ConceptAiDialog";
import { PART_SCHEMAS, partSummary } from "./conceptSchema";
import { PartEditDialog } from "./PartEditDialog";
import { PartView } from "./PartView";
import styles from "./visual.module.css";

interface ConceptUnitCardProps {
  projectId: string;
  unit: VisualConceptUnit;
  concept: VisualConceptControls;
  /** The step cannot be changed now. */
  readOnly: boolean;
  /** Opened from outside (a finding's "Mở phần"). */
  forceOpen: boolean;
  /** The part was folded again, so it is no longer held open from outside. */
  onFold: () => void;
}

/**
 * One of the eight parts of the visual concept: folded to one line once
 * approved, otherwise its content with what the AI read, what fails its
 * checks and the actions on it (hand edit, AI redo with a note, approve).
 */
export function ConceptUnitCard({
  projectId,
  unit,
  concept,
  readOnly,
  forceOpen,
  onFold,
}: ConceptUnitCardProps) {
  const [open, setOpen] = useState(false);
  const [editing, setEditing] = useState(false);
  const [redoing, setRedoing] = useState(false);
  const approved = unit.status === "approved";
  const running =
    unit.status === "running" || concept.running === unit.key || concept.running === "all";
  const disabled = readOnly || concept.busy || concept.running !== null;
  const testId = `concept-unit-${unit.key}`;

  if (approved && !open && !forceOpen && !unit.stale) {
    return (
      <Card data-testid={testId}>
        <div className={styles.folded}>
          <span className={styles.unitNo}>{unit.number}</span>
          <b>{unit.title}</b>
          <span className={styles.summary}>{partSummary(unit.key, unit.content)}</span>
          <UnitStatusChip state="approved" testId={`${testId}-status`} />
          <Button variant="view" onClick={() => setOpen(true)} data-testid={`${testId}-open`}>
            Mở
          </Button>
        </div>
      </Card>
    );
  }

  return (
    <Card data-testid={testId}>
      <div className={researchStyles.body} id={testId}>
        <div className={researchStyles.head}>
          <span className={styles.unitNo}>{unit.number}</span>
          <h3>{unit.title}</h3>
          <UnitStatusChip state={running ? "running" : unit.status} testId={`${testId}-status`} />
          {unit.stale && (
            <span className={`${styles.tag} ${styles.tagWarn}`} data-testid={`${testId}-stale`}>
              cần làm lại: {unit.stale_sources.join(", ")} đã đổi
            </span>
          )}
          <span className={researchStyles.grow} />
          {approved && (
            <Button variant="ghost" onClick={() => {
                setOpen(false);
                onFold();
              }}
              data-testid={`${testId}-fold`}>
              Thu gọn
            </Button>
          )}
        </div>
        <div>
          <span className={researchStyles.hint}>AI đọc: </span>
          {unit.reads.map((r) => (
            <span key={r} className={styles.tag}>
              {r}
            </span>
          ))}
        </div>
        {unit.written && unit.content ? (
          <PartView
            schema={PART_SCHEMAS[unit.key]}
            content={unit.content}
            mine={unit.creator_fields}
          />
        ) : (
          <p className={researchStyles.hint} data-testid={`${testId}-empty`}>
            Chưa viết. Bấm “Viết bằng AI” hoặc “Sửa tay” để viết phần này.
          </p>
        )}
        {unit.problems.length > 0 && (
          <ul className={styles.problems} role="status" data-testid={`${testId}-problems`}>
            {unit.problems.map((p) => (
              <li key={p}>{p}</li>
            ))}
          </ul>
        )}
        <div className={researchStyles.actionsEnd}>
          <Button
            variant="ghost"
            onClick={() => setEditing(true)}
            disabled={disabled}
            data-testid={`${testId}-edit`}
          >
            Sửa tay
          </Button>
          {approved ? (
            <Button
              variant="ghost"
              onClick={() => void concept.unapprove(unit.key)}
              disabled={disabled}
              data-testid={`${testId}-unapprove`}
            >
              Bỏ duyệt
            </Button>
          ) : (
            <>
              <Button
                variant="run"
                onClick={() => setRedoing(true)}
                disabled={disabled}
                data-testid={`${testId}-ai`}
              >
                {unit.written ? "Làm lại kèm ghi chú" : "Viết bằng AI"}
              </Button>
              <Button
                variant="primary"
                onClick={() => {
                  setOpen(false);
                  onFold();
                  void concept.approve(unit.key);
                }}
                disabled={disabled || !unit.written || unit.problems.length > 0}
                data-testid={`${testId}-approve`}
              >
                {unit.stale ? `Giữ và duyệt phần ${unit.number}` : `Duyệt phần ${unit.number}`}
              </Button>
            </>
          )}
        </div>
      </div>
      <PartEditDialog
        open={editing}
        onClose={() => setEditing(false)}
        title={`Sửa tay phần ${unit.number} · ${unit.title}`}
        schema={PART_SCHEMAS[unit.key]}
        content={unit.content}
        approved={approved}
        onSave={(fields) => concept.edit(unit.key, fields)}
        testId={`${testId}-edit-dialog`}
      />
      <ConceptAiDialog
        open={redoing}
        onClose={() => setRedoing(false)}
        title={`${unit.written ? "Làm lại" : "Viết"} phần ${unit.number} · ${unit.title}`}
        impact={
          <p className={researchStyles.hint}>
            Các phần khác giữ nguyên. Trường bạn đã sửa tay giữ nguyên trừ khi ghi chú yêu cầu đổi.
            Phần này về “chờ duyệt”.
          </p>
        }
        onSystem={(note) => void concept.generateUnit(unit.key, note)}
        external={{
          run: (note, external) =>
            generateVisualConceptUnit(projectId, unit.key, note, { external }),
          onDone: concept.reload,
        }}
        testId={`${testId}-ai-dialog`}
      />
    </Card>
  );
}
