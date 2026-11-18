import { useEffect, useState } from "react";
import {
  generateSequenceUnit,
  type SequenceBeat,
  type SequenceUnit,
  type VisualSequenceState,
} from "../../api/client";
import type { VisualSequenceControls } from "../../hooks/useVisualSequence";
import researchStyles from "../research/research.module.css";
import { Button, Card, Dialog, FormField, TextArea } from "../ui";
import { UnitStatusChip } from "../units/UnitStatusChip";
import { ConceptAiDialog } from "../visual/ConceptAiDialog";
import foldStyles from "../visual/visual.module.css";
import planStyles from "../visualplan/plan.module.css";
import { METHOD_LABELS, refsLabel, seconds } from "../visualplan/planModel";
import { SequenceItemDialog } from "./SequenceItemDialog";
import styles from "./sequence.module.css";
import { approvable, motifName, motifSlot, motifsReady, tempoLabel } from "./sequenceModel";

/** Whether the system's AI is writing this unit now (alone, in its beat, or among the empty units). */
function unitRunning(
  sequence: VisualSequenceControls,
  beat: SequenceBeat,
  unit: SequenceUnit,
): boolean {
  const r = sequence.running;
  if (r === "all") return unit.item === null;
  if (r === null || typeof r !== "object") return false;
  return "unit" in r ? r.unit === unit.unit_id : r.beat === beat.key;
}

interface UnitCardProps {
  projectId: string;
  state: VisualSequenceState;
  beat: SequenceBeat;
  unit: SequenceUnit;
  sequence: VisualSequenceControls;
  readOnly: boolean;
}

/**
 * One unit of the plan with its sequence: on the left what step 9 planned, in
 * the middle the transition into it, on the right its pacing, purpose in the
 * flow and motifs; then the actions on it.
 */
function SequenceUnitCard({ projectId, state, beat, unit, sequence, readOnly }: UnitCardProps) {
  const [editing, setEditing] = useState(false);
  const [redoing, setRedoing] = useState(false);
  // An approved unit folds to one line; the Creator opens it to read or change it.
  const [expanded, setExpanded] = useState(false);
  const it = unit.item;
  const approved = it?.status === "approved";
  // Editing and approving stay open while the AI works on another beat; a new AI run waits for the one in flight.
  const running = unitRunning(sequence, beat, unit);
  const disabled = readOnly || sequence.busy || running;
  const aiDisabled = readOnly || sequence.busy || sequence.running !== null;
  const testId = `seq-unit-${unit.unit_id}`;
  const tr = it?.transition_from_previous ?? null;
  const approveBlock = beat.stale
    ? "Beat đã lỗi thời so với kế hoạch hình, xác nhận hoặc viết lại beat trước"
    : unit.problems.join("; ");

  if (approved && !expanded) {
    return (
      <article className={planStyles.unit} id={testId} data-testid={testId}>
        <div className={foldStyles.folded}>
          <span className={planStyles.uid}>{unit.unit_id}</span>
          <span className={foldStyles.summary}>
            {it.pacing.name} · {it.purpose_in_flow}
          </span>
          <UnitStatusChip state="approved" testId={`${testId}-status`} />
          <Button variant="view" onClick={() => setExpanded(true)} data-testid={`${testId}-open`}>
            Mở
          </Button>
        </div>
      </article>
    );
  }

  return (
    <article
      className={`${planStyles.unit} ${unit.problems.length > 0 && !approved ? planStyles.unitWarn : ""}`}
      id={testId}
      data-testid={testId}
    >
      <div className={planStyles.unitHead}>
        <span>
          <span className={planStyles.uid}>{unit.unit_id}</span>{" "}
          <span className={researchStyles.hint}>
            #{unit.order} · {refsLabel(unit.script_refs)} · {seconds(unit.planned_s)}
          </span>
        </span>
        {running ? (
          <UnitStatusChip state="running" testId={`${testId}-status`} />
        ) : (
          it && (
            <UnitStatusChip state={approved ? "approved" : "pending"} testId={`${testId}-status`} />
          )
        )}
      </div>
      <div className={styles.cols3}>
        <div className={`${planStyles.col} ${planStyles.colScript}`}>
          <span className={planStyles.lbl}>Mục tiêu truyền đạt (bước 9)</span>
          <span>{unit.communication_goal}</span>
          <span className={planStyles.lbl}>Cách hiển thị</span>
          <span className={researchStyles.hint}>
            {METHOD_LABELS[unit.visual_approach.method] ?? unit.visual_approach.method} ·{" "}
            {unit.visual_approach.description}
          </span>
        </div>
        <div className={planStyles.col}>
          <span className={planStyles.lbl}>
            Chuyển cảnh vào{unit.previous_unit ? ` · từ ${unit.previous_unit}` : ""}
          </span>
          <div className={styles.trbox} data-testid={`${testId}-transition`}>
            {unit.previous_unit === "" ? (
              <span className={researchStyles.hint}>Unit đầu video, không có chuyển cảnh vào.</span>
            ) : tr ? (
              <>
                <b>{tr.type}</b>
                <span>{tr.description}</span>
                {tr.via_motif && (
                  <span className={researchStyles.hint}>Nối bằng {tr.via_motif}</span>
                )}
              </>
            ) : (
              <span className={researchStyles.hint}>Chưa có.</span>
            )}
          </div>
        </div>
        <div className={planStyles.col}>
          {it ? (
            <>
              <span className={planStyles.lbl}>Nhịp · {tempoLabel(it.pacing.tempo)}</span>
              <span>
                <b>{it.pacing.name}</b> · {it.pacing.description}
              </span>
              <span className={planStyles.lbl}>Vai trò trong mạch</span>
              <span>{it.purpose_in_flow}</span>
              <span className={planStyles.lbl}>Motif</span>
              <div className={planStyles.tags}>
                {it.motif_refs.length === 0 && (
                  <span className={researchStyles.hint}>Không có</span>
                )}
                {it.motif_refs.map((m) => (
                  <span key={m} className={`${researchStyles.chip} ${styles.motifTag}`}>
                    <i
                      className={`${styles.dot} ${styles[`c${motifSlot(state, m)}`] ?? ""}`}
                      aria-hidden="true"
                    />
                    {m} · {motifName(state, m)}
                  </span>
                ))}
              </div>
            </>
          ) : (
            <span className={researchStyles.hint} data-testid={`${testId}-empty`}>
              Chưa có nhịp, chuyển cảnh và motif.
            </span>
          )}
        </div>
      </div>
      {unit.problems.length > 0 && (
        <ul className={planStyles.problems} role="status" data-testid={`${testId}-problems`}>
          {unit.problems.map((p) => (
            <li key={p}>{p}</li>
          ))}
        </ul>
      )}
      <div className={planStyles.unitFoot}>
        <Button
          variant="ghost"
          onClick={() => setEditing(true)}
          disabled={disabled}
          data-testid={`${testId}-edit`}
        >
          {it ? "Sửa" : "Viết tay"}
        </Button>
        {approved ? (
          <>
            <Button
              variant="ghost"
              onClick={() => setExpanded(false)}
              data-testid={`${testId}-fold`}
            >
              Thu gọn
            </Button>
            <Button
              variant="ghost"
              onClick={() => void sequence.unapprove(unit.unit_id)}
              disabled={disabled}
              data-testid={`${testId}-unapprove`}
            >
              Bỏ duyệt
            </Button>
          </>
        ) : (
          <>
            <Button
              variant="run"
              onClick={() => setRedoing(true)}
              disabled={aiDisabled || !motifsReady(state)}
              title={motifsReady(state) ? undefined : "Duyệt motif trước khi AI viết chuỗi"}
              data-testid={`${testId}-redo`}
            >
              {it ? "Làm lại kèm ghi chú" : "Viết chuỗi unit này"}
            </Button>
            {it && (
              <Button
                variant="primary"
                onClick={() => {
                  setExpanded(false);
                  void sequence.approve(unit.unit_id);
                }}
                disabled={disabled || approveBlock !== ""}
                title={approveBlock || undefined}
                data-testid={`${testId}-approve`}
              >
                Duyệt
              </Button>
            )}
          </>
        )}
      </div>

      <SequenceItemDialog
        open={editing}
        onClose={() => setEditing(false)}
        state={state}
        unit={unit}
        onSave={(fields) => sequence.edit(unit.unit_id, fields)}
      />
      <ConceptAiDialog
        open={redoing}
        onClose={() => setRedoing(false)}
        title={it ? `Làm lại chuỗi ${unit.unit_id} kèm ghi chú` : `Viết chuỗi ${unit.unit_id}`}
        impact={
          <p className={researchStyles.hint}>
            AI viết riêng {unit.unit_id} trong một lượt, đọc các unit khác của beat, unit liền trước
            và liền sau. Trường bạn đã sửa tay được giữ; unit về “Chờ duyệt”, các unit khác giữ
            nguyên.
          </p>
        }
        onSystem={(note) => void sequence.generateUnit(unit.unit_id, note)}
        external={{
          run: (note, external) =>
            generateSequenceUnit(projectId, unit.unit_id, note, { external }),
          onDone: sequence.reload,
        }}
        testId={`${testId}-redo-dialog`}
      />
    </article>
  );
}

interface BeatProps {
  projectId: string;
  state: VisualSequenceState;
  beat: SequenceBeat;
  sequence: VisualSequenceControls;
  readOnly: boolean;
}

function SequenceBeatBlock({ projectId, state, beat, sequence, readOnly }: BeatProps) {
  const [open, setOpen] = useState(false);
  // A beat whose every unit is approved folds to its head line; it folds again each time it becomes approved.
  const [expanded, setExpanded] = useState(false);
  useEffect(() => {
    if (beat.approved) setExpanded(false);
  }, [beat.approved]);
  const disabled = readOnly || sequence.busy || sequence.running !== null;
  const ready = motifsReady(state);
  const running =
    sequence.running === "all"
      ? beat.units.some((u) => u.item === null)
      : typeof sequence.running === "object" &&
        sequence.running !== null &&
        "beat" in sequence.running &&
        sequence.running.beat === beat.key;
  const [note, setNote] = useState("");
  const folded = beat.approved && !expanded && !running;

  return (
    <div
      className={planStyles.beat}
      id={`seq-beat-${beat.key}`}
      data-testid={`seq-beat-${beat.key}`}
    >
      <div className={planStyles.beatHead}>
        <span className={planStyles.beatTitle}>
          {beat.key} · {beat.title}
        </span>
        <span className={planStyles.beatMeta}>
          <span data-testid={`seq-beat-${beat.key}-meta`}>
            {beat.units.length} unit{beat.pace ? ` · nhịp bước 8: ${beat.pace}` : ""}
            {!beat.written ? " · chưa có chuỗi" : ""}
          </span>
          {running ? (
            <UnitStatusChip state="running" />
          ) : (
            beat.written && <UnitStatusChip state={beat.approved ? "approved" : "pending"} />
          )}
          {beat.stale && (
            <>
              <span className={researchStyles.chip} data-testid={`seq-beat-${beat.key}-stale`}>
                lỗi thời: {beat.stale_sources.join(", ")} đã đổi
              </span>
              <Button
                variant="primary"
                onClick={() => void sequence.confirmBeat(beat.key)}
                disabled={disabled}
                data-testid={`seq-beat-${beat.key}-confirm`}
              >
                Vẫn hợp, xác nhận
              </Button>
            </>
          )}
          {beat.approved && (
            <Button
              variant={folded ? "view" : "ghost"}
              onClick={() => setExpanded(folded)}
              data-testid={`seq-beat-${beat.key}-${folded ? "open" : "fold"}`}
            >
              {folded ? `Mở ${beat.units.length} unit` : "Thu gọn"}
            </Button>
          )}
          <Button
            variant="run"
            onClick={() => setOpen(true)}
            disabled={disabled || !ready}
            title={ready ? undefined : "Duyệt motif trước khi AI viết chuỗi"}
            data-testid={`seq-beat-${beat.key}-generate`}
          >
            {beat.written ? "Viết lại beat này" : "Viết chuỗi beat này"}
          </Button>
        </span>
      </div>
      {!folded &&
        beat.units.map((u) => (
          <SequenceUnitCard
            key={u.unit_id}
            projectId={projectId}
            state={state}
            beat={beat}
            unit={u}
            sequence={sequence}
            readOnly={readOnly}
          />
        ))}
      <Dialog
        open={open}
        onClose={() => setOpen(false)}
        title={`${beat.written ? "Viết lại" : "Viết"} chuỗi beat ${beat.key}`}
        hasInput
        testId={`seq-beat-${beat.key}-dialog`}
        footer={
          <>
            <Button variant="ghost" onClick={() => setOpen(false)}>
              Huỷ
            </Button>
            <Button
              variant="run"
              onClick={() => {
                const n = note.trim();
                setOpen(false);
                setNote("");
                void sequence.generateBeat(beat.key, n);
              }}
              data-testid={`seq-beat-${beat.key}-confirm-run`}
            >
              Chạy {beat.units.length} unit
            </Button>
          </>
        }
      >
        <p>
          AI hệ thống viết lần lượt {beat.units.length} unit của beat (
          {beat.units.map((u) => u.unit_id).join(", ")}), mỗi unit một lượt; unit sau đọc unit vừa
          viết. Trường bạn đã sửa tay được giữ; mọi unit của beat về “Chờ duyệt”.
        </p>
        <p className={researchStyles.hint}>
          Muốn dùng AI ngoài hoặc DeepSeek: bấm “Làm lại kèm ghi chú” trên từng unit, hoặc “Chạy tất
          cả bằng DeepSeek” ở mục 3.
        </p>
        <FormField label="Ghi chú cho AI (không bắt buộc)">
          <TextArea
            value={note}
            onChange={(e) => setNote(e.target.value)}
            rows={3}
            data-testid={`seq-beat-${beat.key}-note`}
          />
        </FormField>
      </Dialog>
    </div>
  );
}

interface SequenceBeatsCardProps {
  projectId: string;
  state: VisualSequenceState;
  sequence: VisualSequenceControls;
  readOnly: boolean;
}

/** Section 5 of step 10: the units grouped by beat, with approving the rest at the end. */
export function SequenceBeatsCard({
  projectId,
  state,
  sequence,
  readOnly,
}: SequenceBeatsCardProps) {
  // Approving the rest stays open during an AI run: the beats being written have no item to approve yet.
  const disabled = readOnly || sequence.busy;
  const ready = approvable(state);
  return (
    <Card data-testid="seq-beats">
      <div className={researchStyles.body}>
        <div className={researchStyles.head}>
          <h3>5. Chuỗi theo beat</h3>
          <span className={researchStyles.hint}>
            Trái: unit từ bước 9 · giữa: chuyển cảnh vào · phải: nhịp, vai trò và motif
          </span>
        </div>
        {state.beats.map((b) => (
          <SequenceBeatBlock
            key={b.key}
            projectId={projectId}
            state={state}
            beat={b}
            sequence={sequence}
            readOnly={readOnly}
          />
        ))}
        <div className={researchStyles.actionsEnd}>
          <Button
            variant="primary"
            onClick={() => void sequence.approveAll()}
            disabled={disabled || ready.length === 0}
            data-testid="seq-approve-all"
          >
            Duyệt tất cả còn lại{ready.length > 0 ? ` · ${ready.length}` : ""}
          </Button>
        </div>
      </div>
    </Card>
  );
}
